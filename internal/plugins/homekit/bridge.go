package homekit

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/google/uuid"
)

// Bridge acts as an Apple HomeKit Accessory Bridge mapping Domus entities to Apple Home.
type Bridge struct {
	config BridgeConfig
	bus    events.Bus

	mu          sync.RWMutex
	active      bool
	accessories map[string]*Accessory // entityKey -> Accessory
	keyToAID    map[string]uint64
	aidToKey    map[uint64]string
	nextAID     uint64
}

// NewBridge creates a new HomeKit bridge instance.
func NewBridge(cfg BridgeConfig, bus events.Bus) *Bridge {
	if cfg.Port <= 0 {
		cfg.Port = 51827
	}
	if cfg.SetupCode == "" {
		cfg = DefaultBridgeConfig()
	}

	return &Bridge{
		config:      cfg,
		bus:         bus,
		accessories: make(map[string]*Accessory),
		keyToAID:    make(map[string]uint64),
		aidToKey:    make(map[uint64]string),
		nextAID:     2, // AID 1 is the Bridge itself
	}
}

func (b *Bridge) ID() string {
	return "homekit"
}

func (b *Bridge) Type() string {
	return "homekit"
}

func (b *Bridge) GetConfig() BridgeConfig {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.config
}

func (b *Bridge) Start(ctx context.Context) error {
	b.mu.Lock()
	b.active = true
	b.mu.Unlock()

	if b.bus != nil {
		b.bus.Subscribe(events.EntityStateChanged, b.handleEntityStateChanged)
		b.bus.Subscribe(events.CommandRequested, b.handleCommandRequested)
	}

	slog.Info("homekit: bridge started on port", "port", b.config.Port, "pin", b.config.SetupCode, "setup_uri", b.config.SetupURI)
	return nil
}

func (b *Bridge) Stop(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.active = false
	slog.Info("homekit: bridge stopped")
	return nil
}

// RegisterEntity exposes a Domus entity (light, switch, sensor, climate) to Apple Home.
func (b *Bridge) RegisterEntity(entityKey string, domain string, name string) *Accessory {
	b.mu.Lock()
	defer b.mu.Unlock()

	if acc, exists := b.accessories[entityKey]; exists {
		return acc
	}

	aid := b.nextAID
	b.nextAID++
	b.keyToAID[entityKey] = aid
	b.aidToKey[aid] = entityKey

	// 1. Accessory Information Service (mandatory on all accessories)
	infoService := &Service{
		IID:  1,
		Type: ServiceAccessoryInformation,
		Characteristics: []*Characteristic{
			{IID: 2, Type: "00000023-0000-1000-8000-0026BB765291", Format: "string", Value: name, Permissions: []string{"pr"}}, // Name
			{IID: 3, Type: "00000020-0000-1000-8000-0026BB765291", Format: "string", Value: "Domus", Permissions: []string{"pr"}}, // Manufacturer
			{IID: 4, Type: "00000021-0000-1000-8000-0026BB765291", Format: "string", Value: domain, Permissions: []string{"pr"}}, // Model
		},
	}

	// 2. Domain-specific Primary Service
	var primaryService *Service
	switch domain {
	case "light":
		primaryService = &Service{
			IID:  10,
			Type: ServiceLightbulb,
			Characteristics: []*Characteristic{
				{IID: 11, Type: CharOn, Format: "bool", Value: false, Permissions: []string{"pr", "pw", "ev"}},
				{IID: 12, Type: CharBrightness, Format: "int", Value: 100, Permissions: []string{"pr", "pw", "ev"}},
			},
		}
	case "switch":
		primaryService = &Service{
			IID:  10,
			Type: ServiceSwitch,
			Characteristics: []*Characteristic{
				{IID: 11, Type: CharOn, Format: "bool", Value: false, Permissions: []string{"pr", "pw", "ev"}},
			},
		}
	case "climate":
		primaryService = &Service{
			IID:  10,
			Type: ServiceThermostat,
			Characteristics: []*Characteristic{
				{IID: 11, Type: CharCurrentTemperature, Format: "float", Value: 20.0, Permissions: []string{"pr", "ev"}},
				{IID: 12, Type: CharTargetTemperature, Format: "float", Value: 21.0, Permissions: []string{"pr", "pw", "ev"}},
				{IID: 13, Type: CharCurrentHeatingCooling, Format: "int", Value: 0, Permissions: []string{"pr", "ev"}},
			},
		}
	case "sensor":
		primaryService = &Service{
			IID:  10,
			Type: ServiceTemperatureSensor,
			Characteristics: []*Characteristic{
				{IID: 11, Type: CharCurrentTemperature, Format: "float", Value: 20.0, Permissions: []string{"pr", "ev"}},
			},
		}
	default:
		// Default binary switch
		primaryService = &Service{
			IID:  10,
			Type: ServiceSwitch,
			Characteristics: []*Characteristic{
				{IID: 11, Type: CharOn, Format: "bool", Value: false, Permissions: []string{"pr", "pw", "ev"}},
			},
		}
	}

	acc := &Accessory{
		AID:      aid,
		Services: []*Service{infoService, primaryService},
	}

	b.accessories[entityKey] = acc
	return acc
}

