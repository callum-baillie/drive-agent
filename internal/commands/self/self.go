package self

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/callum-baillie/drive-agent/internal/config"
	"github.com/callum-baillie/drive-agent/internal/filesystem"
	"github.com/callum-baillie/drive-agent/internal/ui"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "self", Short: "Self-management commands"}
	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newUpdateCmd())
	cmd.AddCommand(newRollbackCmd())
	return cmd
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use: "version", Short: "Show version details",
		Run: func(cmd *cobra.Command, args []string) {
			ui.Header("Drive Agent Info")
			ui.Label("Version", config.Version)

			exe, _ := os.Executable()
			ui.Label("Install path", exe)

			driveRoot, err := filesystem.FindDriveRoot(exe)
			if err == nil {
				ui.Label("Drive root", driveRoot)

				// Find latest backup
				backupsDir := filepath.Join(filesystem.AgentPath(driveRoot), "backups")
				entries, _ := os.ReadDir(backupsDir)
				if len(entries) > 0 {
					latest := entries[len(entries)-1].Name()
					ui.Label("Latest backup", filepath.Join(backupsDir, latest))
				} else {
					ui.Label("Latest backup", "None")
				}
			} else {
				ui.Label("Drive root", "Not found (running outside initialized drive)")
			}
			fmt.Println()
		},
	}
}

type githubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func newUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "update", Short: "Update drive-agent",
		RunE: runUpdate,
	}
	cmd.Flags().String("version", "", "Specific version to install (e.g. v0.1.1)")
	cmd.Flags().Bool("yes", false, "Skip confirmation")
	cmd.Flags().Bool("dry-run", false, "Show plan without updating")
	return cmd
}

