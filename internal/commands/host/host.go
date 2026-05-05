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
	cmd.Flags().String("docker-mode", "", "Docker storage mode override: prompt, default, bind-mounts, daemon-guidance")
	cmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompts")
	cmd.Flags().Bool("force", false, "Attempt package installs even when catalog checks indicate the tool/app is already installed")
	cmd.Flags().Bool("include-explicit", false, "Include packages marked requiresExplicitApproval when used with --yes")
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
	force, _ := cmd.Flags().GetBool("force")
	includeExplicit, _ := cmd.Flags().GetBool("include-explicit")
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
		pkgPlan := buildPackagePlanWithOptions(profile, cat, registry, packagePlanOptions{
			Force:           force,
			IncludeExplicit: includeExplicit,
		})
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
			if err := runProfileSetupPlan(pkgPlan, cacheActions, dockerActions, setupRunOptions{AutoYes: autoYes}); err != nil {
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
				fmt.Println(shell.ShellBlock(shell.ShellBlockContent(driveRoot)))
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
		if err := applyStorageShellBlock(shellConfig, shellOptions, dryRun, autoYes); err != nil {
			return err
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
			if isPackageCheckInstalled(&p) {
				installed = ui.Green + " (installed)" + ui.Reset
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
		displayName := packageDisplayName(action)
		switch {
		case action.Installed:
			detail := ""
			if action.InstalledDetail != "" {
				detail = " (" + action.InstalledDetail + ")"
			}
			fmt.Printf("  %s%-28s%s already installed%s\n", ui.Green, displayName, ui.Reset, detail)
		case action.SkipReason != "":
			fmt.Printf("  %s%-28s%s skipped: %s\n", ui.Yellow, displayName, ui.Reset, action.SkipReason)
		default:
			fmt.Printf("  %s%-28s%s [%s] %s\n", ui.Cyan, displayName, ui.Reset, action.ManagerID, action.Command)
		}
	}
}

func packageDisplayName(action packageAction) string {
	if strings.TrimSpace(action.Name) != "" {
		return action.Name
	}
	return action.ID
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

type setupRunOptions struct {
	AutoYes       bool
	Confirm       func(prompt string, defaultYes bool) bool
	CommandRunner func(name string, args ...string) (string, error)
}

type packageInstallFailure struct {
	Action packageAction
	Reason string
}

func runProfileSetupPlan(plan packagePlan, cacheActions, dockerActions []setupAction, opts setupRunOptions) error {
	if opts.Confirm == nil {
		opts.Confirm = ui.Confirm
	}
	if opts.CommandRunner == nil {
		opts.CommandRunner = shell.RunCommand
	}
	if !opts.AutoYes && !opts.Confirm("Apply profile package/cache/storage changes?", false) {
		fmt.Println("Skipped profile changes.")
		return nil
	}
	var failures []packageInstallFailure
	for _, action := range plan.Actions {
		if action.Installed || action.SkipReason != "" || action.Command == "" {
			continue
		}
		if !opts.AutoYes && !opts.Confirm(installPrompt(action), false) {
			continue
		}
		parts := strings.Fields(action.Command)
		if len(parts) == 0 {
			continue
		}
		out, err := opts.CommandRunner(parts[0], parts[1:]...)
		if err != nil {
			failure := packageInstallFailure{Action: action, Reason: failureReason(out, err)}
			printPackageInstallFailure(failure)
			failures = append(failures, failure)
			if !opts.AutoYes && !opts.Confirm("Continue with remaining package installs?", true) {
				return fmt.Errorf("failed to install %s", packageDisplayName(action))
			}
			continue
		}
		ui.Success("Installed %s", packageDisplayName(action))
	}
	if err := applySetupActions(cacheActions); err != nil {
		return err
	}
	if err := applySetupActions(dockerActions); err != nil {
		return err
	}
	if len(failures) > 0 {
		printPackageFailureSummary(failures)
		return fmt.Errorf("%d package install(s) failed", len(failures))
	}
	return nil
}

func installPrompt(action packageAction) string {
	name := packageDisplayName(action)
	switch action.ManagerID {
	case "npm", "pnpm", "bun":
		if action.InstallGlobal {
			return fmt.Sprintf("[%s] Install %s globally?", action.ManagerID, name)
		}
	case "homebrew", "homebrew-cask", "cargo", "go-install", "pipx", "uv":
		return fmt.Sprintf("[%s] Install %s?", action.ManagerID, name)
	}
	if action.ManagerID != "" {
		return fmt.Sprintf("[%s] Install %s?", action.ManagerID, name)
	}
	return "Install " + name + "?"
}

func failureReason(output string, err error) string {
	text := strings.TrimSpace(output)
	if text == "" && err != nil {
		text = err.Error()
	}
	if path := appBundleConflictPath(text); path != "" {
		return "app bundle already exists at " + path
	}
	if text == "" {
		return "unknown error"
	}
	return text
}

func appBundleConflictPath(text string) string {
	const marker = "already an App at '"
	idx := strings.Index(text, marker)
	if idx == -1 {
		return ""
	}
	rest := text[idx+len(marker):]
	end := strings.Index(rest, "'")
	if end == -1 {
		return ""
	}
	return rest[:end]
}

func printPackageInstallFailure(failure packageInstallFailure) {
	displayName := packageDisplayName(failure.Action)
	ui.Error("Failed to install %s via %s", displayName, failure.Action.ManagerID)
	fmt.Printf("  Reason: %s\n", failure.Reason)
	fmt.Println("  Suggested action: skip this package or use --force after reviewing manually.")
}

func printPackageFailureSummary(failures []packageInstallFailure) {
	fmt.Println()
	ui.Warning("%d package install(s) failed:", len(failures))
	for _, failure := range failures {
		fmt.Printf("  - %s via %s: %s\n", packageDisplayName(failure.Action), failure.Action.ManagerID, failure.Reason)
	}
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
	block := shell.ShellBlock(shell.ShellBlockContent(driveRoot))
	_, err = fmt.Fprintln(f, "\n"+block)
	return err
}

func applyStorageShellBlock(configPath string, options shell.ShellBlockOptions, dryRun, autoYes bool) error {
	if !shell.StorageShellBlockNeeded(options) {
		return nil
	}
	installed, err := shell.StorageShellBlockAlreadyInstalled(configPath)
	if err != nil {
		return fmt.Errorf("read storage shell config: %w", err)
	}
	if dryRun {
		fmt.Println()
		if installed {
			ui.Info("Would update storage exports in %s:", configPath)
		} else {
			ui.Info("Would add storage exports to %s:", configPath)
		}
		fmt.Println(shell.StorageShellBlock(shell.StorageShellBlockContent(options)))
		return nil
	}
	installStorage := autoYes
	if !autoYes {
		prompt := "\nInstall portable cache/container shell exports?"
		if installed {
			prompt = "\nUpdate portable cache/container shell exports?"
		}
		installStorage = ui.Confirm(prompt, true)
	}
	if !installStorage {
		ui.Info("Skipped portable cache/container shell exports")
		return nil
	}
	backupPath, changed, err := shell.AppendOrUpdateStorageShellBlock(configPath, options)
	if err != nil {
		return fmt.Errorf("install storage shell config: %w", err)
	}
	if !changed {
		ui.Success("Storage shell block already up to date in %s", filepath.Base(configPath))
		return nil
	}
	if backupPath != "" {
		ui.Success("Backed up %s → %s", filepath.Base(configPath), filepath.Base(backupPath))
	}
	ui.Success("Storage shell config updated — restart your shell or run: source %s", configPath)
	return nil
}
