package db

import (
	"context"
	"fmt"
)

// DeploymentRecord tracks each deploy attempt for history + status.
type DeploymentRecord struct {
	ID         string `json:"id"`
	ProjectID  string `json:"project_id"`
	Status     string `json:"status"` // queued|building|live|error|canceled
	CommitSHA  string `json:"commit_sha"`
	Trigger    string `json:"trigger"` // manual|auto|rollback
	StartedAt  string `json:"started_at"`
	FinishedAt string `json:"finished_at"`
	Error      string `json:"error"`
}

// CreateDeployment inserts a new deployment row.
func (d *DB) CreateDeployment(ctx context.Context, rec *DeploymentRecord) error {
	ctx = contextOrBg(ctx)
	if rec.StartedAt == "" {
		rec.StartedAt = nowUTC()
	}
	if rec.Status == "" {
		rec.Status = "queued"
	}
	_, err := d.conn.ExecContext(ctx, `
	    INSERT INTO deployments (id, project_id, status, commit_sha, trigger, started_at, finished_at, error)
	    VALUES (?, ?, ?, ?, ?, ?, ?, ?);`,
		rec.ID, rec.ProjectID, rec.Status, rec.CommitSHA, rec.Trigger,
		rec.StartedAt, rec.FinishedAt, rec.Error)
	if err != nil {
		return fmt.Errorf("db: create deployment: %w", err)
	}
	return nil
}

// UpdateDeploymentStatus patches status/finished_at/error.
func (d *DB) UpdateDeploymentStatus(ctx context.Context, id, status, errMsg string) error {
	ctx = contextOrBg(ctx)
	finished := ""
	if status == "live" || status == "error" || status == "canceled" {
		finished = nowUTC()
	}
	_, err := d.conn.ExecContext(ctx, `
	    UPDATE deployments SET status = ?, error = ?, finished_at = ? WHERE id = ?;`,
		status, errMsg, finished, id)
	if err != nil {
		return fmt.Errorf("db: update deployment %s: %w", id, err)
	}
	return nil
}

// ListDeployments returns deploy history for a project (newest first).
func (d *DB) ListDeployments(ctx context.Context, projectID string) ([]DeploymentRecord, error) {
	ctx = contextOrBg(ctx)
	rows, err := d.conn.QueryContext(ctx, `
	    SELECT id, project_id, status, commit_sha, trigger, started_at, finished_at, error
	    FROM deployments WHERE project_id = ? ORDER BY started_at DESC LIMIT 50;`, projectID)
	if err != nil {
		return nil, fmt.Errorf("db: list deployments: %w", err)
	}
	defer rows.Close()
	var out []DeploymentRecord
	for rows.Next() {
		var r DeploymentRecord
		if err := rows.Scan(&r.ID, &r.ProjectID, &r.Status, &r.CommitSHA, &r.Trigger,
			&r.StartedAt, &r.FinishedAt, &r.Error); err != nil {
			return nil, fmt.Errorf("db: scan deployment: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// GetDeployment fetches a single deployment by ID.
func (d *DB) GetDeployment(ctx context.Context, id string) (*DeploymentRecord, error) {
	ctx = contextOrBg(ctx)
	var r DeploymentRecord
	err := d.conn.QueryRowContext(ctx, `
	    SELECT id, project_id, status, commit_sha, trigger, started_at, finished_at, error
	    FROM deployments WHERE id = ?;`, id).
		Scan(&r.ID, &r.ProjectID, &r.Status, &r.CommitSHA, &r.Trigger,
			&r.StartedAt, &r.FinishedAt, &r.Error)
	if err != nil {
		return nil, fmt.Errorf("db: get deployment: %w", err)
	}
	return &r, nil
}
