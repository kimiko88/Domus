-- 000003_floorplans.up.sql
-- Floor plan management with multi-floor levels and entity positioning

CREATE TABLE floors (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    level       INTEGER NOT NULL DEFAULT 0,
    plan_svg    TEXT,
    plan_image  TEXT,
    sort_order  INTEGER NOT NULL DEFAULT 0,
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    updated_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

CREATE INDEX idx_floors_level ON floors(level, sort_order);

CREATE TABLE floor_markers (
    id            TEXT PRIMARY KEY,
    floor_id      TEXT NOT NULL REFERENCES floors(id) ON DELETE CASCADE,
    entity_key    TEXT NOT NULL,
    x_percent     REAL NOT NULL DEFAULT 50.0,
    y_percent     REAL NOT NULL DEFAULT 50.0,
    icon_override TEXT,
    show_badge    INTEGER NOT NULL DEFAULT 1,
    created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    updated_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    UNIQUE(floor_id, entity_key)
);

CREATE INDEX idx_floor_markers_floor ON floor_markers(floor_id);
