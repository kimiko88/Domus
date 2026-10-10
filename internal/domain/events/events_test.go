package events

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvent_JSONSerialization(t *testing.T) {
	now := time.Date(2026, 10, 10, 8, 30, 0, 0, time.UTC)
	evt := Event{
		ID:         "evt-12345",
		Type:       EntityStateChanged,
		OccurredAt: now,
		Source:     "zigbee2mqtt",
		Payload: map[string]any{
			"entity_key": "sensor.temperature",
			"old_state":  "20.5",
			"new_state":  "21.0",
		},
	}

	data, err := json.Marshal(evt)
	require.NoError(t, err)

	var unmarshaled Event
	require.NoError(t, json.Unmarshal(data, &unmarshaled))

	assert.Equal(t, evt.ID, unmarshaled.ID)
	assert.Equal(t, EntityStateChanged, unmarshaled.Type)
	assert.Equal(t, "zigbee2mqtt", unmarshaled.Source)
	assert.True(t, evt.OccurredAt.Equal(unmarshaled.OccurredAt))

	payloadMap, ok := unmarshaled.Payload.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "sensor.temperature", payloadMap["entity_key"])
	assert.Equal(t, "21.0", payloadMap["new_state"])
}

func TestEventTypes_Completeness(t *testing.T) {
	types := []EventType{
		SystemStarted,
		DeviceDiscovered,
		DeviceOnline,
		DeviceOffline,
		IntegrationStateChanged,
		EntityStateChanged,
		CommandRequested,
		CommandSucceeded,
		CommandFailed,
		AutomationTriggered,
		AutomationRunCompleted,
		AutomationRunFailed,
		SceneActivated,
		NotificationCreated,
	}

	for _, tp := range types {
		assert.NotEmpty(t, string(tp))
	}
}
