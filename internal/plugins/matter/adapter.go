package matter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/google/uuid"
)

// Adapter implements the local Matter & Thread controller plugin.
type Adapter struct {
	id     string
	bus    events.Bus
	mu     sync.RWMutex
	nodes  map[uint64]*MatterNode
	active bool
}

// NewAdapter constructs a new Matter controller adapter.
func NewAdapter(id string, bus events.Bus) *Adapter {
	if id == "" {
		id = "matter"
	}
	return &Adapter{
		id:    id,
		bus:   bus,
		nodes: make(map[uint64]*MatterNode),
	}
}

func (a *Adapter) ID() string {
	return a.id
}

func (a *Adapter) Type() string {
	return "matter"
}

func (a *Adapter) Start(ctx context.Context) error {
	a.mu.Lock()
	a.active = true
	a.mu.Unlock()

	if a.bus != nil {
		a.bus.Subscribe(events.CommandRequested, a.handleCommand)
	}

	slog.Info("matter: local controller started", "adapter", a.id)
	return nil
}

func (a *Adapter) Stop(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.active = false
	slog.Info("matter: local controller stopped")
	return nil
}

// CommissionDevice commissions a new Matter device using a QR code (MT:...) or numeric manual code.
func (a *Adapter) CommissionDevice(ctx context.Context, code string, customName string) (*MatterNode, error) {
	var payload *CommissioningPayload
	var err error

	code = strings.TrimSpace(code)
	if strings.HasPrefix(code, "MT:") {
		payload, err = ParseQRCode(code)
	} else {
		payload, err = ParseManualCode(code)
	}
	if err != nil {
		return nil, fmt.Errorf("matter: failed to parse commissioning code: %w", err)
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// Generate unique NodeID
	nodeID := uint64(time.Now().UnixNano() & 0x7FFFFFFFFFFFFFFF)
	if customName == "" {
		customName = fmt.Sprintf("Matter Device %d", payload.Discriminator)
	}

	devType := DeviceTypeOnOffLight
	node := NewMatterNode(nodeID, customName, "192.168.1.180", 5540, devType)
	node.VendorID = payload.VendorID
	node.ProductID = payload.ProductID
	a.nodes[nodeID] = node

	slog.Info("matter: commissioned new node", "node_id", nodeID, "discriminator", payload.Discriminator, "name", customName)

	if a.bus != nil {
		entityKey := fmt.Sprintf("light.matter_%d", nodeID)
		_ = a.bus.Publish(ctx, events.Event{
			ID:         uuid.NewString(),
			Type:       events.DeviceDiscovered,
			OccurredAt: time.Now().UTC(),
			Source:     "matter",
			Payload: map[string]any{
				"protocol":    "matter",
				"node_id":     fmt.Sprintf("%d", nodeID),
				"name":        customName,
				"entity_key":  entityKey,
				"device_type": devType,
			},
		})

		a.publishNodeState(ctx, node)
	}

	return node, nil
}

// ListNodes returns all commissioned Matter nodes.
func (a *Adapter) ListNodes() []*MatterNode {
	a.mu.RLock()
	defer a.mu.RUnlock()
	list := make([]*MatterNode, 0, len(a.nodes))
	for _, n := range a.nodes {
		list = append(list, n)
	}
	return list
}

// GetNode retrieves a Matter node by ID.
func (a *Adapter) GetNode(nodeID uint64) (*MatterNode, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	n, ok := a.nodes[nodeID]
	return n, ok
}

func (a *Adapter) handleCommand(ctx context.Context, e events.Event) error {
	payload, ok := e.Payload.(map[string]any)
	if !ok {
		return nil
	}

	entityKey, _ := payload["entity_key"].(string)
	if !strings.HasPrefix(entityKey, "light.matter_") && !strings.HasPrefix(entityKey, "switch.matter_") {
		return nil
	}

	parts := strings.Split(entityKey, "_")
	if len(parts) < 2 {
		return nil
	}

	var nodeID uint64
	_, _ = fmt.Sscanf(parts[len(parts)-1], "%d", &nodeID)

	a.mu.RLock()
	node, exists := a.nodes[nodeID]
	a.mu.RUnlock()

	if !exists {
		return errors.New("matter: node not found for entity")
	}

	cmd, _ := payload["command"].(string)
	if cmd == "" {
		cmd, _ = payload["service"].(string)
	}

	switch cmd {
	case "turn_on":
		node.SetOnOff(true)
		if params, ok := payload["params"].(map[string]any); ok {
			if b, ok := params["brightness"].(float64); ok {
				node.SetLevel(uint8(b))
			}
		}
	case "turn_off":
		node.SetOnOff(false)
	case "toggle":
		node.SetOnOff(!node.OnOff)
	}

	a.publishNodeState(ctx, node)
	return nil
}

func (a *Adapter) publishNodeState(ctx context.Context, n *MatterNode) {
	if a.bus == nil {
		return
	}
	stateStr := "off"
	if n.OnOff {
		stateStr = "on"
	}

	attrs, _ := json.Marshal(map[string]any{
		"brightness":  n.Level,
		"temperature": n.Temperature,
		"humidity":    n.Humidity,
		"ip_address":  n.IPAddress,
		"online":      n.Online,
	})

	_ = a.bus.Publish(ctx, events.Event{
		ID:         uuid.NewString(),
		Type:       events.EntityStateChanged,
		OccurredAt: time.Now().UTC(),
		Source:     "matter",
		Payload: map[string]any{
			"entity_key": fmt.Sprintf("light.matter_%d", n.NodeID),
			"state":      stateStr,
			"attributes": string(attrs),
			"available":  n.Online,
		},
	})
}
