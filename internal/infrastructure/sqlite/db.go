// Package sqlite provides SQLite database initialization, PRAGMA configuration,
// and embedded schema migrations for Domus.
package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// Open establishes a connection to the SQLite database and configures required PRAGMAs.
func Open(cfg config.DatabaseConfig) (*sql.DB, error) {
	if cfg.Path == "" {
		return nil, fmt.Errorf("sqlite: database path cannot be empty")
	}

	// Ensure parent directory exists for file-based DBs
	if cfg.Path != ":memory:" {
		dir := filepath.Dir(cfg.Path)
		if dir != "." && dir != "" {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("sqlite: failed to create database directory: %w", err)
			}
		}
	}

	db, err := sql.Open("sqlite", cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("sqlite: failed to open database: %w", err)
	}

	// Configure connection limits (default to 1 writer connection for safety with SQLite)
	maxConns := cfg.MaxOpenConns
	if maxConns <= 0 {
		maxConns = 1
	}
	db.SetMaxOpenConns(maxConns)

	// Apply runtime PRAGMAs
	if cfg.WALMode {
		if _, err := db.Exec("PRAGMA journal_mode = WAL;"); err != nil {
			db.Close()
			return nil, fmt.Errorf("sqlite: failed to set WAL mode: %w", err)
		}
	}

	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		db.Close()
		return nil, fmt.Errorf("sqlite: failed to enable foreign keys: %w", err)
	}

	timeout := cfg.BusyTimeoutMs
	if timeout <= 0 {
		timeout = 5000
	}
	if _, err := db.Exec(fmt.Sprintf("PRAGMA busy_timeout = %d;", timeout)); err != nil {
		db.Close()
		return nil, fmt.Errorf("sqlite: failed to set busy_timeout: %w", err)
	}

	// Set synchronous = NORMAL for optimal WAL performance
	if _, err := db.Exec("PRAGMA synchronous = NORMAL;"); err != nil {
		db.Close()
		return nil, fmt.Errorf("sqlite: failed to set synchronous pragma: %w", err)
	}

	return db, nil
}

// Migrate executes pending migrations embedded in migrationFS in a single transaction.
func Migrate(ctx context.Context, db *sql.DB) error {
	// Ensure schema_version table exists to track migrations
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_version (
			version     INTEGER PRIMARY KEY,
			applied_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
		);
	`)
	if err != nil {
		return fmt.Errorf("sqlite: failed to verify schema_version table: %w", err)
	}

	migrations := []struct {
		version int
		file    string
	}{
		{version: 1, file: "migrations/000001_init.up.sql"},
		{version: 2, file: "migrations/000002_family_invites.up.sql"},
		{version: 3, file: "migrations/000003_floorplans.up.sql"},
		{version: 4, file: "migrations/000004_chores.up.sql"},
		{version: 5, file: "migrations/000005_advanced_features.up.sql"},
		{version: 6, file: "migrations/000006_hardware_and_advanced_dashboards.up.sql"},
		{version: 7, file: "migrations/000007_advanced_ecosystem.up.sql"},
		{version: 8, file: "migrations/000008_shopping_list.up.sql"},
		{version: 9, file: "migrations/000009_advanced_security_and_integrations.up.sql"},
		{version: 10, file: "migrations/000010_home_assistant_parity.up.sql"},
		{version: 11, file: "migrations/000011_scripts_logbook_push.up.sql"},
		{version: 12, file: "migrations/000012_outbound_webhooks_and_inapp_notifications.up.sql"},
		{version: 13, file: "migrations/000013_home_finance.up.sql"},
		{version: 14, file: "migrations/000014_pantry_recipes_mealplan.up.sql"},
		{version: 15, file: "migrations/000015_medicines.up.sql"},
		{version: 16, file: "migrations/000016_predictive_climate.up.sql"},
	}

	for _, m := range migrations {
		var count int
		err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_version WHERE version = ?;", m.version).Scan(&count)
		if err != nil {
			return fmt.Errorf("sqlite: failed to query schema_version for v%d: %w", m.version, err)
		}
		if count > 0 {
			continue // Already applied
		}

		content, err := migrationFS.ReadFile(m.file)
		if err != nil {
			return fmt.Errorf("sqlite: failed to read migration %s: %w", m.file, err)
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("sqlite: failed to start migration transaction for v%d: %w", m.version, err)
		}

		if _, err := tx.ExecContext(ctx, string(content)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("sqlite: failed to execute migration v%d (%s): %w", m.version, m.file, err)
		}

		if _, err := tx.ExecContext(ctx, "INSERT OR IGNORE INTO schema_version (version) VALUES (?);", m.version); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("sqlite: failed to record schema_version %d: %w", m.version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("sqlite: failed to commit migration transaction for v%d: %w", m.version, err)
		}
	}

	return nil
}

// LogWALTrace logs a SQLite WAL transaction or query execution correlating it with W3C distributed trace context.
func LogWALTrace(ctx context.Context, op string, duration time.Duration, err error) {
	attrs := []any{
		"db", "sqlite_wal",
		"operation", op,
		"duration_ms", duration.Milliseconds(),
	}
	if tc := events.TraceFromContext(ctx); tc != nil {
		attrs = append(attrs, "trace_id", tc.TraceID, "span_id", tc.SpanID, "traceparent", tc.String())
	}
	if err != nil {
		attrs = append(attrs, "error", err.Error())
		slog.Error("sqlite wal operation failed", attrs...)
	} else if duration > 50*time.Millisecond {
		slog.Warn("sqlite wal slow query", attrs...)
	} else {
		slog.Debug("sqlite wal operation", attrs...)
	}
}
