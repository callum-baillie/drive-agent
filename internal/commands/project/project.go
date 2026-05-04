package project

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/spf13/cobra"

	"github.com/callum-baillie/drive-agent/internal/config"
	"github.com/callum-baillie/drive-agent/internal/db"
	"github.com/callum-baillie/drive-agent/internal/filesystem"
	"github.com/callum-baillie/drive-agent/internal/ui"
	"github.com/callum-baillie/drive-agent/internal/utils"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "project", Short: "Manage projects"}
	cmd.AddCommand(newProjectAddCmd())
	cmd.AddCommand(newProjectListCmd())
	cmd.AddCommand(newProjectPathCmd())
	cmd.AddCommand(newProjectOpenCmd())
	cmd.AddCommand(newProjectReindexCmd())
	return cmd
}

func newProjectAddCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "add", Short: "Add a project", RunE: runProjectAdd}
	cmd.Flags().String("org", "", "Organization slug")
	cmd.Flags().String("name", "", "Project name")
	cmd.Flags().String("type", "", "Project type (e.g. nextjs, expo)")
	cmd.Flags().String("package-manager", "", "Package manager")
	cmd.Flags().String("git", "", "Git remote URL")
	cmd.Flags().String("tags", "", "Comma-separated tags")
	cmd.Flags().Bool("clone", false, "Clone from git remote")
	return cmd
}

func runProjectAdd(cmd *cobra.Command, args []string) error {
	driveRoot, err := filesystem.FindDriveRoot("")
	if err != nil {
		return fmt.Errorf("not inside a drive-agent managed drive: %w", err)
	}

	orgSlug, _ := cmd.Flags().GetString("org")
	name, _ := cmd.Flags().GetString("name")
	projectType, _ := cmd.Flags().GetString("type")
	pkgMgr, _ := cmd.Flags().GetString("package-manager")
	gitRemote, _ := cmd.Flags().GetString("git")
	tagsStr, _ := cmd.Flags().GetString("tags")
	clone, _ := cmd.Flags().GetBool("clone")

	// Interactive prompts for missing fields
	if orgSlug == "" {
		orgSlug = ui.Prompt("Organization slug", "personal")
	}
	if name == "" {
		name = ui.Prompt("Project name", "")
		if name == "" {
			return fmt.Errorf("project name is required")
		}
	}
	if clone && gitRemote == "" {
		return fmt.Errorf("--clone requires --git <remote-url>")
	}

	slug := utils.Slugify(name)

	database, err := db.Open(filesystem.DBPath(driveRoot))
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer database.Close()

	// Verify org exists
	org, err := database.GetOrganizationBySlug(orgSlug)
	if err != nil {
		return fmt.Errorf("organization %q not found — create it first with: drive-agent org add %s", orgSlug, orgSlug)
	}

	projectPath := filesystem.ProjectPath(driveRoot, orgSlug, slug)

	// Check project doesn't already exist
	if filesystem.Exists(projectPath) {
		return fmt.Errorf("project directory already exists: %s", projectPath)
	}

	// Clone or create directory
	if clone && gitRemote != "" {
		ui.Info("Cloning %s...", gitRemote)
		out, err := exec.Command("git", "clone", gitRemote, projectPath).CombinedOutput()
		if err != nil {
			return fmt.Errorf("git clone failed: %s", string(out))
		}
	} else {
		if err := os.MkdirAll(projectPath, 0755); err != nil {
			return fmt.Errorf("create project directory: %w", err)
		}
	}

	// Parse tags
	var tags []string
	if tagsStr != "" {
		for _, t := range strings.Split(tagsStr, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tags = append(tags, t)
			}
		}
	}

	now := config.NowISO()
	projectID := utils.ProjectID(orgSlug, slug)

	// Write manifest
	manifest := config.ProjectManifestData{
		ID: projectID, Name: name, Slug: slug, Org: orgSlug,
		Type: projectType, PackageManager: pkgMgr,
		Tags: tags, GitRemote: gitRemote, CreatedAt: now,
	}
	manifestPath := filepath.Join(projectPath, config.ProjectManifest)
	f, err := os.Create(manifestPath)
	if err != nil {
		return fmt.Errorf("create manifest: %w", err)
	}
	if err := toml.NewEncoder(f).Encode(manifest); err != nil {
		f.Close()
		return fmt.Errorf("write manifest: %w", err)
	}
	f.Close()

	// Insert into database
	project := &db.Project{
		ID: projectID, OrganizationID: org.ID, Name: name, Slug: slug,
		Path: projectPath, GitRemote: gitRemote, ProjectType: projectType,
		PackageManager: pkgMgr, Tags: tags, CreatedAt: now, UpdatedAt: now,
	}
	if err := database.InsertProject(project); err != nil {
		return fmt.Errorf("register project: %w", err)
	}

	ui.Success("Project %q created", name)
	ui.Label("Path", projectPath)
	ui.Label("Manifest", manifestPath)
	return nil
}

func newProjectListCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "list", Short: "List projects", RunE: runProjectList}
	cmd.Flags().String("org", "", "Filter by organization")
	cmd.Flags().String("tag", "", "Filter by tag")
	return cmd
}

func runProjectList(cmd *cobra.Command, args []string) error {
	driveRoot, err := filesystem.FindDriveRoot("")
	if err != nil {
		return fmt.Errorf("not inside a drive-agent managed drive: %w", err)
	}
	orgFilter, _ := cmd.Flags().GetString("org")
	tagFilter, _ := cmd.Flags().GetString("tag")

	database, err := db.Open(filesystem.DBPath(driveRoot))
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer database.Close()

	projects, err := database.ListProjects(orgFilter, tagFilter)
	if err != nil {
		return fmt.Errorf("list projects: %w", err)
	}
	if len(projects) == 0 {
		ui.Info("No projects found.")
		return nil
	}

	ui.Header("Projects")
	headers := []string{"ORG", "NAME", "TYPE", "TAGS"}
	var rows [][]string
	for _, p := range projects {
		rows = append(rows, []string{
			p.OrgSlug, p.Name,
			p.ProjectType, strings.Join(p.Tags, ", "),
		})
	}
	ui.Table(headers, rows)
	fmt.Println()
	return nil
}

func newProjectPathCmd() *cobra.Command {
	return &cobra.Command{
		Use: "path <project|org/project>", Short: "Print project path",
		Args: cobra.ExactArgs(1), RunE: runProjectPath,
	}
}

func runProjectPath(cmd *cobra.Command, args []string) error {
	driveRoot, err := filesystem.FindDriveRoot("")
	if err != nil {
		return err
	}
	database, err := db.Open(filesystem.DBPath(driveRoot))
	if err != nil {
		return err
	}
	defer database.Close()

	p, err := getProjectByRef(database, args[0])
	if err != nil {
		return err
	}
	fmt.Println(p.Path)
	return nil
}

func newProjectOpenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "open <project|org/project>", Short: "Open project in editor",
		Args: cobra.ExactArgs(1), RunE: runProjectOpen,
	}
	cmd.Flags().String("editor", "", "Editor command (default: cursor, code)")
	return cmd
}

func runProjectOpen(cmd *cobra.Command, args []string) error {
	driveRoot, err := filesystem.FindDriveRoot("")
	if err != nil {
		return err
	}
	editor, _ := cmd.Flags().GetString("editor")

	database, err := db.Open(filesystem.DBPath(driveRoot))
	if err != nil {
		return err
	}
	defer database.Close()

	p, err := getProjectByRef(database, args[0])
	if err != nil {
		return err
	}

	// Detect editor
	if editor == "" {
		for _, e := range []string{"cursor", "code", "zed"} {
			if _, err := exec.LookPath(e); err == nil {
				editor = e
				break
			}
		}
	}
	if editor == "" {
		return fmt.Errorf("no editor found — use --editor flag")
	}

	ui.Info("Opening %s in %s...", p.Name, editor)
	return exec.Command(editor, p.Path).Start()
}

func newProjectReindexCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "reindex", Short: "Rebuild database from project manifests",
		RunE: runProjectReindex,
	}
	cmd.Flags().Bool("dry-run", false, "Show what would change without modifying")
	return cmd
}

