package tuya_local

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/google/uuid"
)

// Config represents configuration for the Tuya Local adapter.
type Config struct {
	ID      string         `json:"id"`
	Devices []DeviceConfig `json:"devices"`
}

// DeviceConfig holds settings for a specific configured Tuya device.
type DeviceConfig struct {
	DeviceID   string         `json:"device_id"`
	IP         string         `json:"ip"`
	LocalKey   string         `json:"local_key"`
	Version    string         `json:"version"`
	Category   DeviceCategory `json:"category"`
	EntityKey  string         `json:"entity_key"`
}

// Adapter manages local Tuya integrations and bridges them to Domus domain events.
type Adapter struct {
	id        string
	bus       events.Bus
	mapper    *DPSMapper
	discovery *DiscoveryListener

	mu        sync.RWMutex
	devices   map[string]*DeviceClient
	configs   map[string]DeviceConfig
	active    bool
}

// NewAdapter creates a new Tuya Local adapter instance.
func NewAdapter(id string, bus events.Bus) *Adapter {
	if id == "" {
		id = "tuya_local"
	}
	adapter := &Adapter{
		id:      id,
		bus:     bus,
		mapper:  NewDPSMapper(),
		devices: make(map[string]*DeviceClient),
		configs: make(map[string]DeviceConfig),
	}

	adapter.discovery = NewDiscoveryListener(adapter.handleDiscoveredDevice)
	return adapter
}

func (a *Adapter) ID() string {
	return a.id
}

func (a *Adapter) Type() string {
	return "tuya_local"
}

// Start begins discovery and establishes connections to configured devices.
func (a *Adapter) Start(ctx context.Context) error {
	a.mu.Lock()
	a.active = true
	a.mu.Unlock()

	// Start UDP discovery
	if err := a.discovery.Start(ctx); err != nil {
		slog.Warn("tuya_local: discovery startup issue", "error", err)
	}

	// Connect to configured devices
	a.mu.RLock()
	defer a.mu.RUnlock()
	for _, cfg := range a.configs {
		go a.connectDevice(cfg)
	}

	// Listen for command events on bus
	if a.bus != nil {
		a.bus.Subscribe(events.CommandRequested, a.handleCommandEvent)
	}

	return nil
}

// Stop gracefully closes all connections and listeners.
func (a *Adapter) Stop(ctx context.Context) error {
	a.mu.Lock()
	a.active = false
	a.discovery.Stop()

	for devID, client := range a.devices {
		_ = client.Close()
		delete(a.devices, devID)
	}
	a.mu.Unlock()

	return nil
}

// HealthCheck verifies adapter health.
func (a *Adapter) HealthCheck(ctx context.Context) error {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if !a.active {
		return errors.New("tuya_local: adapter is not running")
	}
	return nil
}

// RegisterDevice registers or updates a device configuration.
func (a *Adapter) RegisterDevice(cfg DeviceConfig) {
	a.mu.Lock()
	a.configs[cfg.DeviceID] = cfg
	isActive := a.active
	a.mu.Unlock()

	if isActive {
		go a.connectDevice(cfg)
	}
}

func (a *Adapter) connectDevice(cfg DeviceConfig) {
	client := NewDeviceClient(
		cfg.DeviceID,
		cfg.IP,
		[]byte(cfg.LocalKey),
		cfg.Version,
		cfg.Category,
		func(dps map[string]any) {
			a.handleDeviceStatus(cfg, dps)
		},
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		slog.Debug("tuya_local: could not connect to device (will retry on next discovery/event)", "ip", cfg.IP, "error", err)
		return
	}

	a.mu.Lock()
	a.devices[cfg.DeviceID] = client
	a.mu.Unlock()

	// Notify device online
	if a.bus != nil {
		_ = a.bus.Publish(context.Background(), events.Event{
			ID:         uuid.NewString(),
			Type:       events.DeviceOnline,
			OccurredAt: time.Now(),
			Source:     a.id,
			Payload: map[string]any{
				"device_id": cfg.DeviceID,
				"ip":        cfg.IP,
			},
		})
	}
}

func (a *Adapter) handleDeviceStatus(cfg DeviceConfig, dps map[string]any) {
	parsed := a.mapper.MapDPSToEntity(cfg.Category, dps)
	if a.bus == nil {
		return
	}

	entityKey := cfg.EntityKey
	if entityKey == "" {
		entityKey = fmt.Sprintf("%s.tuya_%s", parsed.Domain, cfg.DeviceID)
	}

	_ = a.bus.Publish(context.Background(), events.Event{
		ID:         uuid.NewString(),
		Type:       events.EntityStateChanged,
		OccurredAt: time.Now(),
		Source:     a.id,
		Payload: map[string]any{
			"entity_key": entityKey,
			"state":      parsed.State,
			"attributes": parsed.Attributes,
		},
	})
}

func (a *Adapter) handleDiscoveredDevice(dev DiscoveredDevice) {
	if a.bus != nil {
		_ = a.bus.Publish(context.Background(), events.Event{
			ID:         uuid.NewString(),
			Type:       events.DeviceDiscovered,
			OccurredAt: time.Now(),
			Source:     a.id,
			Payload: map[string]any{
				"device_id":   dev.DeviceID,
				"ip":          dev.IP,
				"version":     dev.Version,
				"product_key": dev.ProductKey,
				"protocol":    "tuya_local",
			},
		})
	}

	// If device IP has changed for a known device, update connection
	a.mu.Lock()
	cfg, known := a.configs[dev.DeviceID]
	if known && cfg.IP != dev.IP {
		cfg.IP = dev.IP
		a.configs[dev.DeviceID] = cfg
		if oldClient, ok := a.devices[dev.DeviceID]; ok {
			_ = oldClient.Close()
			delete(a.devices, dev.DeviceID)
		}
		a.mu.Unlock()
		go a.connectDevice(cfg)
		return
	}
	a.mu.Unlock()
}

func (a *Adapter) handleCommandEvent(ctx context.Context, evt events.Event) error {
	payloadBytes, err := json.Marshal(evt.Payload)
	if err != nil {
		return nil
	}

	var cmd struct {
		EntityKey string         `json:"entity_key"`
		Command   string         `json:"command"`
		Params    map[string]any `json:"params"`
	}
	if err := json.Unmarshal(payloadBytes, &cmd); err != nil {
		return nil
	}

	a.mu.RLock()
	var targetDevID string
	var targetCfg DeviceConfig
	for devID, cfg := range a.configs {
		if cfg.EntityKey == cmd.EntityKey {
			targetDevID = devID
			targetCfg = cfg
			break
		}
	}
	client, exists := a.devices[targetDevID]
	a.mu.RUnlock()

	if !exists || client == nil {
		return nil // Not managed by this adapter
	}

	dps, err := a.mapper.MapCommandToDPS(targetCfg.Category, cmd.Command, cmd.Params)
	if err != nil {
		return err
	}

	return client.SendCommand(ctx, dps)
}
