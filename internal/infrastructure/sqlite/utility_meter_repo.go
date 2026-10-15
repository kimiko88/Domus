// Package sqlite provides SQLite persistence for multi-utility meters.
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	domenergy "github.com/domus-hub/domus/internal/domain/energy"
	"github.com/google/uuid"
)

var _ domenergy.UtilityMeterRepository = (*UtilityMeterRepository)(nil)

// UtilityMeterRepository manages persistence for electricity, gas, and water meters.
type UtilityMeterRepository struct {
	db *sql.DB
}

// NewUtilityMeterRepository constructs a UtilityMeterRepository.
func NewUtilityMeterRepository(db *sql.DB) *UtilityMeterRepository {
	return &UtilityMeterRepository{db: db}
}

// Create inserts a new utility meter record.
func (r *UtilityMeterRepository) Create(ctx context.Context, m *domenergy.UtilityMeter) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	}
	m.UpdatedAt = now

	var resetStr sql.NullString
	if m.LastReset != nil {
		resetStr = sql.NullString{String: m.LastReset.Format(time.RFC3339), Valid: true}
	}

	query := `
		INSERT INTO energy_utility_meters (
			id, name, utility_type, unit, cost_per_unit, source_entity_id,
			current_reading, last_reset, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query,
		m.ID, m.Name, m.UtilityType, m.Unit, m.CostPerUnit, m.SourceEntityID,
		m.CurrentReading, resetStr, m.CreatedAt.Format(time.RFC3339), m.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("sqlite: failed to insert utility meter: %w", err)
	}
	return nil
}

// Get retrieves a meter by ID.
func (r *UtilityMeterRepository) Get(ctx context.Context, id string) (*domenergy.UtilityMeter, error) {
	query := `
		SELECT id, name, utility_type, unit, cost_per_unit, source_entity_id,
		       current_reading, last_reset, created_at, updated_at
		FROM energy_utility_meters
		WHERE id = ?
	`
	row := r.db.QueryRowContext(ctx, query, id)

	var (
		m          domenergy.UtilityMeter
		resetStr   sql.NullString
		createdStr string
		updatedStr string
	)
	err := row.Scan(
		&m.ID, &m.Name, &m.UtilityType, &m.Unit, &m.CostPerUnit, &m.SourceEntityID,
		&m.CurrentReading, &resetStr, &createdStr, &updatedStr,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domenergy.ErrMeterNotFound
		}
		return nil, fmt.Errorf("sqlite: failed to scan utility meter: %w", err)
	}
	if resetStr.Valid && resetStr.String != "" {
		if t, err := time.Parse(time.RFC3339, resetStr.String); err == nil {
			m.LastReset = &t
		}
	}
	m.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	m.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
	return &m, nil
}

// List returns meters optionally filtered by utility_type.
func (r *UtilityMeterRepository) List(ctx context.Context, utilityType string) ([]*domenergy.UtilityMeter, error) {
	query := `
		SELECT id, name, utility_type, unit, cost_per_unit, source_entity_id,
		       current_reading, last_reset, created_at, updated_at
		FROM energy_utility_meters
	`
	var args []any
	if utilityType != "" {
		query += " WHERE utility_type = ?"
		args = append(args, utilityType)
	}
	query += " ORDER BY name ASC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("sqlite: failed to list utility meters: %w", err)
	}
	defer rows.Close()

	var meters []*domenergy.UtilityMeter
	for rows.Next() {
		var (
			m          domenergy.UtilityMeter
			resetStr   sql.NullString
			createdStr string
			updatedStr string
		)
		if err := rows.Scan(
			&m.ID, &m.Name, &m.UtilityType, &m.Unit, &m.CostPerUnit, &m.SourceEntityID,
			&m.CurrentReading, &resetStr, &createdStr, &updatedStr,
		); err != nil {
			return nil, err
		}
		if resetStr.Valid && resetStr.String != "" {
			if t, err := time.Parse(time.RFC3339, resetStr.String); err == nil {
				m.LastReset = &t
			}
		}
		m.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
		m.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
		meters = append(meters, &m)
	}
	return meters, rows.Err()
}

// Update modifies an existing meter record.
func (r *UtilityMeterRepository) Update(ctx context.Context, m *domenergy.UtilityMeter) error {
	m.UpdatedAt = time.Now().UTC()
	var resetStr sql.NullString
	if m.LastReset != nil {
		resetStr = sql.NullString{String: m.LastReset.Format(time.RFC3339), Valid: true}
	}

	query := `
		UPDATE energy_utility_meters
		SET name = ?, utility_type = ?, unit = ?, cost_per_unit = ?,
		    source_entity_id = ?, current_reading = ?, last_reset = ?, updated_at = ?
		WHERE id = ?
	`
	res, err := r.db.ExecContext(ctx, query,
		m.Name, m.UtilityType, m.Unit, m.CostPerUnit,
		m.SourceEntityID, m.CurrentReading, resetStr, m.UpdatedAt.Format(time.RFC3339),
		m.ID,
	)
	if err != nil {
		return fmt.Errorf("sqlite: failed to update utility meter: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return domenergy.ErrMeterNotFound
	}
	return nil
}

// Delete removes a meter.
func (r *UtilityMeterRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, "DELETE FROM energy_utility_meters WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("sqlite: failed to delete utility meter: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return domenergy.ErrMeterNotFound
	}
	return nil
}
