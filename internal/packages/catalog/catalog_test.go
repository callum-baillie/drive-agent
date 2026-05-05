package catalog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadCatalog(t *testing.T) {
	// Find the catalog file relative to test
	// We need to navigate up from internal/packages/catalog to the project root
	wd, _ := os.Getwd()
	catalogPath := filepath.Join(wd, "..", "..", "..", "catalog", "packages.catalog.json")

	cat, err := LoadCatalog(catalogPath)
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}

	if cat.SchemaVersion != 1 {
		t.Errorf("schema version = %d, want 1", cat.SchemaVersion)
	}

	if len(cat.Packages) == 0 {
		t.Fatal("catalog has no packages")
	}

	// Check a known package
	git := cat.GetPackage("git")
	if git == nil {
		t.Fatal("expected 'git' package in catalog")
	}
	if git.Category != "core" {
		t.Errorf("git category = %q, want %q", git.Category, "core")
	}
	if git.Name != "Git" {
		t.Errorf("git name = %q, want %q", git.Name, "Git")
	}

	// Check install config
	brewName := git.GetInstallName("homebrew")
	if brewName != "git" {
		t.Errorf("git brew name = %q, want %q", brewName, "git")
	}

	// Check categories exist
	cats := cat.Categories()
	if len(cats) == 0 {
		t.Error("no categories found")
	}

	// Check a cask package
	cursor := cat.GetPackage("cursor")
	if cursor == nil {
		t.Fatal("expected 'cursor' package in catalog")
	}
	managers := cursor.AvailableOn()
	if len(managers) == 0 {
		t.Error("cursor should have at least one manager")
	}
}

func TestPackageSourceNormalizationCatalogEntries(t *testing.T) {
	wd, _ := os.Getwd()
	catalogPath := filepath.Join(wd, "..", "..", "..", "catalog", "packages.catalog.json")
	cat, err := LoadCatalog(catalogPath)
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}

	tests := []struct {
		id      string
		manager string
		name    string
	}{
		{"cursor", "homebrew-cask", "cursor"},
		{"google-cloud-sdk", "homebrew-cask", "gcloud-cli"},
		{"codex-cli", "npm", "@openai/codex"},
		{"claude-code", "npm", "@anthropic-ai/claude-code"},
		{"checkov", "homebrew", "checkov"},
		{"gopls", "go-install", "golang.org/x/tools/gopls@latest"},
		{"ast-grep", "homebrew", "ast-grep"},
		{"biome", "npm", "@biomejs/biome"},
		{"imagemagick", "homebrew", "imagemagick"},
		{"playwright-cli", "npm", "playwright"},
		{"svgo", "npm", "svgo"},
		{"turbo", "npm", "turbo"},
		{"snyk", "npm", "snyk"},
		{"supabase-cli", "npm", "supabase"},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			pkg := cat.GetPackage(tt.id)
			if pkg == nil {
				t.Fatalf("missing package %q", tt.id)
			}
			if got := pkg.GetInstallName(tt.manager); got != tt.name {
				t.Fatalf("%s via %s = %q, want %q", tt.id, tt.manager, got, tt.name)
			}
		})
	}
}

func TestPackageCheckCommandsForInstallBinaryMismatches(t *testing.T) {
	wd, _ := os.Getwd()
	catalogPath := filepath.Join(wd, "..", "..", "..", "catalog", "packages.catalog.json")
	cat, err := LoadCatalog(catalogPath)
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}

	tests := []struct {
		id      string
		command string
	}{
		{"ripgrep", "rg --version"},
		{"delta", "delta --version"},
		{"ast-grep", "sg --version"},
		{"imagemagick", "magick --version"},
		{"google-cloud-sdk", "gcloud --version"},
		{"vscode", "code --version"},
		{"npm-check-updates", "ncu --version"},
		{"playwright-cli", "playwright --version"},
		{"turbo", "turbo --version"},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			pkg := cat.GetPackage(tt.id)
			if pkg == nil {
				t.Fatalf("missing package %q", tt.id)
			}
			if pkg.Check == nil {
				t.Fatalf("package %q has no check command", tt.id)
			}
			if pkg.Check.Command != tt.command {
				t.Fatalf("%s check command = %q, want %q", tt.id, pkg.Check.Command, tt.command)
			}
		})
	}
}

func TestTurborepoCatalogUsesJavaScriptProvider(t *testing.T) {
	wd, _ := os.Getwd()
	catalogPath := filepath.Join(wd, "..", "..", "..", "catalog", "packages.catalog.json")
	cat, err := LoadCatalog(catalogPath)
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}
	pkg := cat.GetPackage("turbo")
	if pkg == nil {
		t.Fatal("missing turbo package")
	}
	if _, ok := pkg.Install["homebrew"]; ok {
		t.Fatalf("turbo should not have a Homebrew install mapping: %#v", pkg.Install)
	}
	if len(pkg.InstallPreference) == 0 || pkg.InstallPreference[0] != "npm" {
		t.Fatalf("turbo installPreference = %#v, want npm first", pkg.InstallPreference)
	}
	if pkg.GetInstallName("npm") != "turbo" {
		t.Fatalf("turbo npm package = %q, want turbo", pkg.GetInstallName("npm"))
	}
	if pkg.Check == nil || pkg.Check.Command != "turbo --version" {
		t.Fatalf("turbo check = %#v, want turbo --version", pkg.Check)
	}
}

