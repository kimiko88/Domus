package automation_test

import (
	"context"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	appauto "github.com/domus-hub/domus/internal/application/automation"
	"github.com/domus-hub/domus/internal/domain/automation"
	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestControlFlowActions(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "control_flow.db")

	db, err := sqlite.Open(config.DatabaseConfig{
		Path:          dbPath,
		WALMode:       true,
		BusyTimeoutMs: 5000,
		MaxOpenConns:  1,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	require.NoError(t, sqlite.Migrate(ctx, db))

	repos := sqlite.NewRepositories(db)
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 64, Workers: 2})
	engine := appauto.NewEngine(repos.Automations, repos.Entities, bus)
	defer func() {
		_ = engine.Stop(context.Background())
		bus.Close()
		db.Close()
	}()

	var commandCount int32
	bus.Subscribe(events.CommandRequested, func(c context.Context, e events.Event) error {
		atomic.AddInt32(&commandCount, 1)
		return nil
	})

	runCompletedChan := make(chan events.Event, 10)
	bus.Subscribe(events.AutomationRunCompleted, func(c context.Context, e events.Event) error {
		runCompletedChan <- e
		return nil
	})

	// 1. Repeat Count Test
	ruleRepeat := &automation.Automation{
		ID:            "test_repeat",
		Name:          "Test Repeat Action",
		Enabled:       true,
		ExecutionMode: "single",
		Triggers: []automation.Trigger{
			{Platform: "event", EventType: "test.trigger_repeat"},
		},
		Actions: []automation.Action{
			{
				Type: "repeat",
				Repeat: &automation.RepeatConfig{
					Count: 3,
					Sequence: []automation.Action{
						{
							Type:            "service",
							Service:         "toggle",
							TargetEntityKey: "light.living_room",
						},
					},
				},
			},
		},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	require.NoError(t, repos.Automations.Create(ctx, ruleRepeat))

	// 2. Parallel Actions Test
	ruleParallel := &automation.Automation{
		ID:            "test_parallel",
		Name:          "Test Parallel Action",
		Enabled:       true,
		ExecutionMode: "single",
		Triggers: []automation.Trigger{
			{Platform: "event", EventType: "test.trigger_parallel"},
		},
		Actions: []automation.Action{
			{
				Type: "parallel",
				Parallel: []automation.Action{
					{
						Type:            "service",
						Service:         "turn_on",
						TargetEntityKey: "light.bedroom",
					},
					{
						Type:            "service",
						Service:         "turn_on",
						TargetEntityKey: "light.kitchen",
					},
				},
			},
		},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	require.NoError(t, repos.Automations.Create(ctx, ruleParallel))

	require.NoError(t, engine.Start(ctx))

	// Trigger Repeat
	require.NoError(t, bus.Publish(ctx, events.Event{
		ID:         uuid.NewString(),
		Type:       "test.trigger_repeat",
		OccurredAt: time.Now().UTC(),
	}))

	select {
	case <-runCompletedChan:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for repeat run completion")
	}

	for i := 0; i < 50; i++ {
		if atomic.LoadInt32(&commandCount) == 3 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if atomic.LoadInt32(&commandCount) != 3 {
		t.Fatalf("expected 3 commands from repeat, got %d", atomic.LoadInt32(&commandCount))
	}

	// Trigger Parallel
	atomic.StoreInt32(&commandCount, 0)
	require.NoError(t, bus.Publish(ctx, events.Event{
		ID:         uuid.NewString(),
		Type:       "test.trigger_parallel",
		OccurredAt: time.Now().UTC(),
	}))

	select {
	case <-runCompletedChan:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for parallel run completion")
	}

	if atomic.LoadInt32(&commandCount) != 2 {
		t.Fatalf("expected 2 commands from parallel, got %d", atomic.LoadInt32(&commandCount))
	}
}
