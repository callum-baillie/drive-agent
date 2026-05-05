package host

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/callum-baillie/drive-agent/internal/config"
	"github.com/callum-baillie/drive-agent/internal/filesystem"
	"github.com/callum-baillie/drive-agent/internal/packages/catalog"
	"github.com/callum-baillie/drive-agent/internal/packages/providers"
	"github.com/callum-baillie/drive-agent/internal/shell"
)

func TestLoadProfileFromFileParsesHostProfile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "profile.json")
	writeTestProfile(t, path, "external-drive", "bind-mounts")

	profile, err := loadProfileFromFile(path)
	if err != nil {
		t.Fatalf("loadProfileFromFile returned error: %v", err)
	}
	if profile.ProfileName != "test-profile" {
		t.Fatalf("profileName = %q, want test-profile", profile.ProfileName)
	}
	if profile.Caches.NpmCachePath != "/Volumes/Test Drive/Caches/npm" {
		t.Fatalf("npm cache path not parsed: %q", profile.Caches.NpmCachePath)
	}
	if profile.Docker.ExternalDataRoot != "/Volumes/Test Drive/DevData/containers" {
		t.Fatalf("docker data root not parsed: %q", profile.Docker.ExternalDataRoot)
	}
}

func TestGeneratedMacMiniProfileValidity(t *testing.T) {
	path := "/Volumes/ExternalSSD/.drive-agent/config/host-profiles/mac-mini.json"
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skip("mac-mini host profile has not been generated on this host")
	}
	profile, err := loadProfileFromFile(path)
	if err != nil {
		t.Fatalf("generated profile failed to parse: %v", err)
	}
	if profile.ProfileName != "mac-mini" {
		t.Fatalf("profileName = %q, want mac-mini", profile.ProfileName)
	}
	if profile.Target.OS != "macos" || profile.Target.Arch != "arm64" {
		t.Fatalf("target = %+v, want macos/arm64", profile.Target)
	}
	if len(profile.Packages.Include) == 0 {
		t.Fatal("generated profile has no included packages")
	}
}

func TestCacheModeParsing(t *testing.T) {
	tests := map[string]string{
		"":                   cacheModePrompt,
		"prompt":             cacheModePrompt,
		"host-local":         cacheModeHostLocal,
		"local":              cacheModeHostLocal,
		"external-drive":     cacheModeExternal,
		"external":           cacheModeExternal,
		"disabled":           cacheModeDisabled,
		"disabled/no-change": cacheModeDisabled,
		"no-change":          cacheModeDisabled,
	}
	for input, want := range tests {
		got, err := normalizeCacheMode(input)
		if err != nil {
			t.Fatalf("normalizeCacheMode(%q) returned error: %v", input, err)
		}
		if got != want {
			t.Fatalf("normalizeCacheMode(%q) = %q, want %q", input, got, want)
		}
	}
	if _, err := normalizeCacheMode("surprise"); err == nil {
		t.Fatal("expected error for invalid cache mode")
	}
}

func TestExternalCachePlan(t *testing.T) {
	profile := testProfile("external-drive", "bind-mounts")
	actions, shellOptions, err := buildCachePlan(profile, "/Volumes/Test Drive", "external-drive")
	if err != nil {
		t.Fatalf("buildCachePlan returned error: %v", err)
	}
	assertActionContains(t, actions, "Configure npm cache", "npm config set cache '/Volumes/Test Drive/Caches/npm'")
	assertActionContains(t, actions, "Configure pnpm store", "pnpm config set store-dir '/Volumes/Test Drive/Caches/pnpm'")
	assertActionContains(t, actions, "Create Homebrew cache directory", "mkdir -p '/Volumes/Test Drive/Caches/homebrew'")
	if shellOptions.NpmCachePath != "/Volumes/Test Drive/Caches/npm" {
		t.Fatalf("npm shell option = %q", shellOptions.NpmCachePath)
	}
	if shellOptions.HomebrewCachePath != "/Volumes/Test Drive/Caches/homebrew" {
		t.Fatalf("homebrew shell option = %q", shellOptions.HomebrewCachePath)
	}
}

