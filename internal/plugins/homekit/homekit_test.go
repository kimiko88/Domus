package homekit

import (
	"context"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHomeKit_SetupURIAndPIN(t *testing.T) {
	pin := "031-45-154"
	uri := GenerateSetupURI(pin, "DOMU", 2)
	assert.True(t, len(uri) > 10)
	assert.Contains(t, uri, "X-HM://")

	randCode := RandomPairingCode()
	assert.Len(t, randCode, 10) // XXX-XX-XXX
}

func TestHomeKit_BridgeLifecycleAndCommands(t *testing.T) {
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 64, Workers: 1})
	defer bus.Close()

	bridge := NewBridge(DefaultBridgeConfig(), bus)
	ctx := context.Background()
	require.NoError(t, bridge.Start(ctx))
	defer bridge.Stop(ctx)

	assert.Equal(t, "homekit", bridge.ID())
	assert.Equal(t, "homekit", bridge.Type())

	// Expose Light & Thermostat to Apple Home
	accLight := bridge.RegisterEntity("light.kitchen", "light", "Luce Cucina")
	require.NotNil(t, accLight)
	assert.Equal(t, uint64(2), accLight.AID)

	accClimate := bridge.RegisterEntity("climate.living_room", "climate", "Clima Salotto")
	require.NotNil(t, accClimate)
	assert.Equal(t, uint64(3), accClimate.AID)

	// Listen for CommandRequested dispatched by iOS
	cmdCh := make(chan events.Event, 2)
	unsub := bus.Subscribe(events.CommandRequested, func(c context.Context, e events.Event) error {
		cmdCh <- e
		return nil
	})
	defer unsub()

	// Simulate Apple Home user tapping "Turn On" on Light (AID 2, IID 11 = CharOn)
	err := bridge.HandleHomeKitWrite(ctx, 2, 11, true)
	require.NoError(t, err)

	select {
	case cmdEvt := <-cmdCh:
		payload := cmdEvt.Payload.(map[string]any)
		assert.Equal(t, "light.kitchen", payload["entity_key"])
		assert.Equal(t, "turn_on", payload["command"])
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for command requested from HomeKit")
	}

	// Simulate Domus entity state changing to "on"
	err = bridge.handleEntityStateChanged(ctx, events.Event{
		ID:         "st-1",
		Type:       events.EntityStateChanged,
		OccurredAt: time.Now(),
		Payload: map[string]any{
			"entity_key": "light.kitchen",
			"state":      "on",
		},
	})
	require.NoError(t, err)
	assert.True(t, accLight.Services[1].Characteristics[0].Value.(bool))
}