func runUpdate(cmd *cobra.Command, args []string) error {
	targetVersion, _ := cmd.Flags().GetString("version")
	autoYes, _ := cmd.Flags().GetBool("yes")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	ui.Header("Self-Update")
	ui.Info("Current version: v%s", config.Version)

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("detect executable path: %w", err)
	}

	// Refuse if not in .drive-agent/bin
	if filepath.Base(filepath.Dir(exe)) != "bin" || filepath.Base(filepath.Dir(filepath.Dir(exe))) != config.AgentDir {
		return fmt.Errorf("executable not installed in %s/bin. Found at: %s", config.AgentDir, exe)
	}

	driveRoot, err := filesystem.FindDriveRoot(exe)
	if err != nil {
		return fmt.Errorf("detect drive root: %w", err)
	}

	// Fetch release metadata
	var url string
	if targetVersion != "" {
		url = fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/tags/%s", config.RepoOwner, config.RepoName, targetVersion)
	} else {
		url = fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", config.RepoOwner, config.RepoName)
	}

	ui.Info("Fetching release metadata from GitHub...")
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("fetch release metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch release (HTTP %d)", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return fmt.Errorf("decode release metadata: %w", err)
	}

	ui.Label("Target version", release.TagName)

	if release.TagName == "v"+config.Version && targetVersion == "" {
		ui.Success("Already up to date!")
		return nil
	}

	assetName := determineAssetName(runtime.GOOS, runtime.GOARCH)
	var assetURL string
	var checksumsURL string

	for _, a := range release.Assets {
		if a.Name == assetName {
			assetURL = a.BrowserDownloadURL
		} else if a.Name == "checksums.txt" {
			checksumsURL = a.BrowserDownloadURL
		}
	}

	if assetURL == "" {
		return fmt.Errorf("could not find asset %s in release %s", assetName, release.TagName)
	}

	ui.Label("Asset", assetName)

	if dryRun {
		fmt.Println()
		ui.Info("[Dry Run] Would download %s", assetURL)
		ui.Info("[Dry Run] Would verify against %s", checksumsURL)
		ui.Info("[Dry Run] Would replace %s", exe)
		return nil
	}

	if !autoYes {
		if !ui.Confirm(fmt.Sprintf("\nUpdate to %s?", release.TagName), false) {
			fmt.Println("Aborted.")
			return nil
		}
	}

	// Create temp dir
	tmpDir := filepath.Join(filesystem.AgentPath(driveRoot), "releases", "tmp")
	os.MkdirAll(tmpDir, 0755)

	// Download checksums
	ui.Info("Downloading checksums...")
	checksumsData, err := downloadString(checksumsURL)
	if err != nil {
		return fmt.Errorf("download checksums: %w", err)
	}

	expectedChecksum, err := parseChecksums(checksumsData, assetName)
	if err != nil {
		return err
	}

	// Download asset
	ui.Info("Downloading %s...", assetName)
	archivePath := filepath.Join(tmpDir, assetName)
	if err := downloadFile(assetURL, archivePath); err != nil {
		return fmt.Errorf("download asset: %w", err)
	}
	defer os.Remove(archivePath)

	// Verify checksum
	ui.Info("Verifying checksum...")
	if err := verifyFileChecksum(archivePath, expectedChecksum); err != nil {
		return err
	}
	ui.Success("Checksum verified")

	// Extract binary
	ui.Info("Extracting binary...")
	extractedBin := filepath.Join(tmpDir, "drive-agent-new")
	if err := extractArchive(archivePath, extractedBin); err != nil {
		return fmt.Errorf("extract archive: %w", err)
	}
	defer os.Remove(extractedBin)

	// Backup current
	timestamp := time.Now().Format("20060102150405")
	backupPath := filepath.Join(filesystem.AgentPath(driveRoot), "backups", fmt.Sprintf("drive-agent-v%s-%s", config.Version, timestamp))
	ui.Info("Backing up current binary to %s...", filepath.Base(backupPath))

	if err := copyFile(exe, backupPath); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	// Replace atomically
	ui.Info("Replacing binary...")
	if err := os.Rename(extractedBin, exe); err != nil {
		ui.Error("Failed to replace binary: %v", err)
		ui.Warning("Attempting to restore from backup...")
		if restoreErr := copyFile(backupPath, exe); restoreErr != nil {
			ui.Error("Rollback failed! Please manually copy %s to %s", backupPath, exe)
		} else {
			ui.Info("Restored from backup successfully.")
		}
		return err
	}
	os.Chmod(exe, 0755)

	// Update metadata
	newVer := strings.TrimPrefix(release.TagName, "v")
	os.WriteFile(filepath.Join(filesystem.AgentPath(driveRoot), "VERSION"), []byte(newVer), 0644)

	installMeta := map[string]string{
		"installed_at":    time.Now().UTC().Format(time.RFC3339),
		"version":         newVer,
		"method":          "self update",
		"install_path":    exe,
		"drive_root":      driveRoot,
		"release_asset":   assetName,
		"os":              runtime.GOOS,
		"arch":            runtime.GOARCH,
		"repo_owner":      config.RepoOwner,
		"repo_name":       config.RepoName,
		"previous_backup": backupPath,
	}
	if b, err := json.MarshalIndent(installMeta, "", "  "); err == nil {
		os.WriteFile(filepath.Join(filesystem.AgentPath(driveRoot), "install.json"), b, 0644)
	}

	fmt.Println()
	ui.Success("Successfully updated to %s", release.TagName)
	return nil
}

func newRollbackCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "rollback", Short: "Rollback to a previous version",
		RunE: runRollback,
	}
	cmd.Flags().String("backup", "", "Specific backup file to restore")
	cmd.Flags().Bool("list", false, "List available backups")
	cmd.Flags().Bool("yes", false, "Skip confirmation")
	return cmd
}

