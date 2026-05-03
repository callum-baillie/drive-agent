package host

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/callumbaillie/drive-agent/internal/config"
	"github.com/callumbaillie/drive-agent/internal/db"
	"github.com/callumbaillie/drive-agent/internal/filesystem"
	"github.com/callumbaillie/drive-agent/internal/packages/catalog"
	"github.com/callumbaillie/drive-agent/internal/packages/providers"
	"github.com/callumbaillie/drive-agent/internal/shell"
	"github.com/callumbaillie/drive-agent/internal/ui"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "host", Short: "Host setup and management"}
	cmd.AddCommand(newSetupCmd())
	cmd.AddCommand(newDoctorCmd())
	cmd.AddCommand(newPackagesCmd())
	return cmd
}

func newSetupCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "setup", Short: "Set up the current host", RunE: runSetup}
	cmd.Flags().String("profile", "", "Profile name (minimal, developer, ai-developer, full-stack-saas, mobile)")
	cmd.Flags().String("file", "", "Path to a profile JSON file")
	cmd.Flags().Bool("yes", false, "Skip confirmation prompts")
	cmd.Flags().Bool("dry-run", false, "Show planned actions without executing")
	return cmd
}

func runSetup(cmd *cobra.Command, args []string) error {
	driveRoot, err := filesystem.FindDriveRoot("")
	if err != nil {
		return fmt.Errorf("not inside a drive-agent managed drive: %w", err)
	}

	profileName, _ := cmd.Flags().GetString("profile")
	profileFile, _ := cmd.Flags().GetString("file")
	autoYes, _ := cmd.Flags().GetBool("yes")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	ui.Header("Drive Agent — Host Setup")

	// Detect host info
	hostInfo := config.HostInfo{
		HostID:   shell.HostID(),
		Hostname: shell.DetectHostname(),
		OS:       shell.DetectOS(),
		Arch:     shell.DetectArch(),
		Shell:    shell.DetectShell(),
	}

	ui.SubHeader("Detected Host")
	ui.Label("Hostname", hostInfo.Hostname)
	ui.Label("OS", hostInfo.OS)
	ui.Label("Arch", hostInfo.Arch)
	ui.Label("Shell", hostInfo.Shell)

	// Detect tools
	fmt.Println()
	ui.SubHeader("Installed Tools")
	toolChecks := []struct{ name, cmd string }{
		{"Homebrew", "brew"}, {"Git", "git"}, {"GitHub CLI", "gh"},
		{"Node.js", "node"}, {"npm", "npm"}, {"pnpm", "pnpm"},
		{"Docker", "docker"}, {"Python", "python3"},
	}
	for _, t := range toolChecks {
		ui.StatusLine(shell.IsCommandAvailable(t.cmd), t.name)
	}

	// Load profile if specified
	var profile *config.HostProfile
	if profileFile != "" {
		profile, err = loadProfileFromFile(profileFile)
		if err != nil {
			return fmt.Errorf("load profile file: %w", err)
		}
	} else if profileName != "" {
		profilePath := filepath.Join(driveRoot, "profiles", profileName+".json")
		profile, err = loadProfileFromFile(profilePath)
		if err != nil {
			return fmt.Errorf("load profile %q: %w", profileName, err)
		}
	}

	if profile != nil {
		fmt.Println()
		ui.SubHeader("Profile: " + profile.ProfileName)
		ui.Label("Categories", strings.Join(profile.Categories, ", "))
		ui.Label("Includes", strings.Join(profile.Packages.Include, ", "))
		if len(profile.Packages.Exclude) > 0 {
			ui.Label("Excludes", strings.Join(profile.Packages.Exclude, ", "))
		}
	}

	// Shell config setup
	fmt.Println()
	ui.SubHeader("Shell Configuration")
	shellConfig := shell.ShellConfigFile()
	binPath := filepath.Join(driveRoot, config.AgentDir, "bin")
	onPath := shell.IsOnPath(binPath)

	ui.StatusLine(onPath, "drive-agent on PATH")
	ui.Label("Shell config", shellConfig)

	if shellConfig != "" {
		// Check idempotency before prompting
		alreadyInstalled, _ := shell.ShellBlockAlreadyInstalled(shellConfig)
		if alreadyInstalled {
			ui.Success("Shell block already installed in %s (skipping)", filepath.Base(shellConfig))
		} else if !onPath {
			installShell := autoYes
			if !autoYes && !dryRun {
				installShell = ui.Confirm("\nInstall shell aliases and PATH?", true)
			}
			if dryRun {
				fmt.Println()
				ui.Info("Would add to %s:", shellConfig)
				fmt.Println(shell.ShellBlock(shell.ShellBlockContent(driveRoot)))
			} else if installShell {
				backupPath := shell.BackupPathFor(shellConfig)
				if err := shell.AppendShellBlock(shellConfig, driveRoot); err != nil {
					if errors.Is(err, shell.ErrShellBlockAlreadyPresent) {
						ui.Success("Shell block already installed (skipping)")
					} else {
						return fmt.Errorf("install shell config: %w", err)
					}
				} else {
					ui.Success("Backed up %s → %s", filepath.Base(shellConfig), filepath.Base(backupPath))
					ui.Success("Shell config updated — restart your shell or run: source %s", shellConfig)
				}
			}
		}
	}

	// Record host in database
	if !dryRun {
		database, err := db.Open(filesystem.DBPath(driveRoot))
		if err == nil {
			defer database.Close()
			now := config.NowISO()
			database.UpsertHost(&db.Host{
				ID: hostInfo.HostID, Hostname: hostInfo.Hostname,
				OS: hostInfo.OS, Arch: hostInfo.Arch, Shell: hostInfo.Shell,
				LastSeenAt: now, CreatedAt: now, UpdatedAt: now,
			})
		}

		// Save host state JSON
		stateData, _ := json.MarshalIndent(hostInfo, "", "  ")
		statePath := filesystem.HostStatePath(driveRoot, hostInfo.HostID)
		os.MkdirAll(filepath.Dir(statePath), 0755)
		os.WriteFile(statePath, stateData, 0644)
	}

	fmt.Println()
	ui.Success("Host setup complete")
	if dryRun {
		ui.DimText("(dry-run — no changes were made)")
	}
	return nil
}

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{Use: "doctor", Short: "Check host health", RunE: runHostDoctor}
}

