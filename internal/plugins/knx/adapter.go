package knx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/entity"
	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/plugins"
	"github.com/google/uuid"
)

// Ensure Adapter implements plugins.Adapter
var _ plugins.Adapter = (*Adapter)(nil)

// EntityMapping maps a KNX group address pair to a Domus Entity.
type EntityMapping struct {
	EntityKey      string `json:"entity_key"`
	Name           string `json:"name"`
	Domain         string `json:"domain"` // "light", "switch", "sensor", "climate"
	DPT            string `json:"dpt"`    // "1.001", "5.001", "9.001", "14.056"
	CommandAddress string `json:"command_address"`
	StateAddress   string `json:"state_address"`
	Unit           string `json:"unit,omitempty"`
	Writable       bool   `json:"writable"`

	// Parsed internal group address representations
	cmdRaw   uint16
	stateRaw uint16
}

// Config defines the configuration for a KNX integration instance.
type Config struct {
	IntegrationID string          `json:"integration_id"`
	GatewayAddr   string          `json:"gateway_addr"` // e.g. "192.168.1.200:3671"
	Entities      []EntityMapping `json:"entities"`
}

// Adapter bridges European KNX twisted-pair wired installations to the Domus event bus.
type Adapter struct {
	cfg          Config
	client       Client
	entityRepo   entity.Repository
	bus          events.Bus
	logger       *slog.Logger
	mu           sync.RWMutex
	running      bool
	unsubCommand func()
	addrMap      map[uint16]*EntityMapping
	keyMap       map[string]*EntityMapping
}

// NewAdapter creates a new KNX adapter instance.
func NewAdapter(
	cfg Config,
	client Client,
	entityRepo entity.Repository,
	bus events.Bus,
	logger *slog.Logger,
) (*Adapter, error) {
	if logger == nil {
		logger = slog.Default()
	}

	addrMap := make(map[uint16]*EntityMapping)
	keyMap := make(map[string]*EntityMapping)

	for i := range cfg.Entities {
		item := &cfg.Entities[i]
		if item.CommandAddress != "" {
			cmdAddr, err := ParseGroupAddress(item.CommandAddress)
			if err != nil {
				return nil, fmt.Errorf("knx: entity %s invalid command address: %w", item.EntityKey, err)
			}
			item.cmdRaw = cmdAddr
			addrMap[cmdAddr] = item
		}
		if item.StateAddress != "" {
			stAddr, err := ParseGroupAddress(item.StateAddress)
			if err != nil {
				return nil, fmt.Errorf("knx: entity %s invalid state address: %w", item.EntityKey, err)
			}
			item.stateRaw = stAddr
			addrMap[stAddr] = item
		}
		keyMap[item.EntityKey] = item
	}

	return &Adapter{
		cfg:        cfg,
		client:     client,
		entityRepo: entityRepo,
		bus:        bus,
		logger:     logger.With("plugin", "knx", "id", cfg.IntegrationID),
		addrMap:    addrMap,
		keyMap:     keyMap,
	}, nil
}

// ID returns the unique integration ID.
func (a *Adapter) ID() string {
	return a.cfg.IntegrationID
}

// Type returns the adapter type key.
func (a *Adapter) Type() string {
	return "knx"
}

// HealthCheck verifies if the KNX client connection is alive.
func (a *Adapter) HealthCheck(ctx context.Context) error {
	if a.client == nil || !a.client.IsConnected() {
		return fmt.Errorf("knx: gateway %s not connected", a.cfg.GatewayAddr)
	}
	return nil
}

// Start registers entities, subscribes to inbound telegrams, and connects to the bus.
func (a *Adapter) Start(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.running {
		return nil
	}

	// 1. Create entities in database if repository is provided
	if a.entityRepo != nil {
		for _, mapping := range a.keyMap {
			domain := mapping.Domain
			if domain == "" {
				domain = "switch"
			}
			ent := &entity.Entity{
				ID:                mapping.EntityKey,
				EntityKey:         mapping.EntityKey,
				Name:              mapping.Name,
				Domain:            domain,
				UnitOfMeasurement: mapping.Unit,
				Writable:          mapping.Writable,
				State:             "null",
				Attributes:        "{}",
				Available:         a.client.IsConnected(),
				LastChangedAt:     time.Now().UTC(),
				LastUpdatedAt:     time.Now().UTC(),
			}
			_ = a.entityRepo.Create(ctx, ent)
		}
	}

	// 2. Set inbound telegram handler
	if a.client != nil {
		a.client.SetInboundHandler(a.handleInboundTelegram)
		if err := a.client.Connect(ctx); err != nil {
			a.logger.Warn("knx initial connection warning", "err", err)
		}
	}

	// 3. Subscribe to outbound commands from Domus
	if a.bus != nil {
		a.unsubCommand = a.bus.Subscribe(events.CommandRequested, a.handleOutboundCommand)
	}

	a.running = true
	a.logger.Info("knx adapter started", "gateway", a.cfg.GatewayAddr, "entities", len(a.keyMap))
	return nil
}

