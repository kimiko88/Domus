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

func TestHomeKit_SetupURIUsesRandomPairingCode(t *testing.T) {
	cfg := DefaultBridgeConfig()
	assert.Regexp(t, "^[0-9]{3}-[0-9]{2}-[0-9]{3}$", cfg.SetupCode)
	assert.Contains(t, cfg.SetupURI, "X-HM://")
}

func TestHomeKit_BridgeFailsClosedButMappingRemainsTestable(t *testing.T) {
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 64, Workers: 1})
	defer bus.Close()
	bridge := NewBridge(DefaultBridgeConfig(), bus)
	ctx := context.Background()
	require.ErrorIs(t, bridge.Start(ctx), ErrHomeKitTransportUnavailable)
	defer bridge.Stop(ctx)

	assert.Equal(t, "homekit", bridge.ID())
	assert.Equal(t, "homekit", bridge.Type())
	accLight := bridge.RegisterEntity("light.kitchen", "light", "Luce Cucina")
	require.NotNil(t, accLight)
	assert.Equal(t, uint64(2), accLight.AID)

	accClimate := bridge.RegisterEntity("climate.living_room", "climate", "Clima Salotto")
	require.NotNil(t, accClimate)
	assert.Equal(t, uint64(3), accClimate.AID)

	cmdCh := make(chan events.Event, 2)
	unsub := bus.Subscribe(events.CommandRequested, func(_ context.Context, evt events.Event) error {
		cmdCh <- evt
		return nil
	})
	defer unsub()

	require.NoError(t, bridge.HandleHomeKitWrite(ctx, 2, 11, true))
	select {
	case cmdEvt := <-cmdCh:
		payload := cmdEvt.Payload.(map[string]any)
		assert.Equal(t, "light.kitchen", payload["entity_key"])
		assert.Equal(t, "turn_on", payload["command"])
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for the HomeKit mapping command")
	}

	require.NoError(t, bridge.handleEntityStateChanged(ctx, events.Event{
		ID: "st-1", Type: events.EntityStateChanged, OccurredAt: time.Now(),
		Payload: map[string]any{"entity_key": "light.kitchen", "state": "on"},
	}))
	assert.True(t, accLight.Services[1].Characteristics[0].Value.(bool))
}