func TestDisabledCachePlanDoesNotConfigureCaches(t *testing.T) {
	profile := testProfile("disabled", "default")
	actions, shellOptions, err := buildCachePlan(profile, "/Volumes/Test Drive", "disabled")
	if err != nil {
		t.Fatalf("buildCachePlan returned error: %v", err)
	}
	if len(actions) != 1 || actions[0].RequiresRun {
		t.Fatalf("disabled cache mode should not produce mutating actions: %+v", actions)
	}
	if shellOptions != (shell.ShellBlockOptions{}) {
		t.Fatalf("disabled cache mode should not produce shell options: %+v", shellOptions)
	}
}

func TestDockerBindMountRootPlan(t *testing.T) {
	profile := testProfile("host-local", "bind-mounts")
	actions, shellOptions, err := buildDockerPlan(profile, "/Volumes/Test Drive", "bind-mounts")
	if err != nil {
		t.Fatalf("buildDockerPlan returned error: %v", err)
	}
	assertActionContains(t, actions, "Create container data root", "mkdir -p '/Volumes/Test Drive/DevData/containers'")
	assertActionContains(t, actions, "Create Docker build cache root", "mkdir -p '/Volumes/Test Drive/DevData/docker-build-cache'")
	if shellOptions.ContainerDataPath != "/Volumes/Test Drive/DevData/containers" {
		t.Fatalf("container data shell option = %q", shellOptions.ContainerDataPath)
	}
}

func TestDockerModeParsingIncludesDaemonGuidance(t *testing.T) {
	tests := []string{"daemon", "daemon-data-root", "daemon-guidance"}
	for _, input := range tests {
		got, err := normalizeDockerMode(input)
		if err != nil {
			t.Fatalf("normalizeDockerMode(%q) returned error: %v", input, err)
		}
		if got != dockerModeDaemon {
			t.Fatalf("normalizeDockerMode(%q) = %q, want %q", input, got, dockerModeDaemon)
		}
	}
}

func TestCaskAppBundleDetectionTreatsManualInstallAsInstalled(t *testing.T) {
	profile := packageTestProfile("vscode")
	cat := packageTestCatalog(vscodeTestPackage(false))
	registry := testRegistry("homebrew-cask")

	plan := buildPackagePlanWithOptions(profile, cat, registry, packagePlanOptions{
		RunCheck:   func(string) bool { return false },
		PathExists: func(path string) bool { return path == "/Applications/Visual Studio Code.app" },
		HomeDir:    "/Users/example",
	})

	action := requirePackageAction(t, plan, "vscode")
	if !action.Installed {
		t.Fatalf("vscode should be treated as installed: %+v", action)
	}
	if action.InstalledDetail != "/Applications/Visual Studio Code.app" {
		t.Fatalf("installed detail = %q", action.InstalledDetail)
	}
}

func TestAppBundleDetectionExpandsHome(t *testing.T) {
	pkg := vscodeTestPackage(false)
	pkg.Check.AppBundles = []string{"~/Applications/Visual Studio Code.app"}
	status := packageInstalledStatus(&pkg, packagePlanOptions{
		RunCheck:   func(string) bool { return false },
		PathExists: func(path string) bool { return path == "/Users/example/Applications/Visual Studio Code.app" },
		HomeDir:    "/Users/example",
	})
	if !status.Installed {
		t.Fatal("expected app bundle under home to be detected")
	}
	if status.Detail != "/Users/example/Applications/Visual Studio Code.app" {
		t.Fatalf("detail = %q", status.Detail)
	}
}

