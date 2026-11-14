package tuya_local

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/device"
	"github.com/domus-hub/domus/internal/domain/entity"
	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/infrastructure/crypto"
	"github.com/google/uuid"
)

// Config represents configuration for the Tuya Local adapter.
type Config struct {
	ID      string         `json:"id"`
	Devices []DeviceConfig `json:"devices"`
}

// DeviceConfig holds settings for a specific configured Tuya device.
type DeviceConfig struct {
	DeviceID  string         `json:"device_id"`
	Name      string         `json:"name,omitempty"`
	IP        string         `json:"ip"`
	LocalKey  string         `json:"local_key"`
	Version   string         `json:"version"`
	Category  DeviceCategory `json:"category"`
	EntityKey string         `json:"entity_key"`
}

// Adapter manages local Tuya integrations and bridges them to Domus domain events.
type Adapter struct {
	id        string
	bus       events.Bus
	mapper    *DPSMapper
	discovery *DiscoveryListener

	db      *sql.DB
	cipher  *crypto.EnvelopeCipher
	devRepo device.Repository
	entRepo entity.Repository

	mu      sync.RWMutex
	devices map[string]*DeviceClient
	configs map[string]DeviceConfig
	active  bool
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

// WithPersistence attaches SQLite storage and encryption for local_key credentials.
func (a *Adapter) WithPersistence(db *sql.DB, cipher *crypto.EnvelopeCipher) *Adapter {
	a.db = db
	a.cipher = cipher
	return a
}

// WithRepositories attaches Domus core device and entity repositories for auto-provisioning.
func (a *Adapter) WithRepositories(devRepo device.Repository, entRepo entity.Repository) *Adapter {
	a.devRepo = devRepo
	a.entRepo = entRepo
	return a
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

	// Load stored devices if database is attached
	if a.db != nil {
		if err := a.loadStoredDevices(ctx); err != nil {
			slog.Warn("tuya_local: could not load stored devices", "error", err)
		}
	}

	// Start UDP discovery
	if err := a.discovery.Start(ctx); err != nil {
		slog.Warn("tuya_local: discovery startup issue", "error", err)
	}

	// Connect to configured devices that already have an IP
	a.mu.RLock()
	configs := make([]DeviceConfig, 0, len(a.configs))
	for _, cfg := range a.configs {
		configs = append(configs, cfg)
	}
	a.mu.RUnlock()

	for _, cfg := range configs {
		if cfg.IP != "" {
			go a.connectDevice(cfg)
		}
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

// RegisterDevice registers or updates a device configuration in memory.
func (a *Adapter) RegisterDevice(cfg DeviceConfig) {
	a.mu.Lock()
	a.configs[cfg.DeviceID] = cfg
	isActive := a.active
	a.mu.Unlock()

	if isActive && cfg.IP != "" {
		go a.connectDevice(cfg)
	}
}

// RegisterAndPersistDevice persists the device with an encrypted local_key, provisions
// it in Domus repositories, and connects locally.
func (a *Adapter) RegisterAndPersistDevice(ctx context.Context, cfg DeviceConfig, name string) error {
	if cfg.DeviceID == "" || cfg.LocalKey == "" {
		return errors.New("tuya_local: device_id and local_key are required")
	}
	if name == "" {
		name = cfg.Name
	}
	if name == "" {
		name = fmt.Sprintf("Tuya %s", cfg.DeviceID)
	}
	cfg.Name = name

	if cfg.Category == "" {
		cfg.Category = CategorySwitch
	}
	if cfg.Version == "" {
		cfg.Version = "3.3"
	}
	if cfg.EntityKey == "" {
		domain := a.mapper.MapDPSToEntity(cfg.Category, nil).Domain
		cfg.EntityKey = fmt.Sprintf("%s.tuya_%s", domain, cfg.DeviceID)
	}

	// If IP is not provided, look in discovery cache
	if cfg.IP == "" {
		if disc, found := a.discovery.GetDevice(cfg.DeviceID); found && disc.IP != "" {
			cfg.IP = disc.IP
			if disc.Version != "" {
				cfg.Version = disc.Version
			}
		}
	}

	// 1. Store in memory
	a.mu.Lock()
	a.configs[cfg.DeviceID] = cfg
	isActive := a.active
	a.mu.Unlock()

	// 2. Persist in database
	if a.db != nil {
		keyEnc := cfg.LocalKey
		if a.cipher != nil {
			if enc, err := a.cipher.Encrypt(cfg.LocalKey); err == nil {
				keyEnc = enc
			}
		}

		_, err := a.db.ExecContext(ctx, `
			INSERT INTO tuya_local_devices (device_id, name, ip, local_key_enc, version, category, entity_key, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(device_id) DO UPDATE SET
				name = excluded.name,
				ip = excluded.ip,
				local_key_enc = excluded.local_key_enc,
				version = excluded.version,
				category = excluded.category,
				entity_key = excluded.entity_key,
				updated_at = excluded.updated_at
		`, cfg.DeviceID, cfg.Name, cfg.IP, keyEnc, cfg.Version, string(cfg.Category), cfg.EntityKey, time.Now().Unix())
		if err != nil {
			slog.Error("tuya_local: failed to persist device to database", "device_id", cfg.DeviceID, "error", err)
		}
	}

	// 3. Provision Domus core device & entity if repos attached
	if a.devRepo != nil && a.entRepo != nil {
		now := time.Now().UTC()
		devRecordID := fmt.Sprintf("tuya_%s", cfg.DeviceID)
		existingDev, _ := a.devRepo.GetByID(ctx, devRecordID)
		if existingDev == nil {
			dev := &device.Device{
				ID:            devRecordID,
				IntegrationID: a.id,
				ExternalID:    cfg.DeviceID,
				Name:          cfg.Name,
				Manufacturer:  "Tuya",
				Model:         string(cfg.Category),
				Online:        cfg.IP != "",
				Metadata:      fmt.Sprintf(`{"protocol":"tuya_local","version":%q}`, cfg.Version),
				CreatedAt:     now,
				UpdatedAt:     now,
			}
			_ = a.devRepo.Create(ctx, dev)
		}

		existingEnt, _ := a.entRepo.GetByKey(ctx, cfg.EntityKey)
		if existingEnt == nil {
			entDomain := a.mapper.MapDPSToEntity(cfg.Category, nil).Domain
			ent := &entity.Entity{
				ID:            uuid.NewString(),
				DeviceID:      devRecordID,
				EntityKey:     cfg.EntityKey,
				Domain:        entDomain,
				Name:          cfg.Name,
				Writable:      true,
				State:         `{"state":false}`,
				Attributes:    `{}`,
				Available:     true,
				LastChangedAt: now,
				LastUpdatedAt: now,
			}
			_ = a.entRepo.Create(ctx, ent)
		}
	}

	// 4. Connect over LAN if IP is known
	if isActive && cfg.IP != "" {
		go a.connectDevice(cfg)
	}

	return nil
}

func (a *Adapter) loadStoredDevices(ctx context.Context) error {
	_, err := a.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS tuya_local_devices (
			device_id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			ip TEXT NOT NULL,
			local_key_enc TEXT NOT NULL,
			version TEXT NOT NULL,
			category TEXT NOT NULL,
			entity_key TEXT NOT NULL,
			updated_at INTEGER NOT NULL
		);
	`)
	if err != nil {
		return fmt.Errorf("init tuya_local_devices table: %w", err)
	}

	rows, err := a.db.QueryContext(ctx, `
		SELECT device_id, name, ip, local_key_enc, version, category, entity_key
		FROM tuya_local_devices
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	a.mu.Lock()
	defer a.mu.Unlock()

	for rows.Next() {
		var cfg DeviceConfig
		var name, keyEnc, catStr string
		if err := rows.Scan(&cfg.DeviceID, &name, &cfg.IP, &keyEnc, &cfg.Version, &catStr, &cfg.EntityKey); err != nil {
			continue
		}
		cfg.Name = name
		cfg.Category = DeviceCategory(catStr)

		// Decrypt local_key
		localKey := keyEnc
		if a.cipher != nil {
			if dec, err := a.cipher.Decrypt(keyEnc); err == nil {
				localKey = dec
			}
		}
		cfg.LocalKey = localKey
		a.configs[cfg.DeviceID] = cfg
	}
	return rows.Err()
}

func (a *Adapter) connectDevice(cfg DeviceConfig) {
	if cfg.IP == "" {
		return
	}

	a.mu.Lock()
	if existing, ok := a.devices[cfg.DeviceID]; ok && existing != nil {
		_ = existing.Close()
	}
	a.mu.Unlock()

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
	if known && (cfg.IP != dev.IP || a.devices[dev.DeviceID] == nil) {
		cfg.IP = dev.IP
		if dev.Version != "" {
			cfg.Version = dev.Version
		}
		a.configs[dev.DeviceID] = cfg
		if oldClient, ok := a.devices[dev.DeviceID]; ok && oldClient != nil {
			_ = oldClient.Close()
			delete(a.devices, dev.DeviceID)
		}

		// Update IP in DB if persistence is active
		if a.db != nil {
			go func(dID, dIP, ver string) {
				_, _ = a.db.Exec(`UPDATE tuya_local_devices SET ip = ?, version = ?, updated_at = ? WHERE device_id = ?`,
					dIP, ver, time.Now().Unix(), dID)
			}(dev.DeviceID, dev.IP, cfg.Version)
		}

		a.mu.Unlock()
		go a.connectDevice(cfg)
		return
	}
	a.mu.Unlock()
}

// SendCommand executes a LAN command directly to a device using its local_key.
func (a *Adapter) SendCommand(ctx context.Context, target string, command string, params map[string]any) error {
	a.mu.RLock()
	var targetDevID string
	var targetCfg DeviceConfig
	for devID, cfg := range a.configs {
		if devID == target || cfg.EntityKey == target {
			targetDevID = devID
			targetCfg = cfg
			break
		}
	}
	client, exists := a.devices[targetDevID]
	a.mu.RUnlock()

	if targetDevID == "" {
		return fmt.Errorf("tuya_local: target %q not found in local configurations", target)
	}

	if !exists || client == nil {
		// Attempt on-demand connection if IP is present
		if targetCfg.IP != "" {
			client = NewDeviceClient(
				targetCfg.DeviceID,
				targetCfg.IP,
				[]byte(targetCfg.LocalKey),
				targetCfg.Version,
				targetCfg.Category,
				func(dps map[string]any) {
					a.handleDeviceStatus(targetCfg, dps)
				},
			)
			if err := client.Connect(ctx); err == nil {
				a.mu.Lock()
				a.devices[targetDevID] = client
				a.mu.Unlock()
			} else {
				return fmt.Errorf("tuya_local: device %q at %s is unreachable on LAN: %w", targetDevID, targetCfg.IP, err)
			}
		} else {
			return fmt.Errorf("tuya_local: device %q has unknown LAN IP; waiting for UDP broadcast discovery", targetDevID)
		}
	}

	dps, err := a.mapper.MapCommandToDPS(targetCfg.Category, command, params)
	if err != nil {
		return err
	}

	return client.SendCommand(ctx, dps)
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

	return a.SendCommand(ctx, cmd.EntityKey, cmd.Command, cmd.Params)
}

// GetConfiguredDevices returns all devices currently registered for local LAN control.
func (a *Adapter) GetConfiguredDevices() []DeviceConfig {
	a.mu.RLock()
	defer a.mu.RUnlock()

	res := make([]DeviceConfig, 0, len(a.configs))
	for _, c := range a.configs {
		res = append(res, c)
	}
	return res
}

// IsDeviceConfigured returns true if the device is registered with a local_key.
func (a *Adapter) IsDeviceConfigured(deviceID string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	_, ok := a.configs[deviceID]
	return ok
}

// IsDeviceConnected returns true if an active TCP socket exists to the device.
func (a *Adapter) IsDeviceConnected(deviceID string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	c, ok := a.devices[deviceID]
	return ok && c != nil
}

// GetDeviceIP returns the current IP of a configured device.
func (a *Adapter) GetDeviceIP(deviceID string) string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if c, ok := a.configs[deviceID]; ok {
		return c.IP
	}
	return ""
}

// ParseCategory maps Tuya standard cloud product categories to Domus local categories.
func ParseCategory(code string) DeviceCategory {
	switch strings.ToLower(strings.TrimSpace(code)) {
	case "kg", "switch", "qkg", "wkcz":
		return CategorySwitch
	case "cz", "plug", "socket", "pc":
		return CategoryPlug
	case "dj", "dd", "dc", "xdd", "fwd", "light", "cl":
		return CategoryLight
	case "wk", "climate", "thermostat":
		return CategoryClimate
	default:
		return CategorySwitch
	}
}
