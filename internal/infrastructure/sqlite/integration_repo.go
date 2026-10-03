// Package sqlite provides SQLite repository implementations for integrations.
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/domus-hub/domus/internal/domain/integration"
)

// IntegrationRepository implements integration.Repository for SQLite.
type IntegrationRepository struct {
	db *sql.DB
}

// NewIntegrationRepository constructs an IntegrationRepository.
func NewIntegrationRepository(db *sql.DB) *IntegrationRepository {
	return &IntegrationRepository{db: db}
}

// Create inserts a new integration record.
func (r *IntegrationRepository) Create(ctx context.Context, in *integration.Integration) error {
	query := `
		INSERT INTO integrations (id, type, name, enabled, config_encrypted, status, last_error, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);
	`
	enabled := 0
	if in.Enabled {
		enabled = 1
	}

	status := in.Status
	if status == "" {
		status = "disconnected"
	}

	var lastError *string
	if in.LastError != "" {
		lastError = &in.LastError
	}

	createdAt := in.CreatedAt.Format(time.RFC3339)
	updatedAt := in.UpdatedAt.Format(time.RFC3339)

	_, err := r.db.ExecContext(ctx, query,
		in.ID,
		in.Type,
		in.Name,
		enabled,
		in.ConfigEncrypted,
		status,
		lastError,
		createdAt,
		updatedAt,
	)
	if err != nil {
		return fmt.Errorf("sqlite: failed to insert integration: %w", err)
	}
	return nil
}

// GetByID retrieves an integration by its ID.
func (r *IntegrationRepository) GetByID(ctx context.Context, id string) (*integration.Integration, error) {
	query := `
		SELECT id, type, name, enabled, config_encrypted, status, last_error, created_at, updated_at
		FROM integrations
		WHERE id = ?;
	`
	row := r.db.QueryRowContext(ctx, query, id)

	var (
		in        integration.Integration
		enabled   int
		lastError sql.NullString
		createdAt string
		updatedAt string
	)

	err := row.Scan(
		&in.ID,
		&in.Type,
		&in.Name,
		&enabled,
		&in.ConfigEncrypted,
		&in.Status,
		&lastError,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, integration.ErrIntegrationNotFound
		}
		return nil, fmt.Errorf("sqlite: scan integration: %w", err)
	}

	in.Enabled = enabled == 1
	if lastError.Valid {
		in.LastError = lastError.String
	}
	if parsed, err := time.Parse(time.RFC3339, createdAt); err == nil {
		in.CreatedAt = parsed
	}
	if parsed, err := time.Parse(time.RFC3339, updatedAt); err == nil {
		in.UpdatedAt = parsed
	}

	return &in, nil
}

// List retrieves all integrations.
func (r *IntegrationRepository) List(ctx context.Context) ([]*integration.Integration, error) {
	query := `
		SELECT id, type, name, enabled, config_encrypted, status, last_error, created_at, updated_at
		FROM integrations
		ORDER BY name ASC;
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("sqlite: list integrations: %w", err)
	}
	defer rows.Close()

	var integrations []*integration.Integration
	for rows.Next() {
		var (
			in        integration.Integration
			enabled   int
			lastError sql.NullString
			createdAt string
			updatedAt string
		)
		if err := rows.Scan(
			&in.ID,
			&in.Type,
			&in.Name,
			&enabled,
			&in.ConfigEncrypted,
			&in.Status,
			&lastError,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, fmt.Errorf("sqlite: scan integration: %w", err)
		}
		in.Enabled = enabled == 1
		if lastError.Valid {
			in.LastError = lastError.String
		}
		if parsed, err := time.Parse(time.RFC3339, createdAt); err == nil {
			in.CreatedAt = parsed
		}
		if parsed, err := time.Parse(time.RFC3339, updatedAt); err == nil {
			in.UpdatedAt = parsed
		}
		integrations = append(integrations, &in)
	}
	return integrations, rows.Err()
}

// Update modifies an existing integration record.
func (r *IntegrationRepository) Update(ctx context.Context, in *integration.Integration) error {
	query := `
		UPDATE integrations
		SET name = ?, enabled = ?, config_encrypted = ?, updated_at = ?
		WHERE id = ?;
	`
	enabled := 0
	if in.Enabled {
		enabled = 1
	}
	updatedAt := time.Now().UTC().Format(time.RFC3339)

	res, err := r.db.ExecContext(ctx, query,
		in.Name,
		enabled,
		in.ConfigEncrypted,
		updatedAt,
		in.ID,
	)
	if err != nil {
		return fmt.Errorf("sqlite: failed to update integration: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return integration.ErrIntegrationNotFound
	}
	return nil
}

// SetStatus updates status and last_error for an integration.
func (r *IntegrationRepository) SetStatus(ctx context.Context, id string, status, lastError string) error {
	query := `
		UPDATE integrations
		SET status = ?, last_error = ?, updated_at = ?
		WHERE id = ?;
	`
	var errStr *string
	if lastError != "" {
		errStr = &lastError
	}
	updatedAt := time.Now().UTC().Format(time.RFC3339)

	res, err := r.db.ExecContext(ctx, query, status, errStr, updatedAt, id)
	if err != nil {
		return fmt.Errorf("sqlite: failed to set integration status: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return integration.ErrIntegrationNotFound
	}
	return nil
}

// Delete removes an integration by its ID.
func (r *IntegrationRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM integrations WHERE id = ?;`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("sqlite: failed to delete integration: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return integration.ErrIntegrationNotFound
	}
	return nil
}
