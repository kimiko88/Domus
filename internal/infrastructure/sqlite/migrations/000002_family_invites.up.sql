CREATE TABLE IF NOT EXISTS family_invites (
    id                 TEXT PRIMARY KEY,
    token              TEXT NOT NULL UNIQUE,
    role               TEXT NOT NULL DEFAULT 'viewer',
    created_by_user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    expires_at         TEXT NOT NULL,
    used_at            TEXT,
    used_by_user_id    TEXT REFERENCES users(id) ON DELETE SET NULL,
    created_at         TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

CREATE INDEX IF NOT EXISTS idx_invites_token ON family_invites(token);
CREATE INDEX IF NOT EXISTS idx_invites_exp   ON family_invites(expires_at);
