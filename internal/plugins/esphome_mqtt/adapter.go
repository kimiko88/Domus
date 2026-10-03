// Package esphome_mqtt implements the ESPHome MQTT integration adapter.
package esphome_mqtt

import (
	"context"
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

// Config configures the ESPHome MQTT adapter.
type Config struct {
	IntegrationID string
	TopicPrefix   string // Optional base topic prefix, e.g. "esphome"
}

// Adapter is the ESPHome MQTT protocol adapter.
type Adapter struct {
	cfg          Config
	client       mqtt.Client
	deviceRepo   device.Repository
	entityRepo   entity.Repository
	bus          events.Bus
	mapper       *EntityMapper
	mu           sync.Mutex
	unsubCommand func()
	running      bool
}

// NewAdapter constructs an ESPHome MQTT adapter instance.
func NewAdapter(
	cfg Config,
	client mqtt.Client,
	deviceRepo device.Repository,
	entityRepo entity.Repository,
	bus events.Bus,
) *Adapter {
	mapper := NewEntityMapper(cfg.IntegrationID)
	return &Adapter{
		cfg:        cfg,
		client:     client,
		deviceRepo: deviceRepo,
		entityRepo: entityRepo,
		bus:        bus,
		mapper:     mapper,
	}
}

// ID returns the integration instance ID.
func (a *Adapter) ID() string {
	return a.cfg.IntegrationID
}

// Type returns the adapter type.
func (a *Adapter) Type() string {
	return "esphome_mqtt"
}

// HealthCheck verifies MQTT connectivity.
func (a *Adapter) HealthCheck(ctx context.Context) error {
	if !a.client.IsConnected() {
		return mqtt.ErrNotConnected
	}
	return nil
}

// Start begins listening to MQTT topics and event bus commands.
func (a *Adapter) Start(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.running {
		return nil
	}

	stateTopicPattern := "+/+/+/state"
	statusTopicPattern := "+/status"
	if a.cfg.TopicPrefix != "" {
		p := strings.Trim(a.cfg.TopicPrefix, "/")
		stateTopicPattern = fmt.Sprintf("%s/+/%s", p, stateTopicPattern)
		statusTopicPattern = fmt.Sprintf("%s/+/%s", p, statusTopicPattern)
	}

	// 1. Subscribe to component state updates
	if err := a.client.Subscribe(ctx, stateTopicPattern, 1, a.handleStateMessage); err != nil {
		return fmt.Errorf("esphome_mqtt: subscribe %s failed: %w", stateTopicPattern, err)
	}

	// 2. Subscribe to node status (online/offline)
	if err := a.client.Subscribe(ctx, statusTopicPattern, 1, a.handleStatusMessage); err != nil {
		return fmt.Errorf("esphome_mqtt: subscribe %s failed: %w", statusTopicPattern, err)
	}

	// 3. Listen to outbound CommandRequested events
	a.unsubCommand = a.bus.Subscribe(events.CommandRequested, a.handleOutboundCommand)

	a.running = true
	slog.Info("esphome_mqtt adapter started", "integration_id", a.cfg.IntegrationID)
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

	stateTopicPattern := "+/+/+/state"
	statusTopicPattern := "+/status"
	if a.cfg.TopicPrefix != "" {
		p := strings.Trim(a.cfg.TopicPrefix, "/")
		stateTopicPattern = fmt.Sprintf("%s/+/%s", p, stateTopicPattern)
		statusTopicPattern = fmt.Sprintf("%s/+/%s", p, statusTopicPattern)
	}

	_ = a.client.Unsubscribe(ctx, stateTopicPattern, statusTopicPattern)
	a.running = false
	slog.Info("esphome_mqtt adapter stopped", "integration_id", a.cfg.IntegrationID)
	return nil
}

func (a *Adapter) handleStatusMessage(topic string, payload []byte) {
	nodeName, ok := a.mapper.ParseStatusTopic(topic)
	if !ok {
		return
	}

	statusStr := strings.TrimSpace(string(payload))
	isOnline := strings.EqualFold(statusStr, "online")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dev, err := a.deviceRepo.GetByExternalID(ctx, a.cfg.IntegrationID, nodeName)
	if err != nil || dev == nil {
		return
	}

	if err := a.deviceRepo.SetOnline(ctx, dev.ID, isOnline); err != nil {
		slog.Error("esphome_mqtt: failed to update device online state", "device_id", dev.ID, "error", err)
		return
	}

	eventType := events.DeviceOffline
	if isOnline {
		eventType = events.DeviceOnline
	}

	_ = a.bus.Publish(ctx, events.Event{
		ID:         uuid.NewString(),
		Type:       eventType,
		OccurredAt: time.Now().UTC(),
		Source:     "esphome_mqtt",
		Payload: map[string]any{
			"device_id":      dev.ID,
			"integration_id": a.cfg.IntegrationID,
			"external_id":    dev.ExternalID,
			"name":           dev.Name,
		},
	})
}

func (a *Adapter) handleStateMessage(topic string, payload []byte) {
	parsed, ok := a.mapper.ParseTopic(topic)
	if !ok || parsed.SubTopic != "state" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now().UTC()

	// 1. Ensure Device exists
	dev, err := a.deviceRepo.GetByExternalID(ctx, a.cfg.IntegrationID, parsed.NodeName)
	if err != nil || dev == nil {
		dev = &device.Device{
			ID:            uuid.NewString(),
			IntegrationID: a.cfg.IntegrationID,
			ExternalID:    parsed.NodeName,
			Name:          parsed.NodeName,
			Manufacturer:  "ESPHome",
			Model:         "ESP32/ESP8266",
			Online:        true,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		if err := a.deviceRepo.Create(ctx, dev); err != nil {
			slog.Error("esphome_mqtt: failed to create device", "node", parsed.NodeName, "error", err)
			return
		}
		_ = a.bus.Publish(ctx, events.Event{
			ID:         uuid.NewString(),
			Type:       events.DeviceDiscovered,
			OccurredAt: now,
			Source:     "esphome_mqtt",
			Payload: map[string]any{
				"device_id":      dev.ID,
				"integration_id": a.cfg.IntegrationID,
				"external_id":    dev.ExternalID,
				"name":           dev.Name,
			},
		})
	}

	// 2. Ensure Entity exists
	isWritable := parsed.Domain == "switch" || parsed.Domain == "light" || parsed.Domain == "climate"
	ent, err := a.entityRepo.GetByKey(ctx, parsed.EntityKey)
	if err != nil || ent == nil {
		unit := InferUnit(parsed.ObjectID)
		ent = &entity.Entity{
			ID:                uuid.NewString(),
			DeviceID:          dev.ID,
			EntityKey:         parsed.EntityKey,
			Domain:            parsed.Domain,
			Name:              fmt.Sprintf("%s %s", parsed.NodeName, parsed.ObjectID),
			UnitOfMeasurement: unit,
			Writable:          isWritable,
			State:             "null",
			Attributes:        "{}",
			Available:         true,
			LastChangedAt:     now,
			LastUpdatedAt:     now,
		}
		if err := a.entityRepo.Create(ctx, ent); err != nil {
			slog.Error("esphome_mqtt: failed to create entity", "entity_key", parsed.EntityKey, "error", err)
			return
		}
	}

	// 3. Map state and persist
	stateJSON, attrsJSON, err := a.mapper.MapState(parsed.Domain, payload)
	if err != nil {
		slog.Error("esphome_mqtt: failed to map state", "entity_key", parsed.EntityKey, "error", err)
		return
	}

	if attrsJSON == "{}" && ent.Attributes != "" {
		attrsJSON = ent.Attributes
	}

	if err := a.entityRepo.UpdateState(ctx, ent.ID, stateJSON, attrsJSON, true, now); err != nil {
		slog.Error("esphome_mqtt: failed to update entity state in db", "entity_key", parsed.EntityKey, "error", err)
		return
	}

	_ = a.bus.Publish(ctx, events.Event{
		ID:         uuid.NewString(),
		Type:       events.EntityStateChanged,
		OccurredAt: now,
		Source:     "esphome_mqtt",
		Payload: map[string]any{
			"entity_id":       ent.ID,
			"entity_key":      ent.EntityKey,
			"state":           stateJSON,
			"attributes":      attrsJSON,
			"available":       true,
			"last_changed_at": now.Format(time.RFC3339),
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

	ent, err := a.entityRepo.GetByKey(ctx, entityKey)
	if err != nil || !ent.Writable {
		return nil
	}

	dev, err := a.deviceRepo.GetByID(ctx, ent.DeviceID)
	if err != nil || dev.IntegrationID != a.cfg.IntegrationID {
		return nil // Not managed by this ESPHome instance
	}

	// Register node name with mapper so it cleanly splits node vs object_id
	a.mapper.RegisterNode(dev.ExternalID)

	cmd := entity.Command{
		EntityKey: entityKey,
		Service:   service,
		Params:    params,
	}

	topic, payload, err := a.mapper.BuildCommand(cmd)
	if err != nil {
		slog.Error("esphome_mqtt: build command failed", "error", err)
		return err
	}

	return a.client.Publish(ctx, topic, 1, false, payload)
}
