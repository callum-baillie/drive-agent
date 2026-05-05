package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/spf13/cobra"

	backupcore "github.com/callum-baillie/drive-agent/internal/backup"
	"github.com/callum-baillie/drive-agent/internal/backup/restic"
	"github.com/callum-baillie/drive-agent/internal/config"
	"github.com/callum-baillie/drive-agent/internal/db"
	"github.com/callum-baillie/drive-agent/internal/filesystem"
	"github.com/callum-baillie/drive-agent/internal/shell"
	"github.com/callum-baillie/drive-agent/internal/ui"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Backup utilities",
		Long: `Manage Drive Agent backups. Restic is the first implemented provider.

Drive Agent never stores plaintext backup passwords. Provide Restic credentials
through the current shell environment or a local secret manager, then run
backups with --dry-run first. Use backup restore --dry-run against a separate
target before trusting a repository.`,
	}
	cmd.AddCommand(newInitCmd())
	cmd.AddCommand(newStatusCmd())
	cmd.AddCommand(newRunCmd())
	cmd.AddCommand(newSnapshotsCmd())
	cmd.AddCommand(newCheckCmd())
	cmd.AddCommand(newRestoreCmd())
	cmd.AddCommand(newExcludesCmd())
	cmd.AddCommand(newDoctorCmd())
	return cmd
}

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize backup configuration",
		Long:  "Initialize a Restic repository and write .drive-agent/config/backup.json without storing plaintext passwords.",
		RunE:  runInit,
	}
	cmd.Flags().String("provider", "restic", "Backup provider")
	cmd.Flags().String("repo", "", "Restic repository path or URL")
	cmd.Flags().String("name", backupcore.DefaultRepo, "Repo name in backup config")
	cmd.Flags().Bool("allow-same-drive-repo", false, "Allow repository inside the source drive (not a real backup)")
	cmd.Flags().Bool("check-after-init", false, "Run restic check after successful init")
	cmd.Flags().Bool("dry-run", false, "Show the plan without initializing or writing config")
	cmd.Flags().Bool("yes", false, "Skip confirmation")
	return cmd
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show backup status",
		RunE:  runStatus,
	}
}

func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run a backup",
		Long:  "Run a Restic backup. Use --dry-run to show the command and generated exclude file without creating a snapshot.",
		RunE:  runBackup,
	}
	cmd.Flags().String("repo", "", "Configured repo name to use")
	cmd.Flags().Bool("dry-run", false, "Show or run Restic dry-run without creating a snapshot")
	cmd.Flags().StringArray("tag", nil, "Additional Restic tag")
	cmd.Flags().StringArray("exclude", nil, "Additional exclude pattern for this run")
	return cmd
}

func newSnapshotsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snapshots",
		Short: "List Restic snapshots",
		RunE:  runSnapshots,
	}
	cmd.Flags().String("repo", "", "Configured repo name to use")
	cmd.Flags().Bool("json", false, "Print snapshots as JSON")
	return cmd
}

func newCheckCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check Restic repository metadata",
		RunE:  runCheck,
	}
	cmd.Flags().String("repo", "", "Configured repo name to use")
	cmd.Flags().Bool("dry-run", false, "Show the command without running restic check")
	return cmd
}

func newRestoreCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restore",
		Short: "Restore a snapshot to a safe target",
		Long:  "Restore a Restic snapshot to a target directory. This never deletes target contents automatically and refuses active-drive/system paths.",
		RunE:  runRestore,
	}
	cmd.Flags().String("repo", "", "Configured repo name to use")
	cmd.Flags().String("snapshot", "", "Snapshot ID to restore, or latest")
	cmd.Flags().String("target", "", "Restore target directory")
	cmd.Flags().Bool("dry-run", false, "Show or run Restic restore dry-run without writing files")
	cmd.Flags().Bool("yes", false, "Skip confirmation")
	return cmd
}

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose backup configuration and host readiness",
		RunE:  runDoctor,
	}
}

func newExcludesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "excludes",
		Short: "Manage backup excludes",
		Long: `Manage backup excludes.

Without --project, patterns are global defaults stored in backup config and are
matched relative to the drive root. With --project org/project, patterns are
stored in that project's .drive-project.toml and scoped to the project path when
Drive Agent generates the Restic exclude file. Wildcards supported by Restic,
such as apps/*/node_modules and packages/*/dist, are preserved.`,
	}
	listCmd := &cobra.Command{Use: "list", Short: "List global or per-project excludes", RunE: runExcludesList}
	addCmd := &cobra.Command{Use: "add <pattern>", Short: "Add a global or per-project exclude pattern", Args: cobra.ExactArgs(1), RunE: runExcludesAdd}
	removeCmd := &cobra.Command{Use: "remove <pattern>", Short: "Remove a global or per-project exclude pattern", Args: cobra.ExactArgs(1), RunE: runExcludesRemove}
	for _, subcmd := range []*cobra.Command{listCmd, addCmd, removeCmd} {
		subcmd.Flags().String("project", "", "Project ref for project-level excludes, in org/project form")
	}
	cmd.AddCommand(listCmd)
	cmd.AddCommand(addCmd, removeCmd)
	return cmd
}

func runInit(cmd *cobra.Command, args []string) error {
	driveRoot, err := requireDriveRoot()
	if err != nil {
		return err
	}
	provider, _ := cmd.Flags().GetString("provider")
	repoPath, _ := cmd.Flags().GetString("repo")
	repoName, _ := cmd.Flags().GetString("name")
	allowSameDrive, _ := cmd.Flags().GetBool("allow-same-drive-repo")
	checkAfterInit, _ := cmd.Flags().GetBool("check-after-init")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	yes, _ := cmd.Flags().GetBool("yes")

	if provider != "restic" {
		return fmt.Errorf("unsupported backup provider %q (only restic is implemented)", provider)
	}
	if repoPath == "" {
		return fmt.Errorf("--repo is required")
	}

	ui.Header("Backup Init")
	if err := backupcore.ValidateDriveRoot(driveRoot); err != nil {
		return err
	}
	repoSafety, err := backupcore.ValidateRepository(driveRoot, repoPath, allowSameDrive)
	if err != nil {
		return err
	}
	for _, warning := range repoSafety.Warnings {
		ui.Warning(warning)
	}

	password := backupcore.DetectPasswordSource()
	if !password.Configured {
		ui.Warning(password.Warning)
		if !dryRun {
			return fmt.Errorf("restic init requires RESTIC_PASSWORD or RESTIC_PASSWORD_FILE")
		}
	} else {
		ui.Label("Password source", password.Source)
	}

	providerImpl := restic.NewProvider(driveRoot)
	if !dryRun && !providerImpl.IsInstalled() {
		return fmt.Errorf("restic is not installed; run: drive-agent host packages install restic")
	}

	repo := backupcore.Repository{
		Name:               repoName,
		Provider:           provider,
		Repository:         repoPath,
		AllowSameDriveRepo: allowSameDrive,
	}
	plan, err := providerImpl.Init(context.Background(), repo, true)
	if err != nil {
		return err
	}

	ui.Label("Drive root", driveRoot)
	ui.Label("Repository", backupcore.RedactSensitive(repoSafety.Description))
	ui.Label("Command", plan.Command)
	if dryRun {
		ui.DimText("(dry-run - no config written and restic init not run)")
		return nil
	}
	if !yes && !ui.Confirm("Initialize this Restic repository and write backup config?", false) {
		fmt.Println("Aborted.")
		return nil
	}
	plan, err = providerImpl.Init(context.Background(), repo, false)
	if err != nil {
		return err
	}

	cfg := backupcore.NewConfig(provider, repoName, repoPath, allowSameDrive)
	if err := backupcore.SaveConfig(driveRoot, cfg); err != nil {
		return err
	}
	if plan.LogPath != "" {
		ui.Label("Log", plan.LogPath)
	}
	ui.Success("Backup repository initialized")
	ui.Label("Config", backupcore.ConfigPath(driveRoot))

	if checkAfterInit {
		if _, err := providerImpl.Check(context.Background(), repo, false); err != nil {
			return fmt.Errorf("check after init: %w", err)
		}
		state, _ := backupcore.LoadState(driveRoot)
		repoState := state.Repo(repoName)
		repoState.LastCheckAt = config.NowISO()
		state.UpdateRepo(repoName, repoState)
		_ = backupcore.SaveState(driveRoot, state)
		ui.Success("Repository check passed")
	}
	return nil
}

func runStatus(cmd *cobra.Command, args []string) error {
	driveRoot, err := requireDriveRoot()
	if err != nil {
		return err
	}
	ui.Header("Backup Status")
	return printStatus(driveRoot)
}

