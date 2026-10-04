// Package sqlite provides SQLite database initialization, PRAGMA configuration,
// and embedded schema migrations for Domus.
package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"

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
