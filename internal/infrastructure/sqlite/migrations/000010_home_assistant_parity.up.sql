-- 000010_home_assistant_parity.up.sql

-- 1. Template Entities (Virtual computed sensors, binary sensors, and switches)
CREATE TABLE IF NOT EXISTS template_entities (
    id TEXT PRIMARY KEY,
    entity_id TEXT NOT NULL UNIQUE,
    domain TEXT NOT NULL,
    name TEXT NOT NULL,
    state_template TEXT NOT NULL,
    attributes_template TEXT NOT NULL DEFAULT '{}',
    unit_of_measurement TEXT,
    device_class TEXT,
    icon TEXT,
    source_entity_ids TEXT NOT NULL DEFAULT '[]',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 2. Long Term Statistics (LTS Rollup Aggregates)
CREATE TABLE IF NOT EXISTS long_term_statistics (
    id TEXT PRIMARY KEY,
    entity_id TEXT NOT NULL,
    period TEXT NOT NULL, -- 'hourly', 'daily'
    period_start TIMESTAMP NOT NULL,
    period_end TIMESTAMP NOT NULL,
    mean REAL,
    min REAL,
    max REAL,
    sum REAL,
    count INTEGER NOT NULL DEFAULT 0,
    unit TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(entity_id, period, period_start)
);
CREATE INDEX IF NOT EXISTS idx_lts_entity_period ON long_term_statistics (entity_id, period, period_start);

-- 3. Two-Factor Authentication (2FA TOTP RFC 6238)
CREATE TABLE IF NOT EXISTS user_two_factor (
    user_id TEXT PRIMARY KEY,
    secret TEXT NOT NULL,
    is_enabled INTEGER NOT NULL DEFAULT 0,
    backup_codes TEXT NOT NULL DEFAULT '[]',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- 4. Banned IPs & Brute Force Defense
CREATE TABLE IF NOT EXISTS banned_ips (
    ip_address TEXT PRIMARY KEY,
    failed_attempts INTEGER NOT NULL DEFAULT 0,
    banned_until TIMESTAMP,
    reason TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 5. Multi-Utility Energy Meters (Gas, Water, Electricity)
CREATE TABLE IF NOT EXISTS energy_utility_meters (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    utility_type TEXT NOT NULL, -- 'electricity', 'gas', 'water'
    unit TEXT NOT NULL, -- 'kWh', 'm3', 'L'
    cost_per_unit REAL NOT NULL DEFAULT 0.0,
    source_entity_id TEXT NOT NULL,
    current_reading REAL NOT NULL DEFAULT 0.0,
    last_reset TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 6. Spatial Multi-Zones
CREATE TABLE IF NOT EXISTS spatial_zones (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    latitude REAL NOT NULL,
    longitude REAL NOT NULL,
    radius REAL NOT NULL DEFAULT 100.0,
    icon TEXT NOT NULL DEFAULT 'map-pin',
    passive INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Seed default home zone if empty
INSERT OR IGNORE INTO spatial_zones (id, name, latitude, longitude, radius, icon, passive)
VALUES ('zone_home', 'Casa', 45.4642, 9.1900, 150.0, 'home', 0);

-- 7. Tracked Persons (Multi-Device presence aggregation)
CREATE TABLE IF NOT EXISTS tracked_persons (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    user_id TEXT,
    state TEXT NOT NULL DEFAULT 'home', -- 'home', 'not_home', or zone name
    latitude REAL,
    longitude REAL,
    device_trackers_json TEXT NOT NULL DEFAULT '[]',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