// Stop disconnects the client and cleans up listeners.
func (a *Adapter) Stop(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.running {
		return nil
	}

	a.running = false
	if a.unsubCommand != nil {
		a.unsubCommand()
		a.unsubCommand = nil
	}

	if a.client != nil {
		_ = a.client.Close()
	}

	a.logger.Info("knx adapter stopped")
	return nil
}

// handleInboundTelegram is called by the KNX UDP client upon receiving an L_Data.ind telegram.
func (a *Adapter) handleInboundTelegram(frame *CEMIFrame) {
	if frame == nil || !frame.IsGroup {
		return
	}

	a.mu.RLock()
	mapping, exists := a.addrMap[frame.Destination]
	a.mu.RUnlock()

	if !exists {
		return // Telegram not mapped to any Domus entity
	}

	val, stateStr, err := a.decodeTelegramData(mapping.DPT, frame.Data)
	if err != nil {
		a.logger.Debug("failed to decode knx telegram", "entity", mapping.EntityKey, "err", err)
		return
	}

	ctx := context.Background()
	stateJSON, _ := json.Marshal(stateStr)

	if a.entityRepo != nil {
		_ = a.entityRepo.UpdateState(ctx, mapping.EntityKey, string(stateJSON), "{}", true, time.Now().UTC())
	}

	if a.bus != nil {
		_ = a.bus.Publish(ctx, events.Event{
			ID:         uuid.NewString(),
			Type:       events.EntityStateChanged,
			OccurredAt: time.Now().UTC(),
			Source:     "knx",
			Payload: map[string]any{
				"entity_id": mapping.EntityKey,
				"new_state": val,
				"unit":      mapping.Unit,
				"last_seen": time.Now().UTC().Format(time.RFC3339),
			},
		})
	}
}

func (a *Adapter) decodeTelegramData(dpt string, data []byte) (any, string, error) {
	switch dpt {
	case DPT1_Switch:
		if len(data) == 0 {
			return false, "off", errors.New("missing data for DPT1")
		}
		b := DecodeDPT1(data[0])
		if b {
			return true, "on", nil
		}
		return false, "off", nil

	case DPT5_Scaling:
		if len(data) == 0 {
			return 0.0, "0", errors.New("missing data for DPT5")
		}
		pct := DecodeDPT5(data[0])
		return pct, fmt.Sprintf("%.1f", pct), nil

	case DPT9_Temperature:
		temp, err := DecodeDPT9(data)
		if err != nil {
			return 0.0, "0", err
		}
		return temp, fmt.Sprintf("%.2f", temp), nil

	case DPT14_Power:
		pwr, err := DecodeDPT14(data)
		if err != nil {
			return 0.0, "0", err
		}
		return pwr, fmt.Sprintf("%.1f", pwr), nil

	default:
		return nil, "", fmt.Errorf("unsupported DPT %s", dpt)
	}
}

// handleOutboundCommand translates Domus commands into KNX group telegrams.
func (a *Adapter) handleOutboundCommand(ctx context.Context, ev events.Event) error {
	payloadMap, ok := ev.Payload.(map[string]any)
	if !ok {
		return nil
	}

	entityID, _ := payloadMap["entity_id"].(string)
	service, _ := payloadMap["service"].(string)

	a.mu.RLock()
	mapping, exists := a.keyMap[entityID]
	a.mu.RUnlock()

	if !exists || !mapping.Writable || mapping.cmdRaw == 0 {
		return nil // Not handled or read-only
	}

	var data []byte
	apci := uint16(0x0080) // GroupValueWrite

	switch mapping.DPT {
	case DPT1_Switch:
		on := (service == "turn_on")
		data = []byte{EncodeDPT1(on)}

	case DPT5_Scaling:
		val, _ := payloadMap["brightness"].(float64)
		if val == 0 {
			if v, ok := payloadMap["value"].(float64); ok {
				val = v
			}
		}
		data = []byte{EncodeDPT5(val)}

	case DPT9_Temperature:
		val, _ := payloadMap["temperature"].(float64)
		if val == 0 {
			if v, ok := payloadMap["value"].(float64); ok {
				val = v
			}
		}
		var err error
		data, err = EncodeDPT9(val)
		if err != nil {
			return err
		}

	default:
		return fmt.Errorf("knx: writing DPT %s not supported", mapping.DPT)
	}

	if a.client != nil {
		err := a.client.SendTelegram(mapping.cmdRaw, apci, data)
		if err != nil {
			a.logger.Error("failed to send knx telegram", "entity", entityID, "err", err)
			return err
		}
	}

	return nil
}
