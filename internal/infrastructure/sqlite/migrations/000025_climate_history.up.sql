CREATE TABLE IF NOT EXISTS climate_history_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    config_json TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS climate_history_samples (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    entity_id TEXT NOT NULL,
    entity_key TEXT NOT NULL,
    entity_name TEXT NOT NULL,
    device_id TEXT NOT NULL DEFAULT '',
    area_id TEXT NOT NULL DEFAULT '',
    recorded_at TEXT NOT NULL,
    temperature REAL,
    humidity REAL,
    target_temperature REAL,
    hvac_mode TEXT NOT NULL DEFAULT '',
    hvac_action TEXT NOT NULL DEFAULT '',
    UNIQUE(entity_id, recorded_at)
);
CREATE INDEX IF NOT EXISTS idx_climate_samples_area_time ON climate_history_samples(area_id, recorded_at);
CREATE INDEX IF NOT EXISTS idx_climate_samples_entity_time ON climate_history_samples(entity_id, recorded_at);

CREATE TABLE IF NOT EXISTS climate_history_hourly (
    entity_id TEXT NOT NULL,
    entity_key TEXT NOT NULL,
    entity_name TEXT NOT NULL,
    device_id TEXT NOT NULL DEFAULT '',
    area_id TEXT NOT NULL DEFAULT '',
    period_start TEXT NOT NULL,
    period_end TEXT NOT NULL,
    temperature_avg REAL,
    temperature_min REAL,
    temperature_max REAL,
    temperature_sum REAL NOT NULL DEFAULT 0,
    temperature_count INTEGER NOT NULL DEFAULT 0,
    humidity_avg REAL,
    humidity_min REAL,
    humidity_max REAL,
    humidity_sum REAL NOT NULL DEFAULT 0,
    humidity_count INTEGER NOT NULL DEFAULT 0,
    target_temperature_avg REAL,
    target_temperature_min REAL,
    target_temperature_max REAL,
    target_temperature_sum REAL NOT NULL DEFAULT 0,
    target_temperature_count INTEGER NOT NULL DEFAULT 0,
    sample_count INTEGER NOT NULL DEFAULT 0,
    heating_samples INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY(entity_id, period_start)
);
CREATE INDEX IF NOT EXISTS idx_climate_hourly_area_time ON climate_history_hourly(area_id, period_start);

CREATE TABLE IF NOT EXISTS climate_history_daily (
    entity_id TEXT NOT NULL,
    entity_key TEXT NOT NULL,
    entity_name TEXT NOT NULL,
    device_id TEXT NOT NULL DEFAULT '',
    area_id TEXT NOT NULL DEFAULT '',
    period_start TEXT NOT NULL,
    period_end TEXT NOT NULL,
    temperature_avg REAL,
    temperature_min REAL,
    temperature_max REAL,
    temperature_sum REAL NOT NULL DEFAULT 0,
    temperature_count INTEGER NOT NULL DEFAULT 0,
    humidity_avg REAL,
    humidity_min REAL,
    humidity_max REAL,
    humidity_sum REAL NOT NULL DEFAULT 0,
    humidity_count INTEGER NOT NULL DEFAULT 0,
    target_temperature_avg REAL,
    target_temperature_min REAL,
    target_temperature_max REAL,
    target_temperature_sum REAL NOT NULL DEFAULT 0,
    target_temperature_count INTEGER NOT NULL DEFAULT 0,
    sample_count INTEGER NOT NULL DEFAULT 0,
    heating_samples INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY(entity_id, period_start)
);
CREATE INDEX IF NOT EXISTS idx_climate_daily_area_time ON climate_history_daily(area_id, period_start);
