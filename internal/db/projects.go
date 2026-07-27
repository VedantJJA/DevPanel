package db

import (
	"context"
	"database/sql"
	"fmt"
)

// Project represents a deployment project.
type Project struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	RepoURL    string `json:"repo_url"`
	ComposeYML string `json:"compose_yml"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

// CreateProject inserts a new project and returns its ID.
func (d *DB) CreateProject(ctx context.Context, p *Project) (int64, error) {
	ctx = contextOrBg(ctx)
	res, err := d.conn.ExecContext(ctx,
		`INSERT INTO projects (name, repo_url, compose_yml, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		p.Name, p.RepoURL, p.ComposeYML, coalesce(p.Status, "inactive"), nowUTC(), nowUTC(),
	)
	if err != nil {
		return 0, fmt.Errorf("db: create project: %w", err)
	}
	return res.LastInsertId()
}

// GetProject returns a project by ID.
func (d *DB) GetProject(ctx context.Context, id int64) (*Project, error) {
	ctx = contextOrBg(ctx)
	p := &Project{}
	err := d.conn.QueryRowContext(ctx,
		`SELECT id, name, repo_url, compose_yml, status, created_at, updated_at
		 FROM projects WHERE id = ?`, id,
	).Scan(&p.ID, &p.Name, &p.RepoURL, &p.ComposeYML, &p.Status, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("db: get project %d: %w", id, err)
	}
	return p, nil
}

// GetProjectByName returns a project by its unique name.
func (d *DB) GetProjectByName(ctx context.Context, name string) (*Project, error) {
	ctx = contextOrBg(ctx)
	p := &Project{}
	err := d.conn.QueryRowContext(ctx,
		`SELECT id, name, repo_url, compose_yml, status, created_at, updated_at
		 FROM projects WHERE name = ?`, name,
	).Scan(&p.ID, &p.Name, &p.RepoURL, &p.ComposeYML, &p.Status, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("db: get project %q: %w", name, err)
	}
	return p, nil
}

// ListProjects returns all projects ordered by creation date (newest first).
func (d *DB) ListProjects(ctx context.Context) ([]*Project, error) {
	ctx = contextOrBg(ctx)
	rows, err := d.conn.QueryContext(ctx,
		`SELECT id, name, repo_url, compose_yml, status, created_at, updated_at
		 FROM projects ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("db: list projects: %w", err)
	}
	defer rows.Close()

	var out []*Project
	for rows.Next() {
		p := &Project{}
		if err := rows.Scan(&p.ID, &p.Name, &p.RepoURL, &p.ComposeYML, &p.Status, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("db: scan project: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// UpdateProject updates a project's mutable fields.
func (d *DB) UpdateProject(ctx context.Context, p *Project) error {
	ctx = contextOrBg(ctx)
	_, err := d.conn.ExecContext(ctx,
		`UPDATE projects SET name = ?, repo_url = ?, compose_yml = ?, status = ?, updated_at = ?
		 WHERE id = ?`,
		p.Name, p.RepoURL, p.ComposeYML, p.Status, nowUTC(), p.ID,
	)
	if err != nil {
		return fmt.Errorf("db: update project %d: %w", p.ID, err)
	}
	return nil
}

// DeleteProject removes a project by ID (cascades to containers and domains).
func (d *DB) DeleteProject(ctx context.Context, id int64) error {
	ctx = contextOrBg(ctx)
	_, err := d.conn.ExecContext(ctx, `DELETE FROM projects WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("db: delete project %d: %w", id, err)
	}
	return nil
}

// coalesce returns val if non-empty, otherwise fallback.
func coalesce(val, fallback string) string {
	if val == "" {
		return fallback
	}
	return val
}