func runProjectReindex(cmd *cobra.Command, args []string) error {
	driveRoot, err := filesystem.FindDriveRoot("")
	if err != nil {
		return err
	}
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	database, err := db.Open(filesystem.DBPath(driveRoot))
	if err != nil {
		return err
	}
	defer database.Close()

	ui.Header("Reindexing Projects")

	// --- Phase 1: Check DB entries whose folders are missing ---
	existingProjects, _ := database.ListProjects("", "")
	missingCount := 0
	for _, p := range existingProjects {
		if !filesystem.IsDir(p.Path) {
			if dryRun {
				ui.Warning("  Missing folder: %s/%s → %s", p.OrgSlug, p.Slug, p.Path)
			} else {
				ui.Warning("  Missing folder (DB entry kept): %s/%s → %s", p.OrgSlug, p.Slug, p.Path)
			}
			missingCount++
		}
	}

	// --- Phase 2: Scan disk for manifests ---
	orgsDir := filepath.Join(driveRoot, "Orgs")
	orgEntries, err := os.ReadDir(orgsDir)
	if err != nil {
		return fmt.Errorf("read Orgs directory: %w", err)
	}

	found, added, updated, skipped := 0, 0, 0, 0

	for _, orgEntry := range orgEntries {
		if !orgEntry.IsDir() {
			continue
		}
		orgSlug := orgEntry.Name()
		projDir := filepath.Join(orgsDir, orgSlug, "projects")
		projEntries, err := os.ReadDir(projDir)
		if err != nil {
			continue
		}

		// Ensure org exists in DB
		org, err := database.GetOrganizationBySlug(orgSlug)
		if err == sql.ErrNoRows && !dryRun {
			now := config.NowISO()
			org = &db.Organization{
				ID: utils.OrgID(orgSlug), Name: orgSlug, Slug: orgSlug,
				Path: filepath.Join(orgsDir, orgSlug), CreatedAt: now, UpdatedAt: now,
			}
			if insertErr := database.InsertOrganization(org); insertErr != nil {
				ui.Warning("  Could not create org %s: %v", orgSlug, insertErr)
				continue
			}
			ui.Success("  Auto-created org: %s", orgSlug)
		}
		if org == nil {
			continue
		}

		for _, projEntry := range projEntries {
			if !projEntry.IsDir() {
				continue
			}
			projPath := filepath.Join(projDir, projEntry.Name())
			manifestPath := filepath.Join(projPath, config.ProjectManifest)

			if !filesystem.Exists(manifestPath) {
				ui.DimText("  No manifest: %s/%s (skipping)", orgSlug, projEntry.Name())
				skipped++
				continue
			}
			found++

			var manifest config.ProjectManifestData
			if _, err := toml.DecodeFile(manifestPath, &manifest); err != nil {
				ui.Warning("  Malformed manifest: %s: %v", manifestPath, err)
				skipped++
				continue
			}

			// Validate required manifest fields
			if manifest.Slug == "" {
				manifest.Slug = utils.Slugify(projEntry.Name())
			}
			if manifest.Name == "" {
				manifest.Name = projEntry.Name()
			}
			if manifest.ID == "" {
				manifest.ID = utils.ProjectID(orgSlug, manifest.Slug)
			}

			if dryRun {
				_, err := database.GetProjectByPath(projPath)
				if err == sql.ErrNoRows {
					ui.Info("  Would add: %s/%s", orgSlug, manifest.Slug)
					added++
				} else {
					ui.DimText("  Exists: %s/%s", orgSlug, manifest.Slug)
				}
				continue
			}

			now := config.NowISO()
			p := &db.Project{
				ID: manifest.ID, OrganizationID: org.ID,
				Name: manifest.Name, Slug: manifest.Slug, Path: projPath,
				GitRemote: manifest.GitRemote, ProjectType: manifest.Type,
				PackageManager: manifest.PackageManager, Tags: manifest.Tags,
				CreatedAt: manifest.CreatedAt, UpdatedAt: now,
			}
			_, err := database.GetProjectByPath(projPath)
			if err == sql.ErrNoRows {
				if insertErr := database.InsertProject(p); insertErr != nil {
					// Could be a duplicate slug within the same org
					ui.Warning("  Could not add %s/%s: %v", orgSlug, manifest.Slug, insertErr)
					skipped++
					continue
				}
				ui.Success("  Added: %s/%s", orgSlug, manifest.Slug)
				added++
			} else {
				database.UpsertProject(p)
				updated++
			}
		}
	}

	fmt.Println()
	if missingCount > 0 {
		ui.Warning("DB entries with missing folders: %d (run 'project reindex --repair' in a future version to prune)", missingCount)
	}
	ui.Info("Found %d manifests, added %d, updated %d, skipped %d", found, added, updated, skipped)
	if dryRun {
		ui.DimText("(dry-run — no changes made)")
	}
	return nil
}

func parseProjectRef(ref string) (orgSlug, projectSlug string, err error) {
	parts := strings.SplitN(ref, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("use format: project or org/project (e.g. my-app or personal/my-app)")
	}
	return parts[0], parts[1], nil
}

func getProjectByRef(database *db.DB, ref string) (*db.Project, error) {
	if strings.Contains(ref, "/") {
		orgSlug, projSlug, err := parseProjectRef(ref)
		if err != nil {
			return nil, err
		}
		p, err := database.GetProjectBySlug(orgSlug, projSlug)
		if err != nil {
			return nil, fmt.Errorf("project not found: %s/%s", orgSlug, projSlug)
		}
		return p, nil
	}

	projects, err := database.ListProjects("", "")
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}

	var matches []*db.Project
	for _, p := range projects {
		if p.Slug == ref || p.Name == ref {
			matches = append(matches, p)
		}
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("project not found: %s", ref)
	}
	if len(matches) > 1 {
		var refs []string
		for _, p := range matches {
			refs = append(refs, p.OrgSlug+"/"+p.Slug)
		}
		return nil, fmt.Errorf("project %q is ambiguous; use org/project (%s)", ref, strings.Join(refs, ", "))
	}
	return matches[0], nil
}
