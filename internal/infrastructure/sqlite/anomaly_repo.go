package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/domus-hub/domus/internal/application/anomaly"
)

// AnomalyRepository implements anomaly.Repository using SQLite.
type AnomalyRepository struct {
	db *sql.DB
}

// NewAnomalyRepository creates a new SQLite AnomalyRepository.
func NewAnomalyRepository(db *sql.DB) *AnomalyRepository {
	return &AnomalyRepository{db: db}
}

// SaveAlert records a newly detected anomaly alert in SQLite.
func (r *AnomalyRepository) SaveAlert(ctx context.Context, a *anomaly.AnomalyAlert) error {
	dataJSON, _ := json.Marshal(a.Data)
	ackInt := 0
	if a.Acknowledged {
		ackInt = 1
	}

	query := `
		INSERT INTO anomaly_alerts (id, category, severity, entity_id, title, description, acknowledged, created_at, data_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			category = excluded.category,
			severity = excluded.severity,
			title = excluded.title,
			description = excluded.description,
			acknowledged = excluded.acknowledged,
			data_json = excluded.data_json;
	`
	_, err := r.db.ExecContext(ctx, query,
		a.ID, a.Category, a.Severity, a.EntityID, a.Title, a.Description,
		ackInt, a.CreatedAt.Format(time.RFC3339), string(dataJSON),
	)
	if err != nil {
		return fmt.Errorf("sqlite: failed to save anomaly alert: %w", err)
	}
	return nil
}

// ListAlerts fetches anomaly alerts from SQLite with optional filtering by acknowledged state.
func (r *AnomalyRepository) ListAlerts(ctx context.Context, acknowledged *bool, limit int) ([]*anomaly.AnomalyAlert, error) {
	if limit <= 0 {
		limit = 50
	}

	var query string
	var args []any

	if acknowledged != nil {
		ackInt := 0
		if *acknowledged {
			ackInt = 1
		}
		query = `
			SELECT id, category, severity, entity_id, title, description, acknowledged, created_at, data_json
			FROM anomaly_alerts
			WHERE acknowledged = ?
			ORDER BY created_at DESC
			LIMIT ?;
		`
		args = []any{ackInt, limit}
	} else {
		query = `
			SELECT id, category, severity, entity_id, title, description, acknowledged, created_at, data_json
			FROM anomaly_alerts
			ORDER BY created_at DESC
			LIMIT ?;
		`
		args = []any{limit}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("sqlite: query anomaly alerts failed: %w", err)
	}
	defer rows.Close()

	var alerts []*anomaly.AnomalyAlert
	for rows.Next() {
		var (
			a          anomaly.AnomalyAlert
			ackInt     int
			createdStr string
			dataStr    string
		)
		if err := rows.Scan(&a.ID, &a.Category, &a.Severity, &a.EntityID, &a.Title, &a.Description, &ackInt, &createdStr, &dataStr); err != nil {
			return nil, fmt.Errorf("sqlite: scan anomaly alert failed: %w", err)
		}
		a.Acknowledged = ackInt == 1
		if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
			a.CreatedAt = t
		}
		_ = json.Unmarshal([]byte(dataStr), &a.Data)
		alerts = append(alerts, &a)
	}
	if alerts == nil {
		alerts = []*anomaly.AnomalyAlert{}
	}
	return alerts, nil
}

// AcknowledgeAlert marks an alert as acknowledged in SQLite.
func (r *AnomalyRepository) AcknowledgeAlert(ctx context.Context, id string) error {
	query := `UPDATE anomaly_alerts SET acknowledged = 1 WHERE id = ?;`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// SaveWaterMeter stores the latest flow rate and timestamp for a water meter.
func (r *AnomalyRepository) SaveWaterMeter(ctx context.Context, meterID string, flowLMin float64, lastSeen time.Time) error {
	query := `
		INSERT INTO anomaly_water_meters (meter_id, flow_l_min, last_seen_at)
		VALUES (?, ?, ?)
		ON CONFLICT(meter_id) DO UPDATE SET
			flow_l_min = excluded.flow_l_min,
			last_seen_at = excluded.last_seen_at;
	`
	_, err := r.db.ExecContext(ctx, query, meterID, flowLMin, lastSeen.Format(time.RFC3339))
	return err
}

// GetWaterMeter retrieves the latest recorded state of a water meter.
func (r *AnomalyRepository) GetWaterMeter(ctx context.Context, meterID string) (float64, time.Time, error) {
	query := `SELECT flow_l_min, last_seen_at FROM anomaly_water_meters WHERE meter_id = ?;`
	row := r.db.QueryRowContext(ctx, query, meterID)

	var (
		flow        float64
		lastSeenStr string
	)
	if err := row.Scan(&flow, &lastSeenStr); err != nil {
		return 0, time.Time{}, err
	}
	t, _ := time.Parse(time.RFC3339, lastSeenStr)
	return flow, t, nil
}

// SaveAppliance stores monitored electrical appliance metadata and telemetry.
func (r *AnomalyRepository) SaveAppliance(ctx context.Context, id, kind string, nominalWatts, lastWatts, lastTemp float64, lastSeen time.Time) error {
	query := `
		INSERT INTO anomaly_appliances (id, kind, nominal_watts, last_power_watts, last_temp_c, last_seen_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			kind = excluded.kind,
			nominal_watts = excluded.nominal_watts,
			last_power_watts = excluded.last_power_watts,
			last_temp_c = excluded.last_temp_c,
			last_seen_at = excluded.last_seen_at;
	`
	_, err := r.db.ExecContext(ctx, query, id, kind, nominalWatts, lastWatts, lastTemp, lastSeen.Format(time.RFC3339))
	return err
}

// ListAppliances returns all monitored appliances recorded in SQLite.
func (r *AnomalyRepository) ListAppliances(ctx context.Context) ([]map[string]any, error) {
	query := `SELECT id, kind, nominal_watts, last_power_watts, last_temp_c, last_seen_at FROM anomaly_appliances ORDER BY id;`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var appliances []map[string]any
	for rows.Next() {
		var (
			id, kind    string
			nominal     float64
			lastPower   float64
			lastTemp    float64
			lastSeenStr string
		)
		if err := rows.Scan(&id, &kind, &nominal, &lastPower, &lastTemp, &lastSeenStr); err != nil {
			return nil, err
		}
		appliances = append(appliances, map[string]any{
			"id":               id,
			"kind":             kind,
			"nominal_watts":    nominal,
			"last_power_watts": lastPower,
			"last_temp_c":      lastTemp,
			"last_seen_at":     lastSeenStr,
		})
	}
	if appliances == nil {
		appliances = []map[string]any{}
	}
	return appliances, nil
}
