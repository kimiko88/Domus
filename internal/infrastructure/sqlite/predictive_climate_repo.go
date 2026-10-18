// Package sqlite provides SQLite persistence for predictive climate control.
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

// PredictiveClimateRepository implements climate.PredictiveRepository using SQLite.
type PredictiveClimateRepository struct {
	db *sql.DB
}

// NewPredictiveClimateRepository creates a new instance of PredictiveClimateRepository.
func NewPredictiveClimateRepository(db *sql.DB) *PredictiveClimateRepository {
	return &PredictiveClimateRepository{db: db}
}

// Save inserts or updates a predictive zone configuration.
func (r *PredictiveClimateRepository) Save(ctx context.Context, cfg *climate.PredictiveZoneConfig) error {
	if cfg == nil {
		return errors.New("cannot save nil predictive zone config")
	}

	if cfg.ID == "" {
		cfg.ID = "pzone_" + uuid.NewString()
	}
	now := time.Now().UTC()
	if cfg.CreatedAt.IsZero() {
		cfg.CreatedAt = now
	}
	cfg.UpdatedAt = now

	preheat := 0
	if cfg.PreheatEnabled {
		preheat = 1
	}
	solarCutoff := 0
	if cfg.SolarCutoffEnabled {
		solarCutoff = 1
	}
	pressureBoost := 0
	if cfg.PressureBoostEnabled {
		pressureBoost = 1
	}
	enabled := 0
	if cfg.Enabled {
		enabled = 1
	}

	query := `
		INSERT INTO predictive_climate_zones (
			id, name, area_id, trv_entity_key, temp_sensor_key,
			pressure_sensor_key, humidity_sensor_key, window_sensor_key,
			target_temperature, orientation, window_area_sqm, emitter_type,
			thermal_inertia_hours, preheat_enabled, solar_cutoff_enabled,
			pressure_boost_enabled, enabled, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			area_id = excluded.area_id,
			trv_entity_key = excluded.trv_entity_key,
			temp_sensor_key = excluded.temp_sensor_key,
			pressure_sensor_key = excluded.pressure_sensor_key,
			humidity_sensor_key = excluded.humidity_sensor_key,
			window_sensor_key = excluded.window_sensor_key,
			target_temperature = excluded.target_temperature,
			orientation = excluded.orientation,
			window_area_sqm = excluded.window_area_sqm,
			emitter_type = excluded.emitter_type,
			thermal_inertia_hours = excluded.thermal_inertia_hours,
			preheat_enabled = excluded.preheat_enabled,
			solar_cutoff_enabled = excluded.solar_cutoff_enabled,
			pressure_boost_enabled = excluded.pressure_boost_enabled,
			enabled = excluded.enabled,
			updated_at = excluded.updated_at
	`

	_, err := r.db.ExecContext(ctx, query,
		cfg.ID, cfg.Name, cfg.AreaID, cfg.TRVEntityKey, cfg.TempSensorKey,
		cfg.PressureSensorKey, cfg.HumiditySensorKey, cfg.WindowSensorKey,
		cfg.TargetTemperature, string(cfg.Orientation), cfg.WindowAreaSqm, string(cfg.EmitterType),
		cfg.ThermalInertiaHours, preheat, solarCutoff,
		pressureBoost, enabled, cfg.CreatedAt.Format(time.RFC3339), cfg.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("failed to save predictive climate zone: %w", err)
	}
	return nil
}

// GetByID retrieves a predictive zone configuration by its unique ID.
func (r *PredictiveClimateRepository) GetByID(ctx context.Context, id string) (*climate.PredictiveZoneConfig, error) {
	query := `
		SELECT id, name, area_id, trv_entity_key, temp_sensor_key,
		       pressure_sensor_key, humidity_sensor_key, window_sensor_key,
		       target_temperature, orientation, window_area_sqm, emitter_type,
		       thermal_inertia_hours, preheat_enabled, solar_cutoff_enabled,
		       pressure_boost_enabled, enabled, created_at, updated_at
		FROM predictive_climate_zones
		WHERE id = ?
	`
	row := r.db.QueryRowContext(ctx, query, id)
	cfg, err := r.scanZone(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, climate.ErrPredictiveZoneNotFound
		}
		return nil, fmt.Errorf("failed to get predictive climate zone: %w", err)
	}
	return cfg, nil
}

