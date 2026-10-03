// Package sqlite_test tests the SQLite database connection, PRAGMA configuration,
// and embedded schema migrations (TDD red-to-green phase).
package sqlite_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpen_PRAGMAsAndPing(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_domus.db")

	cfg := config.DatabaseConfig{
		Path:          dbPath,
		WALMode:       true,
		BusyTimeoutMs: 5000,
		MaxOpenConns:  1,
	}

	db, err := sqlite.Open(cfg)
	require.NoError(t, err)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	assert.NoError(t, err)

	// Verify journal_mode is WAL
	var journalMode string
	err = db.QueryRowContext(ctx, "PRAGMA journal_mode;").Scan(&journalMode)
	require.NoError(t, err)
	assert.Equal(t, "wal", journalMode)

	// Verify foreign_keys is ON
	var foreignKeys int
	err = db.QueryRowContext(ctx, "PRAGMA foreign_keys;").Scan(&foreignKeys)
	require.NoError(t, err)
	assert.Equal(t, 1, foreignKeys)

	// Verify busy_timeout is set
	var busyTimeout int
	err = db.QueryRowContext(ctx, "PRAGMA busy_timeout;").Scan(&busyTimeout)
	require.NoError(t, err)
	assert.Equal(t, 5000, busyTimeout)
}

func TestMigrate_CreatesAllTablesAndIsIdempotent(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_migrate.db")

	cfg := config.DatabaseConfig{
		Path:          dbPath,
		WALMode:       true,
		BusyTimeoutMs: 5000,
		MaxOpenConns:  1,
	}

	db, err := sqlite.Open(cfg)
	require.NoError(t, err)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1st migration run
	err = sqlite.Migrate(ctx, db)
	require.NoError(t, err)

	// Verify tables exist
	expectedTables := []string{
		"areas",
		"integrations",
		"devices",
		"entities",
		"users",
		"sessions",
		"automations",
		"automation_runs",
		"scenes",
		"state_history",
		"audit_log",
		"schema_version",
	}

	for _, table := range expectedTables {
		var name string
		query := "SELECT name FROM sqlite_master WHERE type='table' AND name=?;"
		err := db.QueryRowContext(ctx, query, table).Scan(&name)
		assert.NoError(t, err, "Table %s should exist in database", table)
		assert.Equal(t, table, name)
	}

	// 2nd migration run (idempotency check)
	err = sqlite.Migrate(ctx, db)
	assert.NoError(t, err, "Migrate must be idempotent")
}

func TestForeignKeyEnforcement(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_fk.db")

	cfg := config.DatabaseConfig{
		Path:          dbPath,
		WALMode:       true,
		BusyTimeoutMs: 5000,
		MaxOpenConns:  1,
	}

	db, err := sqlite.Open(cfg)
	require.NoError(t, err)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	require.NoError(t, sqlite.Migrate(ctx, db))

	// Inserting a device with non-existent integration_id must fail due to FK
	_, err = db.ExecContext(ctx, `
		INSERT INTO devices (id, integration_id, external_id, name)
		VALUES ('dev-1', 'non-existent-integration', 'ext-1', 'Living Room Sensor');
	`)
	assert.Error(t, err, "Inserting child record with invalid FK must fail")
}
