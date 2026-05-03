package org

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/callum-baillie/drive-agent/internal/config"
	"github.com/callum-baillie/drive-agent/internal/db"
	"github.com/callum-baillie/drive-agent/internal/filesystem"
	"github.com/callum-baillie/drive-agent/internal/ui"
	"github.com/callum-baillie/drive-agent/internal/utils"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "org",
		Short: "Manage organizations",
	}
	cmd.AddCommand(newAddCmd())
	cmd.AddCommand(newListCmd())
	return cmd
}

func newAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Add a new organization",
		Args:  cobra.ExactArgs(1),
		RunE:  runAdd,
	}
	cmd.Flags().String("slug", "", "Custom slug (defaults to slugified name)")
	return cmd
}

func runAdd(cmd *cobra.Command, args []string) error {
	name := args[0]
	slugOverride, _ := cmd.Flags().GetString("slug")

	driveRoot, err := filesystem.FindDriveRoot("")
	if err != nil {
		return fmt.Errorf("not inside a drive-agent managed drive: %w", err)
	}

	slug := slugOverride
	if slug == "" {
		slug = utils.Slugify(name)
	}
	if !utils.IsValidSlug(slug) {
		return fmt.Errorf("invalid slug %q", slug)
	}

	orgPath := filesystem.OrgPath(driveRoot, slug)
	for _, dir := range config.OrgLayout {
		if err := os.MkdirAll(filepath.Join(orgPath, dir), 0755); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	dbPath := filesystem.DBPath(driveRoot)
	database, err := db.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer database.Close()

	now := config.NowISO()
	org := &db.Organization{
		ID: utils.OrgID(slug), Name: name, Slug: slug,
		Path: orgPath, CreatedAt: now, UpdatedAt: now,
	}
	if err := database.InsertOrganization(org); err != nil {
		return fmt.Errorf("organization %q may already exist: %w", slug, err)
	}

	ui.Success("Organization %q created", name)
	ui.Label("Slug", slug)
	ui.Label("Path", orgPath)
	return nil
}

func newListCmd() *cobra.Command {
	return &cobra.Command{Use: "list", Short: "List organizations", RunE: runList}
}

func runList(cmd *cobra.Command, args []string) error {
	driveRoot, err := filesystem.FindDriveRoot("")
	if err != nil {
		return fmt.Errorf("not inside a drive-agent managed drive: %w", err)
	}
	dbPath := filesystem.DBPath(driveRoot)
	database, err := db.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer database.Close()

	orgs, err := database.ListOrganizations()
	if err != nil {
		return fmt.Errorf("list organizations: %w", err)
	}
	if len(orgs) == 0 {
		ui.Info("No organizations. Add one: drive-agent org add <name>")
		return nil
	}

	ui.Header("Organizations")
	headers := []string{"SLUG", "NAME", "PATH"}
	var rows [][]string
	for _, o := range orgs {
		rows = append(rows, []string{o.Slug, o.Name, o.Path})
	}
	ui.Table(headers, rows)
	fmt.Println()
	return nil
}
