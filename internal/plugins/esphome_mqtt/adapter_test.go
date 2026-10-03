// Package esphome_mqtt_test contains integration tests for the ESPHome MQTT adapter.
package esphome_mqtt_test

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
	"github.com/domus-hub/domus/internal/plugins/esphome_mqtt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestESPHome(t *testing.T) (
	*sqlite.Repositories,
	*eventbus.MemoryBus,
	*mqtt.MockClient,
	*esphome_mqtt.Adapter,
	func(),
) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "esphome_adapter_test.db")

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
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 128, Workers: 2})
	mockClient := mqtt.NewMockClient()

	now := time.Now().UTC()
	in := &integration.Integration{
		ID:        "esphome_main",
		Type:      "esphome_mqtt",
		Name:      "ESPHome MQTT Hub",
		Enabled:   true,
		Status:    "connected",
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, repos.Integrations.Create(ctx, in))

	adapter := esphome_mqtt.NewAdapter(esphome_mqtt.Config{
		IntegrationID: in.ID,
		TopicPrefix:   "", // Root-level topics: <node>/<domain>/<id>/state
	}, mockClient, repos.Devices, repos.Entities, bus)

	cleanup := func() {
		_ = adapter.Stop(context.Background())
		bus.Close()
		db.Close()
	}

	return repos, bus, mockClient, adapter, cleanup
}

func TestAdapter_AutoDiscoveryAndTelemetryUpdates(t *testing.T) {
	repos, bus, mockClient, adapter, cleanup := setupTestESPHome(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stateChangedChan := make(chan events.Event, 5)
	unsub := bus.Subscribe(events.EntityStateChanged, func(c context.Context, e events.Event) error {
		stateChangedChan <- e
		return nil
	})
	defer unsub()

	require.NoError(t, adapter.Start(ctx))

	// 1. Simulate temperature sensor state
	mockClient.SimulateIncoming("living_esp/sensor/temperature/state", []byte("22.4"))

	select {
	case evt := <-stateChangedChan:
		payload := evt.Payload.(map[string]any)
		assert.Equal(t, "sensor.living_esp_temperature", payload["entity_key"])
		assert.Equal(t, `{"value":22.4}`, payload["state"])
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for EntityStateChanged event")
	}

	// Verify device auto-created in database
	dev, err := repos.Devices.GetByExternalID(ctx, "esphome_main", "living_esp")
	require.NoError(t, err)
	assert.Equal(t, "living_esp", dev.Name)
	assert.Equal(t, "ESPHome", dev.Manufacturer)
	assert.True(t, dev.Online)

	// Verify entity created in database
	ent, err := repos.Entities.GetByKey(ctx, "sensor.living_esp_temperature")
	require.NoError(t, err)
	assert.Equal(t, "sensor", ent.Domain)
	assert.Equal(t, `{"value":22.4}`, ent.State)
	assert.Equal(t, "°C", ent.UnitOfMeasurement)
	assert.False(t, ent.Writable)

	// 2. Simulate switch state update
	mockClient.SimulateIncoming("living_esp/switch/relay_1/state", []byte("ON"))

	select {
	case evt := <-stateChangedChan:
		payload := evt.Payload.(map[string]any)
		assert.Equal(t, "switch.living_esp_relay_1", payload["entity_key"])
		assert.Equal(t, `{"state":"ON"}`, payload["state"])
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for EntityStateChanged event")
	}

	switchEnt, err := repos.Entities.GetByKey(ctx, "switch.living_esp_relay_1")
	require.NoError(t, err)
	assert.Equal(t, "switch", switchEnt.Domain)
	assert.True(t, switchEnt.Writable)
	assert.Equal(t, `{"state":"ON"}`, switchEnt.State)
}

func TestAdapter_NodeStatusOfflineAndOnline(t *testing.T) {
	repos, bus, mockClient, adapter, cleanup := setupTestESPHome(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Pre-create node
	now := time.Now().UTC()
	dev := &device.Device{
		ID:            uuid.NewString(),
		IntegrationID: "esphome_main",
		ExternalID:    "garden_node",
		Name:          "garden_node",
		Online:        true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	require.NoError(t, repos.Devices.Create(ctx, dev))

	offlineChan := make(chan events.Event, 5)
	unsubOffline := bus.Subscribe(events.DeviceOffline, func(c context.Context, e events.Event) error {
		offlineChan <- e
		return nil
	})
	defer unsubOffline()

	onlineChan := make(chan events.Event, 5)
	unsubOnline := bus.Subscribe(events.DeviceOnline, func(c context.Context, e events.Event) error {
		onlineChan <- e
		return nil
	})
	defer unsubOnline()

	require.NoError(t, adapter.Start(ctx))

	// 1. Send status offline
	mockClient.SimulateIncoming("garden_node/status", []byte("offline"))

	select {
	case evt := <-offlineChan:
		payload := evt.Payload.(map[string]any)
		assert.Equal(t, "garden_node", payload["name"])
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for DeviceOffline event")
	}

	d, err := repos.Devices.GetByID(ctx, dev.ID)
	require.NoError(t, err)
	assert.False(t, d.Online)

	// 2. Send status online
	mockClient.SimulateIncoming("garden_node/status", []byte("online"))

	select {
	case evt := <-onlineChan:
		payload := evt.Payload.(map[string]any)
		assert.Equal(t, "garden_node", payload["name"])
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for DeviceOnline event")
	}

	d, err = repos.Devices.GetByID(ctx, dev.ID)
	require.NoError(t, err)
	assert.True(t, d.Online)
}

func TestAdapter_OutboundCommandPublishesToMQTT(t *testing.T) {
	repos, bus, mockClient, adapter, cleanup := setupTestESPHome(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now().UTC()
	dev := &device.Device{
		ID:            uuid.NewString(),
		IntegrationID: "esphome_main",
		ExternalID:    "living_esp",
		Name:          "living_esp",
		Online:        true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	require.NoError(t, repos.Devices.Create(ctx, dev))

	ent := &entity.Entity{
		ID:            uuid.NewString(),
		DeviceID:      dev.ID,
		EntityKey:     "switch.living_esp_fan",
		Domain:        "switch",
		Name:          "living_esp fan",
		Writable:      true,
		State:         `{"state":"OFF"}`,
		Available:     true,
		LastChangedAt: now,
		LastUpdatedAt: now,
	}
	require.NoError(t, repos.Entities.Create(ctx, ent))

	require.NoError(t, adapter.Start(ctx))

	// Publish CommandRequested on bus
	err := bus.Publish(ctx, events.Event{
		ID:         uuid.NewString(),
		Type:       events.CommandRequested,
		OccurredAt: time.Now(),
		Source:     "user:admin",
		Payload: map[string]any{
			"entity_key": "switch.living_esp_fan",
			"service":    "turn_on",
		},
	})
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	published := mockClient.GetPublished()
	require.NotEmpty(t, published)

	var cmdMsg *mqtt.PublishedMessage
	for _, p := range published {
		if p.Topic == "living_esp/switch/fan/command" {
			cmdMsg = &p
			break
		}
	}
	require.NotNil(t, cmdMsg, "Adapter must publish command to living_esp/switch/fan/command")
	assert.Equal(t, "ON", string(cmdMsg.Payload))
}
