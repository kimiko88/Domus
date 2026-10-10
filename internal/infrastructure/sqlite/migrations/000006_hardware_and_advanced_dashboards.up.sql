-- Migration 000006: Hardware standards (Matter, ZWave, BLE), User Helpers, Custom Dashboards & Energy Tariffs

CREATE TABLE IF NOT EXISTS user_helpers (
    id TEXT PRIMARY KEY,
    entity_key TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    domain TEXT NOT NULL, -- 'input_boolean', 'input_number', 'timer'
    state TEXT NOT NULL,
    config_json TEXT NOT NULL DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_user_helpers_domain ON user_helpers(domain);

CREATE TABLE IF NOT EXISTS custom_dashboards (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    icon TEXT NOT NULL DEFAULT 'layout-dashboard',
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    layout_json TEXT NOT NULL DEFAULT '[]',
    kiosk_config_json TEXT NOT NULL DEFAULT '{"enabled": false, "screensaver_timeout_sec": 120, "wake_on_presence": true}',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS energy_tariffs (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    currency TEXT NOT NULL DEFAULT 'EUR',
    f1_price REAL NOT NULL DEFAULT 0.28, -- Peak (Mon-Fri 08:00-19:00)
    f2_price REAL NOT NULL DEFAULT 0.24, -- Mid (Mon-Fri 07:00-08:00, 19:00-23:00, Sat 07:00-23:00)
    f3_price REAL NOT NULL DEFAULT 0.19, -- Off-peak (Nights, Sundays & holidays)
    gas_smc_price REAL NOT NULL DEFAULT 0.95,
    water_m3_price REAL NOT NULL DEFAULT 1.80,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS hardware_nodes (
    id TEXT PRIMARY KEY,
    protocol TEXT NOT NULL, -- 'matter', 'zwave', 'ble', 'homekit'
    node_id TEXT NOT NULL,
    name TEXT NOT NULL,
    model TEXT NOT NULL DEFAULT '',
    manufacturer TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'online', -- 'online', 'offline', 'pairing'
    metadata_json TEXT NOT NULL DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(protocol, node_id)
);

CREATE INDEX IF NOT EXISTS idx_hardware_nodes_protocol ON hardware_nodes(protocol);
