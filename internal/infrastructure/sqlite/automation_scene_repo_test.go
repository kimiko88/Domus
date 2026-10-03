// Package sqlite_test tests SQLite repositories for automations, runs, and scenes.
package sqlite_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/domain/automation"
	"github.com/domus-hub/domus/internal/domain/scene"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAutomationRepository_CRUD_And_Runs(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "automations_test.db")

	db, err := sqlite.Open(config.DatabaseConfig{
		Path:          dbPath,
		WALMode:       true,
		BusyTimeoutMs: 5000,
		MaxOpenConns:  1,
	})
	require.NoError(t, err)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, sqlite.Migrate(ctx, db))

	repo := sqlite.NewAutomationRepository(db)

	now := time.Now().UTC()
	auto := &automation.Automation{
		ID:          uuid.NewString(),
		Name:        "Motion Nightlight",
		Description: "Turn on hallway light on motion",
		Enabled:     true,
		Triggers: []automation.Trigger{
			{
				Platform:  "state",
				EntityKey: "binary_sensor.motion",
				To:        "ON",
			},
		},
		Conditions: []automation.Condition{
			{
				Condition: "state",
				EntityKey: "sun.sun",
				State:     "below_horizon",
			},
		},
		Actions: []automation.Action{
			{
				Type:            "service",
				Service:         "turn_on",
				TargetEntityKey: "light.hallway",
				Params:          map[string]any{"brightness": 128},
			},
		},
		ExecutionMode:   "single",
		CooldownSeconds: 30,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	// 1. Create
	require.NoError(t, repo.Create(ctx, auto))

	// 2. GetByID
	fetched, err := repo.GetByID(ctx, auto.ID)
	require.NoError(t, err)
	assert.Equal(t, auto.Name, fetched.Name)
	assert.Equal(t, auto.Description, fetched.Description)
	assert.True(t, fetched.Enabled)
	require.Len(t, fetched.Triggers, 1)
	assert.Equal(t, "state", fetched.Triggers[0].Platform)
	assert.Equal(t, "binary_sensor.motion", fetched.Triggers[0].EntityKey)
	require.Len(t, fetched.Conditions, 1)
	assert.Equal(t, "below_horizon", fetched.Conditions[0].State)
	require.Len(t, fetched.Actions, 1)
	assert.Equal(t, "turn_on", fetched.Actions[0].Service)

	// 3. List
	enabledTrue := true
	list, err := repo.List(ctx, automation.ListFilter{Enabled: &enabledTrue})
	require.NoError(t, err)
	assert.Len(t, list, 1)

	// 4. Update
	fetched.Name = "Updated Motion Nightlight"
	fetched.Enabled = false
	require.NoError(t, repo.Update(ctx, fetched))

	updated, err := repo.GetByID(ctx, auto.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Motion Nightlight", updated.Name)
	assert.False(t, updated.Enabled)

	// 5. UpdateLastRun
	require.NoError(t, repo.UpdateLastRun(ctx, auto.ID, "completed"))
	withRun, err := repo.GetByID(ctx, auto.ID)
	require.NoError(t, err)
	assert.NotNil(t, withRun.LastRunAt)
	assert.Equal(t, "completed", withRun.LastStatus)

	// 6. Runs history
	run := &automation.AutomationRun{
		ID:             uuid.NewString(),
		AutomationID:   auto.ID,
		StartedAt:      time.Now().UTC(),
		Status:         automation.RunStatusRunning,
		TriggerContext: `{"entity_key":"binary_sensor.motion"}`,
	}
	require.NoError(t, repo.CreateRun(ctx, run))

	finishedAt := time.Now().UTC()
	run.Complete(finishedAt, "Executed turn_on")
	require.NoError(t, repo.UpdateRun(ctx, run))

	runs, err := repo.ListRuns(ctx, auto.ID, 10)
	require.NoError(t, err)
	require.Len(t, runs, 1)
	assert.Equal(t, automation.RunStatusCompleted, runs[0].Status)
	assert.Equal(t, "Executed turn_on", runs[0].Log)

	// 7. Delete (cascades runs)
	require.NoError(t, repo.Delete(ctx, auto.ID))
	_, err = repo.GetByID(ctx, auto.ID)
	assert.ErrorIs(t, err, automation.ErrAutomationNotFound)

	runsAfterDel, err := repo.ListRuns(ctx, auto.ID, 10)
	require.NoError(t, err)
	assert.Empty(t, runsAfterDel)
}

func TestSceneRepository_CRUD(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "scenes_test.db")

	db, err := sqlite.Open(config.DatabaseConfig{
		Path:          dbPath,
		WALMode:       true,
		BusyTimeoutMs: 5000,
		MaxOpenConns:  1,
	})
	require.NoError(t, err)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, sqlite.Migrate(ctx, db))

	repo := sqlite.NewSceneRepository(db)

	now := time.Now().UTC()
	sc := &scene.Scene{
		ID:   uuid.NewString(),
		Name: "Relax Evening",
		Icon: "sofa",
		Actions: []scene.Action{
			{
				EntityKey: "light.living_room",
				Service:   "turn_on",
				Params:    map[string]any{"brightness": 50},
			},
			{
				EntityKey: "switch.ambient_lamp",
				Service:   "turn_on",
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	// 1. Create
	require.NoError(t, repo.Create(ctx, sc))

	// 2. GetByID
	fetched, err := repo.GetByID(ctx, sc.ID)
	require.NoError(t, err)
	assert.Equal(t, sc.Name, fetched.Name)
	assert.Equal(t, "sofa", fetched.Icon)
	require.Len(t, fetched.Actions, 2)
	assert.Equal(t, "light.living_room", fetched.Actions[0].EntityKey)

	// 3. List
	all, err := repo.List(ctx)
	require.NoError(t, err)
	assert.Len(t, all, 1)

	// 4. Update
	fetched.Name = "Relax Evening Dimmed"
	require.NoError(t, repo.Update(ctx, fetched))

	updated, err := repo.GetByID(ctx, sc.ID)
	require.NoError(t, err)
	assert.Equal(t, "Relax Evening Dimmed", updated.Name)

	// 5. Delete
	require.NoError(t, repo.Delete(ctx, sc.ID))
	_, err = repo.GetByID(ctx, sc.ID)
	assert.ErrorIs(t, err, scene.ErrSceneNotFound)
}
