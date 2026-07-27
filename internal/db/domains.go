package db

import (
	"context"
	"database/sql"
	"fmt"
)

// Domain represents a custom domain (FQDN) mapped to a project.
type Domain struct {
	ID        int64  `json:"id"`
	ProjectID int64  `json:"project_id"`
	FQDN      string `json:"fqdn"` // e.g. "app.example.com"
	TLS       bool   `json:"tls"`  // whether to request a certificate
	CreatedAt string `json:"created_at"`
}

// CreateDomain inserts a new domain mapping and returns its ID.
func (d *DB) CreateDomain(ctx context.Context, dom *Domain) (int64, error) {
	ctx = contextOrBg(ctx)
	tlsInt := 0
	if dom.TLS {
		tlsInt = 1
	}
	res, err := d.conn.ExecContext(ctx,
		`INSERT INTO domains (project_id, fqdn, tls, created_at)
		 VALUES (?, ?, ?, ?)`,
		dom.ProjectID, dom.FQDN, tlsInt, nowUTC(),
	)
	if err != nil {
		return 0, fmt.Errorf("db: create domain: %w", err)
	}
	return res.LastInsertId()
}

// GetDomain returns a domain by its database ID.
func (d *DB) GetDomain(ctx context.Context, id int64) (*Domain, error) {
	ctx = contextOrBg(ctx)
	dom := &Domain{}
	var tlsInt int
	err := d.conn.QueryRowContext(ctx,
		`SELECT id, project_id, fqdn, tls, created_at
		 FROM domains WHERE id = ?`, id,
	).Scan(&dom.ID, &dom.ProjectID, &dom.FQDN, &tlsInt, &dom.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("db: get domain %d: %w", id, err)
	}
	dom.TLS = tlsInt == 1
	return dom, nil
}

// GetDomainByFQDN looks up a domain by its fully qualified domain name.
// Used by the /ask endpoint for On-Demand TLS verification.
func (d *DB) GetDomainByFQDN(ctx context.Context, fqdn string) (*Domain, error) {
	ctx = contextOrBg(ctx)
	dom := &Domain{}
	var tlsInt int
	err := d.conn.QueryRowContext(ctx,
		`SELECT id, project_id, fqdn, tls, created_at
		 FROM domains WHERE fqdn = ?`, fqdn,
	).Scan(&dom.ID, &dom.ProjectID, &dom.FQDN, &tlsInt, &dom.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("db: get domain %q: %w", fqdn, err)
	}
	dom.TLS = tlsInt == 1
	return dom, nil
}

// ListDomainsByProject returns all domains for a given project.
func (d *DB) ListDomainsByProject(ctx context.Context, projectID int64) ([]*Domain, error) {
	ctx = contextOrBg(ctx)
	rows, err := d.conn.QueryContext(ctx,
		`SELECT id, project_id, fqdn, tls, created_at
		 FROM domains WHERE project_id = ? ORDER BY fqdn`, projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("db: list domains for project %d: %w", projectID, err)
	}
	defer rows.Close()

	var out []*Domain
	for rows.Next() {
		dom := &Domain{}
		var tlsInt int
		if err := rows.Scan(&dom.ID, &dom.ProjectID, &dom.FQDN, &tlsInt, &dom.CreatedAt); err != nil {
			return nil, fmt.Errorf("db: scan domain: %w", err)
		}
		dom.TLS = tlsInt == 1
		out = append(out, dom)
	}
	return out, rows.Err()
}

// DeleteDomain removes a domain by ID.
func (d *DB) DeleteDomain(ctx context.Context, id int64) error {
	ctx = contextOrBg(ctx)
	_, err := d.conn.ExecContext(ctx, `DELETE FROM domains WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("db: delete domain %d: %w", id, err)
	}
	return nil
}

// DomainExists checks whether a domain with the given FQDN is registered.
// This is the fast-path query used by the /ask TLS verification endpoint.
func (d *DB) DomainExists(ctx context.Context, fqdn string) (bool, error) {
	ctx = contextOrBg(ctx)
	var count int
	err := d.conn.QueryRowContext(ctx,
		`SELECT COUNT(1) FROM domains WHERE fqdn = ?`, fqdn,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("db: domain exists %q: %w", fqdn, err)
	}
	return count > 0, nil
}
