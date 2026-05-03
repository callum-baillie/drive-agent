package cleanup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/callum-baillie/drive-agent/internal/config"
	"github.com/callum-baillie/drive-agent/internal/db"
	"github.com/callum-baillie/drive-agent/internal/filesystem"
	"github.com/callum-baillie/drive-agent/internal/ui"
	"github.com/callum-baillie/drive-agent/internal/utils"
)

type cleanupTarget struct {
	Path string
	Size int64
	Name string
}

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "cleanup", Short: "Scan and clean build artifacts"}
	cmd.AddCommand(newScanCmd())
	cmd.AddCommand(newApplyCmd())
	cmd.AddCommand(newDryRunCmd())
	return cmd
}

func newScanCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "scan", Short: "Scan for removable files", RunE: runScan}
	cmd.Flags().String("org", "", "Filter by org")
	return cmd
}

// newDryRunCmd is an alias for scan — it MUST register its own --org flag because
// Cobra flags are per-command; the flag from sibling scan command is not inherited.
func newDryRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dry-run",
		Short: "Show cleanup plan without deleting (alias for scan)",
		RunE:  runScan,
	}
	cmd.Flags().String("org", "", "Filter by org")
	return cmd
}

func newApplyCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "apply", Short: "Apply cleanup (delete targets)", RunE: runApply}
	cmd.Flags().Bool("yes", false, "Skip confirmation")
	cmd.Flags().String("org", "", "Filter by org")
	return cmd
}

func findTargets(orgFilter string) ([]cleanupTarget, string, error) {
	driveRoot, err := filesystem.FindDriveRoot("")
	if err != nil {
		return nil, "", err
	}

	database, err := db.Open(filesystem.DBPath(driveRoot))
	if err != nil {
		return nil, driveRoot, err
	}
	defer database.Close()

	projects, err := database.ListProjects(orgFilter, "")
	if err != nil {
		return nil, driveRoot, err
	}

	var targets []cleanupTarget
	for _, p := range projects {
		for _, target := range config.CleanupTargets {
			targetPath := filepath.Join(p.Path, target)

			// Safety: skip symlinks (Lstat does not follow the link)
			if utils.IsSymlink(targetPath) {
				continue
			}

			// Safety: must be inside drive root (uses EvalSymlinks internally)
			if !utils.IsPathInsideDrive(targetPath, driveRoot) {
				continue
			}

			// Use Lstat to avoid following symlinks when reading info
			info, err := os.Lstat(targetPath)
			if err != nil {
				continue
			}

			// Only delete directories or regular files — never devices, sockets, etc.
			if info.IsDir() {
				size, _ := filesystem.DirSize(targetPath)
				targets = append(targets, cleanupTarget{
					Path: targetPath, Size: size,
					Name: fmt.Sprintf("%s/%s/%s", p.OrgSlug, p.Slug, target),
				})
			} else if info.Mode().IsRegular() {
				// Handle files like .DS_Store
				targets = append(targets, cleanupTarget{
					Path: targetPath, Size: info.Size(),
					Name: fmt.Sprintf("%s/%s/%s", p.OrgSlug, p.Slug, target),
				})
			}
		}
	}
	return targets, driveRoot, nil
}

func runScan(cmd *cobra.Command, args []string) error {
	orgFilter, _ := cmd.Flags().GetString("org")
	targets, _, err := findTargets(orgFilter)
	if err != nil {
		return err
	}

	if len(targets) == 0 {
		ui.Info("No cleanup targets found.")
		return nil
	}

	ui.Header("Cleanup Scan")
	var total int64
	for i, t := range targets {
		fmt.Printf("  %d. %-50s %s\n", i+1, t.Name, utils.FormatBytes(t.Size))
		total += t.Size
	}
	fmt.Println()
	fmt.Printf("  %sTotal reclaimable: %s%s\n", ui.Bold, utils.FormatBytes(total), ui.Reset)
	fmt.Println()
	ui.DimText("Run 'drive-agent cleanup apply' to delete these.")
	return nil
}

func runApply(cmd *cobra.Command, args []string) error {
	orgFilter, _ := cmd.Flags().GetString("org")
	yes, _ := cmd.Flags().GetBool("yes")
	targets, driveRoot, err := findTargets(orgFilter)
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		ui.Info("Nothing to clean up.")
		return nil
	}

	// Always show sizes before deletion
	ui.Header("Cleanup Plan")
	var total int64
	for i, t := range targets {
		fmt.Printf("  %d. %-50s %s\n", i+1, t.Name, utils.FormatBytes(t.Size))
		total += t.Size
	}
	fmt.Println()
	fmt.Printf("  %sTotal to delete: %s%s\n", ui.Bold, utils.FormatBytes(total), ui.Reset)
	fmt.Println()

	if !yes {
		if !ui.Confirm("Delete these items?", false) {
			fmt.Println("Aborted.")
			return nil
		}
	}

	deleted, failed := 0, 0
	for _, t := range targets {
		// --- Multi-layer safety checks ---

		// 1. Re-check symlink at delete time (the file system may have changed)
		if utils.IsSymlink(t.Path) {
			ui.Warning("  Skipping symlink: %s", t.Path)
			continue
		}

		// 2. Normalize path — filepath.Clean never introduces new path components
		cleanPath := filepath.Clean(t.Path)

		// 3. Reject any path still containing ".." after cleaning
		if strings.Contains(cleanPath, "..") {
			ui.Error("  SAFETY: suspicious path refused: %s", cleanPath)
			failed++
			continue
		}

		// 4. Re-validate the cleaned path is inside the drive root (catches EvalSymlinks escapes)
		if !utils.IsPathInsideDrive(cleanPath, driveRoot) {
			ui.Error("  SAFETY: refusing to delete outside drive root: %s", cleanPath)
			failed++
			continue
		}

		if err := os.RemoveAll(cleanPath); err != nil {
			ui.Error("  Failed: %s: %v", t.Name, err)
			failed++
		} else {
			ui.Success("  Deleted: %s (%s)", t.Name, utils.FormatBytes(t.Size))
			deleted++
		}
	}

	fmt.Println()
	ui.Info("Deleted: %d, Failed: %d", deleted, failed)
	return nil
}
