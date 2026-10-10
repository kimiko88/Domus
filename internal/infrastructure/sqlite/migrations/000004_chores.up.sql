-- 000004_chores.up.sql
-- Domestic chores management with recurrence and reminders

CREATE TABLE chores (
    id                TEXT PRIMARY KEY,
    title             TEXT NOT NULL,
    description       TEXT,
    area_id           TEXT REFERENCES areas(id) ON DELETE SET NULL,
    assigned_user_id  TEXT REFERENCES users(id) ON DELETE SET NULL,
    priority          TEXT NOT NULL DEFAULT 'medium',
    due_date          TEXT,
    recurrence_type   TEXT NOT NULL DEFAULT 'none',
    recurrence_rule   TEXT,
    status            TEXT NOT NULL DEFAULT 'pending',
    last_completed_at TEXT,
    last_completed_by TEXT REFERENCES users(id) ON DELETE SET NULL,
    created_at        TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    updated_at        TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

CREATE INDEX idx_chores_status_due ON chores(status, due_date);
CREATE INDEX idx_chores_assignee   ON chores(assigned_user_id);
CREATE INDEX idx_chores_area       ON chores(area_id);
