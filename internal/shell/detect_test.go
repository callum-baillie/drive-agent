package shell

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShellBlockAlreadyInstalled_NotPresent(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "*.zshrc")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("# existing config\nexport FOO=bar\n")
	f.Close()

	installed, err := ShellBlockAlreadyInstalled(f.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if installed {
		t.Error("expected block to NOT be installed yet")
	}
}

func TestShellBlockAlreadyInstalled_Present(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "*.zshrc")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("# existing config\n")
	f.WriteString(ShellBlock("export PATH=\"/fake:$PATH\""))
	f.WriteString("\n")
	f.Close()

	installed, err := ShellBlockAlreadyInstalled(f.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !installed {
		t.Error("expected block to be detected as already installed")
	}
}

func TestShellBlockAlreadyInstalled_NonExistentFile(t *testing.T) {
	installed, err := ShellBlockAlreadyInstalled(filepath.Join(t.TempDir(), "nonexistent"))
	if err != nil {
		t.Fatalf("should not error for non-existent file: %v", err)
	}
	if installed {
		t.Error("non-existent file should not be considered installed")
	}
}

func TestAppendShellBlock_FirstTime(t *testing.T) {
	tmpDir := t.TempDir()
	driveRoot := tmpDir
	configPath := filepath.Join(tmpDir, ".zshrc")

	// Write initial config
	os.WriteFile(configPath, []byte("# initial\n"), 0644)

	if err := AppendShellBlock(configPath, driveRoot); err != nil {
		t.Fatalf("first AppendShellBlock failed: %v", err)
	}

	// Verify block is present
	data, _ := os.ReadFile(configPath)
	content := string(data)
	if !strings.Contains(content, ">>> drive-agent >>>") {
		t.Error("block start marker not found")
	}
	if !strings.Contains(content, "<<< drive-agent <<<") {
		t.Error("block end marker not found")
	}

	// Verify backup was created
	backupPath := BackupPathFor(configPath)
	if _, err := os.Stat(backupPath); err != nil {
		t.Errorf("backup not created at %s: %v", backupPath, err)
	}
}

func TestAppendShellBlock_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()
	driveRoot := tmpDir
	configPath := filepath.Join(tmpDir, ".zshrc")

	os.WriteFile(configPath, []byte("# initial\n"), 0644)

	// First install
	if err := AppendShellBlock(configPath, driveRoot); err != nil {
		t.Fatalf("first install failed: %v", err)
	}

	data1, _ := os.ReadFile(configPath)

	// Second install should be rejected
	err := AppendShellBlock(configPath, driveRoot)
	if err == nil {
		t.Fatal("expected ErrShellBlockAlreadyPresent on second install, got nil")
	}
	if err != ErrShellBlockAlreadyPresent {
		t.Errorf("expected ErrShellBlockAlreadyPresent, got: %v", err)
	}

	// File should be unchanged
	data2, _ := os.ReadFile(configPath)
	if string(data1) != string(data2) {
		t.Error("file was modified on second AppendShellBlock call (not idempotent)")
	}

	// Count occurrences of the block marker — must be exactly 1
	count := strings.Count(string(data2), ">>> drive-agent >>>")
	if count != 1 {
		t.Errorf("block marker found %d times, want exactly 1", count)
	}
}

func TestShellBlock_ContainsMarkers(t *testing.T) {
	block := ShellBlock("export FOO=bar")
	if !strings.Contains(block, ">>> drive-agent >>>") {
		t.Error("block missing start marker")
	}
	if !strings.Contains(block, "<<< drive-agent <<<") {
		t.Error("block missing end marker")
	}
	if !strings.Contains(block, "export FOO=bar") {
		t.Error("block missing content")
	}
}

func TestShellBlockContent_QuotesDrivePathsWithSpaces(t *testing.T) {
	content := ShellBlockContentWithOptions("/Volumes/External SSD", ShellBlockOptions{
		NpmCachePath:      "/Volumes/External SSD/Caches/npm",
		HomebrewCachePath: "/Volumes/External SSD/Caches/homebrew",
		ContainerDataPath: "/Volumes/External SSD/DevData/containers",
	})

	expected := []string{
		`export PATH='/Volumes/External SSD/.drive-agent/bin':"$PATH"`,
		`export npm_config_cache='/Volumes/External SSD/Caches/npm'`,
		`export HOMEBREW_CACHE='/Volumes/External SSD/Caches/homebrew'`,
		`export DRIVE_AGENT_CONTAINER_DATA='/Volumes/External SSD/DevData/containers'`,
	}
	for _, want := range expected {
		if !strings.Contains(content, want) {
			t.Fatalf("shell block missing %q in:\n%s", want, content)
		}
	}
}