func TestForceChangesInstallDecision(t *testing.T) {
	profile := packageTestProfile("vscode")
	cat := packageTestCatalog(vscodeTestPackage(false))
	registry := testRegistry("homebrew-cask")

	plan := buildPackagePlanWithOptions(profile, cat, registry, packagePlanOptions{
		Force:      true,
		RunCheck:   func(string) bool { return false },
		PathExists: func(string) bool { return true },
		HomeDir:    "/Users/example",
	})

	action := requirePackageAction(t, plan, "vscode")
	if action.Installed {
		t.Fatalf("--force should not mark vscode as installed: %+v", action)
	}
	if action.Command != "brew install --cask visual-studio-code" {
		t.Fatalf("command = %q", action.Command)
	}
}

func TestYesSkipsPromptsForNormalPackages(t *testing.T) {
	confirmCalls := 0
	runCalls := 0
	plan := packagePlan{Actions: []packageAction{{
		ID:            "biome",
		Name:          "Biome",
		ManagerID:     "npm",
		PackageName:   "@biomejs/biome",
		Command:       "npm install -g @biomejs/biome",
		InstallGlobal: true,
	}}}

	if err := runProfileSetupPlan(plan, nil, nil, setupRunOptions{
		AutoYes: true,
		Confirm: func(string, bool) bool {
			confirmCalls++
			return true
		},
		CommandRunner: func(string, ...string) (string, error) {
			runCalls++
			return "", nil
		},
	}); err != nil {
		t.Fatalf("runProfileSetupPlan returned error: %v", err)
	}
	if confirmCalls != 0 {
		t.Fatalf("--yes should skip prompts; confirm calls = %d", confirmCalls)
	}
	if runCalls != 1 {
		t.Fatalf("command runner calls = %d, want 1", runCalls)
	}
}

func TestRequiresExplicitApprovalSkippedUnlessIncluded(t *testing.T) {
	profile := packageTestProfile("playwright-cli")
	cat := packageTestCatalog(playwrightTestPackage())
	registry := testRegistry("npm")

	defaultPlan := buildPackagePlanWithOptions(profile, cat, registry, packagePlanOptions{
		RunCheck: func(string) bool { return false },
	})
	defaultAction := requirePackageAction(t, defaultPlan, "playwright-cli")
	if defaultAction.SkipReason != "requires explicit approval" {
		t.Fatalf("skip reason = %q", defaultAction.SkipReason)
	}

	includedPlan := buildPackagePlanWithOptions(profile, cat, registry, packagePlanOptions{
		IncludeExplicit: true,
		RunCheck:        func(string) bool { return false },
	})
	includedAction := requirePackageAction(t, includedPlan, "playwright-cli")
	if includedAction.SkipReason != "" {
		t.Fatalf("explicit package should be included with flag: %+v", includedAction)
	}
	if includedAction.Command != "npm install -g playwright" {
		t.Fatalf("command = %q", includedAction.Command)
	}
}

func TestTurborepoUsesNpmNotHomebrew(t *testing.T) {
	profile := packageTestProfile("turbo")
	cat := packageTestCatalog(turboTestPackage())
	registry := testRegistry("homebrew", "npm")
	var checks []string

	plan := buildPackagePlanWithOptions(profile, cat, registry, packagePlanOptions{
		RunCheck: func(command string) bool {
			checks = append(checks, command)
			return false
		},
		PathExists: func(string) bool { return false },
	})

	action := requirePackageAction(t, plan, "turbo")
	if action.ManagerID != "npm" {
		t.Fatalf("manager = %q, want npm; action=%+v", action.ManagerID, action)
	}
	if action.Command != "npm install -g turbo" {
		t.Fatalf("command = %q, want npm install -g turbo", action.Command)
	}
	if strings.Contains(action.Command, "brew install") {
		t.Fatalf("turbo should not use Homebrew: %q", action.Command)
	}
	if len(checks) != 1 || checks[0] != "turbo --version" {
		t.Fatalf("checks = %#v, want turbo --version", checks)
	}
}

