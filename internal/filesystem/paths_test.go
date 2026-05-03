package filesystem

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindDriveRoot(t *testing.T) {
	tmpDir := t.TempDir()

	// Create marker
	agentDir := filepath.Join(tmpDir, ".drive-agent")
	os.MkdirAll(agentDir, 0755)
	os.WriteFile(filepath.Join(agentDir, "DRIVE_AGENT_ROOT"), []byte(tmpDir), 0644)

	// Find from root
	root, err := FindDriveRoot(tmpDir)
	if err != nil {
		t.Fatalf("FindDriveRoot: %v", err)
	}
	if root != tmpDir {
		t.Errorf("root = %q, want %q", root, tmpDir)
	}

	// Find from subdirectory
	subDir := filepath.Join(tmpDir, "Orgs", "test")
	os.MkdirAll(subDir, 0755)
	root, err = FindDriveRoot(subDir)
	if err != nil {
		t.Fatalf("FindDriveRoot from subdir: %v", err)
	}
	if root != tmpDir {
		t.Errorf("root from subdir = %q, want %q", root, tmpDir)
	}

	// Not found
	otherDir := t.TempDir()
	_, err = FindDriveRoot(otherDir)
	if err == nil {
		t.Error("expected error for dir without marker")
	}
}

func TestDirSize(t *testing.T) {
	tmpDir := t.TempDir()

	// Create some files
	os.WriteFile(filepath.Join(tmpDir, "a.txt"), make([]byte, 1024), 0644)
	os.WriteFile(filepath.Join(tmpDir, "b.txt"), make([]byte, 2048), 0644)
	os.MkdirAll(filepath.Join(tmpDir, "sub"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "sub", "c.txt"), make([]byte, 512), 0644)

	size, err := DirSize(tmpDir)
	if err != nil {
		t.Fatalf("DirSize: %v", err)
	}
	expected := int64(1024 + 2048 + 512)
	if size != expected {
		t.Errorf("size = %d, want %d", size, expected)
	}
}

func TestExists(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(file, []byte("test"), 0644)

	if !Exists(file) {
		t.Error("existing file not detected")
	}
	if Exists(filepath.Join(tmpDir, "nonexistent")) {
		t.Error("nonexistent file detected as existing")
	}
}
