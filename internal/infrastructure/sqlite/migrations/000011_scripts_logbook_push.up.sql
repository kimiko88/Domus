-- Migration 000011: Scripts, Logbook, Push Tokens, Weekly Schedules, and Climate Compensator
-- Created for complete Home Assistant feature parity and performance hardening.

CREATE TABLE IF NOT EXISTS scripts (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    icon TEXT NOT NULL DEFAULT 'script',
    mode TEXT NOT NULL DEFAULT 'single',
    max_runs INTEGER NOT NULL DEFAULT 10,
    fields_json TEXT NOT NULL DEFAULT '{}',
    sequence_json TEXT NOT NULL DEFAULT '[]',
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS logbook_entries (
    id TEXT PRIMARY KEY,
    entity_id TEXT NOT NULL,
    entity_name TEXT NOT NULL,
    domain TEXT NOT NULL,
    state TEXT NOT NULL,
    message TEXT NOT NULL,
    context_type TEXT NOT NULL DEFAULT 'system',
    context_id TEXT NOT NULL DEFAULT '',
    context_name TEXT NOT NULL DEFAULT '',
    timestamp DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_logbook_timestamp ON logbook_entries(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_logbook_entity_timestamp ON logbook_entries(entity_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_logbook_domain_timestamp ON logbook_entries(domain, timestamp DESC);

CREATE TABLE IF NOT EXISTS device_push_tokens (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    platform TEXT NOT NULL,
    device_token TEXT NOT NULL UNIQUE,
    device_name TEXT NOT NULL DEFAULT '',
    app_version TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_push_tokens_user ON device_push_tokens(user_id);

CREATE TABLE IF NOT EXISTS weekly_schedules (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    icon TEXT NOT NULL DEFAULT 'calendar-clock',
    target_entity_id TEXT NOT NULL DEFAULT '',
    target_service TEXT NOT NULL DEFAULT '',
    grid_json TEXT NOT NULL DEFAULT '{}',
    current_slot_state TEXT NOT NULL DEFAULT 'off',
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS climate_compensations (
    id TEXT PRIMARY KEY,
    trv_entity_key TEXT NOT NULL UNIQUE,
    room_temp_sensor_key TEXT NOT NULL,
    window_sensor_key TEXT NOT NULL DEFAULT '',
    target_temperature REAL NOT NULL DEFAULT 20.0,
    delta_factor REAL NOT NULL DEFAULT 1.0,
    compensated_setpoint REAL NOT NULL DEFAULT 20.0,
    window_open INTEGER NOT NULL DEFAULT 0,
    enabled INTEGER NOT NULL DEFAULT 1,
    updated_at DATETIME NOT NULL
);
