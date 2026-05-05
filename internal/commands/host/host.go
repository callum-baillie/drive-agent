package host

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/callum-baillie/drive-agent/internal/config"
	"github.com/callum-baillie/drive-agent/internal/db"
	"github.com/callum-baillie/drive-agent/internal/filesystem"
	"github.com/callum-baillie/drive-agent/internal/packages/catalog"
	"github.com/callum-baillie/drive-agent/internal/packages/providers"
	"github.com/callum-baillie/drive-agent/internal/shell"
	"github.com/callum-baillie/drive-agent/internal/ui"
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
	cmd.Flags().String("profile", "", "Profile name (minimal, developer, ai-developer, full-stack-saas, mobile, or a drive-local host profile)")
	cmd.Flags().String("file", "", "Path to a profile JSON file")
	cmd.Flags().String("cache-mode", "", "Cache mode override: prompt, host-local, external-drive, disabled")
	cmd.Flags().String("docker-mode", "", "Docker storage mode override: prompt, default, bind-mounts, daemon")
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
	cacheModeFlag, _ := cmd.Flags().GetString("cache-mode")
	dockerModeFlag, _ := cmd.Flags().GetString("docker-mode")
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
		profilePath, err := resolveProfilePath(driveRoot, profileName)
		if err != nil {
			return err
		}
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

	var shellOptions shell.ShellBlockOptions
	if profile != nil {
		cat, err := loadCatalogForSetup(driveRoot)
		if err != nil {
			return fmt.Errorf("load package catalog: %w", err)
		}
		registry := providers.NewRegistry()
		pkgPlan := buildPackagePlan(profile, cat, registry)
		printPackageSetupPlan(pkgPlan)

		cacheMode := profile.Caches.Mode
		if cacheModeFlag != "" {
			cacheMode = cacheModeFlag
		}
		cacheMode, err = resolveInteractiveCacheMode(cacheMode, dryRun, autoYes)
		if err != nil {
			return err
		}
		cacheActions, cacheShellOptions, err := buildCachePlan(profile, driveRoot, cacheMode)
		if err != nil {
			return err
		}
		printSetupActions("Cache Plan", cacheActions)
		shellOptions = mergeShellBlockOptions(shellOptions, cacheShellOptions)

		dockerMode := profile.Docker.Mode
		if dockerModeFlag != "" {
			dockerMode = dockerModeFlag
		}
		dockerMode, err = resolveInteractiveDockerMode(dockerMode, dryRun, autoYes)
		if err != nil {
			return err
		}
		dockerActions, dockerShellOptions, err := buildDockerPlan(profile, driveRoot, dockerMode)
		if err != nil {
			return err
		}
		printSetupActions("Docker / Container Storage Plan", dockerActions)
		shellOptions = mergeShellBlockOptions(shellOptions, dockerShellOptions)

		if !dryRun {
			if err := runProfileSetupPlan(pkgPlan, cacheActions, dockerActions, autoYes); err != nil {
				return err
			}
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
				fmt.Println(shell.ShellBlock(shell.ShellBlockContentWithOptions(driveRoot, shellOptions)))
			} else if installShell {
				backupPath := shell.BackupPathFor(shellConfig)
				if err := appendShellBlockWithOptions(shellConfig, driveRoot, shellOptions); err != nil {
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
	if dryRun {
		ui.Success("Host setup plan complete")
		ui.DimText("(dry-run — no changes were made)")
	} else {
		ui.Success("Host setup complete")
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

func resolveProfilePath(driveRoot, profileName string) (string, error) {
	candidates := []string{
		filepath.Join(driveRoot, config.AgentDir, "config", "host-profiles", profileName+".json"),
		filepath.Join(driveRoot, "profiles", profileName+".json"),
	}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, "profiles", profileName+".json"))
	}
	for _, candidate := range candidates {
		if filesystem.Exists(candidate) {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("profile %q not found; checked %s", profileName, strings.Join(candidates, ", "))
}

func loadCatalogForSetup(driveRoot string) (*catalog.Catalog, error) {
	candidates := []string{
		filepath.Join(filesystem.CatalogPath(driveRoot), "packages.catalog.json"),
		filepath.Join(driveRoot, "catalog", "packages.catalog.json"),
	}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, "catalog", "packages.catalog.json"))
	}
	var lastErr error
	for _, candidate := range candidates {
		cat, err := catalog.LoadCatalog(candidate)
		if err == nil {
			return cat, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

func printPackageSetupPlan(plan packagePlan) {
	fmt.Println()
	ui.SubHeader("Package Manager Plan")
	for _, mgr := range plan.Managers {
		status := "not installed"
		if mgr.Available {
			status = strings.TrimSpace(strings.Join([]string{mgr.Path, mgr.Version}, " "))
		}
		ui.Label(mgr.ID, status)
	}

	fmt.Println()
	ui.SubHeader("Package Install Plan")
	for _, action := range plan.Actions {
		switch {
		case action.Installed:
			fmt.Printf("  %s%-24s%s already installed\n", ui.Green, action.ID, ui.Reset)
		case action.SkipReason != "":
			fmt.Printf("  %s%-24s%s skipped: %s\n", ui.Yellow, action.ID, ui.Reset, action.SkipReason)
		default:
			fmt.Printf("  %s%-24s%s %s\n", ui.Cyan, action.ID, ui.Reset, action.Command)
		}
	}
}

func printSetupActions(title string, actions []setupAction) {
	fmt.Println()
	ui.SubHeader(title)
	for _, action := range actions {
		if action.Command != "" {
			fmt.Printf("  %s%-34s%s %s\n", ui.Cyan, action.Title, ui.Reset, action.Command)
			if action.Current != "" || action.Planned != "" {
				fmt.Printf("  %-34s current: %s -> planned: %s\n", "", action.Current, action.Planned)
			}
			continue
		}
		if action.Current != "" {
			fmt.Printf("  %-34s current: %s -> planned: %s\n", action.Title, action.Current, action.Planned)
			continue
		}
		fmt.Printf("  %-34s %s\n", action.Title, action.Planned)
	}
}

func runProfileSetupPlan(plan packagePlan, cacheActions, dockerActions []setupAction, autoYes bool) error {
	if !autoYes && !ui.Confirm("Apply profile package/cache/storage changes?", false) {
		fmt.Println("Skipped profile changes.")
		return nil
	}
	for _, action := range plan.Actions {
		if action.Installed || action.SkipReason != "" || action.Command == "" {
			continue
		}
		if !autoYes && !ui.Confirm("Run "+action.Command+"?", false) {
			continue
		}
		parts := strings.Fields(action.Command)
		if len(parts) == 0 {
			continue
		}
		out, err := shell.RunCommand(parts[0], parts[1:]...)
		if err != nil {
			return fmt.Errorf("%s: %s", action.ID, out)
		}
		ui.Success("Installed %s", action.ID)
	}
	if err := applySetupActions(cacheActions); err != nil {
		return err
	}
	if err := applySetupActions(dockerActions); err != nil {
		return err
	}
	return nil
}

func resolveInteractiveCacheMode(mode string, dryRun, autoYes bool) (string, error) {
	normalized, err := normalizeCacheMode(mode)
	if err != nil {
		return "", err
	}
	if normalized != cacheModePrompt || dryRun {
		return normalized, nil
	}
	if autoYes {
		return cacheModeHostLocal, nil
	}
	idx, _ := ui.SelectOne("Cache configuration:", []string{
		"Keep host-local caches",
		"Use external drive caches",
		"Disable/avoid cache configuration changes",
	})
	switch idx {
	case 0:
		return cacheModeHostLocal, nil
	case 1:
		return cacheModeExternal, nil
	case 2:
		return cacheModeDisabled, nil
	default:
		return cacheModeHostLocal, nil
	}
}

func resolveInteractiveDockerMode(mode string, dryRun, autoYes bool) (string, error) {
	normalized, err := normalizeDockerMode(mode)
	if err != nil {
		return "", err
	}
	if normalized != dockerModePrompt || dryRun {
		return normalized, nil
	}
	if autoYes {
		return dockerModeDefault, nil
	}
	idx, _ := ui.SelectOne("Docker storage configuration:", []string{
		"Keep Docker's default storage",
		"Use external drive bind-mount roots",
		"Show daemon data-root/build-cache guidance only",
	})
	switch idx {
	case 0:
		return dockerModeDefault, nil
	case 1:
		return dockerModeBindMounts, nil
	case 2:
		return dockerModeDaemon, nil
	default:
		return dockerModeDefault, nil
	}
}

func appendShellBlockWithOptions(configPath, driveRoot string, options shell.ShellBlockOptions) error {
	installed, err := shell.ShellBlockAlreadyInstalled(configPath)
	if err != nil {
		return fmt.Errorf("read shell config: %w", err)
	}
	if installed {
		return shell.ErrShellBlockAlreadyPresent
	}
	backupPath := shell.BackupPathFor(configPath)
	if filesystem.Exists(configPath) {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return fmt.Errorf("backup shell config: %w", err)
		}
		if err := os.WriteFile(backupPath, data, 0644); err != nil {
			return fmt.Errorf("backup shell config: %w", err)
		}
	}
	f, err := os.OpenFile(configPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open shell config: %w", err)
	}
	defer f.Close()
	block := shell.ShellBlock(shell.ShellBlockContentWithOptions(driveRoot, options))
	_, err = fmt.Fprintln(f, "\n"+block)
	return err
}
