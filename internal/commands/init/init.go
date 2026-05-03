package init

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/spf13/cobra"

	"github.com/callum-baillie/drive-agent/internal/config"
	"github.com/callum-baillie/drive-agent/internal/db"
	"github.com/callum-baillie/drive-agent/internal/filesystem"
	"github.com/callum-baillie/drive-agent/internal/ui"
	"github.com/callum-baillie/drive-agent/internal/utils"
)

// NewCmd creates the init command.
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a drive for drive-agent",
		Long: `Initialize a drive for use with drive-agent. Creates the required directory
structure and database. This is a non-destructive operation — it will never
erase, format, or overwrite existing data on the drive.`,
		RunE: runInit,
	}

	cmd.Flags().String("path", "", "Path to initialize (defaults to current directory)")
	cmd.Flags().String("name", "", "Name for this drive")
	cmd.Flags().Bool("repair", false, "Repair an existing initialization")
	cmd.Flags().Bool("allow-non-volume-path", false, "Allow initialization outside /Volumes on macOS")
	cmd.Flags().Bool("non-interactive", false, "Skip interactive prompts")

	return cmd
}

func runInit(cmd *cobra.Command, args []string) error {
	targetPath, _ := cmd.Flags().GetString("path")
	driveName, _ := cmd.Flags().GetString("name")
	repair, _ := cmd.Flags().GetBool("repair")
	allowNonVolume, _ := cmd.Flags().GetBool("allow-non-volume-path")
	nonInteractive, _ := cmd.Flags().GetBool("non-interactive")

	// Default to current directory
	if targetPath == "" {
		var err error
		targetPath, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("get working directory: %w", err)
		}
	}

	// Resolve to absolute path
	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}
	targetPath = absPath

	ui.Header("Drive Agent Init")

	// === SAFETY CHECKS ===

	// Check for dangerous paths
	if dangerous, reason := utils.IsDangerousPath(targetPath); dangerous {
		return fmt.Errorf("refusing to initialize: %s", reason)
	}

	// Check if inside /Volumes on macOS
	if !allowNonVolume && !utils.IsInsideVolumes(targetPath) {
		return fmt.Errorf("path %q is not inside /Volumes. Use --allow-non-volume-path to override", targetPath)
	}

	// Check if path exists
	if !filesystem.Exists(targetPath) {
		return fmt.Errorf("path %q does not exist", targetPath)
	}

	if !filesystem.IsDir(targetPath) {
		return fmt.Errorf("path %q is not a directory", targetPath)
	}

	// Check if already initialized
	agentDir := filepath.Join(targetPath, config.AgentDir)
	markerFile := filepath.Join(agentDir, config.MarkerFile)
	if filesystem.Exists(markerFile) && !repair {
		return fmt.Errorf("drive is already initialized at %q. Use --repair to repair", targetPath)
	}

	// Default drive name from directory name
	if driveName == "" {
		driveName = filepath.Base(targetPath)
	}

	// Show plan
	ui.Info("Target path: %s", targetPath)
	ui.Info("Drive name:  %s", driveName)
	fmt.Println()

	// Confirm if interactive and path has existing files
	if !nonInteractive {
		entries, _ := os.ReadDir(targetPath)
		if len(entries) > 0 && !repair {
			ui.Warning("Target directory contains %d existing items", len(entries))
			ui.Info("Init is non-destructive — no existing files will be modified or deleted")
			if !ui.Confirm("Continue with initialization?", false) {
				fmt.Println("Aborted.")
				return nil
			}
		}
	}

	// === CREATE DIRECTORY STRUCTURE ===

	ui.SubHeader("Creating directory structure...")

	// Create .drive-agent and subdirectories
	for _, dir := range config.AgentLayout {
		dirPath := filepath.Join(agentDir, dir)
		if err := filesystem.EnsureDir(dirPath); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}
		ui.Success("  %s/%s", config.AgentDir, dir)
	}

	// Create top-level directories
	for _, dir := range config.DriveLayout {
		dirPath := filepath.Join(targetPath, dir)
		if err := filesystem.EnsureDir(dirPath); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}
		ui.Success("  %s", dir)
	}

	// === CREATE MARKER FILE ===
	if err := os.WriteFile(markerFile, []byte(targetPath+"\n"), 0644); err != nil {
		return fmt.Errorf("write marker file: %w", err)
	}
	ui.Success("  %s/%s", config.AgentDir, config.MarkerFile)

	// === WRITE VERSION FILE ===
	versionFile := filepath.Join(agentDir, config.VersionFile)
	if err := os.WriteFile(versionFile, []byte(config.Version+"\n"), 0644); err != nil {
		return fmt.Errorf("write version file: %w", err)
	}
	ui.Success("  %s/%s", config.AgentDir, config.VersionFile)

	// === WRITE DRIVE CONFIG ===
	now := config.NowISO()
	driveConfig := config.DriveConfig{
		DriveID:       fmt.Sprintf("drive-%s", time.Now().Format("2006-01-02")),
		Name:          driveName,
		SchemaVersion: 1,
		DefaultOrg:    "personal",
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	configPath := filesystem.ConfigPath(targetPath)
	if !filesystem.Exists(configPath) || repair {
		f, err := os.Create(configPath)
		if err != nil {
			return fmt.Errorf("create drive config: %w", err)
		}
		defer f.Close()
		if err := toml.NewEncoder(f).Encode(driveConfig); err != nil {
			return fmt.Errorf("write drive config: %w", err)
		}
		ui.Success("  %s/config/%s", config.AgentDir, config.DriveConfigFile)
	}

	// === INITIALIZE DATABASE ===
	fmt.Println()
	ui.SubHeader("Initializing database...")

	dbPath := filesystem.DBPath(targetPath)
	database, err := db.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer database.Close()

	if err := database.InitSchema(); err != nil {
		return fmt.Errorf("initialize schema: %w", err)
	}
	ui.Success("  Database schema initialized")

	// Insert drive record
	if err := database.InsertDrive(driveConfig.DriveID, driveConfig.Name, targetPath, now, now); err != nil {
		// Ignore duplicate errors on repair
		if !repair {
			ui.Warning("  Could not insert drive record: %v", err)
		}
	} else {
		ui.Success("  Drive record created")
	}

	// === DONE ===
	fmt.Println()
	ui.Header("Drive initialized successfully!")
	ui.Info("Drive root: %s", targetPath)
	ui.Info("Database:   %s", dbPath)
	fmt.Println()
	ui.DimText("Next steps:")
	ui.DimText("  drive-agent org add personal")
	ui.DimText("  drive-agent host setup")
	fmt.Println()

	return nil
}
