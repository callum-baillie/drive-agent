package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/callum-baillie/drive-agent/internal/commands/backup"
	"github.com/callum-baillie/drive-agent/internal/commands/cleanup"
	gitcmd "github.com/callum-baillie/drive-agent/internal/commands/git"
	"github.com/callum-baillie/drive-agent/internal/commands/host"
	initcmd "github.com/callum-baillie/drive-agent/internal/commands/init"
	"github.com/callum-baillie/drive-agent/internal/commands/org"
	"github.com/callum-baillie/drive-agent/internal/commands/project"
	"github.com/callum-baillie/drive-agent/internal/commands/self"
	"github.com/callum-baillie/drive-agent/internal/config"
)

// NewRootCmd creates the root cobra command.
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "drive-agent",
		Short: "Portable development drive manager",
		Long: `drive-agent is a portable development-drive management system that lives on an
external drive and helps configure, organize, maintain, and back up development
work across multiple host machines.`,
		Version:       config.Version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Set version template
	rootCmd.SetVersionTemplate(fmt.Sprintf("drive-agent v%s\n", config.Version))

	// Add command groups
	rootCmd.AddCommand(initcmd.NewCmd())
	rootCmd.AddCommand(newStatusCmd())
	rootCmd.AddCommand(newDoctorCmd())
	rootCmd.AddCommand(org.NewCmd())
	rootCmd.AddCommand(project.NewCmd())
	rootCmd.AddCommand(host.NewCmd())
	rootCmd.AddCommand(gitcmd.NewCmd())
	rootCmd.AddCommand(cleanup.NewCmd())
	rootCmd.AddCommand(backup.NewCmd())
	rootCmd.AddCommand(self.NewCmd())

	return rootCmd
}

// Execute runs the root command.
func Execute() {
	rootCmd := NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
