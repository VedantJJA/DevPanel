package db

import (
	"context"
	"database/sql"
	"fmt"
)

// Container represents a Docker container tracked by a project.
type Container struct {
	ID          int64  `json:"id"`
	ProjectID   int64  `json:"project_id"`
	ContainerID string `json:"container_id"` // Docker container ID (short hash)
	Name        string `json:"name"`
	Image       string `json:"image"`
	Status      string `json:"status"` // created, running, stopped, etc.
	Port        int    `json:"port"`   // host-exposed port
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// CreateContainer inserts a new container record and returns its ID.
func (d *DB) CreateContainer(ctx context.Context, c *Container) (int64, error) {
	ctx = contextOrBg(ctx)
	res, err := d.conn.ExecContext(ctx,
		`INSERT INTO containers (project_id, container_id, name, image, status, port, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ProjectID, c.ContainerID, c.Name, c.Image,
		coalesce(c.Status, "created"), c.Port, nowUTC(), nowUTC(),
	)
	if err != nil {
		return 0, fmt.Errorf("db: create container: %w", err)
	}
	return res.LastInsertId()
}

// GetContainer returns a container by its database ID.
func (d *DB) GetContainer(ctx context.Context, id int64) (*Container, error) {
	ctx = contextOrBg(ctx)
	c := &Container{}
	err := d.conn.QueryRowContext(ctx,
		`SELECT id, project_id, container_id, name, image, status, port, created_at, updated_at
		 FROM containers WHERE id = ?`, id,
	).Scan(&c.ID, &c.ProjectID, &c.ContainerID, &c.Name, &c.Image, &c.Status, &c.Port, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("db: get container %d: %w", id, err)
	}
	return c, nil
}

// GetContainerByDockerID returns a container by its Docker container ID.
func (d *DB) GetContainerByDockerID(ctx context.Context, containerID string) (*Container, error) {
	ctx = contextOrBg(ctx)
	c := &Container{}
	err := d.conn.QueryRowContext(ctx,
		`SELECT id, project_id, container_id, name, image, status, port, created_at, updated_at
		 FROM containers WHERE container_id = ?`, containerID,
	).Scan(&c.ID, &c.ProjectID, &c.ContainerID, &c.Name, &c.Image, &c.Status, &c.Port, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("db: get container %q: %w", containerID, err)
	}
	return c, nil
}

// ListContainersByProject returns all containers for a given project.
func (d *DB) ListContainersByProject(ctx context.Context, projectID int64) ([]*Container, error) {
	ctx = contextOrBg(ctx)
	rows, err := d.conn.QueryContext(ctx,
		`SELECT id, project_id, container_id, name, image, status, port, created_at, updated_at
		 FROM containers WHERE project_id = ? ORDER BY created_at DESC`, projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("db: list containers for project %d: %w", projectID, err)
	}
	defer rows.Close()

	var out []*Container
	for rows.Next() {
		c := &Container{}
		if err := rows.Scan(&c.ID, &c.ProjectID, &c.ContainerID, &c.Name, &c.Image, &c.Status, &c.Port, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("db: scan container: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// UpdateContainerStatus updates a container's status and timestamp.
func (d *DB) UpdateContainerStatus(ctx context.Context, id int64, status string) error {
	ctx = contextOrBg(ctx)
	_, err := d.conn.ExecContext(ctx,
		`UPDATE containers SET status = ?, updated_at = ? WHERE id = ?`,
		status, nowUTC(), id,
	)
	if err != nil {
		return fmt.Errorf("db: update container %d status: %w", id, err)
	}
	return nil
}

// DeleteContainer removes a container record by ID.
func (d *DB) DeleteContainer(ctx context.Context, id int64) error {
	ctx = contextOrBg(ctx)
	_, err := d.conn.ExecContext(ctx, `DELETE FROM containers WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("db: delete container %d: %w", id, err)
	}
	return nil
}

// DeleteContainerByDockerID removes a container record by Docker ID or Name.
func (d *DB) DeleteContainerByDockerID(ctx context.Context, containerID string) error {
	ctx = contextOrBg(ctx)
	_, err := d.conn.ExecContext(ctx, `DELETE FROM containers WHERE container_id = ? OR name = ?`, containerID, containerID)
	if err != nil {
		return fmt.Errorf("db: delete container by docker id %q: %w", containerID, err)
	}
	return nil
}
