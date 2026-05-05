package host

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/callum-baillie/drive-agent/internal/config"
	"github.com/callum-baillie/drive-agent/internal/filesystem"
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
