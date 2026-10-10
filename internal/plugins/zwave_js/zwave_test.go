package zwave_js_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/domus-hub/domus/internal/plugins/zwave_js"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func TestZWaveClientAndAdapter(t *testing.T) {
	var mu sync.Mutex
	receivedCommands := make([]map[string]any, 0)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			var msg map[string]any
			if err := conn.ReadJSON(&msg); err != nil {
				return
			}

			mu.Lock()
			receivedCommands = append(receivedCommands, msg)
			mu.Unlock()

			cmd, _ := msg["command"].(string)
			if cmd == "start_listening" {
				// Send a synthetic "value updated" event for node 12 (binary switch)
				_ = conn.WriteJSON(map[string]any{
					"event": "value updated",
					"args": map[string]any{
						"nodeId": float64(12),
						"args": map[string]any{
							"commandClass": float64(zwave_js.CCBinarySwitch),
							"property":     "currentValue",
							"newValue":     true,
						},
					},
				})

				// Send sensor multilevel (temperature)
				_ = conn.WriteJSON(map[string]any{
					"event": "value updated",
					"args": map[string]any{
						"nodeId": float64(12),
						"args": map[string]any{
							"commandClass": float64(zwave_js.CCSensorMultilevel),
							"property":     "Temperature",
							"newValue":     float64(21.5),
						},
					},
				})
			}
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 64, Workers: 1})
	adapter := zwave_js.NewAdapter("zwave_test", wsURL, bus)

	var pubMu sync.Mutex
	publishedEvents := make([]events.Event, 0)
	bus.Subscribe(events.EntityStateChanged, func(ctx context.Context, e events.Event) error {
		pubMu.Lock()
		publishedEvents = append(publishedEvents, e)
		pubMu.Unlock()
		return nil
	})

	ctx := context.Background()
	if err := adapter.Start(ctx); err != nil {
		t.Fatalf("failed to start adapter: %v", err)
	}
	defer adapter.Stop(ctx)

	// Wait for websocket messages to be processed
	time.Sleep(150 * time.Millisecond)

	pubMu.Lock()
	count := len(publishedEvents)
	pubMu.Unlock()

	if count < 2 {
		t.Errorf("expected at least 2 events published from zwave updates, got %d", count)
	}

	// Test Command execution: send turn_on command to switch.zwave_node_12
	err := bus.Publish(ctx, events.Event{
		ID:         "test_cmd_1",
		Type:       events.CommandRequested,
		OccurredAt: time.Now(),
		Source:     "user",
		Payload: map[string]any{
			"entity_key": "switch.zwave_node_12",
			"command":    "turn_on",
		},
	})
	if err != nil {
		t.Fatalf("failed to publish command: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	foundSetVal := false
	for _, m := range receivedCommands {
		if cmd, _ := m["command"].(string); cmd == "node.set_value" {
			foundSetVal = true
			break
		}
	}
	mu.Unlock()

	if !foundSetVal {
		t.Errorf("expected node.set_value command to be sent over websocket")
	}
}