func runHostDoctor(cmd *cobra.Command, args []string) error {
	driveRoot, err := filesystem.FindDriveRoot("")
	if err != nil {
		return err
	}

	ui.Header("Host Doctor")

	binPath := filepath.Join(driveRoot, config.AgentDir, "bin")
	ui.StatusLine(shell.IsOnPath(binPath), "drive-agent on PATH")
	ui.StatusLine(shell.IsCommandAvailable("git"), "Git")
	ui.StatusLine(shell.IsCommandAvailable("gh"), "GitHub CLI")
	ui.StatusLine(shell.IsCommandAvailable("brew"), "Homebrew")
	ui.StatusLine(shell.IsCommandAvailable("node"), "Node.js")
	ui.StatusLine(shell.IsCommandAvailable("npm"), "npm")
	ui.StatusLine(shell.IsCommandAvailable("pnpm"), "pnpm")
	ui.StatusLine(shell.IsCommandAvailable("bun"), "Bun")
	ui.StatusLine(shell.IsCommandAvailable("python3"), "Python 3")
	ui.StatusLine(shell.IsCommandAvailable("uv"), "uv")
	ui.StatusLine(shell.IsCommandAvailable("docker") || shell.IsCommandAvailable("orbctl"), "Docker/OrbStack")
	ui.StatusLine(shell.IsCommandAvailable("cursor"), "Cursor")
	ui.StatusLine(shell.IsCommandAvailable("code"), "VS Code")
	fmt.Println()
	return nil
}

func newPackagesCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "packages", Short: "Manage host packages"}
	cmd.AddCommand(newPackagesListCmd())
	cmd.AddCommand(newPackagesInstallCmd())
	return cmd
}

func newPackagesListCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "list", Short: "List available packages", RunE: runPackagesList}
	cmd.Flags().String("category", "", "Filter by category")
	return cmd
}

func runPackagesList(cmd *cobra.Command, args []string) error {
	driveRoot, err := filesystem.FindDriveRoot("")
	if err != nil {
		return err
	}
	catFilter, _ := cmd.Flags().GetString("category")

	catPath := filepath.Join(filesystem.CatalogPath(driveRoot), "packages.catalog.json")
	cat, err := catalog.LoadCatalog(catPath)
	if err != nil {
		// Try the repo-level catalog
		catPath = filepath.Join(driveRoot, "catalog", "packages.catalog.json")
		cat, err = catalog.LoadCatalog(catPath)
		if err != nil {
			return fmt.Errorf("load catalog: %w\nRun 'drive-agent init' first or ensure catalog exists.", err)
		}
	}

	if catFilter != "" {
		pkgs := cat.GetByCategory(catFilter)
		ui.Header("Packages — " + catFilter)
		for _, p := range pkgs {
			installed := ""
			if p.Check != nil {
				parts := strings.Fields(p.Check.Command)
				if len(parts) > 0 && shell.IsCommandAvailable(parts[0]) {
					installed = ui.Green + " (installed)" + ui.Reset
				}
			}
			fmt.Printf("  %-20s %s%s\n", p.ID, p.Description, installed)
		}
	} else {
		ui.Header("Package Categories")
		for _, c := range cat.Categories() {
			pkgs := cat.GetByCategory(c)
			fmt.Printf("  %-25s %d packages\n", c, len(pkgs))
		}
		fmt.Println()
		ui.DimText("Use --category <name> to list packages in a category")
	}
	fmt.Println()
	return nil
}

func newPackagesInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "install <packages...>", Short: "Install packages on the host",
		RunE: runPackagesInstall,
	}
	cmd.Flags().Bool("yes", false, "Skip confirmation")
	cmd.Flags().Bool("dry-run", false, "Show plan without installing")
	cmd.Flags().String("category", "", "Install all packages in categories (comma-separated)")
	return cmd
}

func runPackagesInstall(cmd *cobra.Command, args []string) error {
	driveRoot, err := filesystem.FindDriveRoot("")
	if err != nil {
		return err
	}
	autoYes, _ := cmd.Flags().GetBool("yes")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	catFlag, _ := cmd.Flags().GetString("category")

	// Load catalog
	catPath := filepath.Join(driveRoot, "catalog", "packages.catalog.json")
	cat, err := catalog.LoadCatalog(catPath)
	if err != nil {
		catPath = filepath.Join(filesystem.CatalogPath(driveRoot), "packages.catalog.json")
		cat, err = catalog.LoadCatalog(catPath)
		if err != nil {
			return fmt.Errorf("load catalog: %w", err)
		}
	}

	// Collect package IDs
	var packageIDs []string
	if catFlag != "" {
		for _, c := range strings.Split(catFlag, ",") {
			for _, p := range cat.GetByCategory(strings.TrimSpace(c)) {
				packageIDs = append(packageIDs, p.ID)
			}
		}
	}
	packageIDs = append(packageIDs, args...)

	if len(packageIDs) == 0 {
		return fmt.Errorf("specify packages or use --category")
	}

	registry := providers.NewRegistry()

	// Build install plan
	type installAction struct {
		pkg     *catalog.Package
		mgr     providers.Provider
		pkgName string
		cmd     string
	}
	var plan []installAction

	for _, id := range packageIDs {
		pkg := cat.GetPackage(id)
		if pkg == nil {
			ui.Warning("  Unknown package: %s (skipping)", id)
			continue
		}

		// Find best available manager
		var bestMgr providers.Provider
		var bestName string
		for _, mgrID := range pkg.InstallPreference {
			mgr, ok := registry.Get(mgrID)
			if ok && mgr.IsAvailable() {
				bestMgr = mgr
				bestName = pkg.GetInstallName(mgrID)
				break
			}
		}
		if bestMgr == nil {
			ui.Warning("  No available manager for: %s", id)
			continue
		}

		cmdStr, _ := bestMgr.InstallPackage(bestName, true)
		plan = append(plan, installAction{pkg: pkg, mgr: bestMgr, pkgName: bestName, cmd: cmdStr})
	}

	if len(plan) == 0 {
		ui.Info("Nothing to install.")
		return nil
	}

	// Show plan
	ui.Header("Install Plan")
	for _, a := range plan {
		fmt.Printf("  %s%-20s%s → %s\n", ui.Cyan, a.pkg.ID, ui.Reset, a.cmd)
	}
	fmt.Println()

	if dryRun {
		ui.DimText("(dry-run — no packages installed)")
		return nil
	}

	if !autoYes {
		if !ui.Confirm("Install these packages?", false) {
			fmt.Println("Aborted.")
			return nil
		}
	}

	// Execute installs
	installed, failed := 0, 0
	for _, a := range plan {
		if a.pkg.RequiresApproval {
			ui.Warning("  %s requires explicit approval — skipping auto-install", a.pkg.ID)
			continue
		}
		_, err := a.mgr.InstallPackage(a.pkgName, false)
		if err != nil {
			ui.Error("  Failed: %s: %v", a.pkg.ID, err)
			failed++
		} else {
			ui.Success("  Installed: %s", a.pkg.ID)
			installed++
		}
	}

	fmt.Println()
	ui.Info("Installed: %d, Failed: %d", installed, failed)
	return nil
}

func loadProfileFromFile(path string) (*config.HostProfile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var p config.HostProfile
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return &p, nil
}