func runRollback(cmd *cobra.Command, args []string) error {
	autoYes, _ := cmd.Flags().GetBool("yes")
	listMode, _ := cmd.Flags().GetBool("list")
	backupName, _ := cmd.Flags().GetString("backup")

	ui.Header("Self-Rollback")

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("detect executable path: %w", err)
	}

	driveRoot, err := filesystem.FindDriveRoot(exe)
	if err != nil {
		return fmt.Errorf("detect drive root: %w", err)
	}

	backupsDir := filepath.Join(filesystem.AgentPath(driveRoot), "backups")
	entries, err := os.ReadDir(backupsDir)
	if err != nil || len(entries) == 0 {
		return fmt.Errorf("no backups found in %s", backupsDir)
	}

	var backups []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), "drive-agent-") {
			backups = append(backups, e.Name())
		}
	}
	sort.Strings(backups)

	if listMode {
		ui.SubHeader("Available Backups")
		for _, b := range backups {
			fmt.Println("  " + b)
		}
		return nil
	}

	var selected string
	if backupName != "" {
		for _, b := range backups {
			if b == backupName {
				selected = b
				break
			}
		}
		if selected == "" {
			return fmt.Errorf("backup %q not found", backupName)
		}
	} else {
		selected = backups[len(backups)-1]
	}

	ui.Label("Selected backup", selected)
	sourcePath := filepath.Join(backupsDir, selected)

	if !autoYes {
		if !ui.Confirm(fmt.Sprintf("\nRollback to %s?", selected), false) {
			fmt.Println("Aborted.")
			return nil
		}
	}

	// Backup current state just in case
	timestamp := time.Now().Format("20060102150405")
	failsafe := filepath.Join(backupsDir, fmt.Sprintf("drive-agent-failsafe-%s", timestamp))
	copyFile(exe, failsafe)

	// Perform rollback
	ui.Info("Restoring binary...")
	if err := copyFile(sourcePath, exe); err != nil {
		copyFile(failsafe, exe)
		return fmt.Errorf("failed to restore binary: %w", err)
	}
	os.Chmod(exe, 0755)

	// Update metadata
	installMeta := map[string]string{
		"installed_at":  time.Now().UTC().Format(time.RFC3339),
		"version":       "unknown", // could parse from backup name, but unknown is safer
		"method":        "self rollback",
		"install_path":  exe,
		"drive_root":    driveRoot,
		"source_backup": sourcePath,
		"os":            runtime.GOOS,
		"arch":          runtime.GOARCH,
		"repo_owner":    config.RepoOwner,
		"repo_name":     config.RepoName,
	}
	if b, err := json.MarshalIndent(installMeta, "", "  "); err == nil {
		os.WriteFile(filepath.Join(filesystem.AgentPath(driveRoot), "install.json"), b, 0644)
	}

	fmt.Println()
	ui.Success("Rollback complete.")
	return nil
}

// Helpers

func downloadString(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	return string(data), err
}

func downloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}

func verifyFileChecksum(path, expected string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}

	actual := fmt.Sprintf("%x", h.Sum(nil))
	if actual != expected {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expected, actual)
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

func determineAssetName(osName, archName string) string {
	if osName == "darwin" {
		osName = "Darwin"
	} else if osName == "windows" {
		osName = "Windows"
	} else {
		osName = "Linux"
	}

	if archName == "amd64" {
		archName = "x86_64"
	}

	ext := ".tar.gz"
	if osName == "Windows" {
		ext = ".zip"
	}

	return fmt.Sprintf("%s_%s_%s%s", config.RepoName, osName, archName, ext)
}

func parseChecksums(data, assetName string) (string, error) {
	for _, line := range strings.Split(data, "\n") {
		if strings.Contains(line, assetName) {
			fields := strings.Fields(line)
			if len(fields) > 0 {
				return fields[0], nil
			}
		}
	}
	return "", fmt.Errorf("checksum for %s not found in checksums.txt", assetName)
}

func extractArchive(archivePath, destPath string) error {
	if strings.HasSuffix(archivePath, ".zip") {
		return extractZip(archivePath, destPath)
	}
	return extractTarGz(archivePath, destPath)
}

func extractZip(archivePath, destPath string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		if strings.HasSuffix(f.Name, "drive-agent.exe") || strings.HasSuffix(f.Name, "drive-agent") {
			// Anti path-traversal check
			if strings.Contains(f.Name, "..") {
				return fmt.Errorf("invalid path in zip: %s", f.Name)
			}

			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer rc.Close()

			out, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer out.Close()

			_, err = io.Copy(out, rc)
			return err
		}
	}
	return fmt.Errorf("binary not found in zip archive")
}

func extractTarGz(archivePath, destPath string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if header.Typeflag == tar.TypeReg && (strings.HasSuffix(header.Name, "drive-agent") || strings.HasSuffix(header.Name, "drive-agent.exe")) {
			// Anti path-traversal check
			if strings.Contains(header.Name, "..") {
				return fmt.Errorf("invalid path in tar: %s", header.Name)
			}

			out, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			defer out.Close()

			_, err = io.Copy(out, tr)
			return err
		}
	}
	return fmt.Errorf("binary not found in tar.gz archive")
}
