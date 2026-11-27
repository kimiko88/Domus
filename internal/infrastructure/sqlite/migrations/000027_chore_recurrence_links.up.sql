-- Track which completed chore generated the next recurring instance.
-- This allows an accidental completion to be undone without touching unrelated chores.
ALTER TABLE chores
    ADD COLUMN recurrence_parent_id TEXT REFERENCES chores(id) ON DELETE CASCADE;

CREATE INDEX idx_chores_recurrence_parent
    ON chores(recurrence_parent_id, status);
