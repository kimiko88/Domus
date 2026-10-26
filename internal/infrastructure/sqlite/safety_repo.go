package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/domus-hub/domus/internal/application/safety"
)

// SafetyRepository handles SQLite persistence for emergency valves and hazard incident logs.
type SafetyRepository struct {
	db *sql.DB
}

// NewSafetyRepository creates a new SQLite SafetyRepository.
func NewSafetyRepository(db *sql.DB) *SafetyRepository {
	return &SafetyRepository{db: db}
}

// SaveValve persists an emergency shutoff valve definition.
func (r *SafetyRepository) SaveValve(ctx context.Context, v safety.ValveRegistration) error {
	query := `
		INSERT INTO safety_valves (id, type, name, location, timeout_ms, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			type = excluded.type,
			name = excluded.name,
			location = excluded.location,
			timeout_ms = excluded.timeout_ms;
	`
	_, err := r.db.ExecContext(ctx, query, v.ID, string(v.Type), v.Name, v.Location, v.TimeoutMs, time.Now().UTC().Format(time.RFC3339))
	return err
}

// ListValves returns all registered emergency valves.
func (r *SafetyRepository) ListValves(ctx context.Context) ([]safety.ValveRegistration, error) {
	query := `SELECT id, type, name, location, timeout_ms FROM safety_valves ORDER BY id;`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var valves []safety.ValveRegistration
	for rows.Next() {
		var (
			v       safety.ValveRegistration
			typeStr string
		)
		if err := rows.Scan(&v.ID, &typeStr, &v.Name, &v.Location, &v.TimeoutMs); err != nil {
			return nil, err
		}
		v.Type = safety.ValveType(typeStr)
		valves = append(valves, v)
	}
	if valves == nil {
		valves = []safety.ValveRegistration{}
	}
	return valves, nil
}

// DeleteValve unregisters a valve from SQLite.
func (r *SafetyRepository) DeleteValve(ctx context.Context, id string) error {
	query := `DELETE FROM safety_valves WHERE id = ?;`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// RecordIncident logs an emergency safe-stop cutoff execution into SQLite.
func (r *SafetyRepository) RecordIncident(ctx context.Context, alert safety.EmergencyAlert, result safety.EmergencyResult) error {
	closedJSON, _ := json.Marshal(result.ValvesClosed)
	failedJSON, _ := json.Marshal(result.ValvesFailed)
	successInt := 0
	if result.Success {
		successInt = 1
	}

	query := `
		INSERT INTO emergency_incidents (
			id, hazard, source_sensor, location, severity, duration_ms,
			valves_targeted, valves_closed, valves_failed, success, occurred_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`
	_, err := r.db.ExecContext(ctx, query,
		alert.ID, string(alert.Hazard), alert.SourceSensor, alert.Location, alert.Severity,
		result.DurationMs, result.ValvesTargeted, string(closedJSON), string(failedJSON),
		successInt, alert.OccurredAt.Format(time.RFC3339),
	)
	return err
}

// ListIncidents returns historical safe-stop emergency events.
func (r *SafetyRepository) ListIncidents(ctx context.Context, limit int) ([]map[string]any, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
		SELECT id, hazard, source_sensor, location, severity, duration_ms, valves_targeted,
		       valves_closed, valves_failed, success, occurred_at, cleared_at, cleared_by, clear_reason
		FROM emergency_incidents
		ORDER BY occurred_at DESC
		LIMIT ?;
	`
	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var incidents []map[string]any
	for rows.Next() {
		var (
			id, hazard, source, location, severity string
			durationMs                             float64
			valvesTargeted, success                int
			closedStr, failedStr                   string
			occurredStr                            string
			clearedStr, clearedBy, clearReason     sql.NullString
		)
		if err := rows.Scan(
			&id, &hazard, &source, &location, &severity, &durationMs,
			&valvesTargeted, &closedStr, &failedStr, &success,
			&occurredStr, &clearedStr, &clearedBy, &clearReason,
		); err != nil {
			return nil, err
		}

		var closedList []string
		_ = json.Unmarshal([]byte(closedStr), &closedList)
		var failedMap map[string]string
		_ = json.Unmarshal([]byte(failedStr), &failedMap)

		incidents = append(incidents, map[string]any{
			"id":              id,
			"hazard":          hazard,
			"source_sensor":   source,
			"location":        location,
			"severity":        severity,
			"duration_ms":     durationMs,
			"valves_targeted": valvesTargeted,
			"valves_closed":   closedList,
			"valves_failed":   failedMap,
			"success":         success == 1,
			"occurred_at":     occurredStr,
			"cleared_at":      clearedStr.String,
			"cleared_by":      clearedBy.String,
			"clear_reason":    clearReason.String,
		})
	}
	if incidents == nil {
		incidents = []map[string]any{}
	}
	return incidents, nil
}

// ClearIncident marks an incident as verified and disarmed by an operator.
func (r *SafetyRepository) ClearIncident(ctx context.Context, alertID, clearedBy, reason string) error {
	query := `
		UPDATE emergency_incidents
		SET cleared_at = ?, cleared_by = ?, clear_reason = ?
		WHERE id = ?;
	`
	_, err := r.db.ExecContext(ctx, query, time.Now().UTC().Format(time.RFC3339), clearedBy, reason, alertID)
	return err
}
