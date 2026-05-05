package backup

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"

	"github.com/callum-baillie/drive-agent/internal/config"
)

func TestConfigRoundTrip(t *testing.T) {
	driveRoot := t.TempDir()
	cfg := NewConfig("restic", "local-backup", "/Volumes/Backup/restic/devdrive", false)

	if err := SaveConfig(driveRoot, cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	loaded, err := LoadConfig(driveRoot)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if loaded.Provider != "restic" {
		t.Fatalf("Provider = %q, want restic", loaded.Provider)
	}
	repo, err := loaded.SelectedRepository("local-backup")
	if err != nil {
		t.Fatalf("SelectedRepository: %v", err)
	}
	if repo.Repository != "/Volumes/Backup/restic/devdrive" {
		t.Fatalf("Repository = %q", repo.Repository)
	}
	if len(loaded.Excludes) == 0 {
		t.Fatal("expected default excludes")
	}
}

func TestLoadConfigMergesRequiredSafetyExcludes(t *testing.T) {
	driveRoot := t.TempDir()
	cfg := NewConfig("restic", "local-backup", "/Volumes/Backup/restic/devdrive", false)
	cfg.Excludes = []string{"node_modules"}
	if err := SaveConfig(driveRoot, cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	loaded, err := LoadConfig(driveRoot)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	for _, want := range []string{".env", "**/.env.*", "*.tfstate", "**/*.tfstate.backup", ".terraform", "tmp/**", "node_modules"} {
		if !contains(loaded.Excludes, want) {
			t.Fatalf("loaded excludes missing required pattern %q: %#v", want, loaded.Excludes)
		}
	}
}

func TestDefaultExcludes(t *testing.T) {
	excludes := DefaultExcludes()
	mustContain := []string{
		".env", ".env.*", "**/.env", "**/.env.*",
		"*.tfstate", "*.tfstate.backup", "**/*.tfstate", "**/*.tfstate.backup",
		".terraform", "**/.terraform", "tmp", "tmp/**",
		"node_modules", ".next", "android/.gradle", "ios/Pods",
		".Trashes", ".Spotlight-V100", ".fseventsd", ".drive-agent/releases/tmp",
	}
	mustNotContain := []string{".git", ".env.example", "docs", "prisma", "migrations", "src", "package.json", "pnpm-lock.yaml", "README.md", ".drive-project.toml"}

	for _, pattern := range mustContain {
		if !contains(excludes, pattern) {
			t.Fatalf("default excludes missing %q", pattern)
		}
	}
	for _, pattern := range mustNotContain {
		if contains(excludes, pattern) {
			t.Fatalf("default excludes should not contain %q", pattern)
		}
	}
}

func TestExcludeAddRemoveIdempotent(t *testing.T) {
	excludes := []string{"node_modules"}
	var changed bool
	var err error

	excludes, changed, err = AddExclude(excludes, ".next")
	if err != nil || !changed {
		t.Fatalf("AddExclude changed=%v err=%v", changed, err)
	}
	excludes, changed, err = AddExclude(excludes, ".next")
	if err != nil || changed {
		t.Fatalf("duplicate AddExclude changed=%v err=%v", changed, err)
	}
	excludes, changed, err = RemoveExclude(excludes, ".next")
	if err != nil || !changed {
		t.Fatalf("RemoveExclude changed=%v err=%v", changed, err)
	}
	excludes, changed, err = RemoveExclude(excludes, ".next")
	if err != nil || changed {
		t.Fatalf("duplicate RemoveExclude changed=%v err=%v", changed, err)
	}
	if contains(excludes, ".next") {
		t.Fatal(".next still present")
	}
}

func TestWriteExcludeFile(t *testing.T) {
	driveRoot := t.TempDir()
	path, err := WriteExcludeFile(driveRoot, []string{"node_modules", ".next"})
	if err != nil {
		t.Fatalf("WriteExcludeFile: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if filepath.Base(path) != "restic-excludes.txt" {
		t.Fatalf("exclude file name = %s", path)
	}
	if string(data) != ".next\nnode_modules\n" {
		t.Fatalf("exclude file = %q", data)
	}
}

func TestProjectManifestExcludesRoundTrip(t *testing.T) {
	projectPath := t.TempDir()
	manifest := config.ProjectManifestData{
		ID:             "project-roamar-roamar-turbo",
		Name:           "roamar-turbo",
		Slug:           "roamar-turbo",
		Org:            "roamar",
		Type:           "turborepo",
		PackageManager: "pnpm",
		CreatedAt:      "2026-05-04T00:00:00Z",
	}
	file, err := os.Create(filepath.Join(projectPath, config.ProjectManifest))
	if err != nil {
		t.Fatalf("Create manifest: %v", err)
	}
	if err := toml.NewEncoder(file).Encode(manifest); err != nil {
		t.Fatalf("Encode manifest: %v", err)
	}
	file.Close()

	if changed, err := AddProjectExclude(projectPath, "apps/*/node_modules"); err != nil || !changed {
		t.Fatalf("AddProjectExclude changed=%v err=%v", changed, err)
	}
	if changed, err := AddProjectExclude(projectPath, "apps/*/node_modules"); err != nil || changed {
		t.Fatalf("duplicate AddProjectExclude changed=%v err=%v", changed, err)
	}
	if changed, err := AddProjectExclude(projectPath, "./packages/*/dist"); err != nil || !changed {
		t.Fatalf("AddProjectExclude wildcard changed=%v err=%v", changed, err)
	}

	excludes, err := LoadProjectManifestExcludes(projectPath)
	if err != nil {
		t.Fatalf("LoadProjectManifestExcludes: %v", err)
	}
	for _, want := range []string{"apps/*/node_modules", "packages/*/dist"} {
		if !contains(excludes, want) {
			t.Fatalf("project excludes missing %q: %#v", want, excludes)
		}
	}
}

func TestProjectExcludesAreScopedAndPreserveGlobalExcludes(t *testing.T) {
	projectPath := filepath.Join(t.TempDir(), "Orgs", "roamar", "projects", "roamar-turbo")
	projects := []ProjectExcludeSet{{
		OrgSlug:     "roamar",
		ProjectSlug: "roamar-turbo",
		ProjectPath: projectPath,
		Patterns:    []string{"node_modules", "apps/*/.next", "packages/*/dist"},
	}}
	got, err := MergeProjectExcludes([]string{".next", "node_modules"}, projects)
	if err != nil {
		t.Fatalf("MergeProjectExcludes: %v", err)
	}

	for _, want := range []string{".next", "node_modules"} {
		if !contains(got, want) {
			t.Fatalf("global exclude %q missing: %#v", want, got)
		}
	}
	for _, want := range []string{
		filepath.ToSlash(filepath.Join(projectPath, "node_modules")),
		filepath.ToSlash(filepath.Join(projectPath, "apps/*/.next")),
		filepath.ToSlash(filepath.Join(projectPath, "packages/*/dist")),
	} {
		if !contains(got, want) {
			t.Fatalf("scoped project exclude %q missing: %#v", want, got)
		}
	}
}

func TestProjectExcludeRejectsParentTraversal(t *testing.T) {
	if _, err := ScopeProjectExclude("/Volumes/ExternalSSD/Orgs/roamar/projects/roamar-turbo", "../secrets"); err == nil {
		t.Fatal("expected parent traversal to be rejected")
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
