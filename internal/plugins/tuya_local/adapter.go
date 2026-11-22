package tuya_local

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net"
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
	DeviceID          string         `json:"device_id"`
	Name              string         `json:"name,omitempty"`
	IP                string         `json:"ip"`
	LocalKey          string         `json:"local_key,omitempty"`
	Version           string         `json:"version"`
	Category          DeviceCategory `json:"category"`
	EntityKey         string         `json:"entity_key"`
	IntegrationID     string         `json:"integration_id,omitempty"`
	Online            bool           `json:"online,omitempty"`
	InitialState      string         `json:"initial_state,omitempty"`
	InitialAttributes string         `json:"initial_attributes,omitempty"`
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

// trustedLANIP rejects public, malformed, and unspecified addresses. Tuya Cloud's
// IP field may contain a WAN address and must never become a LocalTuya target.
func trustedLANIP(raw string) string {
	ip := net.ParseIP(strings.TrimSpace(raw))
	if ip == nil || !ip.IsPrivate() {
		return ""
	}
	return ip.String()
}

// RegisterDevice registers or updates a device configuration in memory.
func (a *Adapter) RegisterDevice(cfg DeviceConfig) {
	cfg.IP = trustedLANIP(cfg.IP)
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
// RegisterAndPersistDevice provisions a device for local LAN control.
// A Local Key is mandatory here; Cloud-only devices must use ProvisionCloudDevice.
func (a *Adapter) RegisterAndPersistDevice(ctx context.Context, cfg DeviceConfig, name string) error {
	if strings.TrimSpace(cfg.LocalKey) == "" {
		return errors.New("tuya_local: local_key is required for local device registration")
	}
	return a.provisionDevice(ctx, cfg, name, true)
}

// ProvisionCloudDevice makes a Tuya device and its telemetry entity visible in Domus
// even when Tuya does not expose a Local Key. Such entities are read-only; they can
// be selected as Climate & History sources but cannot receive LAN commands.
func (a *Adapter) ProvisionCloudDevice(ctx context.Context, cfg DeviceConfig, name string) error {
	return a.provisionDevice(ctx, cfg, name, false)
}

func (a *Adapter) provisionDevice(ctx context.Context, cfg DeviceConfig, name string, localControl bool) error {
	if cfg.DeviceID == "" || len(cfg.DeviceID) > 128 || strings.ContainsAny(cfg.DeviceID, "/?#") {
		return errors.New("tuya_local: invalid device_id")
	}
	if localControl && strings.TrimSpace(cfg.LocalKey) == "" {
		return errors.New("tuya_local: local_key is required for local device registration")
	}
	if !localControl {
		// A Cloud-reported address may be the household's public WAN IP. A
		// cloud-only record never opens a socket and therefore stores no address.
		cfg.IP = ""
	} else {
		cfg.IP = trustedLANIP(cfg.IP)
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

	isActive := false
	if localControl {
		// If IP is not provided, use only a trusted address learned from LAN discovery.
		if cfg.IP == "" {
			if disc, found := a.discovery.GetDevice(cfg.DeviceID); found && disc.IP != "" {
				cfg.IP = trustedLANIP(disc.IP)
				if disc.Version != "" {
					cfg.Version = disc.Version
				}
			}
		}

		// Replace a stale LAN socket whenever the address or connection credentials
		// change. Do not put cloud-only records in the LocalTuya connection allow-list.
		var staleClient *DeviceClient
		a.mu.Lock()
		previous, previouslyConfigured := a.configs[cfg.DeviceID]
		if previouslyConfigured && (previous.IP != cfg.IP || previous.LocalKey != cfg.LocalKey || previous.Version != cfg.Version) {
			staleClient = a.devices[cfg.DeviceID]
			delete(a.devices, cfg.DeviceID)
		}
		a.configs[cfg.DeviceID] = cfg
		isActive = a.active
		a.mu.Unlock()
		if staleClient != nil {
			_ = staleClient.Close()
		}

		// Persist the Local Key encrypted at rest. Cloud-only devices never enter
		// this table, which is intentionally reserved for LocalTuya credentials.
		if a.db != nil {
			keyEnc := cfg.LocalKey
			if a.cipher != nil {
				enc, err := a.cipher.Encrypt(cfg.LocalKey)
				if err != nil {
					return fmt.Errorf("tuya_local: encrypt local_key: %w", err)
				}
				keyEnc = enc
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
				return fmt.Errorf("tuya_local: persist local device: %w", err)
			}
		}
	}

	now := time.Now().UTC()
	devRecordID := fmt.Sprintf("tuya_%s", cfg.DeviceID)
	integrationID := strings.TrimSpace(cfg.IntegrationID)
	if integrationID == "" || localControl {
		integrationID = a.id
	}
	protocol := "tuya_cloud"
	if localControl {
		protocol = "tuya_local"
	}

	if a.devRepo != nil {
		existingDev, err := a.devRepo.GetByID(ctx, devRecordID)
		if err != nil && !errors.Is(err, device.ErrDeviceNotFound) {
			return fmt.Errorf("tuya_local: get Domus device: %w", err)
		}
		if existingDev == nil {
			existingDev = &device.Device{
				ID:            devRecordID,
				CreatedAt:     now,
				Manufacturer:  "Tuya",
				ExternalID:    cfg.DeviceID,
			}
		}
		existingDev.IntegrationID = integrationID
		existingDev.ExternalID = cfg.DeviceID
		existingDev.Name = cfg.Name
		existingDev.Manufacturer = "Tuya"
		existingDev.Model = string(cfg.Category)
		existingDev.Online = cfg.Online || cfg.IP != ""
		existingDev.Metadata = fmt.Sprintf(`{"protocol":%q,"version":%q,"local_control":%t}`, protocol, cfg.Version, localControl)
		existingDev.UpdatedAt = now
		if existingDev.CreatedAt.IsZero() {
			existingDev.CreatedAt = now
		}
		if existingDev.CreatedAt.Equal(now) {
			if err := a.devRepo.Create(ctx, existingDev); err != nil {
				return fmt.Errorf("tuya_local: create Domus device: %w", err)
			}
		} else if err := a.devRepo.Update(ctx, existingDev); err != nil {
			return fmt.Errorf("tuya_local: update Domus device: %w", err)
		}
	}

	if a.entRepo != nil {
		entDomain := a.mapper.MapDPSToEntity(cfg.Category, nil).Domain
		hasInitialTelemetry := cfg.InitialState != "" || cfg.InitialAttributes != ""
		initialState := cfg.InitialState
		if initialState == "" {
			parsed := a.mapper.MapDPSToEntity(cfg.Category, nil)
			initialState = parsed.State
		}
		if initialState == "" {
			initialState = "unknown"
		}
		if !json.Valid([]byte(initialState)) {
			// Cloud inference supplies a plain state value (e.g. "heat"); Domus
			// stores entity state as a JSON object.
			stateJSON, _ := json.Marshal(map[string]any{"state": initialState})
			initialState = string(stateJSON)
		}
		initialAttributes := cfg.InitialAttributes
		if initialAttributes == "" || !json.Valid([]byte(initialAttributes)) {
			initialAttributes = `{}`
		}
		existingEnt, err := a.entRepo.GetByKey(ctx, cfg.EntityKey)
		if err != nil && !errors.Is(err, entity.ErrEntityNotFound) {
			return fmt.Errorf("tuya_local: get Domus entity: %w", err)
		}
		writable := localControl || a.IsDeviceConfigured(cfg.DeviceID)
		if existingEnt == nil {
			ent := &entity.Entity{
				ID:            uuid.NewString(),
				DeviceID:      devRecordID,
				EntityKey:     cfg.EntityKey,
				Domain:        entDomain,
				Name:          cfg.Name,
				Writable:      writable,
				State:         initialState,
				Attributes:    initialAttributes,
				Available:     true,
				LastChangedAt: now,
				LastUpdatedAt: now,
			}
			if err := a.entRepo.Create(ctx, ent); err != nil {
				return fmt.Errorf("tuya_local: create Domus entity: %w", err)
			}
		} else {
			existingEnt.Name = cfg.Name
			existingEnt.Writable = writable
			if err := a.entRepo.Update(ctx, existingEnt); err != nil {
				return fmt.Errorf("tuya_local: update entity metadata: %w", err)
			}
			// Keep a previous reading when the Cloud API omits status, but refresh
			// the entity when the provider supplies current telemetry.
			if hasInitialTelemetry {
				if err := a.entRepo.UpdateState(ctx, existingEnt.ID, initialState, initialAttributes, true, now); err != nil {
					return fmt.Errorf("tuya_local: update entity telemetry: %w", err)
				}
			}
		}
	}

	if localControl && isActive && cfg.IP != "" {
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

	// The write pool intentionally allows only one open connection. Do not issue
	// UPDATE statements while this result set is open: doing so would wait forever
	// for the connection held by rows. Collect the sanitized configs first, close
	// the cursor, and only then persist any IP cleanup.
	configs := make([]DeviceConfig, 0)
	untrustedDeviceIDs := make([]string, 0)
	for rows.Next() {
		var cfg DeviceConfig
		var name, keyEnc, catStr string
		if err := rows.Scan(&cfg.DeviceID, &name, &cfg.IP, &keyEnc, &cfg.Version, &catStr, &cfg.EntityKey); err != nil {
			continue
		}
		cfg.Name = name
		cfg.Category = DeviceCategory(catStr)

		// Purge legacy public WAN addresses saved by older releases. The device
		// remains configured but waits for a fresh private LAN discovery packet.
		if cfg.IP != "" {
			if trustedIP := trustedLANIP(cfg.IP); trustedIP == "" {
				cfg.IP = ""
				untrustedDeviceIDs = append(untrustedDeviceIDs, cfg.DeviceID)
			} else {
				cfg.IP = trustedIP
			}
		}

		// Decrypt local_key
		localKey := keyEnc
		if a.cipher != nil {
			if dec, err := a.cipher.Decrypt(keyEnc); err == nil {
				localKey = dec
			}
		}
		cfg.LocalKey = localKey
		configs = append(configs, cfg)
	}

	rowsErr := rows.Err()
	closeErr := rows.Close()
	if rowsErr != nil {
		return fmt.Errorf("read stored Tuya devices: %w", rowsErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close stored Tuya devices query: %w", closeErr)
	}

	a.mu.Lock()
	for _, cfg := range configs {
		a.configs[cfg.DeviceID] = cfg
	}
	a.mu.Unlock()

	// Persist sanitized addresses only after releasing the read cursor so the
	// update can acquire the single connection in the SQLite write pool.
	for _, deviceID := range untrustedDeviceIDs {
		if _, err := a.db.ExecContext(ctx, "UPDATE tuya_local_devices SET ip = '' WHERE device_id = ?", deviceID); err != nil {
			slog.Warn("tuya_local: could not clear untrusted stored IP", "device_id", deviceID, "error", err)
		}
	}

	return nil
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
	entityKey := cfg.EntityKey
	if entityKey == "" {
		entityKey = fmt.Sprintf("%s.tuya_%s", parsed.Domain, cfg.DeviceID)
	}

	now := time.Now().UTC()
	if a.entRepo != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		ent, err := a.entRepo.GetByKey(ctx, entityKey)
		if err == nil && ent != nil {
			// Keep provider-supplied telemetry that is not represented in local
			// DPS mappings (for example Tuya cloud temperature/humidity fields),
			// while refreshing every field the LocalTuya mapper understands.
			attributes := make(map[string]any)
			_ = json.Unmarshal([]byte(ent.Attributes), &attributes)
			for key, value := range parsed.Attributes {
				attributes[key] = value
			}
			attributesJSON, marshalErr := json.Marshal(attributes)

			state := make(map[string]any)
			_ = json.Unmarshal([]byte(ent.State), &state)
			if parsed.State != "" {
				state["state"] = parsed.State
			}
			stateJSON, stateErr := json.Marshal(state)
			if marshalErr == nil && stateErr == nil {
				if updateErr := a.entRepo.UpdateState(ctx, ent.ID, string(stateJSON), string(attributesJSON), true, now); updateErr != nil {
					slog.Warn("tuya_local: failed to persist entity telemetry", "entity_key", entityKey, "error", updateErr)
				}
			}
		} else if err != nil {
			slog.Debug("tuya_local: state received for unregistered entity", "entity_key", entityKey, "error", err)
		}
		cancel()
	}

	if a.bus == nil {
		return
	}
	_ = a.bus.Publish(context.Background(), events.Event{
		ID:         uuid.NewString(),
		Type:       events.EntityStateChanged,
		OccurredAt: now,
		Source:     a.id,
		Payload: map[string]any{
			"entity_key": entityKey,
			"state":      parsed.State,
			"attributes": parsed.Attributes,
		},
	})
}

func (a *Adapter) handleDiscoveredDevice(dev DiscoveredDevice) {
	dev.IP = trustedLANIP(dev.IP)
	if dev.IP == "" || !validDeviceID(dev.DeviceID) {
		return
	}
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

	// If device IP has changed for a known device, update connection.
	a.mu.Lock()
	cfg, known := a.configs[dev.DeviceID]
	isActive := a.active
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
		if isActive {
			go a.connectDevice(cfg)
		}
		return
	}
	a.mu.Unlock()
}

// UnregisterDevice removes a device from the LocalTuya allow-list and releases its LAN connection.
// Domus entities created for this LocalTuya device are removed as well, so a deselected device
// is no longer exposed as an active integration entity.
func (a *Adapter) UnregisterDevice(ctx context.Context, deviceID string) error {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return errors.New("tuya_local: device ID is required")
	}

	a.mu.Lock()
	cfg, exists := a.configs[deviceID]
	client := a.devices[deviceID]
	delete(a.configs, deviceID)
	delete(a.devices, deviceID)
	a.mu.Unlock()

	if !exists {
		return nil
	}
	if client != nil {
		_ = client.Close()
	}
	if a.db != nil {
		if _, err := a.db.ExecContext(ctx, "DELETE FROM tuya_local_devices WHERE device_id = ?", deviceID); err != nil {
			return fmt.Errorf("tuya_local: remove persisted device: %w", err)
		}
	}
	if a.entRepo != nil && cfg.EntityKey != "" {
		if ent, err := a.entRepo.GetByKey(ctx, cfg.EntityKey); err == nil && ent != nil {
			if err := a.entRepo.Delete(ctx, ent.ID); err != nil {
				return fmt.Errorf("tuya_local: remove Domus entity: %w", err)
			}
		}
	}
	if a.devRepo != nil {
		if dev, err := a.devRepo.GetByID(ctx, fmt.Sprintf("tuya_%s", deviceID)); err == nil && dev != nil {
			if err := a.devRepo.Delete(ctx, dev.ID); err != nil {
				return fmt.Errorf("tuya_local: remove Domus device: %w", err)
			}
		}
	}
	return nil
}

// waitForDiscoveredLANIP asks v3.5 devices to identify themselves immediately and
// waits briefly for the UDP listener to associate a trusted private IP with the
// configured device. This closes the race between LocalTuya sync and first use.
func (a *Adapter) waitForDiscoveredLANIP(ctx context.Context, deviceID string) (DeviceConfig, bool) {
	getConfig := func() (DeviceConfig, bool) {
		a.mu.RLock()
		defer a.mu.RUnlock()
		cfg, ok := a.configs[deviceID]
		if !ok {
			return DeviceConfig{}, false
		}
		cfg.IP = trustedLANIP(cfg.IP)
		return cfg, cfg.IP != ""
	}

	if cfg, found := getConfig(); found {
		return cfg, true
	}
	if a.discovery == nil {
		return DeviceConfig{}, false
	}

	// Do not wait for the next six-second background tick: actively solicit
	// a fresh response now. The UDP listener remains responsible for validating
	// the packet and extracting the source IP.
	sent := a.discovery.RequestDiscovery()
	slog.Info("tuya_local: requesting LAN discovery before command",
		"device_id", deviceID, "solicitations_sent", sent)

	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		if cfg, found := getConfig(); found {
			return cfg, true
		}
		select {
		case <-ctx.Done():
			slog.Warn("tuya_local: LAN discovery cancelled before an IP was found",
				"device_id", deviceID, "solicitations_sent", sent, "error", ctx.Err())
			return DeviceConfig{}, false
		case <-timer.C:
			slog.Warn("tuya_local: LAN discovery timed out without a matching device reply",
				"device_id", deviceID, "solicitations_sent", sent, "listening_ports", []int{6666, 6667, 7000})
			return DeviceConfig{}, false
		case <-ticker.C:
		}
	}
}

