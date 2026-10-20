package sqlite_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
)

func TestMaintenanceService_RunCheckpoint(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_maintenance.db")

	cfg := config.DatabaseConfig{
		Path: dbPath,
	}

	pool, err := sqlite.OpenDualPool(cfg)
	if err != nil {
		t.Fatalf("OpenDualPool failed: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()
	if err := sqlite.Migrate(ctx, pool.Write); err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	maint := sqlite.NewMaintenanceService(pool, sqlite.MaintenanceConfig{
		CheckpointInterval: 100 * time.Millisecond,
		DBPath:             dbPath,
	})

	stats, err := maint.RunCheckpoint(ctx)
	if err != nil {
		t.Fatalf("RunCheckpoint returned error: %v", err)
	}

	if !stats.Checkpointed {
		t.Errorf("expected Checkpointed to be true")
	}
	if stats.DBSizeBytes == 0 {
		t.Errorf("expected non-zero DB file size on disk")
	}

	last := maint.LastStats()
	if last == nil || last.DBSizeBytes != stats.DBSizeBytes {
		t.Errorf("expected LastStats to match checkpoint result")
	}
}

func TestMaintenanceService_SchedulerLifecycle(t *testing.T) {
	pool, err := sqlite.OpenDualPool(config.DatabaseConfig{Path: ":memory:"})
	if err != nil {
		t.Fatalf("OpenDualPool failed: %v", err)
	}
	defer pool.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	maint := sqlite.NewMaintenanceService(pool, sqlite.MaintenanceConfig{
		CheckpointInterval: 20 * time.Millisecond,
		DBPath:             ":memory:",
	})

	maint.Start(ctx)
	time.Sleep(60 * time.Millisecond)
	maint.Stop()

	// Should not panic or hang
}
