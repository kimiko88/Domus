-- 000001_init.down.sql
-- Rollback initial Domus schema

DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS state_history;
DROP TABLE IF EXISTS scenes;
DROP TABLE IF EXISTS automation_runs;
DROP TABLE IF EXISTS automations;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS entities;
DROP TABLE IF EXISTS devices;
DROP TABLE IF EXISTS integrations;
DROP TABLE IF EXISTS areas;
DROP TABLE IF EXISTS schema_version;
