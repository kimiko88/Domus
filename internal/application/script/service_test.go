package script_test

import (
	"context"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	appscript "github.com/domus-hub/domus/internal/application/script"
	"github.com/domus-hub/domus/internal/domain/automation"
	"github.com/domus-hub/domus/internal/domain/events"
	domscript "github.com/domus-hub/domus/internal/domain/script"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupScriptTest(t *testing.T) (*appscript.Service, *eventbus.MemoryBus, func()) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "script_test.db")

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

	repo := sqlite.NewScriptRepository(db)
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 64, Workers: 2})
	svc := appscript.NewService(repo, bus)

	cleanup := func() {
		bus.Close()
		db.Close()
	}

	return svc, bus, cleanup
}

func TestScriptExecutionAndVariableSubstitution(t *testing.T) {
	svc, bus, cleanup := setupScriptTest(t)
	defer cleanup()

	ctx := context.Background()

	var receivedEntityKey string
	var commandCount int32
	bus.Subscribe(events.CommandRequested, func(c context.Context, e events.Event) error {
		atomic.AddInt32(&commandCount, 1)
		payload := e.Payload.(map[string]any)
		receivedEntityKey = payload["entity_key"].(string)
		return nil
	})

	sc := &domscript.Script{
		ID:          "script.dim_room",
		Name:        "Dim Room",
		Description: "Dims specified light",
		Mode:        domscript.ModeSingle,
		Sequence: []automation.Action{
			{
				Type:            "service",
				Service:         "turn_on",
				TargetEntityKey: "{target_light}",
				Params:          map[string]any{"brightness": 120},
			},
		},
	}

	require.NoError(t, svc.Create(ctx, sc))

	// Execute with variables
	err := svc.Execute(ctx, "script.dim_room", map[string]any{
		"target_light": "light.bedroom",
	})
	require.NoError(t, err)

	for i := 0; i < 50; i++ {
		if atomic.LoadInt32(&commandCount) == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	assert.Equal(t, int32(1), atomic.LoadInt32(&commandCount))
	assert.Equal(t, "light.bedroom", receivedEntityKey)
}

func TestScriptSingleModeConcurrency(t *testing.T) {
	svc, _, cleanup := setupScriptTest(t)
	defer cleanup()

	ctx := context.Background()

	sc := &domscript.Script{
		ID:   "script.slow",
		Name: "Slow Script",
		Mode: domscript.ModeSingle,
		Sequence: []automation.Action{
			{
				Type:    "delay",
				DelayMs: 300,
			},
		},
	}
	require.NoError(t, svc.Create(ctx, sc))

	// Run first in background
	go func() {
		_ = svc.Execute(ctx, "script.slow", nil)
	}()

	time.Sleep(50 * time.Millisecond)

	// Second concurrent execution should fail with ErrScriptAlreadyRunning
	err := svc.Execute(ctx, "script.slow", nil)
	assert.ErrorIs(t, err, domscript.ErrScriptAlreadyRunning)
}
