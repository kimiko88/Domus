-- Migration 000012: Outbound Webhooks and Persistent In-App Notifications
-- Hardens ecosystem extensibility and notification delivery persistence.

CREATE TABLE IF NOT EXISTS outbound_webhooks (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    target_url TEXT NOT NULL,
    secret_token TEXT NOT NULL DEFAULT '',
    events_json TEXT NOT NULL DEFAULT '["*"]',
    enabled INTEGER NOT NULL DEFAULT 1,
    last_dispatched_at DATETIME,
    last_status INTEGER NOT NULL DEFAULT 0,
    fail_count INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_outbound_webhooks_enabled ON outbound_webhooks(enabled);

CREATE TABLE IF NOT EXISTS inapp_notifications (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    message TEXT NOT NULL,
    category TEXT NOT NULL DEFAULT 'system',
    severity TEXT NOT NULL DEFAULT 'info',
    action TEXT,
    is_read INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_inapp_notifications_created_at ON inapp_notifications(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_inapp_notifications_is_read ON inapp_notifications(is_read);
