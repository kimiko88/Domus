DROP INDEX IF EXISTS idx_chores_recurrence_parent;
ALTER TABLE chores DROP COLUMN recurrence_parent_id;
