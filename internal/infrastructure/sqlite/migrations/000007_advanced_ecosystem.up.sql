-- 000007_advanced_ecosystem.up.sql
-- Cameras and NVR Vision
CREATE TABLE IF NOT EXISTS cameras (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    area_id TEXT,
    stream_url TEXT NOT NULL,
    snapshot_url TEXT,
    stream_type TEXT NOT NULL DEFAULT 'webrtc',
    detection_enabled INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (area_id) REFERENCES areas(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS camera_events (
    id TEXT PRIMARY KEY,
    camera_id TEXT NOT NULL,
    label TEXT NOT NULL,
    score REAL NOT NULL,
    thumbnail_url TEXT,
    detected_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (camera_id) REFERENCES cameras(id) ON DELETE CASCADE
);

-- NFC & QR Tags
CREATE TABLE IF NOT EXISTS nfc_tags (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    tag_id TEXT NOT NULL UNIQUE,
    action_type TEXT NOT NULL, -- 'scene', 'automation', 'service'
    action_target_id TEXT NOT NULL,
    last_scanned_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Weekly Graphical Schedules (Cronotermostato)
CREATE TABLE IF NOT EXISTS schedules (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    entity_key TEXT NOT NULL,
    schedule_json TEXT NOT NULL, -- { "monday": [{"start": "06:00", "end": "08:30", "value": 20.5}], ... }
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Local Calendar Events
CREATE TABLE IF NOT EXISTS calendar_events (
    id TEXT PRIMARY KEY,
    summary TEXT NOT NULL,
    description TEXT,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL,
    location TEXT,
    all_day INTEGER NOT NULL DEFAULT 0,
    source TEXT NOT NULL DEFAULT 'local',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Multi-room Media Players
CREATE TABLE IF NOT EXISTS media_players (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    device_id TEXT,
    area_id TEXT,
    platform TEXT NOT NULL, -- 'cast', 'airplay', 'dlna', 'local'
    state TEXT NOT NULL DEFAULT 'idle', -- 'idle', 'playing', 'paused'
    volume REAL NOT NULL DEFAULT 0.5,
    muted INTEGER NOT NULL DEFAULT 0,
    current_title TEXT,
    current_artist TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (area_id) REFERENCES areas(id) ON DELETE SET NULL
);

-- OTA Firmware Updates
CREATE TABLE IF NOT EXISTS ota_devices (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    device_key TEXT NOT NULL UNIQUE,
    platform TEXT NOT NULL, -- 'zigbee', 'esphome', 'matter', 'zwave'
    current_version TEXT NOT NULL,
    latest_version TEXT NOT NULL,
    upgrade_state TEXT NOT NULL DEFAULT 'idle', -- 'idle', 'downloading', 'flashing', 'success', 'failed'
    release_notes TEXT,
    checked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- User Granular RBAC Permissions
CREATE TABLE IF NOT EXISTS user_permissions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL UNIQUE,
    role TEXT NOT NULL DEFAULT 'member', -- 'admin', 'member', 'guest'
    allowed_areas_json TEXT NOT NULL DEFAULT '[]',
    allowed_dashboards_json TEXT NOT NULL DEFAULT '[]',
    can_manage_automations INTEGER NOT NULL DEFAULT 1,
    can_manage_settings INTEGER NOT NULL DEFAULT 0,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Community Store Packages (WASM, Blueprints, Widgets)
CREATE TABLE IF NOT EXISTS store_packages (
    id TEXT PRIMARY KEY,
    package_id TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    type TEXT NOT NULL, -- 'wasm_plugin', 'blueprint', 'widget'
    version TEXT NOT NULL,
    author TEXT NOT NULL,
    description TEXT,
    installed INTEGER NOT NULL DEFAULT 0,
    installed_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_cameras_area ON cameras(area_id);
CREATE INDEX IF NOT EXISTS idx_camera_events_camera ON camera_events(camera_id);
CREATE INDEX IF NOT EXISTS idx_calendar_events_start ON calendar_events(start_time);
CREATE INDEX IF NOT EXISTS idx_nfc_tags_tag_id ON nfc_tags(tag_id);