// List retrieves all predictive climate zones.
func (r *PredictiveClimateRepository) List(ctx context.Context) ([]*climate.PredictiveZoneConfig, error) {
	query := `
		SELECT id, name, area_id, trv_entity_key, temp_sensor_key,
		       pressure_sensor_key, humidity_sensor_key, window_sensor_key,
		       target_temperature, orientation, window_area_sqm, emitter_type,
		       thermal_inertia_hours, preheat_enabled, solar_cutoff_enabled,
		       pressure_boost_enabled, enabled, created_at, updated_at
		FROM predictive_climate_zones
		ORDER BY name ASC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list predictive climate zones: %w", err)
	}
	defer rows.Close()

	var list []*climate.PredictiveZoneConfig
	for rows.Next() {
		cfg, err := r.scanZone(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, cfg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating predictive zones: %w", err)
	}
	return list, nil
}

// Delete removes a predictive climate zone by ID.
func (r *PredictiveClimateRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, "DELETE FROM predictive_climate_zones WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete predictive climate zone: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return climate.ErrPredictiveZoneNotFound
	}
	return nil
}

// SaveEvaluation records a predictive evaluation run in the history table.
func (r *PredictiveClimateRepository) SaveEvaluation(ctx context.Context, eval *climate.PredictiveEvaluation) error {
	if eval == nil {
		return errors.New("cannot save nil evaluation")
	}
	if eval.ID == "" {
		eval.ID = "peval_" + uuid.NewString()
	}
	if eval.EvaluatedAt.IsZero() {
		eval.EvaluatedAt = time.Now().UTC()
	}

	coldFront := 0
	if eval.ColdFrontDetected {
		coldFront = 1
	}
	windowOpen := 0
	if eval.WindowOpen {
		windowOpen = 1
	}

	query := `
		INSERT INTO predictive_climate_evaluations (
			id, zone_id, zone_name, current_temp, target_temp,
			current_pressure, pressure_trend_hpa_hr, humidity, apparent_temp,
			outdoor_temp_forecast, direct_radiation_forecast, estimated_solar_gain_kw,
			cold_front_detected, window_open, calculated_setpoint, mode, reason, evaluated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query,
		eval.ID, eval.ZoneID, eval.ZoneName, eval.CurrentTemp, eval.TargetTemp,
		eval.CurrentPressure, eval.PressureTrendHpaHr, eval.Humidity, eval.ApparentTemp,
		eval.OutdoorTempForecast, eval.DirectRadiationForecast, eval.EstimatedSolarGainKw,
		coldFront, windowOpen, eval.CalculatedSetpoint, eval.Mode, eval.Reason,
		eval.EvaluatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("failed to save predictive climate evaluation: %w", err)
	}
	return nil
}

