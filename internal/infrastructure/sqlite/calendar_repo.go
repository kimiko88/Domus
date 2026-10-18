package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/domus-hub/domus/internal/domain/calendar"
	"github.com/domus-hub/domus/internal/domain/schedule"
	"github.com/google/uuid"
)

// CalendarRepository implements calendar.Repository for SQLite.
type CalendarRepository struct {
	db *sql.DB
}

// NewCalendarRepository creates a new SQLite-backed CalendarRepository.
func NewCalendarRepository(db *sql.DB) *CalendarRepository {
	return &CalendarRepository{db: db}
}

// --- Schedules ---

func (r *CalendarRepository) SaveSchedule(ctx context.Context, s *schedule.Schedule) error {
	if s.ID == "" {
		s.ID = "sched_" + uuid.NewString()[:8]
	}
	now := time.Now().UTC()
	if s.CreatedAt.IsZero() {
		s.CreatedAt = now
	}
	s.UpdatedAt = now

	gridJSON, err := json.Marshal(s.Grid)
	if err != nil {
		return fmt.Errorf("sqlite: failed to marshal schedule grid: %w", err)
	}

	enabledInt := 0
	if s.Enabled {
		enabledInt = 1
	}

	query := `
		INSERT INTO schedules (id, name, entity_key, schedule_json, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			entity_key = excluded.entity_key,
			schedule_json = excluded.schedule_json,
			enabled = excluded.enabled,
			updated_at = excluded.updated_at
	`
	_, err = r.db.ExecContext(
		ctx, query,
		s.ID,
		s.Name,
		s.EntityKey,
		string(gridJSON),
		enabledInt,
		s.CreatedAt.Format(time.RFC3339),
		s.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("sqlite: failed to save schedule: %w", err)
	}
	return nil
}

func (r *CalendarRepository) GetSchedule(ctx context.Context, id string) (*schedule.Schedule, error) {
	query := `
		SELECT id, name, entity_key, schedule_json, enabled, created_at, updated_at
		FROM schedules
		WHERE id = ?
	`
	var (
		s          schedule.Schedule
		gridJSON   string
		enabledInt int
		createdAt  string
		updatedAt  string
	)
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&s.ID,
		&s.Name,
		&s.EntityKey,
		&gridJSON,
		&enabledInt,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("schedule not found: %s", id)
		}
		return nil, fmt.Errorf("sqlite: failed to get schedule: %w", err)
	}

	s.Enabled = enabledInt == 1
	_ = json.Unmarshal([]byte(gridJSON), &s.Grid)
	if s.Grid == nil {
		s.Grid = make(schedule.WeeklyGrid)
	}
	if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
		s.CreatedAt = t
	}
	if t, err := time.Parse(time.RFC3339, updatedAt); err == nil {
		s.UpdatedAt = t
	}

	return &s, nil
}

func (r *CalendarRepository) ListSchedules(ctx context.Context) ([]*schedule.Schedule, error) {
	query := `
		SELECT id, name, entity_key, schedule_json, enabled, created_at, updated_at
		FROM schedules
		ORDER BY name ASC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("sqlite: failed to list schedules: %w", err)
	}
	defer rows.Close()

	var list []*schedule.Schedule
	for rows.Next() {
		var (
			s          schedule.Schedule
			gridJSON   string
			enabledInt int
			createdAt  string
			updatedAt  string
		)
		if err := rows.Scan(
			&s.ID,
			&s.Name,
			&s.EntityKey,
			&gridJSON,
			&enabledInt,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, fmt.Errorf("sqlite: scan schedule error: %w", err)
		}
		s.Enabled = enabledInt == 1
		_ = json.Unmarshal([]byte(gridJSON), &s.Grid)
		if s.Grid == nil {
			s.Grid = make(schedule.WeeklyGrid)
		}
		if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
			s.CreatedAt = t
		}
		if t, err := time.Parse(time.RFC3339, updatedAt); err == nil {
			s.UpdatedAt = t
		}
		list = append(list, &s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: rows error: %w", err)
	}

	if list == nil {
		list = []*schedule.Schedule{}
	}
	return list, nil
}

func (r *CalendarRepository) DeleteSchedule(ctx context.Context, id string) error {
	query := `DELETE FROM schedules WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("sqlite: failed to delete schedule: %w", err)
	}
	return nil
}

// --- Calendar Events ---

func (r *CalendarRepository) SaveCalendarEvent(ctx context.Context, e *calendar.Event) error {
	if e.ID == "" {
		e.ID = "evt_" + uuid.NewString()[:8]
	}
	now := time.Now().UTC()
	if e.CreatedAt.IsZero() {
		e.CreatedAt = now
	}
	if e.Source == "" {
		e.Source = "local"
	}

	allDayInt := 0
	if e.AllDay {
		allDayInt = 1
	}

	query := `
		INSERT INTO calendar_events (id, summary, description, start_time, end_time, location, all_day, source, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			summary = excluded.summary,
			description = excluded.description,
			start_time = excluded.start_time,
			end_time = excluded.end_time,
			location = excluded.location,
			all_day = excluded.all_day,
			source = excluded.source
	`
	_, err := r.db.ExecContext(
		ctx, query,
		e.ID,
		e.Summary,
		e.Description,
		e.StartTime.Format(time.RFC3339),
		e.EndTime.Format(time.RFC3339),
		e.Location,
		allDayInt,
		e.Source,
		e.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("sqlite: failed to save calendar event: %w", err)
	}
	return nil
}

func (r *CalendarRepository) ListCalendarEvents(ctx context.Context, from, to time.Time) ([]*calendar.Event, error) {
	var query string
	var args []any

	if !from.IsZero() && !to.IsZero() {
		query = `
			SELECT id, summary, description, start_time, end_time, location, all_day, source, created_at
			FROM calendar_events
			WHERE end_time >= ? AND start_time <= ?
			ORDER BY start_time ASC
		`
		args = []any{from.Format(time.RFC3339), to.Format(time.RFC3339)}
	} else {
		query = `
			SELECT id, summary, description, start_time, end_time, location, all_day, source, created_at
			FROM calendar_events
			ORDER BY start_time ASC
		`
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("sqlite: failed to list calendar events: %w", err)
	}
	defer rows.Close()

	var events []*calendar.Event
	for rows.Next() {
		var (
			e           calendar.Event
			description sql.NullString
			location    sql.NullString
			allDayInt   int
			startStr    string
			endStr      string
			createdStr  string
		)
		if err := rows.Scan(
			&e.ID,
			&e.Summary,
			&description,
			&startStr,
			&endStr,
			&location,
			&allDayInt,
			&e.Source,
			&createdStr,
		); err != nil {
			return nil, fmt.Errorf("sqlite: scan calendar event error: %w", err)
		}
		e.Description = description.String
		e.Location = location.String
		e.AllDay = allDayInt == 1
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			e.StartTime = t
		}
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			e.EndTime = t
		}
		if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
			e.CreatedAt = t
		}
		events = append(events, &e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: rows error: %w", err)
	}

	if events == nil {
		events = []*calendar.Event{}
	}
	return events, nil
}

func (r *CalendarRepository) DeleteCalendarEvent(ctx context.Context, id string) error {
	query := `DELETE FROM calendar_events WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("sqlite: failed to delete calendar event: %w", err)
	}
	return nil
}
