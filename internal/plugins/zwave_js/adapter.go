package zwave_js

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/google/uuid"
)

// Command Class constants
const (
	CCBinarySwitch     = 37  // 0x25
	CCMultilevelSwitch = 38  // 0x26
	CCSensorMultilevel = 49  // 0x31
	CCMeter            = 50  // 0x32
	CCNotification     = 113 // 0x71
)

// Adapter implements the Z-Wave JS UI bridge for Domus.
type Adapter struct {
	id     string
	wsURL  string
	bus    events.Bus
	client *Client

	mu     sync.RWMutex
	active bool
	nodes  map[int]*ZWaveNode
}

// NewAdapter constructs a new Z-Wave JS adapter.
func NewAdapter(id string, wsURL string, bus events.Bus) *Adapter {
	if id == "" {
		id = "zwave_js"
	}
	a := &Adapter{
		id:    id,
		wsURL: wsURL,
		bus:   bus,
		nodes: make(map[int]*ZWaveNode),
	}
	a.client = NewClient(wsURL, a.handleValueUpdated)
	return a
}

func (a *Adapter) ID() string {
	return a.id
}

func (a *Adapter) Type() string {
	return "zwave_js"
}

func (a *Adapter) Start(ctx context.Context) error {
	a.mu.Lock()
	a.active = true
	a.mu.Unlock()

	if a.bus != nil {
		a.bus.Subscribe(events.CommandRequested, a.handleCommand)
	}

	// Try connecting in background so start doesn't block if zwave server isn't up yet
	go func() {
		if err := a.client.Connect(context.Background()); err != nil {
			slog.Debug("zwave_js: initial connect failed (will retry)", "url", a.wsURL, "error", err)
		}
	}()

	slog.Info("zwave_js: adapter started", "ws_url", a.wsURL)
	return nil
}

func (a *Adapter) Stop(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.active = false
	_ = a.client.Close()
	slog.Info("zwave_js: adapter stopped")
	return nil
}

// ProcessValueUpdate routes incoming CC property updates to Domus entities.
func (a *Adapter) ProcessValueUpdate(ctx context.Context, nodeID int, cc int, prop string, val any) {
	a.handleValueUpdated(nodeID, cc, prop, val)
}

func (a *Adapter) handleValueUpdated(nodeID int, cc int, prop string, val any) {
	if a.bus == nil {
		return
	}

	ctx := context.Background()
	now := time.Now().UTC()

	switch cc {
	case CCBinarySwitch:
		// Target switch entity
		boolVal, _ := val.(bool)
		state := "off"
		if boolVal {
			state = "on"
		}
		_ = a.bus.Publish(ctx, events.Event{
			ID:         uuid.NewString(),
			Type:       events.EntityStateChanged,
			OccurredAt: now,
			Source:     "zwave_js",
			Payload: map[string]any{
				"entity_key": fmt.Sprintf("switch.zwave_node_%d", nodeID),
				"state":      state,
				"available":  true,
			},
		})

	case CCMultilevelSwitch:
		// Target dimmer / shutter
		numVal, _ := val.(float64)
		state := "off"
		if numVal > 0 {
			state = "on"
		}
		attrs, _ := json.Marshal(map[string]any{"level": int(numVal)})
		_ = a.bus.Publish(ctx, events.Event{
			ID:         uuid.NewString(),
			Type:       events.EntityStateChanged,
			OccurredAt: now,
			Source:     "zwave_js",
			Payload: map[string]any{
				"entity_key": fmt.Sprintf("light.zwave_node_%d", nodeID),
				"state":      state,
				"attributes": string(attrs),
				"available":  true,
			},
		})

	case CCSensorMultilevel:
		// Multilevel Sensor (Temperature, Humidity, Illuminance)
		numVal, _ := val.(float64)
		entityKey := fmt.Sprintf("sensor.zwave_node_%d_%s", nodeID, strings.ToLower(prop))
		_ = a.bus.Publish(ctx, events.Event{
			ID:         uuid.NewString(),
			Type:       events.EntityStateChanged,
			OccurredAt: now,
			Source:     "zwave_js",
			Payload: map[string]any{
				"entity_key": entityKey,
				"state":      fmt.Sprintf("%.2f", numVal),
				"available":  true,
			},
		})

	case CCMeter:
		// Meter Power in Watts or Energy in kWh
		numVal, _ := val.(float64)
		entityKey := fmt.Sprintf("sensor.zwave_node_%d_power", nodeID)
		if strings.Contains(strings.ToLower(prop), "energy") || strings.Contains(strings.ToLower(prop), "kwh") {
			entityKey = fmt.Sprintf("sensor.zwave_node_%d_energy", nodeID)
		}
		_ = a.bus.Publish(ctx, events.Event{
			ID:         uuid.NewString(),
			Type:       events.EntityStateChanged,
			OccurredAt: now,
			Source:     "zwave_js",
			Payload: map[string]any{
				"entity_key": entityKey,
				"state":      fmt.Sprintf("%.1f", numVal),
				"available":  true,
			},
		})
	}
}

func (a *Adapter) handleCommand(ctx context.Context, e events.Event) error {
	payload, ok := e.Payload.(map[string]any)
	if !ok {
		return nil
	}

	entityKey, _ := payload["entity_key"].(string)
	if !strings.Contains(entityKey, ".zwave_node_") {
		return nil
	}

	// Extract node ID from entity_key e.g. switch.zwave_node_12 -> 12
	parts := strings.Split(entityKey, "_")
	if len(parts) < 3 {
		return nil
	}

	var nodeID int
	_, _ = fmt.Sscanf(parts[2], "%d", &nodeID)
	if nodeID <= 0 {
		return nil
	}

	cmd, _ := payload["command"].(string)
	if cmd == "" {
		cmd, _ = payload["service"].(string)
	}

	if strings.HasPrefix(entityKey, "switch.") {
		val := (cmd == "turn_on")
		return a.client.SetValue(ctx, nodeID, CCBinarySwitch, "currentValue", val)
	}

	if strings.HasPrefix(entityKey, "light.") {
		level := 0
		if cmd == "turn_on" {
			level = 99
			if params, ok := payload["params"].(map[string]any); ok {
				if b, ok := params["brightness"].(float64); ok {
					level = int(b * 99.0 / 255.0)
				}
			}
		}
		return a.client.SetValue(ctx, nodeID, CCMultilevelSwitch, "currentValue", level)
	}

	return nil
}
