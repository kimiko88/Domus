-- 000008_shopping_list.up.sql
CREATE TABLE IF NOT EXISTS shopping_items (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    quantity TEXT,
    category TEXT NOT NULL DEFAULT 'alimentari',
    supermarket TEXT DEFAULT '',
    completed INTEGER NOT NULL DEFAULT 0,
    added_by TEXT NOT NULL,
    completed_by TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_shopping_items_completed ON shopping_items(completed);
