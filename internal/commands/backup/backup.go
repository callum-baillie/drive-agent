package backup

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/callum-baillie/drive-agent/internal/shell"
	"github.com/callum-baillie/drive-agent/internal/ui"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Backup utilities",
		Long:  "Backup commands are MVP stubs. They report available tools and planned behavior, but do not yet create or run managed backups.",
	}
	cmd.AddCommand(&cobra.Command{Use: "status", Short: "Show backup status", Long: "Show installed backup tools. Managed backup execution is not implemented yet.", RunE: runStatus})
	cmd.AddCommand(&cobra.Command{Use: "init", Short: "Initialize backup configuration (planned)", Long: "Planned command. It currently prints the intended backup-provider direction without writing configuration.", RunE: runInit})
	cmd.AddCommand(&cobra.Command{Use: "run", Short: "Run a backup (planned)", Long: "Planned command. It currently does not execute backups.", RunE: runBackup})
	cmd.AddCommand(&cobra.Command{Use: "check", Short: "Verify backup integrity (planned)", Long: "Planned command. It currently does not run backup integrity checks.", RunE: runCheck})
	return cmd
}

func runStatus(cmd *cobra.Command, args []string) error {
	ui.Header("Backup Status")
	ui.SubHeader("Available Backup Tools")
	ui.StatusLine(shell.IsCommandAvailable("restic"), "restic")
	ui.StatusLine(shell.IsCommandAvailable("kopia"), "kopia")
	ui.StatusLine(shell.IsCommandAvailable("rclone"), "rclone")
	ui.StatusLine(shell.IsCommandAvailable("rsync"), "rsync")
	ui.StatusLine(shell.IsCommandAvailable("tmutil"), "Time Machine (tmutil)")

	fmt.Println()
	ui.Info("Backup providers are not yet fully implemented.")
	ui.DimText("To set up backups manually:")
	ui.DimText("  1. Install restic or kopia: brew install restic")
	ui.DimText("  2. Initialize a repository: restic init -r /path/to/backup")
	ui.DimText("  3. Run a backup: restic backup /Volumes/YourDrive --exclude node_modules")
	ui.DimText("  4. Verify: restic check -r /path/to/backup")
	fmt.Println()
	return nil
}

func runInit(cmd *cobra.Command, args []string) error {
	ui.Header("Backup Init")
	ui.Info("Backup initialization is planned for a future release.")
	ui.DimText("The backup system will support restic, kopia, rclone, and rsync providers.")
	fmt.Println()
	return nil
}

func runBackup(cmd *cobra.Command, args []string) error {
	ui.Header("Backup Run")
	ui.Info("Automated backup execution is planned for a future release.")
	fmt.Println()
	return nil
}

func runCheck(cmd *cobra.Command, args []string) error {
	ui.Header("Backup Check")
	ui.Info("Backup verification is planned for a future release.")
	fmt.Println()
	return nil
}
