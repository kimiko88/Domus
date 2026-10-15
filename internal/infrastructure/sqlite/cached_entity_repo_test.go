package sqlite_test

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/domain/entity"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
)

func TestCachedEntityRepository(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "cached_test.db")

	cfg := config.DatabaseConfig{
		Path:          dbPath,
		WALMode:       true,
		BusyTimeoutMs: 5000,
	}

	db, err := sqlite.Open(cfg)
	if err != nil {
		t.Fatalf("failed to open sqlite db: %v", err)
	}
	defer db.Close()

	if err := sqlite.Migrate(ctx, db); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}

	baseRepo := sqlite.NewEntityRepository(db)
	cachedRepo := sqlite.NewCachedEntityRepository(baseRepo)

	// Create test entities in base
	e1 := &entity.Entity{
		ID:            "light-1",
		EntityKey:     "light.kitchen",
		Domain:        "light",
		Name:          "Kitchen Light",
		State:         `{"on":true}`,
		Attributes:    `{"brightness":255}`,
		Available:     true,
		LastChangedAt: time.Now().UTC(),
		LastUpdatedAt: time.Now().UTC(),
	}
	if err := baseRepo.Create(ctx, e1); err != nil {
		t.Fatalf("failed to create entity: %v", err)
	}

	// Warmup cache
	if err := cachedRepo.WarmUp(ctx); err != nil {
		t.Fatalf("warmup failed: %v", err)
	}

	// Microsecond read benchmark test
	start := time.Now()
	for i := 0; i < 1000; i++ {
		ent, err := cachedRepo.GetByID(ctx, "light-1")
		if err != nil || ent == nil {
			t.Fatalf("failed to read from cache: %v", err)
		}
	}
	elapsed := time.Since(start)
	avgPerRead := elapsed / 1000
	t.Logf("1000 cached reads took %v (average %v per read)", elapsed, avgPerRead)
	if avgPerRead > 100*time.Microsecond {
		t.Errorf("read too slow: expected < 100µs, got %v", avgPerRead)
	}

	// Test GetByKey
	byKey, err := cachedRepo.GetByKey(ctx, "light.kitchen")
	if err != nil || byKey.ID != "light-1" {
		t.Errorf("expected entity by key, got %v, err: %v", byKey, err)
	}

	// Test concurrent reads and writes
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_, _ = cachedRepo.GetByID(ctx, "light-1")
				_, _ = cachedRepo.List(ctx, entity.ListFilter{Domain: "light"})
			}
		}(i)
	}

	// Concurrent state updates
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_ = cachedRepo.UpdateState(ctx, "light-1", `{"on":false}`, `{}`, true, time.Now().UTC())
		}(i)
	}
	wg.Wait()

	// Verify state after updates
	finalEnt, err := cachedRepo.GetByID(ctx, "light-1")
	if err != nil {
		t.Fatalf("failed to get final entity: %v", err)
	}
	if finalEnt.State != `{"on":false}` {
		t.Errorf("expected final state to be off, got: %s", finalEnt.State)
	}
}
