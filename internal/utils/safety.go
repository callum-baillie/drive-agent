package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// DangerousPaths contains paths that must never be used as a drive root.
var DangerousPaths = []string{
	"/",
	"/bin",
	"/sbin",
	"/usr",
	"/usr/local",
	"/etc",
	"/var",
	"/tmp",
	"/opt",
	"/System",
	"/Library",
	"/Applications",
	"/Users",
	"/private",
	"/dev",
	"/proc",
	"/sys",
	"C:\\",
	"C:\\Windows",
	"C:\\Program Files",
	"C:\\Program Files (x86)",
}

// IsDangerousPath checks if a path is a known dangerous/system path.
// It blocks the exact path AND any descendant of a dangerous directory.
func IsDangerousPath(p string) (bool, string) {
	abs, err := filepath.Abs(p)
	if err != nil {
		return true, fmt.Sprintf("cannot resolve path: %v", err)
	}

	// Resolve symlinks to find the true destination
	realPath, err := filepath.EvalSymlinks(abs)
	if err != nil {
		// If it doesn't exist yet, we still check the absolute path
		realPath = filepath.Clean(abs)
	}

	// Also check the user's home directory
	var homeDir string
	if home, err := os.UserHomeDir(); err == nil {
		homeDir, err = filepath.EvalSymlinks(home)
		if err != nil {
			homeDir = filepath.Clean(home)
		}
	}

	for _, dp := range DangerousPaths {
		// Resolve the dangerous path as well, just in case
		realDP, err := filepath.EvalSymlinks(dp)
		if err != nil {
			realDP = filepath.Clean(dp)
		}

		// We check against the resolved path (handles cases where the user provides
		// a symlink pointing inside a protected path) AND against the raw cleaned path
		// (handles cases where the user provides a path inside a protected symlink
		// but the path doesn't exist yet, so EvalSymlinks failed).
		if isDescendant(realDP, realPath) || isDescendant(filepath.Clean(dp), filepath.Clean(abs)) {
			return true, fmt.Sprintf("path %q is inside protected system path %q", abs, dp)
		}
	}

	if homeDir != "" && isDescendant(homeDir, realPath) {
		return true, fmt.Sprintf("path %q is inside the user's home directory", abs)
	}

	return false, ""
}

// isDescendant checks if child is exactly parent or inside parent.
func isDescendant(parent, child string) bool {
	parent = filepath.Clean(parent)
	child = filepath.Clean(child)

	// If parent is a root like "/" or "C:\", we only block exact matches.
	// Otherwise, we'd block the entire filesystem.
	if parent == "/" || parent == filepath.VolumeName(parent)+"\\" {
		return parent == child
	}

	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	// If rel is ".", it's an exact match.
	// If rel doesn't start with "..", it's inside parent.
	return rel == "." || (!strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel))
}

// IsInsideVolumes checks if a path is inside /Volumes on macOS.
func IsInsideVolumes(p string) bool {
	if runtime.GOOS != "darwin" {
		return true // Not applicable on non-macOS
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return false
	}
	return strings.HasPrefix(abs, "/Volumes/")
}

// IsPathInsideDrive checks if a path is inside the given drive root.
// Used by cleanup to prevent deleting outside drive boundaries.
// It resolves symlinks so a symlink pointing outside the drive is rejected.
func IsPathInsideDrive(path, driveRoot string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	absDrive, err := filepath.Abs(driveRoot)
	if err != nil {
		return false
	}

	// Resolve the drive root to its real location (handles drive itself being a symlink)
	realDrive, err := filepath.EvalSymlinks(absDrive)
	if err != nil {
		realDrive = absDrive
	}

	// Resolve the target path — if it's a symlink, EvalSymlinks follows the chain.
	// If the target doesn't exist yet, fall back to the Abs path.
	realPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		realPath = absPath
	}

	rel, err := filepath.Rel(realDrive, realPath)
	if err != nil {
		return false
	}
	// If rel starts with "..", the real path is outside the drive
	return !strings.HasPrefix(rel, "..")
}

// IsSymlink checks if a path is a symbolic link (Lstat — does not follow links).
func IsSymlink(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeSymlink != 0
}

// FormatBytes formats a byte count into a human-readable string.
func FormatBytes(b int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)
	switch {
	case b >= TB:
		return fmt.Sprintf("%.1f TB", float64(b)/float64(TB))
	case b >= GB:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(GB))
	case b >= MB:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(MB))
	case b >= KB:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(KB))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
