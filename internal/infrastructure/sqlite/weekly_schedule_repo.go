// Package sqlite provides SQLite implementation for weekly matrix schedules.
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/domus-hub/domus/internal/domain/schedule"
	"github.com/google/uuid"
)

// WeeklyScheduleRepository persists weekly schedules in SQLite.
type WeeklyScheduleRepository struct {
	db *sql.DB
}

// NewWeeklyScheduleRepository creates a new WeeklyScheduleRepository.
func NewWeeklyScheduleRepository(db *sql.DB) *WeeklyScheduleRepository {
	return &WeeklyScheduleRepository{db: db}
}

// Save inserts or updates a weekly schedule.
func (r *WeeklyScheduleRepository) Save(ctx context.Context, s *schedule.WeeklySchedule) error {
	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	if s.CreatedAt.IsZero() {
		s.CreatedAt = now
	}
	s.UpdatedAt = now

	gridJSON, err := json.Marshal(s.Grid)
	if err != nil {
		gridJSON = []byte("{}")
	}

	query := `
		INSERT INTO weekly_schedules (
			id, name, icon, target_entity_id, target_service, grid_json, current_slot_state, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			icon = excluded.icon,
			target_entity_id = excluded.target_entity_id,
			target_service = excluded.target_service,
			grid_json = excluded.grid_json,
			current_slot_state = excluded.current_slot_state,
			updated_at = excluded.updated_at
	`
	_, err = r.db.ExecContext(ctx, query,
		s.ID,
		s.Name,
		s.Icon,
		s.TargetEntityID,
		s.TargetService,
		string(gridJSON),
		s.CurrentSlotState,
		s.CreatedAt.Format(time.RFC3339),
		s.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("failed to save weekly schedule: %w", err)
	}
	return nil
}

// GetByID finds a weekly schedule by ID.
func (r *WeeklyScheduleRepository) GetByID(ctx context.Context, id string) (*schedule.WeeklySchedule, error) {
	query := `
		SELECT id, name, icon, target_entity_id, target_service, grid_json, current_slot_state, created_at, updated_at
		FROM weekly_schedules
		WHERE id = ?
	`
	row := r.db.QueryRowContext(ctx, query, id)

	var (
		s            schedule.WeeklySchedule
		gridJSON     string
		createdAtStr string
		updatedAtStr string
	)
	err := row.Scan(
		&s.ID,
		&s.Name,
		&s.Icon,
		&s.TargetEntityID,
		&s.TargetService,
		&gridJSON,
		&s.CurrentSlotState,
		&createdAtStr,
		&updatedAtStr,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, schedule.ErrScheduleNotFound
		}
		return nil, fmt.Errorf("failed to scan weekly schedule: %w", err)
	}

	_ = json.Unmarshal([]byte(gridJSON), &s.Grid)
	if s.Grid == nil {
		s.Grid = make(schedule.WeeklyGrid)
	}
	if parsed, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
		s.CreatedAt = parsed
	}
	if parsed, err := time.Parse(time.RFC3339, updatedAtStr); err == nil {
		s.UpdatedAt = parsed
	}

	return &s, nil
}

// List returns all persisted weekly schedules.
func (r *WeeklyScheduleRepository) List(ctx context.Context) ([]*schedule.WeeklySchedule, error) {
	query := `
		SELECT id, name, icon, target_entity_id, target_service, grid_json, current_slot_state, created_at, updated_at
		FROM weekly_schedules
		ORDER BY name ASC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query weekly schedules: %w", err)
	}
	defer rows.Close()

	var results []*schedule.WeeklySchedule
	for rows.Next() {
		var (
			s            schedule.WeeklySchedule
			gridJSON     string
			createdAtStr string
			updatedAtStr string
		)
		err := rows.Scan(
			&s.ID,
			&s.Name,
			&s.Icon,
			&s.TargetEntityID,
			&s.TargetService,
			&gridJSON,
			&s.CurrentSlotState,
			&createdAtStr,
			&updatedAtStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan weekly schedule row: %w", err)
		}
		_ = json.Unmarshal([]byte(gridJSON), &s.Grid)
		if s.Grid == nil {
			s.Grid = make(schedule.WeeklyGrid)
		}
		if parsed, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			s.CreatedAt = parsed
		}
		if parsed, err := time.Parse(time.RFC3339, updatedAtStr); err == nil {
			s.UpdatedAt = parsed
		}
		results = append(results, &s)
	}
	return results, nil
}

// Delete removes a weekly schedule by ID.
func (r *WeeklyScheduleRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, "DELETE FROM weekly_schedules WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete weekly schedule: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return schedule.ErrScheduleNotFound
	}
	return nil
}
