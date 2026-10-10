package helper_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	appHelper "github.com/domus-hub/domus/internal/application/helper"
	domainHelper "github.com/domus-hub/domus/internal/domain/helper"
	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open memory db: %v", err)
	}

	createTable := `
	CREATE TABLE user_helpers (
		id TEXT PRIMARY KEY,
		entity_key TEXT UNIQUE NOT NULL,
		name TEXT NOT NULL,
		domain TEXT NOT NULL,
		state TEXT NOT NULL,
		config_json TEXT NOT NULL DEFAULT '{}',
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	`
	if _, err := db.Exec(createTable); err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	return db, func() { db.Close() }
}

func TestHelperService_Operations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := sqlite.NewHelperRepository(db)
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 64, Workers: 1})
	svc := appHelper.NewService(repo, bus)

	ctx := context.Background()

	// 1. Test Input Boolean
	boolH := domainHelper.NewInputBoolean("bool_1", "Modalità Ospiti", "input_boolean.guest_mode", "user-check", false)
	if err := svc.Create(ctx, boolH); err != nil {
		t.Fatalf("failed to create boolean: %v", err)
	}

	if err := svc.ExecuteAction(ctx, "input_boolean.guest_mode", "turn_on", nil); err != nil {
		t.Fatalf("turn_on failed: %v", err)
	}
	h, _ := svc.GetByID(ctx, "bool_1")
	if h.State != "on" {
		t.Errorf("expected state 'on', got %s", h.State)
	}

	if err := svc.ExecuteAction(ctx, "input_boolean.guest_mode", "toggle", nil); err != nil {
		t.Fatalf("toggle failed: %v", err)
	}
	h, _ = svc.GetByID(ctx, "bool_1")
	if h.State != "off" {
		t.Errorf("expected state 'off' after toggle, got %s", h.State)
	}

	// 2. Test Input Number
	numH := domainHelper.NewInputNumber("num_1", "Soglia Riscaldamento", "input_number.heating_threshold", 15.0, 25.0, 0.5, 20.0, "°C")
	if err := svc.Create(ctx, numH); err != nil {
		t.Fatalf("failed to create number helper: %v", err)
	}

	// Try setting value above max -> clamped to 25.0
	if err := svc.ExecuteAction(ctx, "input_number.heating_threshold", "set_value", map[string]any{"value": 30.0}); err != nil {
		t.Fatalf("set_value failed: %v", err)
	}
	h, _ = svc.GetByID(ctx, "num_1")
	if h.State != "25.00" {
		t.Errorf("expected clamped value '25.00', got %s", h.State)
	}

	// 3. Test Timer with short duration (1 sec)
	timerH := domainHelper.NewTimer("timer_1", "Timer Pizza", "timer.pizza", 1)
	if err := svc.Create(ctx, timerH); err != nil {
		t.Fatalf("failed to create timer: %v", err)
	}

	finishedCh := make(chan struct{}, 1)
	bus.Subscribe("timer.finished", func(c context.Context, e events.Event) error {
		finishedCh <- struct{}{}
		return nil
	})

	if err := svc.ExecuteAction(ctx, "timer.pizza", "start", map[string]any{"duration": float64(1)}); err != nil {
		t.Fatalf("start timer failed: %v", err)
	}

	select {
	case <-finishedCh:
		// Timer finished successfully
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for timer.finished event")
	}

	h, _ = svc.GetByID(ctx, "timer_1")
	if h.State != "idle" {
		t.Errorf("expected timer state idle after finish, got %s", h.State)
	}
}