func TestCaskCatalogEntriesIncludeAppBundleChecks(t *testing.T) {
	wd, _ := os.Getwd()
	catalogPath := filepath.Join(wd, "..", "..", "..", "catalog", "packages.catalog.json")
	cat, err := LoadCatalog(catalogPath)
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}

	tests := map[string]string{
		"vscode":               "/Applications/Visual Studio Code.app",
		"cursor":               "/Applications/Cursor.app",
		"chatgpt":              "/Applications/ChatGPT.app",
		"docker":               "/Applications/Docker.app",
		"orbstack":             "/Applications/OrbStack.app",
		"postman":              "/Applications/Postman.app",
		"google-chrome":        "/Applications/Google Chrome.app",
		"firefox":              "/Applications/Firefox.app",
		"obsidian":             "/Applications/Obsidian.app",
		"raycast":              "/Applications/Raycast.app",
		"rectangle":            "/Applications/Rectangle.app",
		"magnet":               "/Applications/Magnet.app",
		"hammerspoon":          "/Applications/Hammerspoon.app",
		"appcleaner":           "/Applications/AppCleaner.app",
		"amphetamine":          "/Applications/Amphetamine.app",
		"omnidisksweeper":      "/Applications/OmniDiskSweeper.app",
		"speedtest":            "/Applications/Speedtest.app",
		"vlc":                  "/Applications/VLC.app",
		"bitwarden":            "/Applications/Bitwarden.app",
		"tableplus":            "/Applications/TablePlus.app",
		"tailscale":            "/Applications/Tailscale.app",
		"rustdesk":             "/Applications/RustDesk.app",
		"yubico-authenticator": "/Applications/Yubico Authenticator.app",
		"balenaetcher":         "/Applications/balenaEtcher.app",
		"sublime-text":         "/Applications/Sublime Text.app",
		"github-desktop":       "/Applications/GitHub Desktop.app",
		"fork":                 "/Applications/Fork.app",
		"warp":                 "/Applications/Warp.app",
		"ghostty":              "/Applications/Ghostty.app",
		"iterm2":               "/Applications/iTerm.app",
		"android-studio":       "/Applications/Android Studio.app",
	}

	for id, want := range tests {
		t.Run(id, func(t *testing.T) {
			pkg := cat.GetPackage(id)
			if pkg == nil {
				t.Fatalf("missing package %q", id)
			}
			if pkg.Check == nil {
				t.Fatalf("package %q has no check config", id)
			}
			if !contains(pkg.Check.AppBundles, want) {
				t.Fatalf("%s appBundles = %#v, want %q", id, pkg.Check.AppBundles, want)
			}
			homePath := "~/Applications/" + strings.TrimPrefix(want, "/Applications/")
			if !contains(pkg.Check.AppBundles, homePath) {
				t.Fatalf("%s appBundles = %#v, want %q", id, pkg.Check.AppBundles, homePath)
			}
		})
	}
}

func TestUnavailableCaskOnlyAppsAreDetectionOnly(t *testing.T) {
	wd, _ := os.Getwd()
	catalogPath := filepath.Join(wd, "..", "..", "..", "catalog", "packages.catalog.json")
	cat, err := LoadCatalog(catalogPath)
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}
	for _, id := range []string{"amphetamine", "magnet", "speedtest"} {
		t.Run(id, func(t *testing.T) {
			pkg := cat.GetPackage(id)
			if pkg == nil {
				t.Fatalf("missing package %q", id)
			}
			if len(pkg.Install) != 0 || len(pkg.InstallPreference) != 0 {
				t.Fatalf("%s should not map to unavailable Homebrew casks: install=%#v preference=%#v", id, pkg.Install, pkg.InstallPreference)
			}
			if pkg.Check == nil || len(pkg.Check.AppBundles) == 0 {
				t.Fatalf("%s should remain detectable by app bundle", id)
			}
		})
	}
}

func TestProfileParsing(t *testing.T) {
	wd, _ := os.Getwd()
	profileDir := filepath.Join(wd, "..", "..", "..", "profiles")

	profiles := []string{"minimal.json", "developer.json", "ai-developer.json", "full-stack-saas.json", "mobile.json"}

	for _, name := range profiles {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(profileDir, name)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read profile: %v", err)
			}
			if len(data) == 0 {
				t.Fatal("profile is empty")
			}
			// Basic JSON validation happens through os.ReadFile success
			// More detailed parsing would use config.HostProfile
		})
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestProfilesReferenceKnownHostPackages(t *testing.T) {
	wd, _ := os.Getwd()
	root := filepath.Join(wd, "..", "..", "..")
	cat, err := LoadCatalog(filepath.Join(root, "catalog", "packages.catalog.json"))
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}

	bannedProjectDeps := map[string]bool{
		"next":               true,
		"react":              true,
		"react-dom":          true,
		"tailwindcss":        true,
		"vite":               true,
		"vitest":             true,
		"jest":               true,
		"eslint-config-next": true,
		"@playwright/test":   true,
		"sharp":              true,
	}

	type profileFile struct {
		Packages struct {
			Include []string `json:"include"`
		} `json:"packages"`
	}

	profiles := []string{"minimal.json", "developer.json", "ai-developer.json", "full-stack-saas.json", "mobile.json"}
	for _, name := range profiles {
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(root, "profiles", name))
			if err != nil {
				t.Fatalf("read profile: %v", err)
			}
			var profile profileFile
			if err := json.Unmarshal(data, &profile); err != nil {
				t.Fatalf("parse profile: %v", err)
			}
			for _, id := range profile.Packages.Include {
				if bannedProjectDeps[id] {
					t.Fatalf("profile includes project-level dependency %q", id)
				}
				if cat.GetPackage(id) == nil {
					t.Fatalf("profile references unknown package %q", id)
				}
			}
		})
	}
}