// SendCommand executes a LAN command directly to a device using its local_key.
func (a *Adapter) SendCommand(ctx context.Context, target string, command string, params map[string]any) (map[string]any, error) {
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
		return nil, fmt.Errorf("tuya_local: target %q not found in local configurations", target)
	}

	if !exists || client == nil {
		// A device can be registered before its first UDP announcement arrives.
		// Trigger one immediate discovery request instead of failing before the
		// background listener's next solicitation interval.
		if trustedLANIP(targetCfg.IP) == "" {
			resolved, found := a.waitForDiscoveredLANIP(ctx, targetDevID)
			if !found {
				return nil, fmt.Errorf("tuya_local: device %q has no discovered private LAN IP; UDP discovery did not return a matching device (check UDP 6666/6667/7000 on the same broadcast network)", targetDevID)
			}
			targetCfg = resolved
		}

		// A discovery callback may have completed the connection while this
		// command was waiting. Reuse it rather than opening a duplicate socket.
		a.mu.RLock()
		client = a.devices[targetDevID]
		a.mu.RUnlock()
		if client == nil {
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
				if existing := a.devices[targetDevID]; existing != nil {
					_ = client.Close()
					client = existing
				} else {
					a.devices[targetDevID] = client
				}
				a.mu.Unlock()
			} else {
				return nil, fmt.Errorf("tuya_local: device %q at %s is unreachable on LAN: %w", targetDevID, targetCfg.IP, err)
			}
		}
	}

	if targetCfg.Category == CategoryClimate && command == "set_temperature" {
		// Prefer the device's observed target DP when it advertises DPS 16
		// instead of the common DPS 2 mapping.
		latest := client.LatestDPS()
		if _, hasDP2 := latest["2"]; !hasDP2 {
			if _, hasDP16 := latest["16"]; hasDP16 {
				if params == nil {
					params = make(map[string]any)
				}
				params["dp_id"] = 16
			}
		}
	}

	dps, err := a.mapper.MapCommandToDPS(targetCfg.Category, command, params)
	if err != nil {
		return nil, err
	}
	response, err := client.SendCommand(ctx, dps)
	// If an older configuration guessed DPS 2 but the thermostat reports
	// its setpoint on DPS 16, retry once against the actually advertised point.
	if err != nil && targetCfg.Category == CategoryClimate && command == "set_temperature" {
		if _, hasDP2 := response["2"]; !hasDP2 {
			if _, hasDP16 := response["16"]; hasDP16 {
				if params == nil {
					params = make(map[string]any)
				}
				params["dp_id"] = 16
				retryDPS, mapErr := a.mapper.MapCommandToDPS(targetCfg.Category, command, params)
				if mapErr == nil {
					return client.SendCommand(ctx, retryDPS)
				}
			}
		}
	}
	return response, err
}

