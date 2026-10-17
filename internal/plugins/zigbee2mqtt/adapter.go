// Package zigbee2mqtt implements the Zigbee2MQTT integration adapter for Domus.
package zigbee2mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/device"
	"github.com/domus-hub/domus/internal/domain/entity"
	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/infrastructure/mqtt"
	"github.com/domus-hub/domus/internal/plugins"
	"github.com/google/uuid"
)

// Ensure Adapter implements plugins.Adapter.
var _ plugins.Adapter = (*Adapter)(nil)

// Config configures the Zigbee2MQTT adapter.
type Config struct {
	IntegrationID string
	BaseTopic     string // defaults to "zigbee2mqtt"
}

// Adapter is the Zigbee2MQTT integration adapter.
type Adapter struct {
	cfg          Config
	client       mqtt.Client
	deviceRepo   device.Repository
	entityRepo   entity.Repository
	bus          events.Bus
	mapper       *DeviceMapper
	mu           sync.Mutex
	unsubCommand func()
	running      bool
}

// NewAdapter constructs a Zigbee2MQTT adapter instance.
func NewAdapter(
	cfg Config,
	client mqtt.Client,
	deviceRepo device.Repository,
	entityRepo entity.Repository,
	bus events.Bus,
) *Adapter {
	if cfg.BaseTopic == "" {
		cfg.BaseTopic = "zigbee2mqtt"
	}
	mapper := NewDeviceMapper(cfg.IntegrationID)
	mapper.SetBaseTopic(cfg.BaseTopic)

	return &Adapter{
		cfg:        cfg,
		client:     client,
		deviceRepo: deviceRepo,
		entityRepo: entityRepo,
		bus:        bus,
		mapper:     mapper,
	}
}

// ID returns the unique integration ID.
func (a *Adapter) ID() string {
	return a.cfg.IntegrationID
}

// Type returns the protocol identifier.
func (a *Adapter) Type() string {
	return "zigbee2mqtt"
}

// HealthCheck verifies MQTT connectivity.
func (a *Adapter) HealthCheck(ctx context.Context) error {
	if !a.client.IsConnected() {
		return mqtt.ErrNotConnected
	}
	return nil
}

// Start begins subscriptions on MQTT topics and domain events.
func (a *Adapter) Start(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.running {
		return nil
	}

	// 1. Subscribe to bridge devices discovery
	bridgeDevicesTopic := fmt.Sprintf("%s/bridge/devices", a.cfg.BaseTopic)
	if err := a.client.Subscribe(ctx, bridgeDevicesTopic, 1, a.handleBridgeDevices); err != nil {
		return fmt.Errorf("zigbee2mqtt: subscribe %s failed: %w", bridgeDevicesTopic, err)
	}

	// 2. Subscribe to bridge availability state
	bridgeStateTopic := fmt.Sprintf("%s/bridge/state", a.cfg.BaseTopic)
	if err := a.client.Subscribe(ctx, bridgeStateTopic, 1, a.handleBridgeState); err != nil {
		return fmt.Errorf("zigbee2mqtt: subscribe %s failed: %w", bridgeStateTopic, err)
	}

	// 3. Subscribe to individual device telemetry updates (base_topic/+)
	deviceWildcardTopic := fmt.Sprintf("%s/+", a.cfg.BaseTopic)
	if err := a.client.Subscribe(ctx, deviceWildcardTopic, 1, a.handleDeviceMessage); err != nil {
		return fmt.Errorf("zigbee2mqtt: subscribe %s failed: %w", deviceWildcardTopic, err)
	}

	// 4. Listen to outbound CommandRequested events on the event bus
	a.unsubCommand = a.bus.Subscribe(events.CommandRequested, a.handleOutboundCommand)

	a.running = true
	slog.Info("zigbee2mqtt adapter started", "integration_id", a.cfg.IntegrationID, "base_topic", a.cfg.BaseTopic)
	return nil
}

