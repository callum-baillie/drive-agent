package db

import (
	"fmt"
)

// Organization represents a row in the organizations table.
type Organization struct {
	ID        string
	Name      string
	Slug      string
	Path      string
	Notes     string
	Archived  bool
	CreatedAt string
	UpdatedAt string
}

// InsertOrganization inserts a new organization.
func (d *DB) InsertOrganization(org *Organization) error {
	_, err := d.conn.Exec(
		`INSERT INTO organizations (id, name, slug, path, notes, archived, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		org.ID, org.Name, org.Slug, org.Path, org.Notes,
		boolToInt(org.Archived), org.CreatedAt, org.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert organization: %w", err)
	}
	return nil
}

// ListOrganizations returns all non-archived organizations.
func (d *DB) ListOrganizations() ([]*Organization, error) {
	rows, err := d.conn.Query(
		`SELECT id, name, slug, path, notes, archived, created_at, updated_at
		 FROM organizations ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list organizations: %w", err)
	}
	defer rows.Close()

	var orgs []*Organization
	for rows.Next() {
		o := &Organization{}
		var archived int
		var notes *string
		err := rows.Scan(&o.ID, &o.Name, &o.Slug, &o.Path, &notes, &archived, &o.CreatedAt, &o.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan organization: %w", err)
		}
		o.Archived = archived == 1
		if notes != nil {
			o.Notes = *notes
		}
		orgs = append(orgs, o)
	}
	return orgs, rows.Err()
}

// GetOrganizationBySlug returns an organization by its slug.
func (d *DB) GetOrganizationBySlug(slug string) (*Organization, error) {
	o := &Organization{}
	var archived int
	var notes *string
	err := d.conn.QueryRow(
		`SELECT id, name, slug, path, notes, archived, created_at, updated_at
		 FROM organizations WHERE slug = ?`, slug,
	).Scan(&o.ID, &o.Name, &o.Slug, &o.Path, &notes, &archived, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		return nil, err
	}
	o.Archived = archived == 1
	if notes != nil {
		o.Notes = *notes
	}
	return o, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
