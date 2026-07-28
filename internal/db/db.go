// Package db manages the SQLite database for DevPnl.
//
// It uses modernc.org/sqlite, a pure-Go SQLite implementation that
// requires no CGo — making it ideal for cross-compilation and minimal
// container images.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "modernc.org/sqlite"
)

// DB wraps a *sql.DB with DevPnl-specific operations.
type DB struct {
	conn *sql.DB
}

// Open creates or opens a SQLite database at the given path and runs
// migrations. Use ":memory:" for in-memory databases (testing).
func Open(dsn string) (*DB, error) {
	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("db: open %q: %w", dsn, err)
	}

	// SQLite performance tuning for a single-writer deployment panel.
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA foreign_keys=ON",
		"PRAGMA cache_size=-20000", // 20 MB
	}
	for _, p := range pragmas {
		if _, err := conn.Exec(p); err != nil {
			conn.Close()
			return nil, fmt.Errorf("db: %s: %w", p, err)
		}
	}

	d := &DB{conn: conn}
	if err := d.migrate(); err != nil {
		conn.Close()
		return nil, err
	}

	return d, nil
}

// Close closes the underlying database connection.
func (d *DB) Close() error {
	return d.conn.Close()
}

// Conn returns the underlying *sql.DB for advanced use cases.
func (d *DB) Conn() *sql.DB {
	return d.conn
}

// migrate runs all schema migrations in order. Each migration is
// idempotent (IF NOT EXISTS).
func (d *DB) migrate() error {
	migrations := []string{
		migrationProjects,
		migrationContainers,
		migrationDomains,
		migrationBlueprints,
	}

	for i, m := range migrations {
		if _, err := d.conn.Exec(m); err != nil {
			return fmt.Errorf("db: migration %d: %w", i, err)
		}
	}

	log.Println("db: migrations complete")
	return nil
}

// --- Schema definitions -----------------------------------------------------

const migrationProjects = `
CREATE TABLE IF NOT EXISTS projects (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	name        TEXT    NOT NULL UNIQUE,
	repo_url    TEXT    NOT NULL DEFAULT '',
	compose_yml TEXT    NOT NULL DEFAULT '',
	status      TEXT    NOT NULL DEFAULT 'inactive',
	created_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
	updated_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_projects_name ON projects(name);
`

const migrationContainers = `
CREATE TABLE IF NOT EXISTS containers (
	id           INTEGER PRIMARY KEY AUTOINCREMENT,
	project_id   INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
	container_id TEXT    NOT NULL UNIQUE,
	name         TEXT    NOT NULL,
	image        TEXT    NOT NULL DEFAULT '',
	status       TEXT    NOT NULL DEFAULT 'created',
	port         INTEGER NOT NULL DEFAULT 0,
	created_at   TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
	updated_at   TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_containers_project ON containers(project_id);
CREATE INDEX IF NOT EXISTS idx_containers_cid     ON containers(container_id);
`

const migrationDomains = `
CREATE TABLE IF NOT EXISTS domains (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	project_id INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
	fqdn       TEXT    NOT NULL UNIQUE,
	tls        INTEGER NOT NULL DEFAULT 1,
	created_at TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_domains_fqdn    ON domains(fqdn);
CREATE INDEX IF NOT EXISTS idx_domains_project ON domains(project_id);
`

const migrationBlueprints = `
CREATE TABLE IF NOT EXISTS blueprints (
	id            TEXT PRIMARY KEY,
	name          TEXT NOT NULL,
	repo_url      TEXT NOT NULL UNIQUE,
	status        TEXT NOT NULL DEFAULT 'valid',
	service_count INTEGER NOT NULL DEFAULT 1,
	created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_blueprints_repo ON blueprints(repo_url);
`

// --- Shared helpers ---------------------------------------------------------

// nowUTC returns the current time formatted for SQLite TEXT columns.
func nowUTC() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// contextOrBg returns ctx if non-nil, or context.Background().
func contextOrBg(ctx context.Context) context.Context {
	if ctx != nil {
		return ctx
	}
	return context.Background()
}
