package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

// ServiceRecord stores per-service runtime settings that are editable from the UI
// and merged into the Blueprint at deploy time.
type ServiceRecord struct {
	ID           int64             `json:"id"`
	ProjectID    string            `json:"project_id"`
	Name         string            `json:"name"`
	Type         string            `json:"type"`
	EnvVars      map[string]string `json:"env_vars"`
	Port         int               `json:"port"`
	CustomDomain string            `json:"custom_domain"`
	AutoDeploy   bool              `json:"auto_deploy"`
	BuildCommand string            `json:"build_command"`
	StartCommand string            `json:"start_command"`
	InstanceType string            `json:"instance_type"`
	CreatedAt    string            `json:"created_at"`
	UpdatedAt    string            `json:"updated_at"`
}

// UpsertService inserts or updates a service record keyed by (project_id, name).
func (d *DB) UpsertService(ctx context.Context, s *ServiceRecord) error {
	ctx = contextOrBg(ctx)
	envJSON, err := json.Marshal(s.EnvVars)
	if err != nil {
		return fmt.Errorf("db: marshal env: %w", err)
	}
	if s.EnvVars == nil {
		envJSON = []byte("{}")
	}
	autoDeploy := 0
	if s.AutoDeploy {
		autoDeploy = 1
	}
	q := `
	INSERT INTO services (project_id, name, type, env_json, port, custom_domain,
	                      auto_deploy, build_command, start_command, instance_type)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(project_id, name) DO UPDATE SET
	    type           = excluded.type,
	    env_json       = excluded.env_json,
	    port           = excluded.port,
	    custom_domain  = excluded.custom_domain,
	    auto_deploy    = excluded.auto_deploy,
	    build_command  = excluded.build_command,
	    start_command  = excluded.start_command,
	    instance_type  = excluded.instance_type,
	    updated_at     = strftime('%Y-%m-%dT%H:%M:%SZ','now');`
	_, err = d.conn.ExecContext(ctx, q, s.ProjectID, s.Name, s.Type, string(envJSON),
		s.Port, s.CustomDomain, autoDeploy, s.BuildCommand, s.StartCommand, s.InstanceType)
	if err != nil {
		return fmt.Errorf("db: upsert service %s/%s: %w", s.ProjectID, s.Name, err)
	}
	return nil
}

// ListServices returns all services for a project.
func (d *DB) ListServices(ctx context.Context, projectID string) ([]ServiceRecord, error) {
	ctx = contextOrBg(ctx)
	rows, err := d.conn.QueryContext(ctx, `
	    SELECT id, project_id, name, type, env_json, port, custom_domain,
	           auto_deploy, build_command, start_command, instance_type, created_at, updated_at
	    FROM services WHERE project_id = ? ORDER BY id ASC;`, projectID)
	if err != nil {
		return nil, fmt.Errorf("db: list services: %w", err)
	}
	defer rows.Close()
	var out []ServiceRecord
	for rows.Next() {
		var s ServiceRecord
		var envJSON string
		var autoDeploy int
		if err := rows.Scan(&s.ID, &s.ProjectID, &s.Name, &s.Type, &envJSON, &s.Port,
			&s.CustomDomain, &autoDeploy, &s.BuildCommand, &s.StartCommand, &s.InstanceType,
			&s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("db: scan service: %w", err)
		}
		_ = json.Unmarshal([]byte(envJSON), &s.EnvVars)
		s.AutoDeploy = autoDeploy == 1
		out = append(out, s)
	}
	return out, rows.Err()
}

// GetService returns a single service by project + name.
func (d *DB) GetService(ctx context.Context, projectID, name string) (*ServiceRecord, error) {
	ctx = contextOrBg(ctx)
	var s ServiceRecord
	var envJSON string
	var autoDeploy int
	err := d.conn.QueryRowContext(ctx, `
	    SELECT id, project_id, name, type, env_json, port, custom_domain,
	           auto_deploy, build_command, start_command, instance_type, created_at, updated_at
	    FROM services WHERE project_id = ? AND name = ?;`, projectID, name).
		Scan(&s.ID, &s.ProjectID, &s.Name, &s.Type, &envJSON, &s.Port, &s.CustomDomain,
			&autoDeploy, &s.BuildCommand, &s.StartCommand, &s.InstanceType, &s.CreatedAt, &s.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("db: get service: %w", err)
	}
	_ = json.Unmarshal([]byte(envJSON), &s.EnvVars)
	s.AutoDeploy = autoDeploy == 1
	return &s, nil
}

// DeleteServicesForProject removes all service rows for a project.
func (d *DB) DeleteServicesForProject(ctx context.Context, projectID string) error {
	ctx = contextOrBg(ctx)
	_, err := d.conn.ExecContext(ctx, `DELETE FROM services WHERE project_id = ?`, projectID)
	if err != nil {
		return fmt.Errorf("db: delete services for %s: %w", projectID, err)
	}
	return nil
}
