// Package mqtt_generic_test provides unit and integration tests for the generic MQTT adapter.
package mqtt_generic_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/domain/device"
	"github.com/domus-hub/domus/internal/domain/entity"
	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/domain/integration"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/domus-hub/domus/internal/infrastructure/mqtt"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	"github.com/domus-hub/domus/internal/plugins/mqtt_generic"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMQTTGenericAdapter_IncomingStateUpdatesEntityAndEmitsEvent(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "mqtt_adapter_test.db")

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

	mockClient := mqtt.NewMockClient()

	// 1. Setup Integration, Device, and Entity
	now := time.Now().UTC()
	in := &integration.Integration{
		ID:        "mqtt_main",
		Type:      "mqtt_generic",
		Name:      "MQTT Generic",
		Enabled:   true,
		Status:    "connected",
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, repos.Integrations.Create(ctx, in))

	dev := &device.Device{
		ID:            uuid.NewString(),
		IntegrationID: in.ID,
		ExternalID:    "switch_living_room",
		Name:          "Living Room Switch",
		Online:        true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	require.NoError(t, repos.Devices.Create(ctx, dev))

	ent := &entity.Entity{
		ID:            uuid.NewString(),
		DeviceID:      dev.ID,
		EntityKey:     "switch.living_room",
		Domain:        "switch",
		Name:          "Living Room Light Switch",
		Writable:      true,
		State:         `{"state":"OFF"}`,
		Available:     true,
		LastChangedAt: now,
		LastUpdatedAt: now,
	}
	require.NoError(t, repos.Entities.Create(ctx, ent))

	// 2. Setup MQTT Generic Adapter
	adapter := mqtt_generic.NewAdapter(mqtt_generic.Config{
		IntegrationID: in.ID,
		StateTopicPattern: "domus/+/state", // e.g. domus/switch.living_room/state
		CommandTopicTemplate: "domus/%s/set",
	}, mockClient, repos.Entities, repos.Devices, bus)

	require.NoError(t, adapter.Start(ctx))
	defer adapter.Stop(ctx)

	// Subscribe to event bus for EntityStateChanged
	var stateChangedEvt events.Event
	evtReceived := make(chan struct{}, 1)
	unsub := bus.Subscribe(events.EntityStateChanged, func(c context.Context, e events.Event) error {
		stateChangedEvt = e
		evtReceived <- struct{}{}
		return nil
	})
	defer unsub()

	// 3. Simulate incoming MQTT message
	mockClient.SimulateIncoming("domus/switch.living_room/state", []byte(`{"state":"ON"}`))

	select {
	case <-evtReceived:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for EntityStateChanged event")
	}

	// Verify domain event
	assert.Equal(t, events.EntityStateChanged, stateChangedEvt.Type)
	assert.Equal(t, "switch.living_room", stateChangedEvt.Payload.(map[string]any)["entity_key"])

	// Verify database was updated
	updatedEnt, err := repos.Entities.GetByID(ctx, ent.ID)
	require.NoError(t, err)
	assert.Equal(t, `{"state":"ON"}`, updatedEnt.State)
}

func TestMQTTGenericAdapter_OutboundCommandPublishesToMQTT(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "mqtt_adapter_cmd_test.db")

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

	mockClient := mqtt.NewMockClient()

	adapter := mqtt_generic.NewAdapter(mqtt_generic.Config{
		IntegrationID: "mqtt_main",
		StateTopicPattern: "domus/+/state",
		CommandTopicTemplate: "domus/%s/set",
	}, mockClient, repos.Entities, repos.Devices, bus)

	require.NoError(t, adapter.Start(ctx))
	defer adapter.Stop(ctx)

	// Emit CommandRequested event on bus
	err = bus.Publish(ctx, events.Event{
		ID:         uuid.NewString(),
		Type:       events.CommandRequested,
		OccurredAt: time.Now(),
		Source:     "user:admin",
		Payload: map[string]any{
			"entity_key": "light.desk_lamp",
			"service":    "turn_on",
			"params":     map[string]any{"brightness": 180},
		},
	})
	require.NoError(t, err)

	// Allow event loop to process
	time.Sleep(100 * time.Millisecond)

	published := mockClient.GetPublished()
	require.NotEmpty(t, published, "Adapter must publish to MQTT when CommandRequested is fired")
	assert.Equal(t, "domus/light.desk_lamp/set", published[0].Topic)
	assert.Contains(t, string(published[0].Payload), "turn_on")
}