func (a *Adapter) handleCommandEvent(ctx context.Context, evt events.Event) error {
	payloadBytes, err := json.Marshal(evt.Payload)
	if err != nil {
		return nil
	}

	var cmd struct {
		EntityID  string         `json:"entity_id"`
		EntityKey string         `json:"entity_key"`
		Command   string         `json:"command"`
		Service   string         `json:"service"`
		Params    map[string]any `json:"params"`
		// Legacy assistant integrations used a top-level temperature field.
		Temperature *float64 `json:"temperature"`
	}
	if err := json.Unmarshal(payloadBytes, &cmd); err != nil {
		return nil
	}

	if cmd.Command == "" {
		cmd.Command = cmd.Service
	}
	if cmd.Params == nil {
		cmd.Params = make(map[string]any)
	}
	if cmd.Temperature != nil {
		if _, exists := cmd.Params["temperature"]; !exists {
			cmd.Params["temperature"] = *cmd.Temperature
		}
	}
	if cmd.EntityKey == "" && cmd.EntityID != "" && a.entRepo != nil {
		if ent, getErr := a.entRepo.GetByID(ctx, cmd.EntityID); getErr == nil && ent != nil {
			cmd.EntityKey = ent.EntityKey
		}
	}
	if cmd.EntityKey == "" || cmd.Command == "" {
		return nil
	}

	_, err = a.SendCommand(ctx, cmd.EntityKey, cmd.Command, cmd.Params)
	if errors.Is(err, ErrStateNotReported) {
		// The TCP peer acknowledged the command. Do not turn a missing telemetry
		// update into a failed automation/action; the entity will update when the
		// device next publishes its state.
		return nil
	}
	return err
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
