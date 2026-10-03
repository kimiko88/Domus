// Package automation_test provides integration tests for the automation engine runner.
package automation_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	appauto "github.com/domus-hub/domus/internal/application/automation"
	"github.com/domus-hub/domus/internal/domain/automation"
	"github.com/domus-hub/domus/internal/domain/device"
	"github.com/domus-hub/domus/internal/domain/entity"
	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/domain/integration"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupEngineTest(t *testing.T) (
	*sqlite.Repositories,
	*eventbus.MemoryBus,
	*appauto.Engine,
	func(),
) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "engine_test.db")

	db, err := sqlite.Open(config.DatabaseConfig{
		Path:          dbPath,
		WALMode:       true,
		BusyTimeoutMs: 5000,
		MaxOpenConns:  1,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, sqlite.Migrate(ctx, db))

	repos := sqlite.NewRepositories(db)
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 64, Workers: 2})

	engine := appauto.NewEngine(repos.Automations, repos.Entities, bus)

	cleanup := func() {
		_ = engine.Stop(context.Background())
		bus.Close()
		db.Close()
	}

	return repos, bus, engine, cleanup
}

func TestEngine_ExecutionFlow(t *testing.T) {
	repos, bus, engine, cleanup := setupEngineTest(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now().UTC()
	in := &integration.Integration{ID: "in1", Type: "test", Name: "Test", Status: "connected", CreatedAt: now, UpdatedAt: now}
	require.NoError(t, repos.Integrations.Create(ctx, in))
	dev := &device.Device{ID: "dev1", IntegrationID: "in1", ExternalID: "d1", Name: "D1", Online: true, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, repos.Devices.Create(ctx, dev))

	// Motion sensor entity
	sensorEnt := &entity.Entity{
		ID:        uuid.NewString(),
		DeviceID:  dev.ID,
		EntityKey: "binary_sensor.hallway_motion",
		Domain:    "binary_sensor",
		Name:      "Hallway Motion",
		State:     `{"state":false}`,
		Available: true,
	}
	require.NoError(t, repos.Entities.Create(ctx, sensorEnt))

	// Light entity
	lightEnt := &entity.Entity{
		ID:        uuid.NewString(),
		DeviceID:  dev.ID,
		EntityKey: "light.hallway",
		Domain:    "light",
		Name:      "Hallway Light",
		Writable:  true,
		State:     `{"state":"OFF"}`,
		Available: true,
	}
	require.NoError(t, repos.Entities.Create(ctx, lightEnt))

	// Create Automation: When motion == true, turn on light
	auto := &automation.Automation{
		ID:      uuid.NewString(),
		Name:    "Motion Light Auto",
		Enabled: true,
		Triggers: []automation.Trigger{
			{
				Platform:  "state",
				EntityKey: "binary_sensor.hallway_motion",
				To:        "true",
			},
		},
		Actions: []automation.Action{
			{
				Type:            "service",
				Service:         "turn_on",
				TargetEntityKey: "light.hallway",
				Params:          map[string]any{"brightness": 150},
			},
		},
		ExecutionMode: "single",
	}
	require.NoError(t, repos.Automations.Create(ctx, auto))

	// Channel to capture CommandRequested on event bus
	cmdChan := make(chan events.Event, 5)
	unsubCmd := bus.Subscribe(events.CommandRequested, func(c context.Context, e events.Event) error {
		cmdChan <- e
		return nil
	})
	defer unsubCmd()

	// Channel to capture AutomationRunCompleted
	runCompletedChan := make(chan events.Event, 5)
	unsubRun := bus.Subscribe(events.AutomationRunCompleted, func(c context.Context, e events.Event) error {
		runCompletedChan <- e
		return nil
	})
	defer unsubRun()

	require.NoError(t, engine.Start(ctx))

	// Trigger by publishing EntityStateChanged
	err := bus.Publish(ctx, events.Event{
		ID:         uuid.NewString(),
		Type:       events.EntityStateChanged,
		OccurredAt: time.Now().UTC(),
		Source:     "test",
		Payload: map[string]any{
			"entity_key": "binary_sensor.hallway_motion",
			"state":      `{"state":true}`,
		},
	})
	require.NoError(t, err)

	// Verify CommandRequested was emitted
	select {
	case cmdEvt := <-cmdChan:
		payload := cmdEvt.Payload.(map[string]any)
		assert.Equal(t, "light.hallway", payload["entity_key"])
		assert.Equal(t, "turn_on", payload["service"])
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for CommandRequested")
	}

	// Verify AutomationRunCompleted was emitted
	select {
	case runEvt := <-runCompletedChan:
		payload := runEvt.Payload.(map[string]any)
		assert.Equal(t, auto.ID, payload["automation_id"])
		assert.Equal(t, "completed", payload["status"])
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for AutomationRunCompleted")
	}

	// Verify run was recorded in SQLite
	runs, err := repos.Automations.ListRuns(ctx, auto.ID, 5)
	require.NoError(t, err)
	require.Len(t, runs, 1)
	assert.Equal(t, automation.RunStatusCompleted, runs[0].Status)
}

func TestEngine_Cooldown(t *testing.T) {
	repos, bus, engine, cleanup := setupEngineTest(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	auto := &automation.Automation{
		ID:              uuid.NewString(),
		Name:            "Cooldown Auto",
		Enabled:         true,
		CooldownSeconds: 60, // 60 seconds cooldown
		Triggers: []automation.Trigger{
			{Platform: "event", EventType: "test.trigger"},
		},
		Actions: []automation.Action{
			{Type: "event", EventType: "test.action_fired"},
		},
		ExecutionMode: "single",
	}
	require.NoError(t, repos.Automations.Create(ctx, auto))

	actionFiredCount := 0
	unsub := bus.Subscribe("test.action_fired", func(c context.Context, e events.Event) error {
		actionFiredCount++
		return nil
	})
	defer unsub()

	require.NoError(t, engine.Start(ctx))

	// First trigger -> should execute
	_ = bus.Publish(ctx, events.Event{Type: "test.trigger", OccurredAt: time.Now()})
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 1, actionFiredCount)

	// Second immediate trigger -> blocked by cooldown
	_ = bus.Publish(ctx, events.Event{Type: "test.trigger", OccurredAt: time.Now()})
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 1, actionFiredCount, "Second execution must be blocked by cooldown")
}