// GetLatestEvaluations returns the most recent evaluation for each active predictive zone.
func (r *PredictiveClimateRepository) GetLatestEvaluations(ctx context.Context) ([]*climate.PredictiveEvaluation, error) {
	query := `
		SELECT e.id, e.zone_id, e.zone_name, e.current_temp, e.target_temp,
		       e.current_pressure, e.pressure_trend_hpa_hr, e.humidity, e.apparent_temp,
		       e.outdoor_temp_forecast, e.direct_radiation_forecast, e.estimated_solar_gain_kw,
		       e.cold_front_detected, e.window_open, e.calculated_setpoint, e.mode, e.reason, e.evaluated_at
		FROM predictive_climate_evaluations e
		INNER JOIN (
			SELECT zone_id, MAX(evaluated_at) as max_date
			FROM predictive_climate_evaluations
			GROUP BY zone_id
		) latest ON e.zone_id = latest.zone_id AND e.evaluated_at = latest.max_date
		ORDER BY e.zone_name ASC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest predictive climate evaluations: %w", err)
	}
	defer rows.Close()

	var evals []*climate.PredictiveEvaluation
	for rows.Next() {
		ev, err := r.scanEvaluation(rows)
		if err != nil {
			return nil, err
		}
		evals = append(evals, ev)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating latest evaluations: %w", err)
	}
	return evals, nil
}

// GetZoneEvaluations returns historical evaluations for a specific zone.
func (r *PredictiveClimateRepository) GetZoneEvaluations(ctx context.Context, zoneID string, limit int) ([]*climate.PredictiveEvaluation, error) {
	if limit <= 0 {
		limit = 20
	}
	query := `
		SELECT id, zone_id, zone_name, current_temp, target_temp,
		       current_pressure, pressure_trend_hpa_hr, humidity, apparent_temp,
		       outdoor_temp_forecast, direct_radiation_forecast, estimated_solar_gain_kw,
		       cold_front_detected, window_open, calculated_setpoint, mode, reason, evaluated_at
		FROM predictive_climate_evaluations
		WHERE zone_id = ?
		ORDER BY evaluated_at DESC
		LIMIT ?
	`
	rows, err := r.db.QueryContext(ctx, query, zoneID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get zone evaluations: %w", err)
	}
	defer rows.Close()

	var evals []*climate.PredictiveEvaluation
	for rows.Next() {
		ev, err := r.scanEvaluation(rows)
		if err != nil {
			return nil, err
		}
		evals = append(evals, ev)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating zone evaluations: %w", err)
	}
	return evals, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func (r *PredictiveClimateRepository) scanZone(s rowScanner) (*climate.PredictiveZoneConfig, error) {
	var (
		cfg           climate.PredictiveZoneConfig
		orientStr     string
		emitterStr    string
		preheat       int
		solarCutoff   int
		pressureBoost int
		enabled       int
		createdStr    string
		updatedStr    string
	)

	err := s.Scan(
		&cfg.ID, &cfg.Name, &cfg.AreaID, &cfg.TRVEntityKey, &cfg.TempSensorKey,
		&cfg.PressureSensorKey, &cfg.HumiditySensorKey, &cfg.WindowSensorKey,
		&cfg.TargetTemperature, &orientStr, &cfg.WindowAreaSqm, &emitterStr,
		&cfg.ThermalInertiaHours, &preheat, &solarCutoff,
		&pressureBoost, &enabled, &createdStr, &updatedStr,
	)
	if err != nil {
		return nil, err
	}

	cfg.Orientation = climate.RoomOrientation(orientStr)
	cfg.EmitterType = climate.EmitterType(emitterStr)
	cfg.PreheatEnabled = preheat == 1
	cfg.SolarCutoffEnabled = solarCutoff == 1
	cfg.PressureBoostEnabled = pressureBoost == 1
	cfg.Enabled = enabled == 1

	if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
		cfg.CreatedAt = t
	}
	if t, err := time.Parse(time.RFC3339, updatedStr); err == nil {
		cfg.UpdatedAt = t
	}
	return &cfg, nil
}

func (r *PredictiveClimateRepository) scanEvaluation(s rowScanner) (*climate.PredictiveEvaluation, error) {
	var (
		ev           climate.PredictiveEvaluation
		coldFront    int
		windowOpen   int
		evaluatedStr string
	)

	err := s.Scan(
		&ev.ID, &ev.ZoneID, &ev.ZoneName, &ev.CurrentTemp, &ev.TargetTemp,
		&ev.CurrentPressure, &ev.PressureTrendHpaHr, &ev.Humidity, &ev.ApparentTemp,
		&ev.OutdoorTempForecast, &ev.DirectRadiationForecast, &ev.EstimatedSolarGainKw,
		&coldFront, &windowOpen, &ev.CalculatedSetpoint, &ev.Mode, &ev.Reason, &evaluatedStr,
	)
	if err != nil {
		return nil, err
	}

	ev.ColdFrontDetected = coldFront == 1
	ev.WindowOpen = windowOpen == 1
	if t, err := time.Parse(time.RFC3339, evaluatedStr); err == nil {
		ev.EvaluatedAt = t
	}
	return &ev, nil
}

// SaveProfile inserts or updates a learned thermal profile for a zone.
func (r *PredictiveClimateRepository) SaveProfile(ctx context.Context, profile *climate.ThermalProfile) error {
	if profile == nil {
		return errors.New("cannot save nil thermal profile")
	}
	if profile.ZoneID == "" {
		return errors.New("cannot save thermal profile without zone_id")
	}
	if profile.LastTrainedAt.IsZero() {
		profile.LastTrainedAt = time.Now().UTC()
	}

	query := `
		INSERT INTO predictive_climate_profiles (
			zone_id, heat_rate_per_hr, cool_rate_per_hr,
			outdoor_loss_coeff, solar_boost_coeff, samples_count, last_trained_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(zone_id) DO UPDATE SET
			heat_rate_per_hr = excluded.heat_rate_per_hr,
			cool_rate_per_hr = excluded.cool_rate_per_hr,
			outdoor_loss_coeff = excluded.outdoor_loss_coeff,
			solar_boost_coeff = excluded.solar_boost_coeff,
			samples_count = excluded.samples_count,
			last_trained_at = excluded.last_trained_at
	`
	_, err := r.db.ExecContext(ctx, query,
		profile.ZoneID, profile.HeatRatePerHr, profile.CoolRatePerHr,
		profile.OutdoorLossCoeff, profile.SolarBoostCoeff, profile.SamplesCount,
		profile.LastTrainedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("failed to save predictive climate profile: %w", err)
	}
	return nil
}

// GetProfile retrieves a learned thermal profile by zone ID.
func (r *PredictiveClimateRepository) GetProfile(ctx context.Context, zoneID string) (*climate.ThermalProfile, error) {
	query := `
		SELECT zone_id, heat_rate_per_hr, cool_rate_per_hr,
		       outdoor_loss_coeff, solar_boost_coeff, samples_count, last_trained_at
		FROM predictive_climate_profiles
		WHERE zone_id = ?
	`
	row := r.db.QueryRowContext(ctx, query, zoneID)
	p, err := r.scanProfile(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, climate.ErrThermalProfileNotFound
		}
		return nil, fmt.Errorf("failed to get predictive climate profile: %w", err)
	}
	return p, nil
}

// ListProfiles retrieves all learned thermal profiles.
func (r *PredictiveClimateRepository) ListProfiles(ctx context.Context) ([]*climate.ThermalProfile, error) {
	query := `
		SELECT zone_id, heat_rate_per_hr, cool_rate_per_hr,
		       outdoor_loss_coeff, solar_boost_coeff, samples_count, last_trained_at
		FROM predictive_climate_profiles
		ORDER BY zone_id ASC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list predictive climate profiles: %w", err)
	}
	defer rows.Close()

	var profiles []*climate.ThermalProfile
	for rows.Next() {
		p, err := r.scanProfile(rows)
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating predictive profiles: %w", err)
	}
	return profiles, nil
}

func (r *PredictiveClimateRepository) scanProfile(s rowScanner) (*climate.ThermalProfile, error) {
	var (
		p          climate.ThermalProfile
		trainedStr string
	)
	err := s.Scan(
		&p.ZoneID,
		&p.HeatRatePerHr,
		&p.CoolRatePerHr,
		&p.OutdoorLossCoeff,
		&p.SolarBoostCoeff,
		&p.SamplesCount,
		&trainedStr,
	)
	if err != nil {
		return nil, err
	}
	if t, err := time.Parse(time.RFC3339, trainedStr); err == nil {
		p.LastTrainedAt = t
	}
	return &p, nil
}

