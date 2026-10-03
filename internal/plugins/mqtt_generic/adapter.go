// Package mqtt_generic implements the generic MQTT protocol adapter for Domus.
package mqtt_generic

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

// Ensure Adapter implements plugins.Adapter
var _ plugins.Adapter = (*Adapter)(nil)

// Config configures the generic MQTT adapter.
type Config struct {
	IntegrationID        string
	StateTopicPattern    string // e.g. "domus/+/state"
	CommandTopicTemplate string // e.g. "domus/%s/set"
}

// Adapter is the generic MQTT integration adapter.
type Adapter struct {
	cfg          Config
	client       mqtt.Client
	entityRepo   entity.Repository
	deviceRepo   device.Repository
	bus          events.Bus
	mu           sync.Mutex
	unsubCommand func()
	running      bool
}

// NewAdapter constructs a generic MQTT adapter.
func NewAdapter(
	cfg Config,
	client mqtt.Client,
	entityRepo entity.Repository,
	deviceRepo device.Repository,
	bus events.Bus,
) *Adapter {
	if cfg.StateTopicPattern == "" {
		cfg.StateTopicPattern = "domus/+/state"
	}
	if cfg.CommandTopicTemplate == "" {
		cfg.CommandTopicTemplate = "domus/%s/set"
	}
	return &Adapter{
		cfg:        cfg,
		client:     client,
		entityRepo: entityRepo,
		deviceRepo: deviceRepo,
		bus:        bus,
	}
}

// ID returns the integration instance identifier.
func (a *Adapter) ID() string {
	return a.cfg.IntegrationID
}

// Type returns the adapter type key.
func (a *Adapter) Type() string {
	return "mqtt_generic"
}

// HealthCheck verifies whether the underlying MQTT client is connected.
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

	// 1. Subscribe to state updates from MQTT
	err := a.client.Subscribe(ctx, a.cfg.StateTopicPattern, 1, a.handleIncomingState)
	if err != nil {
		return fmt.Errorf("mqtt_generic: subscribe failed on %s: %w", a.cfg.StateTopicPattern, err)
	}

	// 2. Subscribe to outbound commands from the event bus
	a.unsubCommand = a.bus.Subscribe(events.CommandRequested, a.handleOutboundCommand)

	a.running = true
	slog.Info("mqtt_generic adapter started", "integration_id", a.cfg.IntegrationID, "topic", a.cfg.StateTopicPattern)
	return nil
}

// Stop terminates MQTT subscriptions and event listeners.
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

	_ = a.client.Unsubscribe(ctx, a.cfg.StateTopicPattern)
	a.running = false
	slog.Info("mqtt_generic adapter stopped", "integration_id", a.cfg.IntegrationID)
	return nil
}

func (a *Adapter) handleIncomingState(topic string, payload []byte) {
	entityKey := extractEntityKey(a.cfg.StateTopicPattern, topic)
	if entityKey == "" {
		slog.Warn("mqtt_generic: could not extract entity_key from topic", "topic", topic)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ent, err := a.entityRepo.GetByKey(ctx, entityKey)
	if err != nil {
		slog.Debug("mqtt_generic: state received for unregistered entity", "entity_key", entityKey)
		return
	}

	now := time.Now().UTC()
	stateStr := string(payload)

	// Update SQLite database
	if err := a.entityRepo.UpdateState(ctx, ent.ID, stateStr, ent.Attributes, true, now); err != nil {
		slog.Error("mqtt_generic: failed to update entity state in db", "entity_key", entityKey, "error", err)
		return
	}

	// Emit EntityStateChanged on event bus
	_ = a.bus.Publish(ctx, events.Event{
		ID:         uuid.NewString(),
		Type:       events.EntityStateChanged,
		OccurredAt: now,
		Source:     "mqtt_generic",
		Payload: map[string]any{
			"entity_id":       ent.ID,
			"entity_key":      ent.EntityKey,
			"state":           stateStr,
			"attributes":      ent.Attributes,
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
	if entityKey == "" {
		return nil
	}

	topic := fmt.Sprintf(a.cfg.CommandTopicTemplate, entityKey)
	jsonPayload, err := json.Marshal(payloadMap)
	if err != nil {
		return err
	}

	return a.client.Publish(ctx, topic, 1, false, jsonPayload)
}

func extractEntityKey(pattern, topic string) string {
	patternParts := strings.Split(pattern, "/")
	topicParts := strings.Split(topic, "/")

	if len(patternParts) != len(topicParts) {
		return ""
	}

	for i, part := range patternParts {
		if part == "+" {
			return topicParts[i]
		}
	}
	return ""
}
