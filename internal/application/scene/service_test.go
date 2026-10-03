// Package scene_test provides unit tests for the scene application service.
package scene_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	appscene "github.com/domus-hub/domus/internal/application/scene"
	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/domain/scene"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSceneService_ApplyScene(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "scene_svc_test.db")

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

	repos := sqlite.NewRepositories(db)
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 64, Workers: 2})
	defer bus.Close()

	service := appscene.NewService(repos.Scenes, bus)

	// Create Scene
	sc := &scene.Scene{
		ID:   uuid.NewString(),
		Name: "Dinner Scene",
		Icon: "utensils",
		Actions: []scene.Action{
			{
				EntityKey: "light.dining_room",
				Service:   "turn_on",
				Params:    map[string]any{"brightness": 120},
			},
			{
				EntityKey: "switch.background_music",
				Service:   "turn_on",
			},
		},
	}
	require.NoError(t, repos.Scenes.Create(ctx, sc))

	// Listen for CommandRequested and SceneActivated
	cmdChan := make(chan events.Event, 5)
	unsubCmd := bus.Subscribe(events.CommandRequested, func(c context.Context, e events.Event) error {
		cmdChan <- e
		return nil
	})
	defer unsubCmd()

	activatedChan := make(chan events.Event, 5)
	unsubAct := bus.Subscribe(events.SceneActivated, func(c context.Context, e events.Event) error {
		activatedChan <- e
		return nil
	})
	defer unsubAct()

	// Apply scene
	require.NoError(t, service.ApplyScene(ctx, sc.ID))

	// Verify 2 command requested events
	var receivedEntities []string
	for i := 0; i < 2; i++ {
		select {
		case cmdEvt := <-cmdChan:
			payload := cmdEvt.Payload.(map[string]any)
			receivedEntities = append(receivedEntities, payload["entity_key"].(string))
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for command requested event")
		}
	}
	assert.Contains(t, receivedEntities, "light.dining_room")
	assert.Contains(t, receivedEntities, "switch.background_music")

	// Verify scene activated event
	select {
	case actEvt := <-activatedChan:
		payload := actEvt.Payload.(map[string]any)
		assert.Equal(t, sc.ID, payload["scene_id"])
		assert.Equal(t, sc.Name, payload["name"])
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for scene activated event")
	}

	// Non-existent scene returns error
	err = service.ApplyScene(ctx, "non-existent")
	assert.ErrorIs(t, err, scene.ErrSceneNotFound)
}
