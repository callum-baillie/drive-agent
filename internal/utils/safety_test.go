package utils

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// --- Extended safety tests ---

func TestIsDangerousPath_Descendants(t *testing.T) {
	dangerousPaths := []string{
		"/",
		"/Users",
		"/Users/callum",
		"/Users/callum/Desktop",
		"/Users/callum/Documents/DevDriveTest",
		"/System",
		"/System/Library",
		"/Library",
		"/Library/Frameworks",
		"/private",
		"/private/var",
		"/etc",
		"/bin",
		"/usr",
		"/usr/local",
		"/opt",
		"/opt/homebrew",
		"/tmp/DevDriveTest", // /tmp is protected, so this must be rejected
	}

	for _, p := range dangerousPaths {
		t.Run(p, func(t *testing.T) {
			dangerous, reason := IsDangerousPath(p)
			if !dangerous {
				t.Errorf("IsDangerousPath(%q) = false, want true", p)
			}
			if reason == "" {
				t.Errorf("IsDangerousPath(%q) returned empty reason", p)
			}
		})
	}
}

func TestIsDangerousPath_SafePaths(t *testing.T) {
	// These should NOT be considered dangerous
	safePaths := []string{
		"/Volumes/DevDrive",
		"/Volumes/DevDrive/Orgs",
		"/Volumes/DevDrive/Orgs/roamar/projects/user-web",
		// /tmp/DevDriveTest is intentionally omitted here because /tmp is protected
		// by DangerousPaths, so it is blocked (tested above).
	}
	for _, p := range safePaths {
		t.Run(p, func(t *testing.T) {
			dangerous, _ := IsDangerousPath(p)
			if dangerous {
				t.Errorf("IsDangerousPath(%q) = true, want false", p)
			}
		})
	}
}

func TestIsDangerousPath_HomeDirectory(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home dir")
	}
	dangerous, _ := IsDangerousPath(home)
	if !dangerous {
		t.Errorf("IsDangerousPath(home=%q) = false, want true", home)
	}
}



func TestIsInsideVolumes(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("IsInsideVolumes is macOS specific")
	}
	tests := []struct {
		path string
		want bool
	}{
		{"/Volumes/MyDrive", true},
		{"/Volumes/MyDrive/Orgs/foo", true},
		{"/tmp/test", false},
		{"/Users/me/test", false},
		{"/", false},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := IsInsideVolumes(tt.path)
			if got != tt.want {
				t.Errorf("IsInsideVolumes(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestIsPathInsideDrive(t *testing.T) {
	tests := []struct {
		path      string
		driveRoot string
		want      bool
	}{
		{"/Volumes/Drive/Orgs/foo", "/Volumes/Drive", true},
		{"/Volumes/Drive/.drive-agent/db", "/Volumes/Drive", true},
		{"/Volumes/Drive", "/Volumes/Drive", true},
		{"/Volumes/OtherDrive/foo", "/Volumes/Drive", false},
		{"/tmp/evil", "/Volumes/Drive", false},
		{"/", "/Volumes/Drive", false},
		{"/Volumes/DriveExtra/foo", "/Volumes/Drive", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := IsPathInsideDrive(tt.path, tt.driveRoot)
			if got != tt.want {
				t.Errorf("IsPathInsideDrive(%q, %q) = %v, want %v",
					tt.path, tt.driveRoot, got, tt.want)
			}
		})
	}
}

func TestIsPathInsideDrive_SymlinkTraversal(t *testing.T) {
	// Create a temp dir simulating a drive root
	driveRoot := t.TempDir()
	outsideDir := t.TempDir()

	// Create a symlink inside the drive pointing outside
	symlinkPath := filepath.Join(driveRoot, "evil-link")
	if err := os.Symlink(outsideDir, symlinkPath); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	// The symlink target resolves outside the drive root — should be rejected
	inside := IsPathInsideDrive(symlinkPath, driveRoot)
	if inside {
		t.Errorf("IsPathInsideDrive accepted symlink pointing outside drive root: %s -> %s",
			symlinkPath, outsideDir)
	}
}

func TestIsSymlink(t *testing.T) {
	tmpDir := t.TempDir()

	// Regular file
	regularFile := filepath.Join(tmpDir, "regular.txt")
	os.WriteFile(regularFile, []byte("test"), 0644)
	if IsSymlink(regularFile) {
		t.Error("regular file should not be symlink")
	}

	// Symlink
	linkFile := filepath.Join(tmpDir, "link.txt")
	os.Symlink(regularFile, linkFile)
	if !IsSymlink(linkFile) {
		t.Error("symlink should be detected as symlink")
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1024 * 1024, "1.0 MB"},
		{1024 * 1024 * 1024, "1.0 GB"},
		{1024 * 1024 * 1024 * 2, "2.0 GB"},
	}

	for _, tt := range tests {
		got := FormatBytes(tt.input)
		if got != tt.want {
			t.Errorf("FormatBytes(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
