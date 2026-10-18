-- 000016_predictive_climate.up.sql
-- Multi-zone predictive climate control tables (MPC with weather, barometric nowcasting, and solar feedforward).

CREATE TABLE IF NOT EXISTS predictive_climate_zones (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    area_id TEXT NOT NULL DEFAULT '',
    trv_entity_key TEXT NOT NULL,
    temp_sensor_key TEXT NOT NULL,
    pressure_sensor_key TEXT NOT NULL DEFAULT '',
    humidity_sensor_key TEXT NOT NULL DEFAULT '',
    window_sensor_key TEXT NOT NULL DEFAULT '',
    target_temperature REAL NOT NULL DEFAULT 20.0,
    orientation TEXT NOT NULL DEFAULT 'south',
    window_area_sqm REAL NOT NULL DEFAULT 2.0,
    emitter_type TEXT NOT NULL DEFAULT 'radiator',
    thermal_inertia_hours REAL NOT NULL DEFAULT 2.0,
    preheat_enabled INTEGER NOT NULL DEFAULT 1,
    solar_cutoff_enabled INTEGER NOT NULL DEFAULT 1,
    pressure_boost_enabled INTEGER NOT NULL DEFAULT 1,
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_predictive_zones_area ON predictive_climate_zones(area_id);
CREATE INDEX IF NOT EXISTS idx_predictive_zones_trv ON predictive_climate_zones(trv_entity_key);

CREATE TABLE IF NOT EXISTS predictive_climate_evaluations (
    id TEXT PRIMARY KEY,
    zone_id TEXT NOT NULL,
    zone_name TEXT NOT NULL,
    current_temp REAL NOT NULL,
    target_temp REAL NOT NULL,
    current_pressure REAL NOT NULL DEFAULT 1013.25,
    pressure_trend_hpa_hr REAL NOT NULL DEFAULT 0.0,
    humidity REAL NOT NULL DEFAULT 50.0,
    apparent_temp REAL NOT NULL DEFAULT 20.0,
    outdoor_temp_forecast REAL NOT NULL DEFAULT 10.0,
    direct_radiation_forecast REAL NOT NULL DEFAULT 0.0,
    estimated_solar_gain_kw REAL NOT NULL DEFAULT 0.0,
    cold_front_detected INTEGER NOT NULL DEFAULT 0,
    window_open INTEGER NOT NULL DEFAULT 0,
    calculated_setpoint REAL NOT NULL,
    mode TEXT NOT NULL DEFAULT 'comfort',
    reason TEXT NOT NULL,
    evaluated_at DATETIME NOT NULL,
    FOREIGN KEY(zone_id) REFERENCES predictive_climate_zones(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_predictive_evals_zone ON predictive_climate_evaluations(zone_id);
CREATE INDEX IF NOT EXISTS idx_predictive_evals_evaluated_at ON predictive_climate_evaluations(evaluated_at);