func runDoctor(cmd *cobra.Command, args []string) error {
	driveRoot, err := requireDriveRoot()
	if err != nil {
		return err
	}
	ui.Header("Backup Doctor")
	return printStatus(driveRoot)
}

func runBackup(cmd *cobra.Command, args []string) error {
	driveRoot, err := requireDriveRoot()
	if err != nil {
		return err
	}
	cfg, repoName, repo, err := loadRepoFromFlags(cmd, driveRoot)
	if err != nil {
		return err
	}
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	extraTags, _ := cmd.Flags().GetStringArray("tag")
	extraExcludes, _ := cmd.Flags().GetStringArray("exclude")

	password := backupcore.DetectPasswordSource()
	if !password.Configured && !dryRun {
		return fmt.Errorf("%s", password.Warning)
	}
	excludes, err := backupcore.MergeExcludes(cfg.Excludes, nil)
	if err != nil {
		return err
	}
	projectExcludes, err := loadProjectExcludeSets(driveRoot)
	if err != nil {
		return err
	}
	excludes, err = backupcore.MergeProjectExcludes(excludes, projectExcludes)
	if err != nil {
		return err
	}
	excludes, err = backupcore.MergeExcludes(excludes, extraExcludes)
	if err != nil {
		return err
	}
	excludeFile := filepath.Join(filesystem.AgentPath(driveRoot), "state", "backup", "restic-excludes.txt")
	excludeFile, err = backupcore.WriteExcludeFile(driveRoot, excludes)
	if err != nil {
		return err
	}

	driveLabel := loadDriveLabel(driveRoot)
	req := backupcore.BackupRequest{
		DriveRoot:   driveRoot,
		DriveLabel:  driveLabel,
		HostLabel:   shell.DetectHostname(),
		Repo:        repo,
		ExcludeFile: excludeFile,
		ExtraTags:   extraTags,
		DryRun:      dryRun,
	}
	providerImpl := restic.NewProvider(driveRoot)
	plan, err := providerImpl.Backup(context.Background(), req)
	if err != nil {
		return err
	}
	ui.Header("Backup Run")
	ui.Label("Repository", backupcore.RedactSensitive(repo.Repository))
	ui.Label("Command", plan.Command)
	ui.Label("Exclude file", excludeFile)
	if dryRun {
		ui.DimText("(dry-run - no snapshot created)")
		return nil
	}

	snapshots, snapErr := providerImpl.Snapshots(context.Background(), repo)
	state, _ := backupcore.LoadState(driveRoot)
	repoState := state.Repo(repoName)
	repoState.LastSuccessfulBackupAt = config.NowISO()
	if snapErr == nil {
		repoState.SnapshotCount = len(snapshots)
		if len(snapshots) > 0 {
			repoState.LatestSnapshotID = snapshots[len(snapshots)-1].ID
			repoState.LastSnapshotID = snapshots[len(snapshots)-1].ID
		}
	}
	state.UpdateRepo(repoName, repoState)
	if err := backupcore.SaveState(driveRoot, state); err != nil {
		return err
	}
	if plan.LogPath != "" {
		ui.Label("Log", plan.LogPath)
	}
	ui.Success("Backup completed")
	return nil
}

