package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/callumbaillie/drive-agent/internal/config"
	"github.com/callumbaillie/drive-agent/internal/db"
	"github.com/callumbaillie/drive-agent/internal/filesystem"
	"github.com/callumbaillie/drive-agent/internal/shell"
	"github.com/callumbaillie/drive-agent/internal/ui"
	"github.com/callumbaillie/drive-agent/internal/utils"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show drive status summary",
		RunE:  runStatus,
	}
}

func runStatus(cmd *cobra.Command, args []string) error {
	driveRoot, err := filesystem.FindDriveRoot("")
	if err != nil {
		return fmt.Errorf("not inside a drive-agent managed drive: %w", err)
	}

	ui.Header("Drive Agent Status")

	// Drive info
	ui.SubHeader("Drive")
	ui.Label("Root", driveRoot)
	ui.Label("Version", config.Version)

	// Check free space
	if stat, err := os.Stat(driveRoot); err == nil {
		_ = stat // We'd need syscall for disk space, show path instead
	}

	// Database info
	dbPath := filesystem.DBPath(driveRoot)
	database, err := db.Open(dbPath)
	if err != nil {
		ui.Label("Database", "error: "+err.Error())
		return nil
	}
	defer database.Close()

	schemaVer, _ := database.SchemaVersion()
	ui.Label("Schema Version", fmt.Sprintf("%d", schemaVer))

	// Count orgs
	orgs, err := database.ListOrganizations()
	if err == nil {
		ui.Label("Organizations", fmt.Sprintf("%d", len(orgs)))
	}

	// Count projects
	projects, err := database.ListProjects("", "")
	if err == nil {
		ui.Label("Projects", fmt.Sprintf("%d", len(projects)))
	}

	// Host info
	fmt.Println()
	ui.SubHeader("Host")
	ui.Label("Hostname", shell.DetectHostname())
	ui.Label("OS", shell.DetectOS())
	ui.Label("Arch", shell.DetectArch())
	ui.Label("Shell", shell.DetectShell())

	// Check if drive-agent is on PATH
	binPath := filepath.Join(driveRoot, config.AgentDir, "bin")
	ui.Label("On PATH", fmt.Sprintf("%v", shell.IsOnPath(binPath)))

	// Quick Git summary
	if projects != nil && len(projects) > 0 {
		fmt.Println()
		ui.SubHeader("Git Quick Status")
		dirtyCount := 0
		for _, p := range projects {
			if filesystem.IsDir(filepath.Join(p.Path, ".git")) {
				out, err := shell.RunCommandInDir(p.Path, "git", "status", "--porcelain")
				if err == nil && strings.TrimSpace(out) != "" {
					dirtyCount++
				}
			}
		}
		ui.Label("Dirty repos", fmt.Sprintf("%d", dirtyCount))
	}

	// Cleanup estimate
	if projects != nil && len(projects) > 0 {
		fmt.Println()
		ui.SubHeader("Cleanup Estimate")
		var totalSize int64
		count := 0
		for _, p := range projects {
			for _, target := range config.CleanupTargets {
				targetPath := filepath.Join(p.Path, target)
				if filesystem.IsDir(targetPath) {
					size, _ := filesystem.DirSize(targetPath)
					totalSize += size
					count++
				}
			}
		}
		ui.Label("Reclaimable items", fmt.Sprintf("%d", count))
		ui.Label("Reclaimable size", utils.FormatBytes(totalSize))
	}

	fmt.Println()
	return nil
}

func newDoctorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Run health checks on the drive and host",
		RunE:  runDoctor,
	}
	cmd.Flags().String("path", "", "Path to drive root (optional)")
	return cmd
}

func runDoctor(cmd *cobra.Command, args []string) error {
	targetPath, _ := cmd.Flags().GetString("path")
	driveRoot, err := filesystem.FindDriveRoot(targetPath)
	if err != nil {
		return fmt.Errorf("not inside a drive-agent managed drive: %w", err)
	}

	ui.Header("Drive Agent Doctor")

	// Drive checks
	ui.SubHeader("Drive Checks")

	markerPath := filepath.Join(driveRoot, config.AgentDir, config.MarkerFile)
	ui.StatusLine(filesystem.Exists(markerPath), "Drive marker exists")

	agentDir := filepath.Join(driveRoot, config.AgentDir)
	ui.StatusLine(filesystem.IsDir(agentDir), "Agent directory exists")

	dbPath := filesystem.DBPath(driveRoot)
	ui.StatusLine(filesystem.Exists(dbPath), "Database exists")

	// Check required folders
	allFoldersExist := true
	for _, dir := range config.DriveLayout {
		if !filesystem.IsDir(filepath.Join(driveRoot, dir)) {
			allFoldersExist = false
			break
		}
	}
	ui.StatusLine(allFoldersExist, "All required folders exist")

	// Check writable
	testFile := filepath.Join(driveRoot, config.AgentDir, ".write-test")
	writable := false
	if err := os.WriteFile(testFile, []byte("test"), 0644); err == nil {
		os.Remove(testFile)
		writable = true
	}
	ui.StatusLine(writable, "Drive is writable")

	// Database checks
	fmt.Println()
	ui.SubHeader("Database Checks")

	database, err := db.Open(dbPath)
	if err != nil {
		ui.StatusLine(false, "Database opens: "+err.Error())
	} else {
		defer database.Close()
		ui.StatusLine(true, "Database opens successfully")

		schemaVer, err := database.SchemaVersion()
		ui.StatusLine(err == nil, fmt.Sprintf("Schema version: %d", schemaVer))

		// Check projects exist on disk
		projects, err := database.ListProjects("", "")
		if err == nil {
			missingCount := 0
			for _, p := range projects {
				if !filesystem.IsDir(p.Path) {
					missingCount++
				}
			}
			ui.StatusLine(missingCount == 0,
				fmt.Sprintf("Projects on disk: %d/%d", len(projects)-missingCount, len(projects)))

			// Check for orphan projects (folders without DB entries)
			orgs, _ := database.ListOrganizations()
			orphanCount := 0
			for _, o := range orgs {
				projDir := filepath.Join(o.Path, "projects")
				entries, err := os.ReadDir(projDir)
				if err != nil {
					continue
				}
				for _, entry := range entries {
					if !entry.IsDir() {
						continue
					}
					projPath := filepath.Join(projDir, entry.Name())
					if _, err := database.GetProjectByPath(projPath); err != nil {
						orphanCount++
					}
				}
			}
			if orphanCount > 0 {
				ui.WarnLine(fmt.Sprintf("Orphan project folders: %d (run 'project reindex' to fix)", orphanCount))
			} else {
				ui.StatusLine(true, "No orphan project folders")
			}
		}
	}

	// Host checks
	fmt.Println()
	ui.SubHeader("Host Checks")

	binPath := filepath.Join(driveRoot, config.AgentDir, "bin")
	ui.StatusLine(shell.IsOnPath(binPath), "drive-agent on PATH")

	ui.StatusLine(shell.IsCommandAvailable("git"), "Git available")
	ui.StatusLine(shell.IsCommandAvailable("gh"), "GitHub CLI available")
	ui.StatusLine(shell.IsCommandAvailable("brew"), "Homebrew available")
	ui.StatusLine(shell.IsCommandAvailable("node"), "Node.js available")
	ui.StatusLine(shell.IsCommandAvailable("docker") || shell.IsCommandAvailable("orbctl"), "Docker/OrbStack available")

	fmt.Println()
	return nil
}
