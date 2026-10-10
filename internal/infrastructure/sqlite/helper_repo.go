package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/domus-hub/domus/internal/domain/helper"
	"github.com/google/uuid"
)

var _ helper.Repository = (*HelperRepository)(nil)

// HelperRepository implements helper.Repository for SQLite.
type HelperRepository struct {
	db *sql.DB
}

// NewHelperRepository constructs a HelperRepository.
func NewHelperRepository(db *sql.DB) *HelperRepository {
	return &HelperRepository{db: db}
}

// Create inserts a new user helper.
func (r *HelperRepository) Create(ctx context.Context, h *helper.Helper) error {
	if h.ID == "" {
		h.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	if h.CreatedAt.IsZero() {
		h.CreatedAt = now
	}
	if h.UpdatedAt.IsZero() {
		h.UpdatedAt = now
	}

	query := `
		INSERT INTO user_helpers (id, entity_key, name, domain, state, config_json, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(
		ctx, query,
		h.ID, h.EntityKey, h.Name, h.Domain, h.State, h.ConfigJSON,
		h.CreatedAt.Format(time.RFC3339),
		h.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("sqlite: failed to insert helper: %w", err)
	}
	return nil
}

// GetByID finds a helper by its primary ID.
func (r *HelperRepository) GetByID(ctx context.Context, id string) (*helper.Helper, error) {
	query := `
		SELECT id, entity_key, name, domain, state, config_json, created_at, updated_at
		FROM user_helpers
		WHERE id = ?
	`
	row := r.db.QueryRowContext(ctx, query, id)

	var h helper.Helper
	var createdAtStr, updatedAtStr string
	err := row.Scan(&h.ID, &h.EntityKey, &h.Name, &h.Domain, &h.State, &h.ConfigJSON, &createdAtStr, &updatedAtStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, helper.ErrHelperNotFound
		}
		return nil, fmt.Errorf("sqlite: query helper by id failed: %w", err)
	}
	h.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	h.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
	return &h, nil
}

// GetByEntityKey retrieves a helper by entity key e.g. input_boolean.cleaning_mode.
func (r *HelperRepository) GetByEntityKey(ctx context.Context, entityKey string) (*helper.Helper, error) {
	query := `
		SELECT id, entity_key, name, domain, state, config_json, created_at, updated_at
		FROM user_helpers
		WHERE entity_key = ?
	`
	row := r.db.QueryRowContext(ctx, query, entityKey)

	var h helper.Helper
	var createdAtStr, updatedAtStr string
	err := row.Scan(&h.ID, &h.EntityKey, &h.Name, &h.Domain, &h.State, &h.ConfigJSON, &createdAtStr, &updatedAtStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, helper.ErrHelperNotFound
		}
		return nil, fmt.Errorf("sqlite: query helper by entity_key failed: %w", err)
	}
	h.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	h.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
	return &h, nil
}

// List returns all helpers, optionally filtered by domain.
func (r *HelperRepository) List(ctx context.Context, domain string) ([]*helper.Helper, error) {
	var query string
	var rows *sql.Rows
	var err error

	if domain != "" {
		query = `
			SELECT id, entity_key, name, domain, state, config_json, created_at, updated_at
			FROM user_helpers
			WHERE domain = ?
			ORDER BY name ASC
		`
		rows, err = r.db.QueryContext(ctx, query, domain)
	} else {
		query = `
			SELECT id, entity_key, name, domain, state, config_json, created_at, updated_at
			FROM user_helpers
			ORDER BY domain ASC, name ASC
		`
		rows, err = r.db.QueryContext(ctx, query)
	}
	if err != nil {
		return nil, fmt.Errorf("sqlite: list helpers failed: %w", err)
	}
	defer rows.Close()

	var helpers []*helper.Helper
	for rows.Next() {
		var h helper.Helper
		var createdAtStr, updatedAtStr string
		if err := rows.Scan(&h.ID, &h.EntityKey, &h.Name, &h.Domain, &h.State, &h.ConfigJSON, &createdAtStr, &updatedAtStr); err != nil {
			return nil, fmt.Errorf("sqlite: scan helper row failed: %w", err)
		}
		h.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		h.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
		helpers = append(helpers, &h)
	}
	return helpers, rows.Err()
}

// Update updates state, name, and config_json.
func (r *HelperRepository) Update(ctx context.Context, h *helper.Helper) error {
	h.UpdatedAt = time.Now().UTC()
	query := `
		UPDATE user_helpers
		SET name = ?, state = ?, config_json = ?, updated_at = ?
		WHERE id = ?
	`
	res, err := r.db.ExecContext(ctx, query, h.Name, h.State, h.ConfigJSON, h.UpdatedAt.Format(time.RFC3339), h.ID)
	if err != nil {
		return fmt.Errorf("sqlite: update helper failed: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return helper.ErrHelperNotFound
	}
	return nil
}

// Delete removes a helper by ID.
func (r *HelperRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM user_helpers WHERE id = ?`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("sqlite: delete helper failed: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return helper.ErrHelperNotFound
	}
	return nil
}
