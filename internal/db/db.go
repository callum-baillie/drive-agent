package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// Schema is the initial database schema (version 1).
const Schema = `
CREATE TABLE IF NOT EXISTS drive (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    root_path TEXT NOT NULL,
    schema_version INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS hosts (
    id TEXT PRIMARY KEY,
    hostname TEXT NOT NULL,
    os TEXT NOT NULL,
    arch TEXT,
    shell TEXT,
    last_seen_at TEXT,
    setup_completed INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS organizations (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    path TEXT NOT NULL,
    notes TEXT,
    archived INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    name TEXT NOT NULL,
    slug TEXT NOT NULL,
    path TEXT NOT NULL UNIQUE,
    git_remote TEXT,
    default_branch TEXT,
    project_type TEXT,
    package_manager TEXT,
    framework TEXT,
    archived INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    last_opened_at TEXT,
    FOREIGN KEY (organization_id) REFERENCES organizations(id)
);

CREATE TABLE IF NOT EXISTS project_tags (
    project_id TEXT NOT NULL,
    tag TEXT NOT NULL,
    PRIMARY KEY (project_id, tag),
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS package_install_records (
    id TEXT PRIMARY KEY,
    host_id TEXT NOT NULL,
    package_id TEXT NOT NULL,
    manager TEXT NOT NULL,
    installed_at TEXT NOT NULL,
    status TEXT NOT NULL,
    version TEXT,
    FOREIGN KEY (host_id) REFERENCES hosts(id)
);

CREATE TABLE IF NOT EXISTS command_runs (
    id TEXT PRIMARY KEY,
    command TEXT NOT NULL,
    status TEXT NOT NULL,
    started_at TEXT NOT NULL,
    completed_at TEXT,
    log_path TEXT
);

CREATE TABLE IF NOT EXISTS schema_version (
    version INTEGER NOT NULL
);
`

// DB wraps the SQLite database connection.
type DB struct {
	conn *sql.DB
	path string
}

// Open opens or creates the SQLite database at the given path.
func Open(dbPath string) (*DB, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	conn, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Test connection
	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &DB{conn: conn, path: dbPath}, nil
}

// Close closes the database connection.
func (d *DB) Close() error {
	return d.conn.Close()
}

// Conn returns the raw database connection.
func (d *DB) Conn() *sql.DB {
	return d.conn
}

// InitSchema creates all tables if they don't exist and sets the schema version.
func (d *DB) InitSchema() error {
	_, err := d.conn.Exec(Schema)
	if err != nil {
		return fmt.Errorf("create schema: %w", err)
	}

	// Set schema version if not present
	var count int
	err = d.conn.QueryRow("SELECT COUNT(*) FROM schema_version").Scan(&count)
	if err != nil {
		return fmt.Errorf("check schema version: %w", err)
	}
	if count == 0 {
		_, err = d.conn.Exec("INSERT INTO schema_version (version) VALUES (1)")
		if err != nil {
			return fmt.Errorf("set schema version: %w", err)
		}
	}

	return nil
}

// SchemaVersion returns the current schema version.
func (d *DB) SchemaVersion() (int, error) {
	var v int
	err := d.conn.QueryRow("SELECT version FROM schema_version LIMIT 1").Scan(&v)
	if err != nil {
		return 0, err
	}
	return v, nil
}

// RecordCommandRun inserts a command run record.
func (d *DB) RecordCommandRun(id, command, status, startedAt string) error {
	_, err := d.conn.Exec(
		"INSERT INTO command_runs (id, command, status, started_at) VALUES (?, ?, ?, ?)",
		id, command, status, startedAt,
	)
	return err
}

// CompleteCommandRun marks a command run as completed.
func (d *DB) CompleteCommandRun(id, status, completedAt string) error {
	_, err := d.conn.Exec(
		"UPDATE command_runs SET status = ?, completed_at = ? WHERE id = ?",
		status, completedAt, id,
	)
	return err
}