// HandleHomeKitWrite handles a characteristic modification requested from an iOS device (Home app or Siri).
func (b *Bridge) HandleHomeKitWrite(ctx context.Context, aid uint64, iid uint64, val any) error {
	b.mu.RLock()
	entityKey, exists := b.aidToKey[aid]
	b.mu.RUnlock()

	if !exists {
		return fmt.Errorf("homekit: unknown accessory AID %d", aid)
	}

	if b.bus == nil {
		return nil
	}

	// Characteristic IID 11 is 'On' for Light & Switch
	if iid == 11 {
		boolVal, _ := val.(bool)
		cmd := "turn_off"
		if boolVal {
			cmd = "turn_on"
		}
		return b.bus.Publish(ctx, events.Event{
			ID:         uuid.NewString(),
			Type:       events.CommandRequested,
			OccurredAt: time.Now().UTC(),
			Source:     "homekit",
			Payload: map[string]any{
				"entity_key": entityKey,
				"command":    cmd,
				"service":    cmd,
			},
		})
	}

	// Characteristic IID 12 is 'Brightness' for Light, or 'TargetTemperature' for Climate
	if iid == 12 {
		if floatVal, ok := val.(float64); ok {
			return b.bus.Publish(ctx, events.Event{
				ID:         uuid.NewString(),
				Type:       events.CommandRequested,
				OccurredAt: time.Now().UTC(),
				Source:     "homekit",
				Payload: map[string]any{
					"entity_key": entityKey,
					"command":    "set_level",
					"service":    "set_level",
					"params":     map[string]any{"value": floatVal},
				},
			})
		}
	}

	return nil
}

func (b *Bridge) handleEntityStateChanged(ctx context.Context, e events.Event) error {
	payload, ok := e.Payload.(map[string]any)
	if !ok {
		return nil
	}

	entityKey, _ := payload["entity_key"].(string)
	b.mu.Lock()
	acc, exists := b.accessories[entityKey]
	b.mu.Unlock()

	if !exists || len(acc.Services) < 2 {
		return nil
	}

	stateStr, _ := payload["state"].(string)
	primarySvc := acc.Services[1]

	// Update characteristic value
	for _, char := range primarySvc.Characteristics {
		if char.Type == CharOn {
			char.Value = (strings.ToLower(stateStr) == "on")
		} else if char.Type == CharCurrentTemperature {
			if f, err := strconv.ParseFloat(stateStr, 64); err == nil {
				char.Value = f
			}
		}
	}

	return nil
}

func (b *Bridge) handleCommandRequested(ctx context.Context, e events.Event) error {
	return nil
}
