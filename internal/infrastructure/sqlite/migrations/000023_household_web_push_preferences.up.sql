-- Migration 000023: Household task, medicine and food Web Push reminder preferences.
ALTER TABLE web_push_preferences ADD COLUMN task_alerts INTEGER NOT NULL DEFAULT 1;
ALTER TABLE web_push_preferences ADD COLUMN medicine_expiry_alerts INTEGER NOT NULL DEFAULT 1;
ALTER TABLE web_push_preferences ADD COLUMN medicine_low_stock_alerts INTEGER NOT NULL DEFAULT 1;
ALTER TABLE web_push_preferences ADD COLUMN medicine_expiry_days INTEGER NOT NULL DEFAULT 7;
ALTER TABLE web_push_preferences ADD COLUMN food_expiry_alerts INTEGER NOT NULL DEFAULT 1;
ALTER TABLE web_push_preferences ADD COLUMN food_low_stock_alerts INTEGER NOT NULL DEFAULT 1;
ALTER TABLE web_push_preferences ADD COLUMN food_expiry_days INTEGER NOT NULL DEFAULT 3;

CREATE TABLE IF NOT EXISTS web_push_delivery_log (
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    dedupe_key TEXT NOT NULL,
    sent_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    PRIMARY KEY(user_id, dedupe_key)
);

CREATE INDEX IF NOT EXISTS idx_web_push_delivery_sent_at
    ON web_push_delivery_log(sent_at);