// Stop terminates all subscriptions.
func (a *Adapter) Stop(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.running {
		return nil
	}

	if a.unsubCommand != nil {
		a.unsubCommand()
		a.unsubCommand = nil
	}

	_ = a.client.Unsubscribe(ctx,
		fmt.Sprintf("%s/bridge/devices", a.cfg.BaseTopic),
		fmt.Sprintf("%s/bridge/state", a.cfg.BaseTopic),
		fmt.Sprintf("%s/+", a.cfg.BaseTopic),
	)

	a.running = false
	slog.Info("zigbee2mqtt adapter stopped", "integration_id", a.cfg.IntegrationID)
	return nil
}

func (a *Adapter) handleBridgeDevices(topic string, payload []byte) {
	mappedDevs, mappedEntities, err := a.mapper.MapBridgeDevices(payload)
	if err != nil {
		slog.Error("zigbee2mqtt: error mapping bridge devices", "error", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Map of IEEE address -> Device ID
	ieeeToID := make(map[string]string)

	for _, md := range mappedDevs {
		existing, err := a.deviceRepo.GetByExternalID(ctx, a.cfg.IntegrationID, md.Device.ExternalID)
		if err == nil && existing != nil {
			// Update existing
			existing.Name = md.Device.Name
			existing.Manufacturer = md.Device.Manufacturer
			existing.Model = md.Device.Model
			existing.Online = md.Device.Online
			_ = a.deviceRepo.Update(ctx, existing)
			ieeeToID[md.Device.ExternalID] = existing.ID
		} else {
			// Create new
			if err := a.deviceRepo.Create(ctx, md.Device); err != nil {
				slog.Error("zigbee2mqtt: failed to save discovered device", "error", err)
				continue
			}
			ieeeToID[md.Device.ExternalID] = md.Device.ID

			// Emit DeviceDiscovered
			_ = a.bus.Publish(ctx, events.Event{
				ID:         uuid.NewString(),
				Type:       events.DeviceDiscovered,
				OccurredAt: time.Now().UTC(),
				Source:     "zigbee2mqtt",
				Payload: map[string]any{
					"device_id":      md.Device.ID,
					"integration_id": a.cfg.IntegrationID,
					"external_id":    md.Device.ExternalID,
					"name":           md.Device.Name,
					"manufacturer":   md.Device.Manufacturer,
					"model":          md.Device.Model,
				},
			})
		}
	}

	for _, ent := range mappedEntities {
		existing, err := a.entityRepo.GetByKey(ctx, ent.EntityKey)
		if err == nil && existing != nil {
			continue // Already registered
		}
		if err := a.entityRepo.Create(ctx, ent); err != nil {
			slog.Error("zigbee2mqtt: failed to save entity", "entity_key", ent.EntityKey, "error", err)
		}
	}
}

func (a *Adapter) handleDeviceMessage(topic string, payload []byte) {
	parts := strings.Split(topic, "/")
	if len(parts) != 2 {
		return
	}
	friendlyName := parts[1]
	if friendlyName == "bridge" {
		return // Handled by separate subscription
	}

	updates, err := a.mapper.MapTelemetry(friendlyName, payload)
	if err != nil {
		slog.Debug("zigbee2mqtt: failed to parse telemetry", "topic", topic, "error", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now().UTC()

	for _, u := range updates {
		ent, err := a.entityRepo.GetByKey(ctx, u.EntityKey)
		if err != nil {
			continue // Entity not found in repository
		}

		attrs := u.AttributesJSON
		if attrs == "" {
			attrs = ent.Attributes
		}

		if err := a.entityRepo.UpdateState(ctx, ent.ID, u.StateJSON, attrs, true, now); err != nil {
			slog.Error("zigbee2mqtt: failed to update entity state in db", "entity_key", u.EntityKey, "error", err)
			continue
		}

		_ = a.bus.Publish(ctx, events.Event{
			ID:         uuid.NewString(),
			Type:       events.EntityStateChanged,
			OccurredAt: now,
			Source:     "zigbee2mqtt",
			Payload: map[string]any{
				"entity_id":       ent.ID,
				"entity_key":      ent.EntityKey,
				"state":           u.StateJSON,
				"attributes":      attrs,
				"available":       true,
				"last_changed_at": now.Format(time.RFC3339),
			},
		})
	}
}

func (a *Adapter) handleBridgeState(topic string, payload []byte) {
	strPayload := strings.TrimSpace(string(payload))
	// Can be raw string "online"/"offline" or JSON {"state": "online"}
	var status string
	if strings.HasPrefix(strPayload, "{") {
		var stateObj struct {
			State string `json:"state"`
		}
		if err := json.Unmarshal(payload, &stateObj); err == nil {
			status = stateObj.State
		}
	} else {
		status = strPayload
	}

	isOnline := strings.EqualFold(status, "online")
	normalizedStatus := "disconnected"
	if isOnline {
		normalizedStatus = "connected"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = a.bus.Publish(ctx, events.Event{
		ID:         uuid.NewString(),
		Type:       events.IntegrationStateChanged,
		OccurredAt: time.Now().UTC(),
		Source:     "zigbee2mqtt",
		Payload: map[string]any{
			"integration_id": a.cfg.IntegrationID,
			"status":         normalizedStatus,
		},
	})
}

func (a *Adapter) handleOutboundCommand(ctx context.Context, e events.Event) error {
	payloadMap, ok := e.Payload.(map[string]any)
	if !ok {
		return nil
	}

	entityKey, _ := payloadMap["entity_key"].(string)
	service, _ := payloadMap["service"].(string)
	params, _ := payloadMap["params"].(map[string]any)

	if entityKey == "" || service == "" {
		return nil
	}

	// Verify if entity exists and belongs to this integration
	ent, err := a.entityRepo.GetByKey(ctx, entityKey)
	if err != nil || !ent.Writable {
		return nil
	}

	dev, err := a.deviceRepo.GetByID(ctx, ent.DeviceID)
	if err != nil || dev.IntegrationID != a.cfg.IntegrationID {
		return nil // Not managed by this Z2M instance
	}

	cmd := entity.Command{
		EntityKey: entityKey,
		Service:   service,
		Params:    params,
	}

	topic, payload, err := a.mapper.BuildCommand(cmd)
	if err != nil {
		slog.Error("zigbee2mqtt: build command error", "error", err)
		return err
	}

	return a.client.Publish(ctx, topic, 1, false, payload)
}

// BindDevices sends a cluster-level direct binding request to Zigbee2MQTT bridge.
func (a *Adapter) BindDevices(ctx context.Context, sourceIEEE, targetIEEE string, clusters []string) error {
	topic := fmt.Sprintf("%s/bridge/request/device/bind", a.cfg.BaseTopic)
	if len(clusters) == 0 {
		clusters = []string{"genOnOff"}
	}
	payload, err := json.Marshal(map[string]any{
		"from":     sourceIEEE,
		"to":       targetIEEE,
		"clusters": clusters,
	})
	if err != nil {
		return err
	}
	return a.client.Publish(ctx, topic, 1, false, payload)
}

// UnbindDevices sends an unbind request to Zigbee2MQTT bridge.
func (a *Adapter) UnbindDevices(ctx context.Context, sourceIEEE, targetIEEE string, clusters []string) error {
	topic := fmt.Sprintf("%s/bridge/request/device/unbind", a.cfg.BaseTopic)
	if len(clusters) == 0 {
		clusters = []string{"genOnOff"}
	}
	payload, err := json.Marshal(map[string]any{
		"from":     sourceIEEE,
		"to":       targetIEEE,
		"clusters": clusters,
	})
	if err != nil {
		return err
	}
	return a.client.Publish(ctx, topic, 1, false, payload)
}
