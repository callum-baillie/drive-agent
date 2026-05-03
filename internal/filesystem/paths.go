package filesystem

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/callum-baillie/drive-agent/internal/config"
)

// FindDriveRoot walks upward from the given path to find a .drive-agent directory.
// If path is empty, it uses the current working directory.
func FindDriveRoot(fromPath string) (string, error) {
	if fromPath == "" {
		var err error
		fromPath, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("get working directory: %w", err)
		}
	}

	abs, err := filepath.Abs(fromPath)
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}

	for {
		marker := filepath.Join(abs, config.AgentDir, config.MarkerFile)
		if _, err := os.Stat(marker); err == nil {
			return abs, nil
		}

		parent := filepath.Dir(abs)
		if parent == abs {
			// Reached filesystem root
			break
		}
		abs = parent
	}

	return "", fmt.Errorf("no drive-agent root found (looked for %s/%s from %s)",
		config.AgentDir, config.MarkerFile, fromPath)
}

// AgentPath returns the path to the .drive-agent directory.
func AgentPath(driveRoot string) string {
	return filepath.Join(driveRoot, config.AgentDir)
}

// DBPath returns the path to the SQLite database.
func DBPath(driveRoot string) string {
	return filepath.Join(driveRoot, config.AgentDir, "db", config.DatabaseFile)
}

// ConfigPath returns the path to the drive.toml config.
func ConfigPath(driveRoot string) string {
	return filepath.Join(driveRoot, config.AgentDir, "config", config.DriveConfigFile)
}

// OrgPath returns the path to an organization directory.
func OrgPath(driveRoot, orgSlug string) string {
	return filepath.Join(driveRoot, "Orgs", orgSlug)
}

// ProjectPath returns the path to a project directory.
func ProjectPath(driveRoot, orgSlug, projectSlug string) string {
	return filepath.Join(driveRoot, "Orgs", orgSlug, "projects", projectSlug)
}

// CatalogPath returns the path to the catalog directory.
func CatalogPath(driveRoot string) string {
	return filepath.Join(driveRoot, config.AgentDir, "catalog")
}

// LogsPath returns the path to the logs directory.
func LogsPath(driveRoot string) string {
	return filepath.Join(driveRoot, config.AgentDir, "logs")
}

// HostStatePath returns the path to a host state JSON file.
func HostStatePath(driveRoot, hostID string) string {
	return filepath.Join(driveRoot, config.AgentDir, "state", "hosts", hostID+".json")
}

// EnsureDir creates a directory and all parents if they don't exist.
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// Exists checks if a path exists.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// IsDir checks if a path is a directory.
func IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// DirSize calculates the total size of all files in a directory recursively.
func DirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors (permission denied, etc.)
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}
