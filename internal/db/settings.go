package db

import (
	"context"
	"fmt"
)

// GetSetting retrieves a single key-value setting from the settings table.
func (d *DB) GetSetting(ctx context.Context, key string) (string, error) {
	ctx = contextOrBg(ctx)
	var value string
	err := d.conn.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	if err != nil {
		return "", fmt.Errorf("db: get setting %s: %w", key, err)
	}
	return value, nil
}

// SetSetting upserts a key-value setting.
func (d *DB) SetSetting(ctx context.Context, key, value string) error {
	ctx = contextOrBg(ctx)
	_, err := d.conn.ExecContext(ctx, `
		INSERT INTO settings (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value;
	`, key, value)
	if err != nil {
		return fmt.Errorf("db: set setting %s: %w", key, err)
	}
	return nil
}

// GetAllSettings returns all settings as a map.
func (d *DB) GetAllSettings(ctx context.Context) (map[string]string, error) {
	ctx = contextOrBg(ctx)
	rows, err := d.conn.QueryContext(ctx, `SELECT key, value FROM settings`)
	if err != nil {
		return nil, fmt.Errorf("db: list settings: %w", err)
	}
	defer rows.Close()
	m := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		m[k] = v
	}
	return m, rows.Err()
}

// DeleteSetting removes a setting by key.
func (d *DB) DeleteSetting(ctx context.Context, key string) error {
	ctx = contextOrBg(ctx)
	_, err := d.conn.ExecContext(ctx, `DELETE FROM settings WHERE key = ?`, key)
	if err != nil {
		return fmt.Errorf("db: delete setting %s: %w", key, err)
	}
	return nil
}