func runSnapshots(cmd *cobra.Command, args []string) error {
	driveRoot, err := requireDriveRoot()
	if err != nil {
		return err
	}
	_, repoName, repo, err := loadRepoFromFlags(cmd, driveRoot)
	if err != nil {
		return err
	}
	jsonOut, _ := cmd.Flags().GetBool("json")
	if password := backupcore.DetectPasswordSource(); !password.Configured {
		return fmt.Errorf("%s", password.Warning)
	}
	snapshots, err := restic.NewProvider(driveRoot).Snapshots(context.Background(), repo)
	if err != nil {
		return err
	}
	state, _ := backupcore.LoadState(driveRoot)
	repoState := state.Repo(repoName)
	repoState.SnapshotCount = len(snapshots)
	if len(snapshots) > 0 {
		repoState.LatestSnapshotID = snapshots[len(snapshots)-1].ID
	}
	state.UpdateRepo(repoName, repoState)
	_ = backupcore.SaveState(driveRoot, state)

	if jsonOut {
		data, err := json.MarshalIndent(snapshots, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}
	ui.Header("Backup Snapshots")
	if len(snapshots) == 0 {
		ui.Info("No snapshots found.")
		return nil
	}
	for _, snapshot := range snapshots {
		fmt.Printf("  %-12s %-24s %-24s %s\n", snapshot.ID, snapshot.Time, snapshot.Hostname, strings.Join(snapshot.Tags, ", "))
	}
	return nil
}

func runCheck(cmd *cobra.Command, args []string) error {
	driveRoot, err := requireDriveRoot()
	if err != nil {
		return err
	}
	_, repoName, repo, err := loadRepoFromFlags(cmd, driveRoot)
	if err != nil {
		return err
	}
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if password := backupcore.DetectPasswordSource(); !password.Configured && !dryRun {
		return fmt.Errorf("%s", password.Warning)
	}
	plan, err := restic.NewProvider(driveRoot).Check(context.Background(), repo, dryRun)
	if err != nil {
		return err
	}
	ui.Header("Backup Check")
	ui.Label("Command", plan.Command)
	if dryRun {
		ui.DimText("(dry-run - restic check not run)")
		return nil
	}
	state, _ := backupcore.LoadState(driveRoot)
	repoState := state.Repo(repoName)
	repoState.LastCheckAt = config.NowISO()
	state.UpdateRepo(repoName, repoState)
	_ = backupcore.SaveState(driveRoot, state)
	if plan.LogPath != "" {
		ui.Label("Log", plan.LogPath)
	}
	ui.Success("Repository check passed")
	return nil
}

func runRestore(cmd *cobra.Command, args []string) error {
	driveRoot, err := requireDriveRoot()
	if err != nil {
		return err
	}
	_, _, repo, err := loadRepoFromFlags(cmd, driveRoot)
	if err != nil {
		return err
	}
	snapshot, _ := cmd.Flags().GetString("snapshot")
	target, _ := cmd.Flags().GetString("target")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	yes, _ := cmd.Flags().GetBool("yes")
	if snapshot == "" {
		return fmt.Errorf("--snapshot is required (use latest or a snapshot ID)")
	}
	if target == "" {
		return fmt.Errorf("--target is required")
	}
	if err := backupcore.ValidateRestoreTarget(driveRoot, target); err != nil {
		return err
	}
	absTarget, _ := filepath.Abs(target)
	empty, err := backupcore.IsDirEmpty(absTarget)
	if err != nil {
		return fmt.Errorf("inspect restore target: %w", err)
	}
	if !empty {
		ui.Warning("Restore target is not empty. Drive Agent will not delete existing contents.")
	}
	if password := backupcore.DetectPasswordSource(); !password.Configured && !dryRun {
		return fmt.Errorf("%s", password.Warning)
	}
	providerImpl := restic.NewProvider(driveRoot)
	if err := restic.ValidateSnapshot(snapshot); err != nil {
		return err
	}
	plan := backupcore.Plan{
		Command: backupcore.RedactSensitive(restic.FormatCommand("restic", restic.RestoreArgs(repo.Repository, snapshot, absTarget, dryRun))),
	}
	ui.Header("Backup Restore")
	ui.Label("Snapshot", snapshot)
	ui.Label("Target", absTarget)
	ui.Label("Command", plan.Command)
	if dryRun {
		ui.DimText("(dry-run - no files restored)")
		return nil
	}
	if !yes && !ui.Confirm("Restore snapshot to this target?", false) {
		fmt.Println("Aborted.")
		return nil
	}
	req := backupcore.RestoreRequest{Repo: repo, Snapshot: snapshot, Target: absTarget, DryRun: false}
	plan, err = providerImpl.Restore(context.Background(), req)
	if err != nil {
		return err
	}
	if plan.LogPath != "" {
		ui.Label("Log", plan.LogPath)
	}
	ui.Success("Restore completed")
	return nil
}

func runExcludesList(cmd *cobra.Command, args []string) error {
	driveRoot, err := requireDriveRoot()
	if err != nil {
		return err
	}
	projectRef, _ := cmd.Flags().GetString("project")
	if projectRef != "" {
		project, err := resolveProjectRef(driveRoot, projectRef)
		if err != nil {
			return err
		}
		excludes, err := backupcore.LoadProjectManifestExcludes(project.Path)
		if err != nil {
			return err
		}
		ui.Header("Project Backup Excludes")
		ui.Label("Project", project.OrgSlug+"/"+project.Slug)
		ui.Label("Manifest", filepath.Join(project.Path, config.ProjectManifest))
		for _, pattern := range excludes {
			fmt.Println("  " + pattern)
		}
		return nil
	}
	cfg, err := backupcore.LoadConfig(driveRoot)
	if err != nil {
		return err
	}
	ui.Header("Backup Excludes")
	for _, pattern := range cfg.Excludes {
		fmt.Println("  " + pattern)
	}
	return nil
}

func runExcludesAdd(cmd *cobra.Command, args []string) error {
	driveRoot, err := requireDriveRoot()
	if err != nil {
		return err
	}
	projectRef, _ := cmd.Flags().GetString("project")
	if projectRef != "" {
		project, err := resolveProjectRef(driveRoot, projectRef)
		if err != nil {
			return err
		}
		if _, err := backupcore.AddProjectExclude(project.Path, args[0]); err != nil {
			return err
		}
		ui.Success("Project exclude configured for %s/%s: %s", project.OrgSlug, project.Slug, args[0])
		return nil
	}
	cfg, err := backupcore.LoadConfig(driveRoot)
	if err != nil {
		return err
	}
	cfg.Excludes, _, err = backupcore.AddExclude(cfg.Excludes, args[0])
	if err != nil {
		return err
	}
	cfg.UpdatedAt = config.NowISO()
	if err := backupcore.SaveConfig(driveRoot, cfg); err != nil {
		return err
	}
	ui.Success("Exclude configured: %s", args[0])
	return nil
}

func runExcludesRemove(cmd *cobra.Command, args []string) error {
	driveRoot, err := requireDriveRoot()
	if err != nil {
		return err
	}
	projectRef, _ := cmd.Flags().GetString("project")
	if projectRef != "" {
		project, err := resolveProjectRef(driveRoot, projectRef)
		if err != nil {
			return err
		}
		if _, err := backupcore.RemoveProjectExclude(project.Path, args[0]); err != nil {
			return err
		}
		ui.Success("Project exclude removed if present for %s/%s: %s", project.OrgSlug, project.Slug, args[0])
		return nil
	}
	cfg, err := backupcore.LoadConfig(driveRoot)
	if err != nil {
		return err
	}
	cfg.Excludes, _, err = backupcore.RemoveExclude(cfg.Excludes, args[0])
	if err != nil {
		return err
	}
	cfg.UpdatedAt = config.NowISO()
	if err := backupcore.SaveConfig(driveRoot, cfg); err != nil {
		return err
	}
	ui.Success("Exclude removed if present: %s", args[0])
	return nil
}

func loadProjectExcludeSets(driveRoot string) ([]backupcore.ProjectExcludeSet, error) {
	database, err := db.Open(filesystem.DBPath(driveRoot))
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	defer database.Close()

	projects, err := database.ListProjects("", "")
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	var out []backupcore.ProjectExcludeSet
	for _, project := range projects {
		manifestPath := filepath.Join(project.Path, config.ProjectManifest)
		if _, err := os.Stat(manifestPath); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("inspect project manifest: %w", err)
		}
		excludes, err := backupcore.LoadProjectManifestExcludes(project.Path)
		if err != nil {
			return nil, err
		}
		if len(excludes) == 0 {
			continue
		}
		out = append(out, backupcore.ProjectExcludeSet{
			OrgSlug:     project.OrgSlug,
			ProjectSlug: project.Slug,
			ProjectPath: project.Path,
			Patterns:    excludes,
		})
	}
	return out, nil
}

