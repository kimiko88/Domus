-- 000009_advanced_security_and_integrations.up.sql

-- 1. Alarm Control Panel
CREATE TABLE IF NOT EXISTS alarm_control_panels (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    state TEXT NOT NULL DEFAULT 'disarmed', -- disarmed, armed_home, armed_away, armed_night, pending, triggered
    pin_hash TEXT,
    entry_delay_seconds INTEGER NOT NULL DEFAULT 30,
    exit_delay_seconds INTEGER NOT NULL DEFAULT 60,
    trigger_delay_seconds INTEGER NOT NULL DEFAULT 0,
    perimeter_sensors_json TEXT NOT NULL DEFAULT '[]',
    interior_sensors_json TEXT NOT NULL DEFAULT '[]',
    siren_entities_json TEXT NOT NULL DEFAULT '[]',
    last_changed_by TEXT,
    last_triggered_reason TEXT,
    triggered_at TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Seed default home alarm if not exists
INSERT OR IGNORE INTO alarm_control_panels (
    id, name, state, pin_hash, entry_delay_seconds, exit_delay_seconds,
    perimeter_sensors_json, interior_sensors_json, siren_entities_json
) VALUES (
    'alarm_main', 'Allarme Domus Casa', 'disarmed', '1234', 30, 60,
    '[]', '[]', '[]'
);

-- 2. Inbound Webhooks
CREATE TABLE IF NOT EXISTS webhooks (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    webhook_id TEXT NOT NULL UNIQUE,
    secret_token TEXT,
    allowed_methods TEXT NOT NULL DEFAULT 'POST',
    enabled INTEGER NOT NULL DEFAULT 1,
    last_triggered_at TIMESTAMP,
    trigger_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 3. Zigbee Direct Mesh Bindings
CREATE TABLE IF NOT EXISTS zigbee_bindings (
    id TEXT PRIMARY KEY,
    source_device_id TEXT NOT NULL,
    source_device_name TEXT NOT NULL,
    target_device_id TEXT NOT NULL,
    target_device_name TEXT NOT NULL,
    clusters_json TEXT NOT NULL DEFAULT '["genOnOff"]',
    bound_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 4. Guest Mode & Kiosk PIN on Users
ALTER TABLE users ADD COLUMN is_guest INTEGER NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN guest_expires_at TIMESTAMP;
ALTER TABLE users ADD COLUMN pin_code TEXT;
ALTER TABLE users ADD COLUMN kiosk_mode INTEGER NOT NULL DEFAULT 0;
