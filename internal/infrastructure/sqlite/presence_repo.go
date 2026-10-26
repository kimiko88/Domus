package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/domus-hub/domus/internal/application/presence"
)

// VacationRepository handles SQLite persistence for presence simulation actions and vacation mode state.
type VacationRepository struct {
	db *sql.DB
}

// NewVacationRepository creates a new SQLite VacationRepository.
func NewVacationRepository(db *sql.DB) *VacationRepository {
	return &VacationRepository{db: db}
}

// GetConfig returns the vacation mode configuration from SQLite.
func (r *VacationRepository) GetConfig(ctx context.Context) (bool, int, error) {
	query := `SELECT enabled, jitter_minutes FROM vacation_config WHERE id = 'default';`
	row := r.db.QueryRowContext(ctx, query)

	var (
		enabledInt int
		jitter     int
	)
	if err := row.Scan(&enabledInt, &jitter); err != nil {
		if err == sql.ErrNoRows {
			return false, 15, nil
		}
		return false, 15, err
	}
	return enabledInt == 1, jitter, nil
}

// SetConfig updates vacation mode state and jitter settings in SQLite.
func (r *VacationRepository) SetConfig(ctx context.Context, enabled bool, jitterMinutes int) error {
	enabledInt := 0
	if enabled {
		enabledInt = 1
	}
	if jitterMinutes <= 0 {
		jitterMinutes = 15
	}

	query := `
		INSERT INTO vacation_config (id, enabled, jitter_minutes, updated_at)
		VALUES ('default', ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			enabled = excluded.enabled,
			jitter_minutes = excluded.jitter_minutes,
			updated_at = excluded.updated_at;
	`
	_, err := r.db.ExecContext(ctx, query, enabledInt, jitterMinutes, time.Now().UTC().Format(time.RFC3339))
	return err
}

// SaveAction stores a simulated vacation habit action in SQLite.
func (r *VacationRepository) SaveAction(ctx context.Context, act presence.SimulatedAction) error {
	payloadJSON, _ := json.Marshal(map[string]any{"jitter_min": act.JitterMin})
	query := `
		INSERT INTO vacation_actions (id, entity_id, command, time_of_day, day_of_week, payload_json, enabled)
		VALUES (?, ?, ?, ?, ?, ?, 1)
		ON CONFLICT(id) DO UPDATE SET
			entity_id = excluded.entity_id,
			command = excluded.command,
			time_of_day = excluded.time_of_day,
			payload_json = excluded.payload_json;
	`
	timeOfDayStr := fmt.Sprintf("%02d:%02d:%02d", act.TimeOfDaySec/3600, (act.TimeOfDaySec%3600)/60, act.TimeOfDaySec%60)
	_, err := r.db.ExecContext(ctx, query, act.ID, act.EntityID, act.Command, timeOfDayStr, -1, string(payloadJSON))
	return err
}

// ListActions returns all enrolled simulated vacation habit actions.
func (r *VacationRepository) ListActions(ctx context.Context) ([]presence.SimulatedAction, error) {
	query := `SELECT id, entity_id, command, time_of_day, payload_json FROM vacation_actions WHERE enabled = 1;`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var actions []presence.SimulatedAction
	for rows.Next() {
		var (
			act        presence.SimulatedAction
			timeStr    string
			payloadStr string
		)
		if err := rows.Scan(&act.ID, &act.EntityID, &act.Command, &timeStr, &payloadStr); err != nil {
			return nil, err
		}

		var h, m, s int
		_, _ = fmt.Sscanf(timeStr, "%d:%d:%d", &h, &m, &s)
		act.TimeOfDaySec = h*3600 + m*60 + s
		act.JitterMin = 15

		var payload map[string]any
		if err := json.Unmarshal([]byte(payloadStr), &payload); err == nil {
			if j, ok := payload["jitter_min"].(float64); ok {
				act.JitterMin = int(j)
			}
		}

		actions = append(actions, act)
	}
	if actions == nil {
		actions = []presence.SimulatedAction{}
	}
	return actions, nil
}

// CommuteRepository handles SQLite persistence for real-time user commute locations and ETA.
type CommuteRepository struct {
	db *sql.DB
}

// NewCommuteRepository creates a new SQLite CommuteRepository.
func NewCommuteRepository(db *sql.DB) *CommuteRepository {
	return &CommuteRepository{db: db}
}

// SaveLocation records the user's latest GPS transit telemetry in SQLite.
func (r *CommuteRepository) SaveLocation(ctx context.Context, status presence.CommuteStatus) error {
	approachingInt := 0
	if status.IsApproaching {
		approachingInt = 1
	}

	query := `
		INSERT INTO commute_user_locations (
			user_id, latitude, longitude, speed_kmh, distance_km, eta_minutes, is_heading_home, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			latitude = excluded.latitude,
			longitude = excluded.longitude,
			speed_kmh = excluded.speed_kmh,
			distance_km = excluded.distance_km,
			eta_minutes = excluded.eta_minutes,
			is_heading_home = excluded.is_heading_home,
			updated_at = excluded.updated_at;
	`
	_, err := r.db.ExecContext(ctx, query,
		status.UserID, status.CurrentLat, status.CurrentLon, status.EstimatedSpeedKmh,
		status.DistanceMeters/1000.0, status.MinutesToHome, approachingInt,
		status.LastUpdate.Format(time.RFC3339),
	)
	return err
}

// GetLocation retrieves the latest recorded commute status for a user.
func (r *CommuteRepository) GetLocation(ctx context.Context, userID string) (*presence.CommuteStatus, error) {
	query := `
		SELECT user_id, latitude, longitude, speed_kmh, distance_km, eta_minutes, is_heading_home, updated_at
		FROM commute_user_locations
		WHERE user_id = ?;
	`
	row := r.db.QueryRowContext(ctx, query, userID)

	var (
		st             presence.CommuteStatus
		distKm         float64
		approachingInt int
		updatedStr     string
	)
	if err := row.Scan(
		&st.UserID, &st.CurrentLat, &st.CurrentLon, &st.EstimatedSpeedKmh,
		&distKm, &st.MinutesToHome, &approachingInt, &updatedStr,
	); err != nil {
		return nil, err
	}

	st.DistanceMeters = distKm * 1000.0
	st.IsApproaching = approachingInt == 1
	if t, err := time.Parse(time.RFC3339, updatedStr); err == nil {
		st.LastUpdate = t
	}
	return &st, nil
}
