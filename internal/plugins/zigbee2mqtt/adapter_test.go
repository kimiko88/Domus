// Package zigbee2mqtt_test contains integration tests for the Zigbee2MQTT adapter.
package zigbee2mqtt_test

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
	"github.com/domus-hub/domus/internal/plugins/zigbee2mqtt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestZ2M(t *testing.T) (
	*sqlite.Repositories,
	*eventbus.MemoryBus,
	*mqtt.MockClient,
	*zigbee2mqtt.Adapter,
	func(),
) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "z2m_adapter_test.db")

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
		ID:        "z2m_main",
		Type:      "zigbee2mqtt",
		Name:      "Zigbee2MQTT Hub",
		Enabled:   true,
		Status:    "connected",
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, repos.Integrations.Create(ctx, in))

	adapter := zigbee2mqtt.NewAdapter(zigbee2mqtt.Config{
		IntegrationID: in.ID,
		BaseTopic:     "zigbee2mqtt",
	}, mockClient, repos.Devices, repos.Entities, bus)

	cleanup := func() {
		_ = adapter.Stop(context.Background())
		bus.Close()
		db.Close()
	}

	return repos, bus, mockClient, adapter, cleanup
}

func TestAdapter_AutoDiscoveryFromBridgeDevices(t *testing.T) {
	repos, bus, mockClient, adapter, cleanup := setupTestZ2M(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Track DeviceDiscovered events on the bus
	discoveredChan := make(chan events.Event, 10)
	unsub := bus.Subscribe(events.DeviceDiscovered, func(c context.Context, e events.Event) error {
		discoveredChan <- e
		return nil
	})
	defer unsub()

	require.NoError(t, adapter.Start(ctx))

	bridgeDevicesJSON := `[
		{
			"ieee_address": "0x00124b0018e21a22",
			"type": "Coordinator",
			"friendly_name": "Coordinator"
		},
		{
			"ieee_address": "0x00158d0001a2b3c4",
			"type": "Router",
			"friendly_name": "living_room_light",
			"definition": {
				"model": "LED1624G9",
				"vendor": "IKEA",
				"description": "TRADFRI LED bulb E27 1000 lumen",
				"exposes": [
					{
						"type": "light",
						"features": [
							{"name": "state", "property": "state", "type": "binary"},
							{"name": "brightness", "property": "brightness", "type": "numeric"}
						]
					}
				]
			}
		}
	]`

	// Simulate incoming message on zigbee2mqtt/bridge/devices
	mockClient.SimulateIncoming("zigbee2mqtt/bridge/devices", []byte(bridgeDevicesJSON))

	// Verify DeviceDiscovered event emitted
	select {
	case evt := <-discoveredChan:
		assert.Equal(t, events.DeviceDiscovered, evt.Type)
		payload := evt.Payload.(map[string]any)
		assert.Equal(t, "living_room_light", payload["name"])
		assert.Equal(t, "0x00158d0001a2b3c4", payload["external_id"])
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for DeviceDiscovered event")
	}

	// Verify device in database
	dev, err := repos.Devices.GetByExternalID(ctx, "z2m_main", "0x00158d0001a2b3c4")
	require.NoError(t, err)
	assert.Equal(t, "living_room_light", dev.Name)
	assert.Equal(t, "IKEA", dev.Manufacturer)
	assert.Equal(t, "LED1624G9", dev.Model)

	// Verify entity in database
	ent, err := repos.Entities.GetByKey(ctx, "light.living_room_light")
	require.NoError(t, err)
	assert.Equal(t, "light", ent.Domain)
	assert.True(t, ent.Writable)
}

func TestAdapter_IncomingTelemetryUpdatesEntitiesAndEmitsEvent(t *testing.T) {
	repos, bus, mockClient, adapter, cleanup := setupTestZ2M(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now().UTC()
	// Pre-create device and entity
	dev := &device.Device{
		ID:            uuid.NewString(),
		IntegrationID: "z2m_main",
		ExternalID:    "0x00158d0002b3c4d5",
		Name:          "bedroom_sensor",
		Online:        true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	require.NoError(t, repos.Devices.Create(ctx, dev))

	entTemp := &entity.Entity{
		ID:                uuid.NewString(),
		DeviceID:          dev.ID,
		EntityKey:         "sensor.bedroom_sensor_temperature",
		Domain:            "sensor",
		Name:              "bedroom_sensor temperature",
		UnitOfMeasurement: "°C",
		State:             `{"value":null}`,
		Available:         true,
		LastChangedAt:     now,
		LastUpdatedAt:     now,
	}
	require.NoError(t, repos.Entities.Create(ctx, entTemp))

	stateChangedChan := make(chan events.Event, 5)
	unsub := bus.Subscribe(events.EntityStateChanged, func(c context.Context, e events.Event) error {
		stateChangedChan <- e
		return nil
	})
	defer unsub()

	require.NoError(t, adapter.Start(ctx))

	// Publish telemetry
	mockClient.SimulateIncoming("zigbee2mqtt/bedroom_sensor", []byte(`{"temperature": 21.4, "humidity": 55.0, "linkquality": 88}`))

	select {
	case evt := <-stateChangedChan:
		payload := evt.Payload.(map[string]any)
		assert.Equal(t, "sensor.bedroom_sensor_temperature", payload["entity_key"])
		assert.Equal(t, `{"value":21.4}`, payload["state"])
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for EntityStateChanged event")
	}

	// Verify database record
	updatedEnt, err := repos.Entities.GetByKey(ctx, "sensor.bedroom_sensor_temperature")
	require.NoError(t, err)
	assert.Equal(t, `{"value":21.4}`, updatedEnt.State)
	assert.Contains(t, updatedEnt.Attributes, "linkquality")
}

func TestAdapter_OutboundCommandPublishesToMQTT(t *testing.T) {
	repos, bus, mockClient, adapter, cleanup := setupTestZ2M(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now().UTC()
	dev := &device.Device{
		ID:            uuid.NewString(),
		IntegrationID: "z2m_main",
		ExternalID:    "0x00158d0001a2b3c4",
		Name:          "living_room_light",
		Online:        true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	require.NoError(t, repos.Devices.Create(ctx, dev))

	ent := &entity.Entity{
		ID:            uuid.NewString(),
		DeviceID:      dev.ID,
		EntityKey:     "light.living_room_light",
		Domain:        "light",
		Name:          "living_room_light",
		Writable:      true,
		State:         `{"state":"OFF"}`,
		Available:     true,
		LastChangedAt: now,
		LastUpdatedAt: now,
	}
	require.NoError(t, repos.Entities.Create(ctx, ent))

	require.NoError(t, adapter.Start(ctx))

	// Emit CommandRequested on event bus
	err := bus.Publish(ctx, events.Event{
		ID:         uuid.NewString(),
		Type:       events.CommandRequested,
		OccurredAt: time.Now(),
		Source:     "user:admin",
		Payload: map[string]any{
			"entity_key": "light.living_room_light",
			"service":    "turn_on",
			"params":     map[string]any{"brightness": 190},
		},
	})
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	published := mockClient.GetPublished()
	require.NotEmpty(t, published)

	var setMsg *mqtt.PublishedMessage
	for _, p := range published {
		if p.Topic == "zigbee2mqtt/living_room_light/set" {
			setMsg = &p
			break
		}
	}
	require.NotNil(t, setMsg, "Adapter must publish to zigbee2mqtt/living_room_light/set")
	assert.Contains(t, string(setMsg.Payload), `"state":"ON"`)
	assert.Contains(t, string(setMsg.Payload), `"brightness":190`)
}

func TestAdapter_BridgeAvailabilityState(t *testing.T) {
	_, bus, mockClient, adapter, cleanup := setupTestZ2M(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	integrationChan := make(chan events.Event, 5)
	unsub := bus.Subscribe(events.IntegrationStateChanged, func(c context.Context, e events.Event) error {
		integrationChan <- e
		return nil
	})
	defer unsub()

	require.NoError(t, adapter.Start(ctx))

	// Publish offline bridge state
	mockClient.SimulateIncoming("zigbee2mqtt/bridge/state", []byte(`offline`))

	select {
	case evt := <-integrationChan:
		payload := evt.Payload.(map[string]any)
		assert.Equal(t, "disconnected", payload["status"])
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for IntegrationStateChanged event")
	}
}
