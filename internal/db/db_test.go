package db

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestDB(t *testing.T) *DB {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.sqlite")
	database, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := database.InitSchema(); err != nil {
		t.Fatalf("init schema: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return database
}

func TestInitSchema(t *testing.T) {
	db := setupTestDB(t)
	v, err := db.SchemaVersion()
	if err != nil {
		t.Fatalf("schema version: %v", err)
	}
	if v != 1 {
		t.Errorf("schema version = %d, want 1", v)
	}
}

func TestOpenConfiguresSQLitePragmas(t *testing.T) {
	db := setupTestDB(t)

	var foreignKeys int
	if err := db.Conn().QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys); err != nil {
		t.Fatalf("query foreign_keys pragma: %v", err)
	}
	if foreignKeys != 1 {
		t.Errorf("foreign_keys = %d, want 1", foreignKeys)
	}

	var journalMode string
	if err := db.Conn().QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		t.Fatalf("query journal_mode pragma: %v", err)
	}
	if journalMode != "wal" {
		t.Errorf("journal_mode = %q, want %q", journalMode, "wal")
	}

	var busyTimeout int
	if err := db.Conn().QueryRow("PRAGMA busy_timeout").Scan(&busyTimeout); err != nil {
		t.Fatalf("query busy_timeout pragma: %v", err)
	}
	if busyTimeout != 5000 {
		t.Errorf("busy_timeout = %d, want 5000", busyTimeout)
	}
}

func TestOrganizationCRUD(t *testing.T) {
	db := setupTestDB(t)

	org := &Organization{
		ID: "org_test", Name: "Test Org", Slug: "test-org",
		Path:      "/tmp/test/Orgs/test-org",
		CreatedAt: "2026-01-01T00:00:00Z", UpdatedAt: "2026-01-01T00:00:00Z",
	}
	if err := db.InsertOrganization(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	orgs, err := db.ListOrganizations()
	if err != nil {
		t.Fatalf("list orgs: %v", err)
	}
	if len(orgs) != 1 {
		t.Fatalf("expected 1 org, got %d", len(orgs))
	}
	if orgs[0].Slug != "test-org" {
		t.Errorf("slug = %q, want %q", orgs[0].Slug, "test-org")
	}

	found, err := db.GetOrganizationBySlug("test-org")
	if err != nil {
		t.Fatalf("get by slug: %v", err)
	}
	if found.Name != "Test Org" {
		t.Errorf("name = %q, want %q", found.Name, "Test Org")
	}
}

func TestProjectCRUD(t *testing.T) {
	db := setupTestDB(t)

	// Create org first
	org := &Organization{
		ID: "org_test", Name: "Test", Slug: "test",
		Path:      "/tmp/test/Orgs/test",
		CreatedAt: "2026-01-01T00:00:00Z", UpdatedAt: "2026-01-01T00:00:00Z",
	}
	db.InsertOrganization(org)

	proj := &Project{
		ID: "proj_test_myapp", OrganizationID: "org_test",
		Name: "My App", Slug: "my-app",
		Path:        "/tmp/test/Orgs/test/projects/my-app",
		GitRemote:   "git@github.com:test/my-app.git",
		ProjectType: "nextjs", PackageManager: "pnpm",
		Tags:      []string{"web", "nextjs"},
		CreatedAt: "2026-01-01T00:00:00Z", UpdatedAt: "2026-01-01T00:00:00Z",
	}
	if err := db.InsertProject(proj); err != nil {
		t.Fatalf("insert project: %v", err)
	}

	projects, err := db.ListProjects("", "")
	if err != nil {
		t.Fatalf("list projects: %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(projects))
	}
	if projects[0].Slug != "my-app" {
		t.Errorf("slug = %q, want %q", projects[0].Slug, "my-app")
	}
	if len(projects[0].Tags) != 2 {
		t.Errorf("tags count = %d, want 2", len(projects[0].Tags))
	}

	// Filter by org
	filtered, err := db.ListProjects("test", "")
	if err != nil {
		t.Fatalf("filter by org: %v", err)
	}
	if len(filtered) != 1 {
		t.Errorf("filtered count = %d, want 1", len(filtered))
	}

	// Filter by tag
	tagged, err := db.ListProjects("", "web")
	if err != nil {
		t.Fatalf("filter by tag: %v", err)
	}
	if len(tagged) != 1 {
		t.Errorf("tagged count = %d, want 1", len(tagged))
	}

	// Get by slug
	found, err := db.GetProjectBySlug("test", "my-app")
	if err != nil {
		t.Fatalf("get by slug: %v", err)
	}
	if found.GitRemote != "git@github.com:test/my-app.git" {
		t.Errorf("remote = %q", found.GitRemote)
	}
}

func TestHostUpsert(t *testing.T) {
	db := setupTestDB(t)

	host := &Host{
		ID: "test-host", Hostname: "test-host.local",
		OS: "darwin", Arch: "arm64", Shell: "zsh",
		LastSeenAt: "2026-01-01T00:00:00Z",
		CreatedAt:  "2026-01-01T00:00:00Z", UpdatedAt: "2026-01-01T00:00:00Z",
	}

	if err := db.UpsertHost(host); err != nil {
		t.Fatalf("upsert host: %v", err)
	}

	found, err := db.GetHost("test-host")
	if err != nil {
		t.Fatalf("get host: %v", err)
	}
	if found.OS != "darwin" {
		t.Errorf("os = %q, want %q", found.OS, "darwin")
	}

	// Upsert again (update)
	host.LastSeenAt = "2026-02-01T00:00:00Z"
	if err := db.UpsertHost(host); err != nil {
		t.Fatalf("upsert host again: %v", err)
	}
}

func TestDriveRecord(t *testing.T) {
	db := setupTestDB(t)
	err := db.InsertDrive("drive-test", "TestDrive", "/tmp/test", "2026-01-01T00:00:00Z", "2026-01-01T00:00:00Z")
	if err != nil {
		t.Fatalf("insert drive: %v", err)
	}

	id, name, rootPath, _, schemaVer, err := db.GetDrive()
	if err != nil {
		t.Fatalf("get drive: %v", err)
	}
	if id != "drive-test" || name != "TestDrive" || rootPath != "/tmp/test" || schemaVer != 1 {
		t.Errorf("unexpected drive: id=%q name=%q root=%q ver=%d", id, name, rootPath, schemaVer)
	}
}

func TestInitDirectoryCreation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create standard directories
	dirs := []string{
		filepath.Join(tmpDir, ".drive-agent", "bin"),
		filepath.Join(tmpDir, ".drive-agent", "db"),
		filepath.Join(tmpDir, ".drive-agent", "config"),
		filepath.Join(tmpDir, "Orgs"),
		filepath.Join(tmpDir, "DevData"),
		filepath.Join(tmpDir, "Caches"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("create dir %s: %v", dir, err)
		}
	}

	// Verify
	for _, dir := range dirs {
		info, err := os.Stat(dir)
		if err != nil {
			t.Errorf("dir %s does not exist: %v", dir, err)
		} else if !info.IsDir() {
			t.Errorf("%s is not a directory", dir)
		}
	}
}