func TestInstallPromptFormatting(t *testing.T) {
	tests := []struct {
		action packageAction
		want   string
	}{
		{packageAction{Name: "Visual Studio Code", ManagerID: "homebrew-cask"}, "[homebrew-cask] Install Visual Studio Code?"},
		{packageAction{Name: "ripgrep", ManagerID: "homebrew"}, "[homebrew] Install ripgrep?"},
		{packageAction{Name: "Biome", ManagerID: "npm", InstallGlobal: true}, "[npm] Install Biome globally?"},
		{packageAction{Name: "tsx", ManagerID: "pnpm", InstallGlobal: true}, "[pnpm] Install tsx globally?"},
		{packageAction{Name: "gopls", ManagerID: "go-install"}, "[go-install] Install gopls?"},
	}
	for _, tt := range tests {
		if got := installPrompt(tt.action); got != tt.want {
			t.Fatalf("installPrompt(%+v) = %q, want %q", tt.action, got, tt.want)
		}
	}
}

func TestFailedInstallContinuesInYesModeAndSummarizes(t *testing.T) {
	var commands []string
	plan := packagePlan{Actions: []packageAction{
		{ID: "vscode", Name: "Visual Studio Code", ManagerID: "homebrew-cask", Command: "brew install --cask visual-studio-code"},
		{ID: "ripgrep", Name: "ripgrep", ManagerID: "homebrew", Command: "brew install ripgrep"},
	}}

	err := runProfileSetupPlan(plan, nil, nil, setupRunOptions{
		AutoYes: true,
		CommandRunner: func(name string, args ...string) (string, error) {
			commands = append(commands, name+" "+strings.Join(args, " "))
			if strings.Contains(strings.Join(args, " "), "visual-studio-code") {
				return "Error: It seems there is already an App at '/Applications/Visual Studio Code.app'.", errors.New("failed")
			}
			return "", nil
		},
	})
	if err == nil {
		t.Fatal("expected failed install summary error")
	}
	if len(commands) != 2 {
		t.Fatalf("expected yes mode to continue after failure, ran %d commands: %v", len(commands), commands)
	}
}

func TestDryRunDoesNotMutateHostConfig(t *testing.T) {
	driveRoot := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/zsh")
	filesystem.SetDriveRootOverride(driveRoot)
	t.Cleanup(func() { filesystem.SetDriveRootOverride("") })

	mustMkdir(t, filepath.Join(driveRoot, ".drive-agent", "config", "host-profiles"))
	mustMkdir(t, filepath.Join(driveRoot, ".drive-agent", "catalog"))
	mustMkdir(t, filepath.Join(driveRoot, ".drive-agent", "state", "hosts"))
	if err := os.WriteFile(filepath.Join(driveRoot, ".drive-agent", "DRIVE_AGENT_ROOT"), []byte(driveRoot+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	writeTestProfile(t, filepath.Join(driveRoot, ".drive-agent", "config", "host-profiles", "test-profile.json"), "external-drive", "bind-mounts")
	writeMinimalCatalog(t, filepath.Join(driveRoot, ".drive-agent", "catalog", "packages.catalog.json"))

	cmd := newSetupCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--profile", "test-profile", "--dry-run"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("dry-run host setup returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".zshrc")); !os.IsNotExist(err) {
		t.Fatalf("dry-run created shell config: %v", err)
	}
	if _, err := os.Stat(filepath.Join(driveRoot, "Caches")); !os.IsNotExist(err) {
		t.Fatalf("dry-run created cache directories: %v", err)
	}
	entries, err := os.ReadDir(filepath.Join(driveRoot, ".drive-agent", "state", "hosts"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("dry-run wrote host state entries: %d", len(entries))
	}
}

func TestApplyStorageShellBlockDryRunDoesNotWrite(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), ".zshrc")
	options := shell.ShellBlockOptions{
		HomebrewCachePath: "/Volumes/Test Drive/Caches/homebrew",
		BunCachePath:      "/Volumes/Test Drive/Caches/bun",
		ContainerDataPath: "/Volumes/Test Drive/DevData/containers",
		DockerCachePath:   "/Volumes/Test Drive/DevData/docker-build-cache",
	}

	if err := applyStorageShellBlock(configPath, options, true, true); err != nil {
		t.Fatalf("applyStorageShellBlock dry-run returned error: %v", err)
	}
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Fatalf("dry-run should not write shell config; stat err=%v", err)
	}
}

