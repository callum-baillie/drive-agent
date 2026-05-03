package git

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/callum-baillie/drive-agent/internal/db"
	"github.com/callum-baillie/drive-agent/internal/filesystem"
	"github.com/callum-baillie/drive-agent/internal/shell"
	"github.com/callum-baillie/drive-agent/internal/ui"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "git", Short: "Git utilities across projects"}
	cmd.AddCommand(newStatusAllCmd())
	cmd.AddCommand(newFetchAllCmd())
	cmd.AddCommand(newPullAllCmd())
	return cmd
}

func getProjects(orgFilter, tagFilter string) ([]*db.Project, string, error) {
	driveRoot, err := filesystem.FindDriveRoot("")
	if err != nil {
		return nil, "", err
	}
	database, err := db.Open(filesystem.DBPath(driveRoot))
	if err != nil {
		return nil, "", err
	}
	defer database.Close()
	projects, err := database.ListProjects(orgFilter, tagFilter)
	return projects, driveRoot, err
}

// hasUpstream returns true if the repo has a configured remote tracking branch for HEAD.
func hasUpstream(dir string) bool {
	// git rev-parse --abbrev-ref --symbolic-full-name @{u} exits non-zero if no upstream
	_, err := shell.RunCommandInDir(dir, "git", "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
	return err == nil
}

func newStatusAllCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "status-all", Short: "Show git status for all projects", RunE: runStatusAll}
	cmd.Flags().String("org", "", "Filter by organization")
	cmd.Flags().String("tag", "", "Filter by tag")
	return cmd
}

func runStatusAll(cmd *cobra.Command, args []string) error {
	orgFilter, _ := cmd.Flags().GetString("org")
	tagFilter, _ := cmd.Flags().GetString("tag")
	projects, _, err := getProjects(orgFilter, tagFilter)
	if err != nil {
		return err
	}

	ui.Header("Git Status — All Projects")
	clean, dirty, skipped := 0, 0, 0

	for _, p := range projects {
		gitDir := filepath.Join(p.Path, ".git")
		if !filesystem.IsDir(gitDir) {
			skipped++
			continue
		}

		out, err := shell.RunCommandInDir(p.Path, "git", "status", "--porcelain")
		if err != nil {
			ui.Warning("  %s/%s: error running git status: %v", p.OrgSlug, p.Slug, err)
			skipped++
			continue
		}

		branch, _ := shell.RunCommandInDir(p.Path, "git", "branch", "--show-current")
		branch = strings.TrimSpace(branch)
		if branch == "" {
			branch = "(detached)"
		}

		if strings.TrimSpace(out) == "" {
			clean++
			fmt.Printf("  %s%s%s %s/%s %s[%s]%s\n",
				ui.Green, ui.SymbolCheck, ui.Reset, p.OrgSlug, p.Slug, ui.Dim, branch, ui.Reset)
		} else {
			dirty++
			changes := len(strings.Split(strings.TrimSpace(out), "\n"))
			fmt.Printf("  %s%s%s %s/%s %s[%s]%s — %d changed file(s)\n",
				ui.Yellow, ui.SymbolWarn, ui.Reset, p.OrgSlug, p.Slug, ui.Dim, branch, ui.Reset, changes)
		}
	}

	fmt.Println()
	ui.Info("Clean: %d, Dirty: %d, Skipped (not git or error): %d", clean, dirty, skipped)
	return nil
}

func newFetchAllCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "fetch-all", Short: "Fetch all projects", RunE: runFetchAll}
	cmd.Flags().String("org", "", "Filter by organization")
	cmd.Flags().String("tag", "", "Filter by tag")
	cmd.Flags().Bool("dry-run", false, "Show what would be fetched")
	return cmd
}

func runFetchAll(cmd *cobra.Command, args []string) error {
	orgFilter, _ := cmd.Flags().GetString("org")
	tagFilter, _ := cmd.Flags().GetString("tag")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	projects, _, err := getProjects(orgFilter, tagFilter)
	if err != nil {
		return err
	}

	ui.Header("Git Fetch — All Projects")
	fetched, skipped := 0, 0

	for _, p := range projects {
		if !filesystem.IsDir(filepath.Join(p.Path, ".git")) {
			skipped++
			continue
		}
		if dryRun {
			ui.Info("  Would fetch: %s/%s", p.OrgSlug, p.Slug)
			fetched++
			continue
		}
		out, err := shell.RunCommandInDir(p.Path, "git", "fetch", "--all", "--prune")
		if err != nil {
			ui.Warning("  %s/%s: fetch failed: %s", p.OrgSlug, p.Slug, out)
		} else {
			ui.Success("  %s/%s", p.OrgSlug, p.Slug)
			fetched++
		}
	}

	fmt.Println()
	ui.Info("Fetched: %d, Skipped (not git): %d", fetched, skipped)
	if dryRun {
		ui.DimText("(dry-run)")
	}
	return nil
}

func newPullAllCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "pull-all", Short: "Pull all clean projects", RunE: runPullAll}
	cmd.Flags().String("org", "", "Filter by organization")
	cmd.Flags().String("tag", "", "Filter by tag")
	cmd.Flags().Bool("dry-run", false, "Show what would be pulled")
	return cmd
}

func runPullAll(cmd *cobra.Command, args []string) error {
	orgFilter, _ := cmd.Flags().GetString("org")
	tagFilter, _ := cmd.Flags().GetString("tag")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	projects, _, err := getProjects(orgFilter, tagFilter)
	if err != nil {
		return err
	}

	ui.Header("Git Pull — All Projects")
	pulled, skipped, noGit := 0, 0, 0

	for _, p := range projects {
		if !filesystem.IsDir(filepath.Join(p.Path, ".git")) {
			noGit++
			continue
		}

		// Check dirty (untracked + modified files)
		status, _ := shell.RunCommandInDir(p.Path, "git", "status", "--porcelain")
		if strings.TrimSpace(status) != "" {
			ui.Warning("  Skipping (dirty): %s/%s", p.OrgSlug, p.Slug)
			skipped++
			continue
		}

		// Check detached HEAD — `git branch --show-current` returns empty on detached HEAD
		branch, _ := shell.RunCommandInDir(p.Path, "git", "branch", "--show-current")
		branch = strings.TrimSpace(branch)
		if branch == "" {
			ui.Warning("  Skipping (detached HEAD): %s/%s", p.OrgSlug, p.Slug)
			skipped++
			continue
		}

		// Check for configured upstream tracking branch
		if !hasUpstream(p.Path) {
			ui.Warning("  Skipping (no upstream): %s/%s [%s]", p.OrgSlug, p.Slug, branch)
			skipped++
			continue
		}

		if dryRun {
			ui.Info("  Would pull: %s/%s [%s]", p.OrgSlug, p.Slug, branch)
			pulled++
			continue
		}

		out, err := shell.RunCommandInDir(p.Path, "git", "pull", "--ff-only")
		if err != nil {
			ui.Warning("  %s/%s: pull failed: %s", p.OrgSlug, p.Slug, out)
		} else {
			ui.Success("  %s/%s [%s]", p.OrgSlug, p.Slug, branch)
			pulled++
		}
	}

	fmt.Println()
	ui.Info("Pulled: %d, Skipped (dirty/detached/no-upstream): %d, Not git: %d",
		pulled, skipped, noGit)
	if dryRun {
		ui.DimText("(dry-run)")
	}
	return nil
}
