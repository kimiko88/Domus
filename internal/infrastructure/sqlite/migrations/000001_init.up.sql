-- 000001_init.up.sql
-- Initial Domus schema — version 1
-- PRAGMAs (WAL, FK, busy_timeout) are set programmatically in db.go, not here.

CREATE TABLE areas (
    id             TEXT PRIMARY KEY,
    name           TEXT NOT NULL,
    icon           TEXT,
    parent_area_id TEXT REFERENCES areas(id) ON DELETE SET NULL,
    sort_order     INTEGER NOT NULL DEFAULT 0,
    created_at     TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    updated_at     TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

CREATE TABLE integrations (
    id               TEXT PRIMARY KEY,
    type             TEXT NOT NULL,
    name             TEXT NOT NULL,
    enabled          INTEGER NOT NULL DEFAULT 1,
    config_encrypted BLOB,
    status           TEXT NOT NULL DEFAULT 'disconnected',
    last_error       TEXT,
    created_at       TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    updated_at       TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

CREATE TABLE devices (
    id             TEXT PRIMARY KEY,
    integration_id TEXT NOT NULL REFERENCES integrations(id) ON DELETE CASCADE,
    external_id    TEXT NOT NULL,
    name           TEXT NOT NULL,
    manufacturer   TEXT,
    model          TEXT,
    area_id        TEXT REFERENCES areas(id) ON DELETE SET NULL,
    online         INTEGER NOT NULL DEFAULT 0,
    metadata       TEXT NOT NULL DEFAULT '{}',
    created_at     TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    updated_at     TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    UNIQUE(integration_id, external_id)
);

CREATE INDEX idx_devices_integration ON devices(integration_id);
CREATE INDEX idx_devices_area        ON devices(area_id);

CREATE TABLE entities (
    id                  TEXT PRIMARY KEY,
    device_id           TEXT REFERENCES devices(id) ON DELETE CASCADE,
    entity_key          TEXT NOT NULL UNIQUE,
    domain              TEXT NOT NULL,
    name                TEXT NOT NULL,
    icon                TEXT,
    unit_of_measurement TEXT,
    writable            INTEGER NOT NULL DEFAULT 0,
    state               TEXT NOT NULL DEFAULT 'null',
    attributes          TEXT NOT NULL DEFAULT '{}',
    available           INTEGER NOT NULL DEFAULT 1,
    area_id             TEXT REFERENCES areas(id) ON DELETE SET NULL,
    last_changed_at     TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    last_updated_at     TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

CREATE INDEX idx_entities_domain ON entities(domain);
CREATE INDEX idx_entities_area   ON entities(area_id);
CREATE INDEX idx_entities_device ON entities(device_id);

CREATE TABLE users (
    id            TEXT PRIMARY KEY,
    username      TEXT NOT NULL UNIQUE,
    email         TEXT UNIQUE,
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL DEFAULT 'viewer',
    enabled       INTEGER NOT NULL DEFAULT 1,
    created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    updated_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

CREATE TABLE sessions (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    ip_address TEXT,
    user_agent TEXT
);

CREATE INDEX idx_sessions_user ON sessions(user_id);
CREATE INDEX idx_sessions_exp  ON sessions(expires_at);

CREATE TABLE automations (
    id               TEXT PRIMARY KEY,
    name             TEXT NOT NULL,
    description      TEXT,
    enabled          INTEGER NOT NULL DEFAULT 1,
    trigger          TEXT NOT NULL DEFAULT '{}',
    conditions       TEXT NOT NULL DEFAULT '[]',
    actions          TEXT NOT NULL DEFAULT '[]',
    execution_mode   TEXT NOT NULL DEFAULT 'single',
    cooldown_seconds INTEGER NOT NULL DEFAULT 0,
    last_run_at      TEXT,
    last_status      TEXT,
    created_at       TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    updated_at       TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

CREATE TABLE automation_runs (
    id              TEXT PRIMARY KEY,
    automation_id   TEXT NOT NULL REFERENCES automations(id) ON DELETE CASCADE,
    started_at      TEXT NOT NULL,
    finished_at     TEXT,
    status          TEXT NOT NULL DEFAULT 'running',
    trigger_context TEXT NOT NULL DEFAULT '{}',
    log             TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_runs_automation ON automation_runs(automation_id, started_at DESC);

CREATE TABLE scenes (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    icon       TEXT,
    actions    TEXT NOT NULL DEFAULT '[]',
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

CREATE TABLE state_history (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    entity_id   TEXT NOT NULL,
    state       TEXT NOT NULL,
    recorded_at TEXT NOT NULL
);

CREATE INDEX idx_history_entity_time ON state_history(entity_id, recorded_at DESC);

CREATE TABLE audit_log (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    actor_type  TEXT NOT NULL,
    actor_id    TEXT,
    action      TEXT NOT NULL,
    target_type TEXT,
    target_id   TEXT,
    payload     TEXT NOT NULL DEFAULT '{}',
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

CREATE INDEX idx_audit_time   ON audit_log(created_at DESC);
CREATE INDEX idx_audit_target ON audit_log(target_type, target_id);
