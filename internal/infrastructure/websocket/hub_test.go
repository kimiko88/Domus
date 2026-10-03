// Package websocket_test tests the WebSocket broadcast hub (TDD red-to-green phase).
package websocket_test

import (
	"context"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/domus-hub/domus/internal/infrastructure/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebSocketHub_BroadcastToClients(t *testing.T) {
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 64, Workers: 2})
	defer bus.Close()

	hub := websocket.NewHub(bus)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)

	// Register 2 test client channels
	clientCh1 := make(chan websocket.Message, 10)
	clientCh2 := make(chan websocket.Message, 10)

	c1 := hub.Register("client-1", clientCh1)
	defer hub.Unregister(c1)

	c2 := hub.Register("client-2", clientCh2)
	defer hub.Unregister(c2)

	assert.Equal(t, 2, hub.ClientCount())

	// Broadcast message
	hub.Broadcast(websocket.Message{
		Type:    "system.ping",
		Payload: map[string]any{"time": time.Now().Unix()},
	})

	select {
	case msg := <-clientCh1:
		assert.Equal(t, "system.ping", msg.Type)
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for client 1 message")
	}

	select {
	case msg := <-clientCh2:
		assert.Equal(t, "system.ping", msg.Type)
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for client 2 message")
	}

	// Unregister client 1
	hub.Unregister(c1)
	assert.Equal(t, 1, hub.ClientCount())
}

func TestWebSocketHub_ForwardsEventBusEvents(t *testing.T) {
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 64, Workers: 2})
	defer bus.Close()

	hub := websocket.NewHub(bus)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)

	clientCh := make(chan websocket.Message, 10)
	c := hub.Register("client-listener", clientCh)
	defer hub.Unregister(c)

	// Publish domain event on event bus
	err := bus.Publish(ctx, events.Event{
		ID:         "evt-123",
		Type:       events.EntityStateChanged,
		OccurredAt: time.Now(),
		Source:     "zigbee",
		Payload: map[string]any{
			"entity_key": "light.kitchen",
			"state":      "ON",
		},
	})
	require.NoError(t, err)

	select {
	case msg := <-clientCh:
		assert.Equal(t, string(events.EntityStateChanged), msg.Type)
		payloadMap := msg.Payload.(map[string]any)
		assert.Equal(t, "light.kitchen", payloadMap["entity_key"])
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for forwarded event bus message")
	}
}
