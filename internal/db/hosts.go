package db

import "fmt"

// Host represents a row in the hosts table.
type Host struct {
	ID             string
	Hostname       string
	OS             string
	Arch           string
	Shell          string
	LastSeenAt     string
	SetupCompleted bool
	CreatedAt      string
	UpdatedAt      string
}

// UpsertHost inserts or updates a host record.
func (d *DB) UpsertHost(h *Host) error {
	_, err := d.conn.Exec(
		`INSERT INTO hosts (id, hostname, os, arch, shell, last_seen_at, setup_completed, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   hostname = excluded.hostname,
		   os = excluded.os,
		   arch = excluded.arch,
		   shell = excluded.shell,
		   last_seen_at = excluded.last_seen_at,
		   updated_at = excluded.updated_at`,
		h.ID, h.Hostname, h.OS, h.Arch, h.Shell, h.LastSeenAt,
		boolToInt(h.SetupCompleted), h.CreatedAt, h.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert host: %w", err)
	}
	return nil
}

// GetHost returns a host by ID.
func (d *DB) GetHost(id string) (*Host, error) {
	h := &Host{}
	var setupCompleted int
	err := d.conn.QueryRow(
		`SELECT id, hostname, os, arch, shell, last_seen_at, setup_completed, created_at, updated_at
		 FROM hosts WHERE id = ?`, id,
	).Scan(&h.ID, &h.Hostname, &h.OS, &h.Arch, &h.Shell, &h.LastSeenAt, &setupCompleted, &h.CreatedAt, &h.UpdatedAt)
	if err != nil {
		return nil, err
	}
	h.SetupCompleted = setupCompleted == 1
	return h, nil
}

// ListHosts returns all known hosts.
func (d *DB) ListHosts() ([]*Host, error) {
	rows, err := d.conn.Query(
		`SELECT id, hostname, os, arch, shell, last_seen_at, setup_completed, created_at, updated_at
		 FROM hosts ORDER BY hostname`)
	if err != nil {
		return nil, fmt.Errorf("list hosts: %w", err)
	}
	defer rows.Close()

	var hosts []*Host
	for rows.Next() {
		h := &Host{}
		var setupCompleted int
		err := rows.Scan(&h.ID, &h.Hostname, &h.OS, &h.Arch, &h.Shell, &h.LastSeenAt, &setupCompleted, &h.CreatedAt, &h.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan host: %w", err)
		}
		h.SetupCompleted = setupCompleted == 1
		hosts = append(hosts, h)
	}
	return hosts, rows.Err()
}

// InsertDrive inserts the drive record.
func (d *DB) InsertDrive(id, name, rootPath, createdAt, updatedAt string) error {
	_, err := d.conn.Exec(
		`INSERT OR IGNORE INTO drive (id, name, root_path, schema_version, created_at, updated_at)
		 VALUES (?, ?, ?, 1, ?, ?)`,
		id, name, rootPath, createdAt, updatedAt,
	)
	return err
}

// GetDrive returns the first drive record.
func (d *DB) GetDrive() (id, name, rootPath, createdAt string, schemaVersion int, err error) {
	err = d.conn.QueryRow(
		"SELECT id, name, root_path, created_at, schema_version FROM drive LIMIT 1",
	).Scan(&id, &name, &rootPath, &createdAt, &schemaVersion)
	return
}