func TestApplyStorageShellBlockYesWritesExpectedExports(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), ".zshrc")
	if err := os.WriteFile(configPath, []byte("# existing\n"), 0644); err != nil {
		t.Fatal(err)
	}
	options := shell.ShellBlockOptions{
		HomebrewCachePath: "/Volumes/Test Drive/Caches/homebrew",
		BunCachePath:      "/Volumes/Test Drive/Caches/bun",
		ContainerDataPath: "/Volumes/Test Drive/DevData/containers",
		DockerCachePath:   "/Volumes/Test Drive/DevData/docker-build-cache",
	}

	if err := applyStorageShellBlock(configPath, options, false, true); err != nil {
		t.Fatalf("applyStorageShellBlock returned error: %v", err)
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	expected := []string{
		`export HOMEBREW_CACHE='/Volumes/Test Drive/Caches/homebrew'`,
		`export BUN_INSTALL_CACHE_DIR='/Volumes/Test Drive/Caches/bun'`,
		`export DRIVE_AGENT_CONTAINER_DATA='/Volumes/Test Drive/DevData/containers'`,
		`export DRIVE_AGENT_DOCKER_BUILD_CACHE='/Volumes/Test Drive/DevData/docker-build-cache'`,
	}
	for _, want := range expected {
		if !strings.Contains(content, want) {
			t.Fatalf("missing %q in:\n%s", want, content)
		}
	}
	if strings.Count(content, ">>> drive-agent storage >>>") != 1 {
		t.Fatalf("storage block count = %d, want 1\n%s", strings.Count(content, ">>> drive-agent storage >>>"), content)
	}

	if err := applyStorageShellBlock(configPath, options, false, true); err != nil {
		t.Fatalf("second applyStorageShellBlock returned error: %v", err)
	}
	data, err = os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(data), ">>> drive-agent storage >>>") != 1 {
		t.Fatalf("repeated setup duplicated storage block:\n%s", string(data))
	}
}

func testProfile(cacheMode, dockerMode string) *config.HostProfile {
	return &config.HostProfile{
		SchemaVersion: 1,
		ProfileName:   "test-profile",
		Caches: config.ProfileCaches{
			Mode:              cacheMode,
			ExternalDriveRoot: "/Volumes/Test Drive",
			NpmCachePath:      "/Volumes/Test Drive/Caches/npm",
			PnpmStorePath:     "/Volumes/Test Drive/Caches/pnpm",
			BunCachePath:      "/Volumes/Test Drive/Caches/bun",
			HomebrewCachePath: "/Volumes/Test Drive/Caches/homebrew",
		},
		Docker: config.ProfileDocker{
			Mode:                   dockerMode,
			ExternalDataRoot:       "/Volumes/Test Drive/DevData/containers",
			ExternalBuildCacheRoot: "/Volumes/Test Drive/DevData/docker-build-cache",
		},
	}
}

