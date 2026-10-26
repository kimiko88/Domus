-- 000018_advanced_features_persistence.up.sql
-- Persistence tables for AI Vision NVR, Predictive Anomaly Engine, Emergency Safe-Stop,
-- Remote Backup, Vacation Simulator, Commute Tracking, and Wallbox EV Charging.

-- 1. AI Vision NVR Tables
CREATE TABLE IF NOT EXISTS camera_family_faces (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT '',
    photo_url TEXT NOT NULL DEFAULT '',
    embedding TEXT NOT NULL DEFAULT '[]',
    enrolled_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS camera_authorized_plates (
    plate_number TEXT PRIMARY KEY,
    owner_name TEXT NOT NULL,
    vehicle_model TEXT NOT NULL DEFAULT '',
    auto_open_gate INTEGER NOT NULL DEFAULT 1,
    enrolled_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS camera_motion_zones (
    id TEXT PRIMARY KEY,
    camera_id TEXT NOT NULL,
    name TEXT NOT NULL,
    sensitivity INTEGER NOT NULL DEFAULT 80,
    labels TEXT NOT NULL DEFAULT '[]',
    coordinates TEXT NOT NULL DEFAULT '[]',
    enabled INTEGER NOT NULL DEFAULT 1,
    FOREIGN KEY(camera_id) REFERENCES cameras(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS camera_event_clips (
    event_id TEXT PRIMARY KEY,
    camera_id TEXT NOT NULL,
    duration_sec REAL NOT NULL DEFAULT 10.0,
    thumbnail_url TEXT NOT NULL,
    clip_url TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    FOREIGN KEY(camera_id) REFERENCES cameras(id) ON DELETE CASCADE
);

-- 2. Predictive Anomaly & Household Safety
CREATE TABLE IF NOT EXISTS anomaly_alerts (
    id TEXT PRIMARY KEY,
    category TEXT NOT NULL,
    severity TEXT NOT NULL,
    entity_id TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    acknowledged INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL,
    data_json TEXT NOT NULL DEFAULT '{}'
);

CREATE TABLE IF NOT EXISTS anomaly_water_meters (
    meter_id TEXT PRIMARY KEY,
    flow_l_min REAL NOT NULL DEFAULT 0,
    last_seen_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS anomaly_appliances (
    id TEXT PRIMARY KEY,
    kind TEXT NOT NULL,
    nominal_watts REAL NOT NULL,
    last_power_watts REAL NOT NULL DEFAULT 0,
    last_temp_c REAL NOT NULL DEFAULT 0,
    last_seen_at DATETIME NOT NULL
);

-- 3. Emergency Safe-Stop Hardware
CREATE TABLE IF NOT EXISTS safety_valves (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    name TEXT NOT NULL,
    location TEXT NOT NULL,
    timeout_ms INTEGER NOT NULL DEFAULT 35,
    created_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS emergency_incidents (
    id TEXT PRIMARY KEY,
    hazard TEXT NOT NULL,
    source_sensor TEXT NOT NULL,
    location TEXT NOT NULL,
    severity TEXT NOT NULL,
    duration_ms REAL NOT NULL,
    valves_targeted INTEGER NOT NULL,
    valves_closed TEXT NOT NULL DEFAULT '[]',
    valves_failed TEXT NOT NULL DEFAULT '{}',
    success INTEGER NOT NULL DEFAULT 1,
    occurred_at DATETIME NOT NULL,
    cleared_at DATETIME,
    cleared_by TEXT,
    clear_reason TEXT
);

-- 4. Disaster Recovery & Remote Backup
CREATE TABLE IF NOT EXISTS remote_backup_configs (
    id TEXT PRIMARY KEY,
    enabled INTEGER NOT NULL DEFAULT 0,
    destination_type TEXT NOT NULL DEFAULT 'webdav',
    endpoint TEXT NOT NULL DEFAULT '',
    path TEXT NOT NULL DEFAULT '/DomusBackups',
    bucket TEXT NOT NULL DEFAULT '',
    retention_days INTEGER NOT NULL DEFAULT 7,
    encryption_enabled INTEGER NOT NULL DEFAULT 1,
    updated_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS backup_records (
    id TEXT PRIMARY KEY,
    filename TEXT NOT NULL,
    size_bytes INTEGER NOT NULL,
    checksum TEXT NOT NULL,
    destination TEXT NOT NULL,
    status TEXT NOT NULL,
    error_message TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL
);

-- 5. Vacation Presence Simulation & Commute Tracking
CREATE TABLE IF NOT EXISTS vacation_config (
    id TEXT PRIMARY KEY,
    enabled INTEGER NOT NULL DEFAULT 0,
    jitter_minutes INTEGER NOT NULL DEFAULT 15,
    updated_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS vacation_actions (
    id TEXT PRIMARY KEY,
    entity_id TEXT NOT NULL,
    command TEXT NOT NULL,
    time_of_day TEXT NOT NULL,
    day_of_week INTEGER NOT NULL DEFAULT -1,
    payload_json TEXT NOT NULL DEFAULT '{}',
    enabled INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS commute_user_locations (
    user_id TEXT PRIMARY KEY,
    latitude REAL NOT NULL,
    longitude REAL NOT NULL,
    speed_kmh REAL NOT NULL DEFAULT 0,
    distance_km REAL NOT NULL DEFAULT 0,
    eta_minutes REAL NOT NULL DEFAULT 0,
    is_heading_home INTEGER NOT NULL DEFAULT 0,
    updated_at DATETIME NOT NULL
);

-- 6. Wallbox EV Smart Charging
CREATE TABLE IF NOT EXISTS wallbox_charge_points (
    charge_point_id TEXT PRIMARY KEY,
    mode TEXT NOT NULL DEFAULT 'solar_optimized',
    status TEXT NOT NULL DEFAULT 'Available',
    active_power_w REAL NOT NULL DEFAULT 0,
    offered_amps REAL NOT NULL DEFAULT 16,
    delivered_amps REAL NOT NULL DEFAULT 0,
    vehicle_soc REAL NOT NULL DEFAULT 0,
    energy_wh REAL NOT NULL DEFAULT 0,
    last_seen_at DATETIME NOT NULL
);
