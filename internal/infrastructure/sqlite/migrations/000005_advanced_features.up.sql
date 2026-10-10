-- 000005_advanced_features.up.sql
-- Tables for downsampled telemetry, load shedding configuration, and presence tracking

CREATE TABLE hourly_telemetry (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    entity_id   TEXT NOT NULL,
    hour_bucket TEXT NOT NULL,           -- RFC3339 truncated to hour (e.g. 2026-10-10T08:00:00Z)
    val_avg     REAL NOT NULL,
    val_min     REAL NOT NULL,
    val_max     REAL NOT NULL,
    samples     INTEGER NOT NULL,
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    UNIQUE(entity_id, hour_bucket)
);

CREATE INDEX idx_hourly_telemetry_entity ON hourly_telemetry(entity_id, hour_bucket DESC);

CREATE TABLE load_shedding_config (
    id                 TEXT PRIMARY KEY,
    meter_entity_key   TEXT NOT NULL,    -- Sensor entity measuring total power (e.g. sensor.main_power)
    max_power_watts    REAL NOT NULL,    -- Threshold (e.g. 3300.0 or 4500.0)
    overload_duration  INTEGER NOT NULL, -- Seconds of sustained overload before shedding (e.g. 60)
    recovery_watts     REAL NOT NULL,    -- Safety threshold before reconnecting (e.g. 2500.0)
    recovery_delay_sec INTEGER NOT NULL, -- Seconds below recovery threshold before re-enabling loads (e.g. 180)
    priority_loads     TEXT NOT NULL,    -- JSON array of entity keys in shedding order: ["switch.ev_charger", "switch.water_heater", "switch.dehumidifier"]
    enabled            INTEGER NOT NULL DEFAULT 1,
    updated_at         TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

CREATE TABLE presence_tracking (
    user_id            TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    state              TEXT NOT NULL DEFAULT 'away', -- 'home', 'away', 'just_arrived', 'just_left'
    gps_lat            REAL,
    gps_lon            REAL,
    wifi_ip            TEXT,
    wifi_mac           TEXT,
    wifi_connected     INTEGER NOT NULL DEFAULT 0,
    room_zone          TEXT,                         -- Room where mmWave or BLE detected presence
    last_seen_at       TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);