func writeTestProfile(t *testing.T, path, cacheMode, dockerMode string) {
	t.Helper()
	profile := testProfile(cacheMode, dockerMode)
	profile.PackageManagers.Preferred = []string{"homebrew", "homebrew-cask", "npm"}
	profile.Packages.Include = []string{"git"}
	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func writeMinimalCatalog(t *testing.T, path string) {
	t.Helper()
	data := []byte(`{"schemaVersion":1,"packages":[{"id":"git","name":"Git","category":"core","description":"Version control","kind":"cli","installPreference":["homebrew"],"install":{"homebrew":{"type":"formula","name":"git"}},"check":{"command":"git --version"}}]}`)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func packageTestProfile(ids ...string) *config.HostProfile {
	profile := testProfile("host-local", "default")
	profile.PackageManagers.Preferred = []string{"homebrew", "homebrew-cask", "npm"}
	profile.Packages.Include = ids
	return profile
}

func packageTestCatalog(pkgs ...catalog.Package) *catalog.Catalog {
	return &catalog.Catalog{SchemaVersion: 1, Packages: pkgs}
}

func vscodeTestPackage(requiresApproval bool) catalog.Package {
	return catalog.Package{
		ID:                "vscode",
		Name:              "Visual Studio Code",
		Category:          "editors",
		Description:       "Code editor",
		Kind:              "gui",
		InstallPreference: []string{"homebrew-cask"},
		Install: map[string]catalog.InstallConfig{
			"homebrew-cask": {Type: "cask", Name: "visual-studio-code"},
		},
		Check: &catalog.CheckConfig{
			Command:    "code --version",
			AppBundles: []string{"/Applications/Visual Studio Code.app", "~/Applications/Visual Studio Code.app"},
		},
		RequiresApproval: requiresApproval,
	}
}

func playwrightTestPackage() catalog.Package {
	return catalog.Package{
		ID:                "playwright-cli",
		Name:              "Playwright CLI",
		Category:          "api-testing",
		Description:       "Optional host-level Playwright CLI",
		Kind:              "cli",
		InstallPreference: []string{"npm"},
		Install: map[string]catalog.InstallConfig{
			"npm": {Global: true, Name: "playwright"},
		},
		Check:            &catalog.CheckConfig{Command: "playwright --version"},
		RequiresApproval: true,
	}
}

func turboTestPackage() catalog.Package {
	return catalog.Package{
		ID:                "turbo",
		Name:              "Turborepo",
		Category:          "javascript",
		Description:       "Monorepo build tool",
		Kind:              "cli",
		InstallPreference: []string{"npm", "pnpm"},
		Install: map[string]catalog.InstallConfig{
			"npm":  {Global: true, Name: "turbo"},
			"pnpm": {Global: true, Name: "turbo"},
		},
		Check: &catalog.CheckConfig{Command: "turbo --version"},
	}
}

func testRegistry(ids ...string) *providers.Registry {
	registry := providers.NewRegistry()
	for _, id := range ids {
		registry.Register(fakeProvider{id: id})
	}
	return registry
}

func requirePackageAction(t *testing.T, plan packagePlan, id string) packageAction {
	t.Helper()
	for _, action := range plan.Actions {
		if action.ID == id {
			return action
		}
	}
	t.Fatalf("missing package action %q in %+v", id, plan.Actions)
	return packageAction{}
}

type fakeProvider struct {
	id string
}

func (p fakeProvider) ID() string            { return p.id }
func (p fakeProvider) Name() string          { return p.id }
func (p fakeProvider) SupportedOS() []string { return []string{"test"} }
func (p fakeProvider) IsAvailable() bool     { return true }
func (p fakeProvider) ManagerPath() string   { return "/bin/" + p.id }
func (p fakeProvider) ManagerVersion() string {
	return "test"
}
func (p fakeProvider) InstallManager(bool) (string, error) { return "", nil }
func (p fakeProvider) IsPackageInstalled(string) bool      { return false }
func (p fakeProvider) InstallPackage(packageName string, dryRun bool) (string, error) {
	switch p.id {
	case "homebrew-cask":
		return "brew install --cask " + packageName, nil
	case "homebrew":
		return "brew install " + packageName, nil
	case "npm":
		return "npm install -g " + packageName, nil
	default:
		return p.id + " install " + packageName, nil
	}
}
func (p fakeProvider) InstallPackages(packages []string, dryRun bool) (string, error) {
	return "", nil
}

func assertActionContains(t *testing.T, actions []setupAction, title, command string) {
	t.Helper()
	for _, action := range actions {
		if action.Title == title && action.Command == command {
			return
		}
	}
	t.Fatalf("missing action %q command %q in %+v", title, command, actions)
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatal(err)
	}
}
