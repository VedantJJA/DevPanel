package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// ServiceRecord stores per-service runtime settings that are editable from the UI
// and merged into the Blueprint at deploy time.
type ServiceRecord struct {
	ID           int64             `json:"id"`
	ProjectID    string            `json:"project_id"`
	Name         string            `json:"name"`
	Slug         string            `json:"slug"`
	FQDN         string            `json:"fqdn"`          // Full URL, e.g. "https://myapp.klouds.online" (Coolify-style)
	Type         string            `json:"type"`
	Image        string            `json:"image"`
	EnvVars      map[string]string `json:"env_vars"`
	Port         int               `json:"port"`
	CustomDomain string            `json:"custom_domain"` // Legacy: kept for backwards compat
	AutoDeploy   bool              `json:"auto_deploy"`
	BuildCommand string            `json:"build_command"`
	StartCommand string            `json:"start_command"`
	InstanceType string            `json:"instance_type"`
	Runtime      string            `json:"runtime"`
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
	INSERT INTO services (project_id, name, slug, fqdn, type, image, env_json, port, custom_domain,
	                      auto_deploy, build_command, start_command, instance_type, runtime)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(project_id, name) DO UPDATE SET
	    slug           = excluded.slug,
	    fqdn           = excluded.fqdn,
	    type           = excluded.type,
	    image          = excluded.image,
	    env_json       = excluded.env_json,
	    port           = excluded.port,
	    custom_domain  = excluded.custom_domain,
	    auto_deploy    = excluded.auto_deploy,
	    build_command  = excluded.build_command,
	    start_command  = excluded.start_command,
	    instance_type  = excluded.instance_type,
	    runtime        = excluded.runtime,
	    updated_at     = strftime('%Y-%m-%dT%H:%M:%SZ','now');`
	_, err = d.conn.ExecContext(ctx, q, s.ProjectID, s.Name, s.Slug, s.FQDN, s.Type, s.Image, string(envJSON),
		s.Port, s.CustomDomain, autoDeploy, s.BuildCommand, s.StartCommand, s.InstanceType, s.Runtime)
	if err != nil {
		return fmt.Errorf("db: upsert service %s/%s: %w", s.ProjectID, s.Name, err)
	}
	return nil
}

// ListServices returns all services for a project.
func (d *DB) ListServices(ctx context.Context, projectID string) ([]ServiceRecord, error) {
	ctx = contextOrBg(ctx)
	q := `
	SELECT id, name, slug, fqdn, type, image, env_json, port, custom_domain,
	       auto_deploy, build_command, start_command, instance_type, runtime, created_at, updated_at
	FROM services
	WHERE project_id = ?
	ORDER BY created_at ASC`
	rows, err := d.conn.QueryContext(ctx, q, projectID)
	if err != nil {
		return nil, fmt.Errorf("db: list services for %s: %w", projectID, err)
	}
	defer rows.Close()

	var out []ServiceRecord
	for rows.Next() {
		var s ServiceRecord
		var envJSON string
		var ad int
		s.ProjectID = projectID
		if err := rows.Scan(&s.ID, &s.Name, &s.Slug, &s.FQDN, &s.Type, &s.Image, &envJSON, &s.Port, &s.CustomDomain,
			&ad, &s.BuildCommand, &s.StartCommand, &s.InstanceType, &s.Runtime, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("db: scan service: %w", err)
		}
		_ = json.Unmarshal([]byte(envJSON), &s.EnvVars)
		s.AutoDeploy = ad == 1
		out = append(out, s)
	}
	return out, rows.Err()
}

// GetService returns a single service by project + name.
func (d *DB) GetService(ctx context.Context, projectID, name string) (*ServiceRecord, error) {
	ctx = contextOrBg(ctx)
	var s ServiceRecord
	var envJSON string
	var ad int
	q := `
	SELECT id, slug, fqdn, type, image, env_json, port, custom_domain,
	       auto_deploy, build_command, start_command, instance_type, runtime, created_at, updated_at
	FROM services WHERE project_id = ? AND name = ?`
	err := d.conn.QueryRowContext(ctx, q, projectID, name).Scan(
		&s.ID, &s.Slug, &s.FQDN, &s.Type, &s.Image, &envJSON, &s.Port, &s.CustomDomain,
		&ad, &s.BuildCommand, &s.StartCommand, &s.InstanceType, &s.Runtime, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("db: get service: %w", err)
	}
	s.ProjectID = projectID
	s.Name = name
	_ = json.Unmarshal([]byte(envJSON), &s.EnvVars)
	s.AutoDeploy = ad == 1
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

// FindServiceByName returns a service record by name, slug, or custom domain across all projects.
func (d *DB) FindServiceByName(ctx context.Context, nameOrDomain string) (*ServiceRecord, error) {
	ctx = contextOrBg(ctx)
	var s ServiceRecord
	var envJSON string
	var ad int
	q := `
	SELECT id, project_id, name, slug, fqdn, type, image, env_json, port, custom_domain,
	       auto_deploy, build_command, start_command, instance_type, runtime, created_at, updated_at
	FROM services WHERE name = ? OR slug = ? OR custom_domain = ? LIMIT 1`
	err := d.conn.QueryRowContext(ctx, q, nameOrDomain, nameOrDomain, nameOrDomain).Scan(
		&s.ID, &s.ProjectID, &s.Name, &s.Slug, &s.FQDN, &s.Type, &s.Image, &envJSON, &s.Port, &s.CustomDomain,
		&ad, &s.BuildCommand, &s.StartCommand, &s.InstanceType, &s.Runtime, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("db: find service: %w", err)
	}
	_ = json.Unmarshal([]byte(envJSON), &s.EnvVars)
	s.AutoDeploy = ad == 1
	return &s, nil
}

// FindServiceBySlug returns a service record by its unique URL slug.
func (d *DB) FindServiceBySlug(ctx context.Context, slug string) (*ServiceRecord, error) {
	ctx = contextOrBg(ctx)
	var s ServiceRecord
	var envJSON string
	var ad int
	q := `
	SELECT id, project_id, name, slug, fqdn, type, image, env_json, port, custom_domain,
	       auto_deploy, build_command, start_command, instance_type, runtime, created_at, updated_at
	FROM services WHERE slug = ? LIMIT 1`
	err := d.conn.QueryRowContext(ctx, q, slug).Scan(
		&s.ID, &s.ProjectID, &s.Name, &s.Slug, &s.FQDN, &s.Type, &s.Image, &envJSON, &s.Port, &s.CustomDomain,
		&ad, &s.BuildCommand, &s.StartCommand, &s.InstanceType, &s.Runtime, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("db: find service by slug %q: %w", slug, err)
	}
	_ = json.Unmarshal([]byte(envJSON), &s.EnvVars)
	s.AutoDeploy = ad == 1
	return &s, nil
}

// FindServiceByFQDN looks up a service whose FQDN host matches the given hostname.
// This is the primary Coolify-style routing lookup: Host header → service.
func (d *DB) FindServiceByFQDN(ctx context.Context, host string) (*ServiceRecord, error) {
	ctx = contextOrBg(ctx)
	// host may be "myapp.klouds.online" or "myapp.klouds.online:8090"
	hostOnly := strings.Split(host, ":")[0]

	// Query all services that have a non-empty fqdn
	q := `
	SELECT id, project_id, name, slug, fqdn, type, image, env_json, port, custom_domain,
	       auto_deploy, build_command, start_command, instance_type, runtime, created_at, updated_at
	FROM services WHERE fqdn != '' `
	rows, err := d.conn.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("db: find service by fqdn: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var s ServiceRecord
		var envJSON string
		var ad int
		if err := rows.Scan(&s.ID, &s.ProjectID, &s.Name, &s.Slug, &s.FQDN, &s.Type, &s.Image, &envJSON, &s.Port, &s.CustomDomain,
			&ad, &s.BuildCommand, &s.StartCommand, &s.InstanceType, &s.Runtime, &s.CreatedAt, &s.UpdatedAt); err != nil {
			continue
		}
		_ = json.Unmarshal([]byte(envJSON), &s.EnvVars)
		s.AutoDeploy = ad == 1

		// Compare host portion of stored FQDN against incoming host header
		if s.FQDN != "" {
			parsed, parseErr := url.Parse(s.FQDN)
			if parseErr == nil {
				fqdnHost := strings.Split(parsed.Host, ":")[0]
				if strings.EqualFold(fqdnHost, hostOnly) {
					return &s, nil
				}
			}
		}
	}
	return nil, nil
}

// SlugExists returns true if any service already uses the given URL slug.
func (d *DB) SlugExists(ctx context.Context, slug string) (bool, error) {
	ctx = contextOrBg(ctx)
	var count int
	err := d.conn.QueryRowContext(ctx,
		`SELECT COUNT(1) FROM services WHERE slug = ?`, slug,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("db: slug exists %q: %w", slug, err)
	}
	return count > 0, nil
}

// FQDNExists returns true if any service already uses a FQDN with the given host.
func (d *DB) FQDNExists(ctx context.Context, host string) (bool, error) {
	svc, err := d.FindServiceByFQDN(ctx, host)
	if err != nil {
		return false, err
	}
	return svc != nil, nil
}
