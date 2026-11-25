-- 000026: signed plugin repositories and staged artifact catalog.
CREATE TABLE IF NOT EXISTS marketplace_repositories (
    id TEXT PRIMARY KEY,
    url TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL DEFAULT '',
    owner TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    key_id TEXT NOT NULL DEFAULT '',
    trusted INTEGER NOT NULL DEFAULT 0,
    enabled INTEGER NOT NULL DEFAULT 1,
    index_sha256 TEXT NOT NULL DEFAULT '',
    last_synced_at TEXT,
    last_error TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS marketplace_packages (
    id TEXT PRIMARY KEY,
    repository_id TEXT NOT NULL,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    version TEXT NOT NULL,
    author TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    license TEXT NOT NULL DEFAULT '',
    runtime TEXT NOT NULL,
    min_domus_version TEXT NOT NULL DEFAULT '',
    max_domus_version TEXT NOT NULL DEFAULT '',
    capabilities_json TEXT NOT NULL DEFAULT '[]',
    artifact_url TEXT NOT NULL,
    artifact_sha256 TEXT NOT NULL,
    artifact_size INTEGER NOT NULL,
    trusted INTEGER NOT NULL DEFAULT 0,
    compatible INTEGER NOT NULL DEFAULT 1,
    installed_path TEXT NOT NULL DEFAULT '',
    installed_at TEXT,
    updated_at TEXT NOT NULL,
    FOREIGN KEY(repository_id) REFERENCES marketplace_repositories(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_marketplace_packages_repository
    ON marketplace_packages(repository_id);
CREATE INDEX IF NOT EXISTS idx_marketplace_packages_type
    ON marketplace_packages(type);
