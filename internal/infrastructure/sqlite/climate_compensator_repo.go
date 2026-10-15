// Package sqlite provides SQLite implementation for climate compensator repository.
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/domus-hub/domus/internal/domain/climate"
	"github.com/google/uuid"
)

// ClimateCompensatorRepository persists climate compensation configurations in SQLite.
type ClimateCompensatorRepository struct {
	db *sql.DB
}

// NewClimateCompensatorRepository creates a new ClimateCompensatorRepository.
func NewClimateCompensatorRepository(db *sql.DB) *ClimateCompensatorRepository {
	return &ClimateCompensatorRepository{db: db}
}

// Save inserts or updates a climate compensation record.
func (r *ClimateCompensatorRepository) Save(ctx context.Context, c *climate.ClimateCompensation) error {
	if c.ID == "" {
		c.ID = uuid.NewString()
	}
	c.UpdatedAt = time.Now().UTC()

	var windowOpenInt int
	if c.WindowOpen {
		windowOpenInt = 1
	}
	var enabledInt int
	if c.Enabled {
		enabledInt = 1
	}

	query := `
		INSERT INTO climate_compensations (
			id, trv_entity_key, room_temp_sensor_key, window_sensor_key,
			target_temperature, delta_factor, compensated_setpoint, window_open, enabled, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(trv_entity_key) DO UPDATE SET
			room_temp_sensor_key = excluded.room_temp_sensor_key,
			window_sensor_key = excluded.window_sensor_key,
			target_temperature = excluded.target_temperature,
			delta_factor = excluded.delta_factor,
			compensated_setpoint = excluded.compensated_setpoint,
			window_open = excluded.window_open,
			enabled = excluded.enabled,
			updated_at = excluded.updated_at
	`
	_, err := r.db.ExecContext(ctx, query,
		c.ID,
		c.TRVEntityKey,
		c.RoomTempSensorKey,
		c.WindowSensorKey,
		c.TargetTemperature,
		c.DeltaFactor,
		c.CompensatedSetpoint,
		windowOpenInt,
		enabledInt,
		c.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("failed to save climate compensation: %w", err)
	}
	return nil
}

// GetByID finds a compensation record by ID.
func (r *ClimateCompensatorRepository) GetByID(ctx context.Context, id string) (*climate.ClimateCompensation, error) {
	query := `
		SELECT id, trv_entity_key, room_temp_sensor_key, window_sensor_key,
		       target_temperature, delta_factor, compensated_setpoint, window_open, enabled, updated_at
		FROM climate_compensations
		WHERE id = ?
	`
	row := r.db.QueryRowContext(ctx, query, id)
	return scanCompensation(row)
}

// GetByTRV finds a compensation record by TRV entity key.
func (r *ClimateCompensatorRepository) GetByTRV(ctx context.Context, trvKey string) (*climate.ClimateCompensation, error) {
	query := `
		SELECT id, trv_entity_key, room_temp_sensor_key, window_sensor_key,
		       target_temperature, delta_factor, compensated_setpoint, window_open, enabled, updated_at
		FROM climate_compensations
		WHERE trv_entity_key = ?
	`
	row := r.db.QueryRowContext(ctx, query, trvKey)
	return scanCompensation(row)
}

// List returns all registered climate compensations.
func (r *ClimateCompensatorRepository) List(ctx context.Context) ([]*climate.ClimateCompensation, error) {
	query := `
		SELECT id, trv_entity_key, room_temp_sensor_key, window_sensor_key,
		       target_temperature, delta_factor, compensated_setpoint, window_open, enabled, updated_at
		FROM climate_compensations
		ORDER BY trv_entity_key ASC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query climate compensations: %w", err)
	}
	defer rows.Close()

	var results []*climate.ClimateCompensation
	for rows.Next() {
		var (
			c            climate.ClimateCompensation
			windowOpen   int
			enabled      int
			updatedAtStr string
		)
		err := rows.Scan(
			&c.ID,
			&c.TRVEntityKey,
			&c.RoomTempSensorKey,
			&c.WindowSensorKey,
			&c.TargetTemperature,
			&c.DeltaFactor,
			&c.CompensatedSetpoint,
			&windowOpen,
			&enabled,
			&updatedAtStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan climate compensation: %w", err)
		}
		c.WindowOpen = windowOpen == 1
		c.Enabled = enabled == 1
		if parsed, err := time.Parse(time.RFC3339, updatedAtStr); err == nil {
			c.UpdatedAt = parsed
		}
		results = append(results, &c)
	}
	return results, nil
}

// Delete removes a climate compensation record by ID.
func (r *ClimateCompensatorRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, "DELETE FROM climate_compensations WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete climate compensation: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return climate.ErrCompensationNotFound
	}
	return nil
}

type scannable interface {
	Scan(dest ...any) error
}

func scanCompensation(row scannable) (*climate.ClimateCompensation, error) {
	var (
		c            climate.ClimateCompensation
		windowOpen   int
		enabled      int
		updatedAtStr string
	)
	err := row.Scan(
		&c.ID,
		&c.TRVEntityKey,
		&c.RoomTempSensorKey,
		&c.WindowSensorKey,
		&c.TargetTemperature,
		&c.DeltaFactor,
		&c.CompensatedSetpoint,
		&windowOpen,
		&enabled,
		&updatedAtStr,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, climate.ErrCompensationNotFound
		}
		return nil, fmt.Errorf("failed to scan climate compensation: %w", err)
	}
	c.WindowOpen = windowOpen == 1
	c.Enabled = enabled == 1
	if parsed, err := time.Parse(time.RFC3339, updatedAtStr); err == nil {
		c.UpdatedAt = parsed
	}
	return &c, nil
}
