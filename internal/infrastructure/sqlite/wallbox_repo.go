package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/domus-hub/domus/internal/plugins/ocpp"
)

// WallboxRepository manages SQLite storage for OCPP EV charge points and live telemetry.
type WallboxRepository struct {
	db *sql.DB
}

// NewWallboxRepository creates a new SQLite WallboxRepository.
func NewWallboxRepository(db *sql.DB) *WallboxRepository {
	return &WallboxRepository{db: db}
}

// SaveChargePoint records or updates an enrolled EV wallbox telemetry snapshot.
func (r *WallboxRepository) SaveChargePoint(ctx context.Context, telem ocpp.WallboxTelemetry, mode ocpp.ChargingMode) error {
	query := `
		INSERT INTO wallbox_charge_points (
			charge_point_id, mode, status, active_power_w, offered_amps, delivered_amps, vehicle_soc, energy_wh, last_seen_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(charge_point_id) DO UPDATE SET
			mode = excluded.mode,
			status = excluded.status,
			active_power_w = excluded.active_power_w,
			offered_amps = excluded.offered_amps,
			delivered_amps = excluded.delivered_amps,
			vehicle_soc = excluded.vehicle_soc,
			energy_wh = excluded.energy_wh,
			last_seen_at = excluded.last_seen_at;
	`
	_, err := r.db.ExecContext(ctx, query,
		telem.ChargePointID, string(mode), string(telem.Status), telem.ActivePowerW,
		telem.OfferedAmps, telem.DeliveredAmps, telem.VehicleSoC, telem.EnergyWh,
		telem.LastSeenAt.Format(time.RFC3339),
	)
	return err
}

// GetChargePoint retrieves the stored telemetry and mode of a wallbox by charge point ID.
func (r *WallboxRepository) GetChargePoint(ctx context.Context, cpID string) (*ocpp.WallboxTelemetry, ocpp.ChargingMode, error) {
	query := `
		SELECT charge_point_id, mode, status, active_power_w, offered_amps, delivered_amps, vehicle_soc, energy_wh, last_seen_at
		FROM wallbox_charge_points
		WHERE charge_point_id = ?;
	`
	row := r.db.QueryRowContext(ctx, query, cpID)

	var (
		telem       ocpp.WallboxTelemetry
		modeStr     string
		statusStr   string
		lastSeenStr string
	)
	if err := row.Scan(
		&telem.ChargePointID, &modeStr, &statusStr, &telem.ActivePowerW,
		&telem.OfferedAmps, &telem.DeliveredAmps, &telem.VehicleSoC, &telem.EnergyWh,
		&lastSeenStr,
	); err != nil {
		return nil, "", err
	}

	telem.Status = ocpp.ChargePointStatus(statusStr)
	if t, err := time.Parse(time.RFC3339, lastSeenStr); err == nil {
		telem.LastSeenAt = t
	}
	return &telem, ocpp.ChargingMode(modeStr), nil
}

// ListChargePoints returns all enrolled wallbox charge points.
func (r *WallboxRepository) ListChargePoints(ctx context.Context) ([]ocpp.WallboxTelemetry, error) {
	query := `
		SELECT charge_point_id, status, active_power_w, offered_amps, delivered_amps, vehicle_soc, energy_wh, last_seen_at
		FROM wallbox_charge_points
		ORDER BY charge_point_id;
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ocpp.WallboxTelemetry
	for rows.Next() {
		var (
			telem       ocpp.WallboxTelemetry
			statusStr   string
			lastSeenStr string
		)
		if err := rows.Scan(
			&telem.ChargePointID, &statusStr, &telem.ActivePowerW, &telem.OfferedAmps,
			&telem.DeliveredAmps, &telem.VehicleSoC, &telem.EnergyWh, &lastSeenStr,
		); err != nil {
			return nil, err
		}
		telem.Status = ocpp.ChargePointStatus(statusStr)
		if t, err := time.Parse(time.RFC3339, lastSeenStr); err == nil {
			telem.LastSeenAt = t
		}
		result = append(result, telem)
	}
	if result == nil {
		result = []ocpp.WallboxTelemetry{}
	}
	return result, nil
}