func resolveProjectRef(driveRoot, ref string) (*db.Project, error) {
	database, err := db.Open(filesystem.DBPath(driveRoot))
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	defer database.Close()

	if strings.Contains(ref, "/") {
		parts := strings.SplitN(ref, "/", 2)
		if parts[0] == "" || parts[1] == "" {
			return nil, fmt.Errorf("project must use org/project format")
		}
		project, err := database.GetProjectBySlug(parts[0], parts[1])
		if err != nil {
			return nil, fmt.Errorf("project not found: %s", ref)
		}
		return project, nil
	}

	projects, err := database.ListProjects("", "")
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	var matches []*db.Project
	for _, project := range projects {
		if project.Slug == ref || project.Name == ref {
			matches = append(matches, project)
		}
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("project not found: %s", ref)
	case 1:
		return matches[0], nil
	default:
		var refs []string
		for _, project := range matches {
			refs = append(refs, project.OrgSlug+"/"+project.Slug)
		}
		return nil, fmt.Errorf("project %q is ambiguous; use org/project (%s)", ref, strings.Join(refs, ", "))
	}
}

func printStatus(driveRoot string) error {
	cfg, cfgErr := backupcore.LoadConfig(driveRoot)
	state, _ := backupcore.LoadState(driveRoot)
	providerImpl := restic.NewProvider(driveRoot)
	password := backupcore.DetectPasswordSource()

	ui.Label("Source drive", driveRoot)
	ui.Label("Restic installed", yesNo(providerImpl.IsInstalled()))
	if password.Configured {
		ui.Label("Password source", password.Source)
	} else {
		ui.Label("Password source", "missing")
		ui.Warning(password.Warning)
	}
	if cfgErr != nil {
		ui.Warning("Backup config unavailable: %v", cfgErr)
		ui.Warning("Backup config not initialized. Run: drive-agent backup init --provider restic --repo <repo>")
		if !providerImpl.IsInstalled() {
			ui.Warning("Restic missing. Run: drive-agent host packages install restic")
		}
		return nil
	}

	repo, err := cfg.SelectedRepository("")
	if err != nil {
		return err
	}
	ui.Label("Provider", cfg.Provider)
	ui.Label("Selected repo", cfg.SelectedRepo)
	ui.Label("Repo path", backupcore.RedactSensitive(repo.Repository))
	ui.Label("Exclude count", fmt.Sprintf("%d", len(cfg.Excludes)))

	repoSafety, err := backupcore.ValidateRepository(driveRoot, repo.Repository, repo.AllowSameDriveRepo)
	if err != nil {
		ui.Warning("Repository warning: %v", err)
	} else if repoSafety.SameDrive {
		ui.Warning("Repository is on the same drive. This is not a real backup.")
	}
	if !providerImpl.IsInstalled() {
		ui.Warning("Restic missing. Run: drive-agent host packages install restic")
	}

	repoState := state.Repo(cfg.SelectedRepo)
	ui.Label("Last backup", valueOrNone(repoState.LastSuccessfulBackupAt))
	ui.Label("Last check", valueOrNone(repoState.LastCheckAt))
	ui.Label("Snapshot count", fmt.Sprintf("%d", repoState.SnapshotCount))
	ui.Label("Latest snapshot", valueOrNone(repoState.LatestSnapshotID))

	if providerImpl.IsInstalled() && password.Configured {
		snapshots, err := providerImpl.Snapshots(context.Background(), repo)
		if err != nil {
			ui.Warning("Repository unreachable or snapshots unavailable: %v", err)
		} else {
			ui.Label("Snapshot count (live)", fmt.Sprintf("%d", len(snapshots)))
			if len(snapshots) > 0 {
				ui.Label("Latest snapshot (live)", snapshots[len(snapshots)-1].ID)
			}
		}
	}
	return nil
}

