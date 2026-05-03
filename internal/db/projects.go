package db

import (
	"database/sql"
	"fmt"
)

// Project represents a row in the projects table.
type Project struct {
	ID             string
	OrganizationID string
	Name           string
	Slug           string
	Path           string
	GitRemote      string
	DefaultBranch  string
	ProjectType    string
	PackageManager string
	Framework      string
	Archived       bool
	CreatedAt      string
	UpdatedAt      string
	LastOpenedAt   string
	Tags           []string
	// Denormalized for display
	OrgSlug string
}

// InsertProject inserts a new project.
func (d *DB) InsertProject(p *Project) error {
	tx, err := d.conn.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		`INSERT INTO projects (id, organization_id, name, slug, path, git_remote, default_branch,
		 project_type, package_manager, framework, archived, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.OrganizationID, p.Name, p.Slug, p.Path, p.GitRemote, p.DefaultBranch,
		p.ProjectType, p.PackageManager, p.Framework, boolToInt(p.Archived), p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert project: %w", err)
	}

	// Insert tags
	for _, tag := range p.Tags {
		_, err = tx.Exec(
			"INSERT OR IGNORE INTO project_tags (project_id, tag) VALUES (?, ?)",
			p.ID, tag,
		)
		if err != nil {
			return fmt.Errorf("insert tag %q: %w", tag, err)
		}
	}

	return tx.Commit()
}

// ListProjects returns projects, optionally filtered by org slug or tag.
func (d *DB) ListProjects(orgSlug, tag string) ([]*Project, error) {
	query := `SELECT p.id, p.organization_id, p.name, p.slug, p.path, 
			  COALESCE(p.git_remote, ''), COALESCE(p.default_branch, ''),
			  COALESCE(p.project_type, ''), COALESCE(p.package_manager, ''),
			  COALESCE(p.framework, ''), p.archived, p.created_at, p.updated_at,
			  COALESCE(p.last_opened_at, ''), o.slug
			  FROM projects p
			  JOIN organizations o ON p.organization_id = o.id`

	var args []interface{}
	var conditions []string

	if orgSlug != "" {
		conditions = append(conditions, "o.slug = ?")
		args = append(args, orgSlug)
	}
	if tag != "" {
		conditions = append(conditions, "EXISTS (SELECT 1 FROM project_tags pt WHERE pt.project_id = p.id AND pt.tag = ?)")
		args = append(args, tag)
	}

	if len(conditions) > 0 {
		query += " WHERE "
		for i, c := range conditions {
			if i > 0 {
				query += " AND "
			}
			query += c
		}
	}

	query += " ORDER BY o.slug, p.name"

	rows, err := d.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	var projects []*Project
	for rows.Next() {
		p := &Project{}
		var archived int
		err := rows.Scan(
			&p.ID, &p.OrganizationID, &p.Name, &p.Slug, &p.Path,
			&p.GitRemote, &p.DefaultBranch, &p.ProjectType, &p.PackageManager,
			&p.Framework, &archived, &p.CreatedAt, &p.UpdatedAt, &p.LastOpenedAt, &p.OrgSlug,
		)
		if err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		p.Archived = archived == 1

		// Load tags
		p.Tags, err = d.getProjectTags(p.ID)
		if err != nil {
			return nil, err
		}

		projects = append(projects, p)
	}
	return projects, rows.Err()
}

// GetProjectBySlug returns a project by org slug and project slug.
func (d *DB) GetProjectBySlug(orgSlug, projectSlug string) (*Project, error) {
	p := &Project{}
	var archived int
	err := d.conn.QueryRow(
		`SELECT p.id, p.organization_id, p.name, p.slug, p.path,
		 COALESCE(p.git_remote, ''), COALESCE(p.default_branch, ''),
		 COALESCE(p.project_type, ''), COALESCE(p.package_manager, ''),
		 COALESCE(p.framework, ''), p.archived, p.created_at, p.updated_at,
		 COALESCE(p.last_opened_at, ''), o.slug
		 FROM projects p
		 JOIN organizations o ON p.organization_id = o.id
		 WHERE o.slug = ? AND p.slug = ?`,
		orgSlug, projectSlug,
	).Scan(
		&p.ID, &p.OrganizationID, &p.Name, &p.Slug, &p.Path,
		&p.GitRemote, &p.DefaultBranch, &p.ProjectType, &p.PackageManager,
		&p.Framework, &archived, &p.CreatedAt, &p.UpdatedAt, &p.LastOpenedAt, &p.OrgSlug,
	)
	if err != nil {
		return nil, err
	}
	p.Archived = archived == 1
	p.Tags, _ = d.getProjectTags(p.ID)
	return p, nil
}

// GetProjectByPath returns a project by its filesystem path.
func (d *DB) GetProjectByPath(path string) (*Project, error) {
	p := &Project{}
	var archived int
	err := d.conn.QueryRow(
		`SELECT p.id, p.organization_id, p.name, p.slug, p.path,
		 COALESCE(p.git_remote, ''), COALESCE(p.default_branch, ''),
		 COALESCE(p.project_type, ''), COALESCE(p.package_manager, ''),
		 COALESCE(p.framework, ''), p.archived, p.created_at, p.updated_at,
		 COALESCE(p.last_opened_at, ''), o.slug
		 FROM projects p
		 JOIN organizations o ON p.organization_id = o.id
		 WHERE p.path = ?`,
		path,
	).Scan(
		&p.ID, &p.OrganizationID, &p.Name, &p.Slug, &p.Path,
		&p.GitRemote, &p.DefaultBranch, &p.ProjectType, &p.PackageManager,
		&p.Framework, &archived, &p.CreatedAt, &p.UpdatedAt, &p.LastOpenedAt, &p.OrgSlug,
	)
	if err != nil {
		return nil, err
	}
	p.Archived = archived == 1
	p.Tags, _ = d.getProjectTags(p.ID)
	return p, nil
}

// UpsertProject inserts or updates a project (used by reindex).
func (d *DB) UpsertProject(p *Project) error {
	existing, err := d.GetProjectByPath(p.Path)
	if err == sql.ErrNoRows {
		return d.InsertProject(p)
	}
	if err != nil {
		return err
	}

	// Update existing project
	_, err = d.conn.Exec(
		`UPDATE projects SET name = ?, slug = ?, git_remote = ?, project_type = ?,
		 package_manager = ?, updated_at = ? WHERE id = ?`,
		p.Name, p.Slug, p.GitRemote, p.ProjectType, p.PackageManager, p.UpdatedAt, existing.ID,
	)
	if err != nil {
		return fmt.Errorf("update project: %w", err)
	}

	// Update tags
	_, _ = d.conn.Exec("DELETE FROM project_tags WHERE project_id = ?", existing.ID)
	for _, tag := range p.Tags {
		_, _ = d.conn.Exec(
			"INSERT OR IGNORE INTO project_tags (project_id, tag) VALUES (?, ?)",
			existing.ID, tag,
		)
	}

	return nil
}

// DeleteProjectByPath removes a project by path.
func (d *DB) DeleteProjectByPath(path string) error {
	_, err := d.conn.Exec("DELETE FROM projects WHERE path = ?", path)
	return err
}

func (d *DB) getProjectTags(projectID string) ([]string, error) {
	rows, err := d.conn.Query("SELECT tag FROM project_tags WHERE project_id = ? ORDER BY tag", projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}
