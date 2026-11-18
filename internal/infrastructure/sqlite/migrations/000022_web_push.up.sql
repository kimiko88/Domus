-- Migration 000022: Browser Web Push subscriptions and per-user alert preferences.
CREATE TABLE IF NOT EXISTS web_push_subscriptions (
    id              TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    endpoint        TEXT NOT NULL UNIQUE,
    p256dh          TEXT NOT NULL,
    auth_secret     TEXT NOT NULL,
    locale          TEXT NOT NULL DEFAULT 'it',
    expiration_time INTEGER,
    created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    updated_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

CREATE INDEX IF NOT EXISTS idx_web_push_subscriptions_user
    ON web_push_subscriptions(user_id);

CREATE TABLE IF NOT EXISTS web_push_preferences (
    user_id                    TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    alarm_alerts               INTEGER NOT NULL DEFAULT 1,
    emergency_alerts           INTEGER NOT NULL DEFAULT 1,
    device_offline_alerts      INTEGER NOT NULL DEFAULT 1,
    automation_failure_alerts  INTEGER NOT NULL DEFAULT 1,
    command_failure_alerts     INTEGER NOT NULL DEFAULT 0,
    network_alerts             INTEGER NOT NULL DEFAULT 1,
    camera_alerts              INTEGER NOT NULL DEFAULT 0,
    doorbell_alerts            INTEGER NOT NULL DEFAULT 1,
    pantry_alerts              INTEGER NOT NULL DEFAULT 0,
    finance_alerts             INTEGER NOT NULL DEFAULT 0,
    created_at                 TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    updated_at                 TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);
