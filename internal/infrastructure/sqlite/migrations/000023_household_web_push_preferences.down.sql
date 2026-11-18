-- Migration 000023 rollback: remove household Web Push delivery ledger.
DROP TABLE IF EXISTS web_push_delivery_log;
-- SQLite versions predating DROP COLUMN retain the additional preference columns.
