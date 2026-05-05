package backup

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/callum-baillie/drive-agent/internal/utils"
)

type RepoSafety struct {
	IsLocal     bool
	LocalPath   string
	SameDrive   bool
	Warnings    []string
	Description string
}

func ValidateDriveRoot(driveRoot string) error {
	if dangerous, reason := utils.IsDangerousPath(driveRoot); dangerous {
		return fmt.Errorf("unsafe drive root: %s", reason)
	}
	return nil
}

func ValidateRepository(driveRoot, repo string, allowSameDrive bool) (RepoSafety, error) {
	repo = strings.TrimSpace(repo)
	if repo == "" {
		return RepoSafety{}, fmt.Errorf("backup repository is required")
	}
	if strings.Contains(repo, "\x00") {
		return RepoSafety{}, fmt.Errorf("backup repository contains invalid characters")
	}

	if isRemoteRepository(repo) {
		return RepoSafety{Description: repo}, nil
	}

	localPath, err := expandLocalRepoPath(repo)
	if err != nil {
		return RepoSafety{}, err
	}
	if !filepath.IsAbs(localPath) {
		return RepoSafety{}, fmt.Errorf("local backup repository must be an absolute path")
	}
	if dangerous, reason := utils.IsDangerousPath(localPath); dangerous {
		return RepoSafety{}, fmt.Errorf("refusing dangerous backup repository path: %s", reason)
	}

	safety := RepoSafety{IsLocal: true, LocalPath: localPath, Description: localPath}
	if utils.IsPathInsideDrive(localPath, driveRoot) {
		safety.SameDrive = true
		if !allowSameDrive {
			return safety, fmt.Errorf("backup repository %q is inside the source drive; use --allow-same-drive-repo only for non-backup tests", localPath)
		}
		safety.Warnings = append(safety.Warnings, "Repository is on the same drive. This is not a real backup.")
	}
	return safety, nil
}

func ValidateRestoreTarget(driveRoot, target string) error {
	target = strings.TrimSpace(target)
	if target == "" {
		return fmt.Errorf("restore target is required")
	}
	abs, err := filepath.Abs(target)
	if err != nil {
		return fmt.Errorf("resolve restore target: %w", err)
	}
	if dangerous, reason := utils.IsDangerousPath(abs); dangerous {
		return fmt.Errorf("refusing dangerous restore target: %s", reason)
	}
	if utils.IsPathInsideDrive(abs, driveRoot) {
		return fmt.Errorf("refusing to restore inside the active drive root: %s", abs)
	}
	if runtime.GOOS == "darwin" && !utils.IsInsideVolumes(abs) {
		return fmt.Errorf("restore target %q is not under /Volumes on macOS", abs)
	}
	return nil
}

func IsDirEmpty(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, err
	}
	return len(entries) == 0, nil
}

func isRemoteRepository(repo string) bool {
	lower := strings.ToLower(repo)
	remotePrefixes := []string{"sftp:", "s3:", "b2:", "azure:", "gs:", "rest:", "rclone:", "swift:"}
	for _, prefix := range remotePrefixes {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	if parsed, err := url.Parse(repo); err == nil && parsed.Scheme != "" && parsed.Host != "" {
		return true
	}
	return false
}

func expandLocalRepoPath(path string) (string, error) {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		if path == "~" {
			path = home
		} else if strings.HasPrefix(path, "~/") {
			path = filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve backup repository path: %w", err)
	}
	return filepath.Clean(abs), nil
}