func requireDriveRoot() (string, error) {
	driveRoot, err := filesystem.FindDriveRoot("")
	if err != nil {
		return "", fmt.Errorf("not inside a drive-agent managed drive: %w", err)
	}
	if err := backupcore.ValidateDriveRoot(driveRoot); err != nil {
		return "", err
	}
	return driveRoot, nil
}

func loadRepoFromFlags(cmd *cobra.Command, driveRoot string) (*backupcore.Config, string, backupcore.Repository, error) {
	cfg, err := backupcore.LoadConfig(driveRoot)
	if err != nil {
		return nil, "", backupcore.Repository{}, err
	}
	repoName, _ := cmd.Flags().GetString("repo")
	repo, err := cfg.SelectedRepository(repoName)
	if err != nil {
		return nil, "", backupcore.Repository{}, err
	}
	if repoName == "" {
		repoName = cfg.SelectedRepo
	}
	if _, err := backupcore.ValidateRepository(driveRoot, repo.Repository, repo.AllowSameDriveRepo); err != nil {
		return nil, "", backupcore.Repository{}, err
	}
	return cfg, repoName, repo, nil
}

func loadDriveLabel(driveRoot string) string {
	var driveConfig config.DriveConfig
	if _, err := toml.DecodeFile(filesystem.ConfigPath(driveRoot), &driveConfig); err == nil {
		if driveConfig.Name != "" {
			return driveConfig.Name
		}
		if driveConfig.DriveID != "" {
			return driveConfig.DriveID
		}
	}
	return filepath.Base(driveRoot)
}

func yesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

func valueOrNone(value string) string {
	if value == "" {
		return "None"
	}
	return value
}
