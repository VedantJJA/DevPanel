package db

import (
	"context"
	"fmt"
	"strings"
)

// BlueprintRecord represents a persisted devpanel.yaml application blueprint in SQLite.
type BlueprintRecord struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	RepoURL      string `json:"repo_url"`
	Status       string `json:"status"`
	ServiceCount int    `json:"serviceCount"`
	CreatedAt    string `json:"createdAt"`
}

// CreateBlueprint inserts or replaces a blueprint record in SQLite.
func (d *DB) CreateBlueprint(ctx context.Context, bp *BlueprintRecord) error {
	ctx = contextOrBg(ctx)

	if bp.CreatedAt == "" {
		bp.CreatedAt = nowUTC()
	}
	if bp.Status == "" {
		bp.Status = "valid"
	}

	// Remove any existing blueprint with the same repo_url so the new id takes effect.
	// This is necessary because id is the PRIMARY KEY and can't be updated via ON CONFLICT.
	_, _ = d.conn.ExecContext(ctx, `DELETE FROM blueprints WHERE repo_url = ? AND id != ?`, bp.RepoURL, bp.ID)

	query := `
		INSERT INTO blueprints (id, name, repo_url, status, service_count, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			repo_url = excluded.repo_url,
			status = excluded.status,
			service_count = excluded.service_count;
	`

	_, err := d.conn.ExecContext(ctx, query, bp.ID, bp.Name, bp.RepoURL, bp.Status, bp.ServiceCount, bp.CreatedAt)
	if err != nil {
		return fmt.Errorf("db: create blueprint %s: %w", bp.Name, err)
	}

	return nil
}

// ListBlueprints returns all dynamic blueprint records stored in SQLite.
func (d *DB) ListBlueprints(ctx context.Context) ([]BlueprintRecord, error) {
	ctx = contextOrBg(ctx)

	query := `
		SELECT id, name, repo_url, status, service_count, created_at
		FROM blueprints
		ORDER BY created_at DESC;
	`

	rows, err := d.conn.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("db: list blueprints: %w", err)
	}
	defer rows.Close()

	var records []BlueprintRecord
	for rows.Next() {
		var rec BlueprintRecord
		if err := rows.Scan(&rec.ID, &rec.Name, &rec.RepoURL, &rec.Status, &rec.ServiceCount, &rec.CreatedAt); err != nil {
			return nil, fmt.Errorf("db: scan blueprint: %w", err)
		}
		records = append(records, rec)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("db: rows error: %w", err)
	}

	return records, nil
}

// GetBlueprint fetches a single blueprint by id or name from SQLite.
func (d *DB) GetBlueprint(ctx context.Context, idOrName string) (*BlueprintRecord, error) {
	ctx = contextOrBg(ctx)

	query := `
		SELECT id, name, repo_url, status, service_count, created_at
		FROM blueprints
		WHERE id = ? OR name = ? OR id = ? OR id = ?;
	`
	cleanID := strings.TrimPrefix(idOrName, "bp-")
	prefixedID := "bp-" + cleanID

	var rec BlueprintRecord
	err := d.conn.QueryRowContext(ctx, query, idOrName, idOrName, cleanID, prefixedID).
		Scan(&rec.ID, &rec.Name, &rec.RepoURL, &rec.Status, &rec.ServiceCount, &rec.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("db: get blueprint %s: %w", idOrName, err)
	}

	return &rec, nil
}

// DeleteBlueprint removes a blueprint record by ID from SQLite.
func (d *DB) DeleteBlueprint(ctx context.Context, id string) error {
	ctx = contextOrBg(ctx)

	_, err := d.conn.ExecContext(ctx, `DELETE FROM blueprints WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("db: delete blueprint %s: %w", id, err)
	}

	return nil
}
