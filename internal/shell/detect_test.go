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

func TestStorageShellBlockContent_QuotesDrivePathsWithSpaces(t *testing.T) {
	content := StorageShellBlockContent(ShellBlockOptions{
		HomebrewCachePath: "/Volumes/External SSD/Caches/homebrew",
		BunCachePath:      "/Volumes/External SSD/Caches/bun",
		ContainerDataPath: "/Volumes/External SSD/DevData/containers",
		DockerCachePath:   "/Volumes/External SSD/DevData/docker-build-cache",
	})

	expected := []string{
		`export HOMEBREW_CACHE='/Volumes/External SSD/Caches/homebrew'`,
		`export BUN_INSTALL_CACHE_DIR='/Volumes/External SSD/Caches/bun'`,
		`export DRIVE_AGENT_CONTAINER_DATA='/Volumes/External SSD/DevData/containers'`,
		`export DRIVE_AGENT_DOCKER_BUILD_CACHE='/Volumes/External SSD/DevData/docker-build-cache'`,
	}
	for _, want := range expected {
		if !strings.Contains(content, want) {
			t.Fatalf("storage shell block missing %q in:\n%s", want, content)
		}
	}
	if strings.Contains(content, "npm_config_cache") {
		t.Fatalf("storage shell block should not persist npm_config_cache; npm is configured with npm config set:\n%s", content)
	}
}

func TestAppendOrUpdateStorageShellBlock_IdempotentAndBacksUp(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".zshrc")
	initial := "# initial\nexport FOO=bar\n"
	if err := os.WriteFile(configPath, []byte(initial), 0644); err != nil {
		t.Fatal(err)
	}
	options := ShellBlockOptions{
		HomebrewCachePath: "/Volumes/Test Drive/Caches/homebrew",
		BunCachePath:      "/Volumes/Test Drive/Caches/bun",
		ContainerDataPath: "/Volumes/Test Drive/DevData/containers",
		DockerCachePath:   "/Volumes/Test Drive/DevData/docker-build-cache",
	}

	backupPath, changed, err := AppendOrUpdateStorageShellBlock(configPath, options)
	if err != nil {
		t.Fatalf("AppendOrUpdateStorageShellBlock returned error: %v", err)
	}
	if !changed {
		t.Fatal("expected first call to write storage block")
	}
	if backupPath != BackupPathFor(configPath) {
		t.Fatalf("backup path = %q, want %q", backupPath, BackupPathFor(configPath))
	}
	backupData, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("read backup: %v", err)
	}
	if string(backupData) != initial {
		t.Fatalf("backup content = %q, want %q", string(backupData), initial)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if strings.Count(content, ">>> drive-agent storage >>>") != 1 {
		t.Fatalf("storage block count = %d, want 1\n%s", strings.Count(content, ">>> drive-agent storage >>>"), content)
	}

	_, changed, err = AppendOrUpdateStorageShellBlock(configPath, options)
	if err != nil {
		t.Fatalf("second AppendOrUpdateStorageShellBlock returned error: %v", err)
	}
	if changed {
		t.Fatal("second call should be a no-op")
	}
	data, err = os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(data), ">>> drive-agent storage >>>") != 1 {
		t.Fatalf("storage block was duplicated:\n%s", string(data))
	}
}

func TestAppendOrUpdateStorageShellBlock_UpdatesExistingBlock(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), ".zshrc")
	if err := os.WriteFile(configPath, []byte(StorageShellBlock("export HOMEBREW_CACHE='/old'")+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	_, changed, err := AppendOrUpdateStorageShellBlock(configPath, ShellBlockOptions{
		HomebrewCachePath: "/Volumes/New/Caches/homebrew",
	})
	if err != nil {
		t.Fatalf("AppendOrUpdateStorageShellBlock returned error: %v", err)
	}
	if !changed {
		t.Fatal("expected existing storage block to be updated")
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if strings.Count(content, ">>> drive-agent storage >>>") != 1 {
		t.Fatalf("storage block count = %d, want 1\n%s", strings.Count(content, ">>> drive-agent storage >>>"), content)
	}
	if strings.Contains(content, "/old") || !strings.Contains(content, "/Volumes/New/Caches/homebrew") {
		t.Fatalf("storage block was not replaced correctly:\n%s", content)
	}
}
