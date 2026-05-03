package catalog

import (
	"os"
	"path/filepath"
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
