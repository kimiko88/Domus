// Package broadlink provides a local-first integration for BroadLink RM IR/RF remotes.
package broadlink

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/device"
	"github.com/domus-hub/domus/internal/domain/entity"
	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/infrastructure/crypto"
	"github.com/google/uuid"
)

const (
	PluginType = "broadlink_local"
	DefaultIntegrationID = "broadlink_local"
)

type DeviceConfig struct {
	MAC         string
	IP          string
	DeviceType  uint16
	Model       string
	DeviceID    uint32
	Key         [16]byte
	Name        string
	SupportsIR  bool
	SupportsRF  bool
}

type DeviceSummary struct {
	ID          string `json:"id"`
	MAC         string `json:"mac"`
	IP          string `json:"ip"`
	Name        string `json:"name"`
	Model       string `json:"model"`
	DeviceType  uint16 `json:"device_type"`
	Online      bool   `json:"online"`
	SupportsIR  bool   `json:"supports_ir"`
	SupportsRF  bool   `json:"supports_rf"`
	EntityKey   string `json:"entity_key"`
}

type SavedCommand struct {
	Name      string `json:"name"`
	Medium    string `json:"medium"`
	CreatedAt int64  `json:"created_at"`
}

type Adapter struct {
	id      string
	bus     events.Bus
	db      *sql.DB
	cipher  *crypto.EnvelopeCipher
	devRepo device.Repository
	entRepo entity.Repository

	mu          sync.RWMutex
	devices     map[string]DeviceConfig
	discovered  map[string]DiscoveredDevice
	onlineByMAC map[string]bool
	active      bool
	unsubscribe func()
}

func NewAdapter(id string, bus events.Bus) *Adapter {
	if id == "" {
		id = DefaultIntegrationID
	}
	return &Adapter{id: id, bus: bus, devices: make(map[string]DeviceConfig), discovered: make(map[string]DiscoveredDevice), onlineByMAC: make(map[string]bool)}
}

func (a *Adapter) WithPersistence(db *sql.DB, cipher *crypto.EnvelopeCipher) *Adapter {
	a.db, a.cipher = db, cipher
	return a
}

func (a *Adapter) WithRepositories(devRepo device.Repository, entRepo entity.Repository) *Adapter {
	a.devRepo, a.entRepo = devRepo, entRepo
	return a
}

func (a *Adapter) ID() string { return a.id }
func (a *Adapter) Type() string { return PluginType }

func (a *Adapter) Start(ctx context.Context) error {
	if a.db != nil {
		if err := a.initStorage(ctx); err != nil {
			return err
		}
		if err := a.loadDevices(ctx); err != nil {
			return err
		}
	}
	a.mu.Lock()
	a.active = true
	a.mu.Unlock()
	if a.bus != nil {
		a.unsubscribe = a.bus.Subscribe(events.CommandRequested, a.handleCommandEvent)
	}
	return nil
}

func (a *Adapter) Stop(ctx context.Context) error {
	a.mu.Lock()
	a.active = false
	unsubscribe := a.unsubscribe
	a.unsubscribe = nil
	a.mu.Unlock()
	if unsubscribe != nil {
		unsubscribe()
	}
	return nil
}

func (a *Adapter) HealthCheck(ctx context.Context) error {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if !a.active {
		return errors.New("broadlink: adapter is not running")
	}
	return nil
}

func (a *Adapter) initStorage(ctx context.Context) error {
	_, err := a.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS broadlink_devices (
			mac TEXT PRIMARY KEY, name TEXT NOT NULL, ip TEXT NOT NULL,
			device_type INTEGER NOT NULL, device_id INTEGER NOT NULL,
			aes_key_enc TEXT NOT NULL, updated_at INTEGER NOT NULL
		);
		CREATE TABLE IF NOT EXISTS broadlink_commands (
			mac TEXT NOT NULL, name TEXT NOT NULL, medium TEXT NOT NULL,
			code_base64 TEXT NOT NULL, created_at INTEGER NOT NULL,
			PRIMARY KEY(mac, name, medium)
		);
	`)
	return err
}

func (a *Adapter) loadDevices(ctx context.Context) error {
	rows, err := a.db.QueryContext(ctx, `SELECT mac, name, ip, device_type, device_id, aes_key_enc FROM broadlink_devices`)
	if err != nil { return err }
	var loaded []DeviceConfig
	for rows.Next() {
		var cfg DeviceConfig
		var keyEnc string
		if err := rows.Scan(&cfg.MAC, &cfg.Name, &cfg.IP, &cfg.DeviceType, &cfg.DeviceID, &keyEnc); err != nil { _ = rows.Close(); return err }
		if ip := net.ParseIP(cfg.IP); ip == nil || !ip.IsPrivate() {
			continue
		}
		keyText := keyEnc
		if a.cipher != nil {
			keyText, err = a.cipher.Decrypt(keyEnc)
			if err != nil {
				slog.Warn("broadlink: skipped device with unreadable encrypted key", "mac", cfg.MAC, "error", err)
				continue
			}
		}
		key, decodeErr := hex.DecodeString(keyText)
		if decodeErr != nil || len(key) != 16 {
			slog.Warn("broadlink: skipped device with invalid stored AES key", "mac", cfg.MAC)
			continue
		}
		copy(cfg.Key[:], key)
		cfg.Model, cfg.SupportsIR, cfg.SupportsRF = DeviceModel(cfg.DeviceType)
		loaded = append(loaded, cfg)
	}
	if err := rows.Err(); err != nil { _ = rows.Close(); return err }
	if err := rows.Close(); err != nil { return err }
	a.mu.Lock()
	for _, cfg := range loaded {
		a.devices[cfg.MAC] = cfg
		a.onlineByMAC[cfg.MAC] = false
	}
	a.mu.Unlock()
	for _, cfg := range loaded { a.provisionEntity(ctx, cfg, false) }
	return nil
}

func (a *Adapter) Scan(ctx context.Context, target string) ([]DiscoveredDevice, error) {
	devices, err := discover(ctx, strings.TrimSpace(target))
	if err != nil { return nil, err }
	a.mu.Lock()
	for _, dev := range devices {
		a.discovered[dev.MAC] = dev
		if cfg, paired := a.devices[dev.MAC]; paired {
			cfg.IP = dev.IP
			a.devices[dev.MAC] = cfg
			a.onlineByMAC[dev.MAC] = true
		}
	}
	paired := make([]DeviceConfig, 0)
	for _, dev := range devices {
		if cfg, ok := a.devices[dev.MAC]; ok {
			paired = append(paired, cfg)
		}
	}
	a.mu.Unlock()
	for _, cfg := range paired {
		if a.db != nil {
			if _, err := a.db.ExecContext(ctx, "UPDATE broadlink_devices SET ip=?, updated_at=? WHERE mac=?", cfg.IP, time.Now().Unix(), cfg.MAC); err != nil {
				slog.Warn("broadlink: could not refresh paired remote IP", "mac", cfg.MAC, "error", err)
			}
		}
		a.provisionEntity(ctx, cfg, true)
	}
	return devices, nil
}

func (a *Adapter) PairDevice(ctx context.Context, ip, name string) (DeviceSummary, error) {
	ip = strings.TrimSpace(ip)
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil || !parsedIP.IsPrivate() {
		return DeviceSummary{}, fmt.Errorf("broadlink: device IP must be a private LAN address")
	}
	var discovered DiscoveredDevice
	found := false
	a.mu.RLock()
	for _, dev := range a.discovered {
		if dev.IP == parsedIP.String() { discovered, found = dev, true; break }
	}
	a.mu.RUnlock()
	if !found {
		results, err := a.Scan(ctx, parsedIP.String())
		if err != nil { return DeviceSummary{}, err }
		for _, dev := range results {
			if dev.IP == parsedIP.String() { discovered, found = dev, true; break }
		}
	}
	if !found { return DeviceSummary{}, fmt.Errorf("broadlink: no supported RM remote replied at %s", ip) }

	client, err := NewClient(discovered.IP, discovered.DeviceType, discovered.MAC, 0, nil)
	if err != nil { return DeviceSummary{}, err }
	authCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := client.Auth(authCtx); err != nil { return DeviceSummary{}, fmt.Errorf("broadlink: authentication failed; verify that device lock is disabled: %w", err) }
	cfg := DeviceConfig{
		MAC: discovered.MAC, IP: discovered.IP, DeviceType: discovered.DeviceType,
		Model: discovered.Model, Name: strings.TrimSpace(name), SupportsIR: discovered.SupportsIR,
		SupportsRF: discovered.SupportsRF, DeviceID: client.deviceID, Key: client.key,
	}
	if cfg.Name == "" { cfg.Name = cfg.Model + " " + cfg.MAC[len(cfg.MAC)-5:] }

	if a.db == nil { return DeviceSummary{}, fmt.Errorf("broadlink: encrypted persistence is required before pairing devices") }
	keyText := hex.EncodeToString(cfg.Key[:])
	if a.cipher == nil { return DeviceSummary{}, fmt.Errorf("broadlink: encryption service is required before pairing devices") }
	keyEnc, err := a.cipher.Encrypt(keyText)
	if err != nil { return DeviceSummary{}, fmt.Errorf("broadlink: encrypt session key: %w", err) }
	_, err = a.db.ExecContext(ctx, `
		INSERT INTO broadlink_devices(mac, name, ip, device_type, device_id, aes_key_enc, updated_at)
		VALUES(?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(mac) DO UPDATE SET name=excluded.name, ip=excluded.ip,
		device_type=excluded.device_type, device_id=excluded.device_id,
		aes_key_enc=excluded.aes_key_enc, updated_at=excluded.updated_at
	`, cfg.MAC, cfg.Name, cfg.IP, cfg.DeviceType, cfg.DeviceID, keyEnc, time.Now().Unix())
	if err != nil { return DeviceSummary{}, fmt.Errorf("broadlink: save device: %w", err) }
	a.mu.Lock()
	a.devices[cfg.MAC] = cfg
	a.onlineByMAC[cfg.MAC] = true
	a.mu.Unlock()
	a.provisionEntity(ctx, cfg, true)
	if a.bus != nil {
		_ = a.bus.Publish(ctx, events.Event{ID: uuid.NewString(), Type: events.DeviceOnline, OccurredAt: time.Now(), Source: a.id,
			Payload: map[string]any{"device_id": deviceRecordID(cfg.MAC), "ip": cfg.IP}})
	}
	return summary(cfg, true), nil
}

func (a *Adapter) provisionEntity(ctx context.Context, cfg DeviceConfig, online bool) {
	if a.devRepo == nil || a.entRepo == nil { return }
	now := time.Now().UTC()
	deviceID := deviceRecordID(cfg.MAC)
	meta, _ := json.Marshal(map[string]any{"protocol":"broadlink_local","mac":cfg.MAC,"ip":cfg.IP,"device_type":cfg.DeviceType,"supports_ir":cfg.SupportsIR,"supports_rf":cfg.SupportsRF})
	dev, err := a.devRepo.GetByID(ctx, deviceID)
	if dev == nil || err != nil {
		_ = a.devRepo.Create(ctx, &device.Device{ID:deviceID, IntegrationID:a.id, ExternalID:cfg.MAC, Name:cfg.Name, Manufacturer:"BroadLink", Model:cfg.Model, Online:online, Metadata:string(meta), CreatedAt:now, UpdatedAt:now})
	} else {
		dev.Name, dev.Model, dev.Online, dev.Metadata, dev.UpdatedAt = cfg.Name, cfg.Model, online, string(meta), now
		_ = a.devRepo.Update(ctx, dev)
	}
	key := entityKey(cfg.MAC)
	attributes, _ := json.Marshal(map[string]any{"model":cfg.Model,"ip":cfg.IP,"mac":cfg.MAC,"supports_ir":cfg.SupportsIR,"supports_rf":cfg.SupportsRF,"protocol":"broadlink_local"})
	ent, err := a.entRepo.GetByKey(ctx, key)
	if ent == nil || err != nil {
		_ = a.entRepo.Create(ctx, &entity.Entity{ID:uuid.NewString(),DeviceID:deviceID,EntityKey:key,Domain:"remote",Name:cfg.Name,Writable:true,State:`{"state":"idle"}`,Attributes:string(attributes),Available:true,LastChangedAt:now,LastUpdatedAt:now})
	} else {
		ent.DeviceID, ent.Name, ent.Writable, ent.Available, ent.Attributes, ent.LastUpdatedAt = deviceID, cfg.Name, true, online, string(attributes), now
		_ = a.entRepo.Update(ctx, ent)
	}
}

func (a *Adapter) ListDevices() []DeviceSummary {
	a.mu.RLock()
	defer a.mu.RUnlock()
	out := make([]DeviceSummary, 0, len(a.devices))
	for mac, cfg := range a.devices {
		out = append(out, summary(cfg, a.onlineByMAC[mac]))
	}
	return out
}

func (a *Adapter) markOnline(mac string, online bool) {
	a.mu.Lock()
	if _, exists := a.devices[mac]; exists {
		a.onlineByMAC[mac] = online
	}
	a.mu.Unlock()
}

func summary(cfg DeviceConfig, online bool) DeviceSummary {
	return DeviceSummary{ID:deviceRecordID(cfg.MAC),MAC:cfg.MAC,IP:cfg.IP,Name:cfg.Name,Model:cfg.Model,DeviceType:cfg.DeviceType,Online:online,SupportsIR:cfg.SupportsIR,SupportsRF:cfg.SupportsRF,EntityKey:entityKey(cfg.MAC)}
}

func deviceRecordID(mac string) string { return "broadlink_" + strings.NewReplacer(":","", "-","").Replace(strings.ToLower(mac)) }
func entityKey(mac string) string { return "remote." + deviceRecordID(mac) }

func (a *Adapter) resolve(target string) (DeviceConfig, error) {
	target = strings.TrimSpace(target)
	a.mu.RLock()
	defer a.mu.RUnlock()
	for _, cfg := range a.devices {
		if strings.EqualFold(cfg.MAC, target) || deviceRecordID(cfg.MAC) == target || entityKey(cfg.MAC) == target {
			return cfg, nil
		}
	}
	return DeviceConfig{}, fmt.Errorf("broadlink: configured device %q not found", target)
}

func (a *Adapter) client(cfg DeviceConfig) (*Client, error) {
	return NewClient(cfg.IP, cfg.DeviceType, cfg.MAC, cfg.DeviceID, cfg.Key[:])
}

func (a *Adapter) StartLearning(ctx context.Context, target, medium string) error {
	cfg, err := a.resolve(target); if err != nil { return err }
	if medium == "rf" && !cfg.SupportsRF { return fmt.Errorf("broadlink: RF is not supported by %s", cfg.Model) }
	client, err := a.client(cfg); if err != nil { return err }
	err = client.StartLearning(ctx, medium)
	a.markOnline(cfg.MAC, err == nil)
	return err
}

func (a *Adapter) SaveLearnedCommand(ctx context.Context, target, name, medium string) (SavedCommand, error) {
	cfg, err := a.resolve(target); if err != nil { return SavedCommand{}, err }
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 80 { return SavedCommand{}, fmt.Errorf("broadlink: command name must contain 1 to 80 characters") }
	if medium != "ir" && medium != "rf" { return SavedCommand{}, fmt.Errorf("broadlink: command medium must be ir or rf") }
	if medium == "rf" && !cfg.SupportsRF { return SavedCommand{}, fmt.Errorf("broadlink: RF is not supported by %s", cfg.Model) }
	client, err := a.client(cfg); if err != nil { return SavedCommand{}, err }
	captureCtx, cancel := context.WithTimeout(ctx, 4*time.Second); defer cancel()
	code, err := client.ReadLearning(captureCtx)
	if err != nil {
		a.markOnline(cfg.MAC, false)
		return SavedCommand{}, err
	}
	a.markOnline(cfg.MAC, true)
	if len(code) == 0 || allZero(code) {
		return SavedCommand{}, fmt.Errorf("broadlink: no signal captured; press the physical remote and retry")
	}
	if a.db == nil { return SavedCommand{}, fmt.Errorf("broadlink: persistence is unavailable") }
	now := time.Now().Unix()
	_, err = a.db.ExecContext(ctx, `INSERT INTO broadlink_commands(mac,name,medium,code_base64,created_at) VALUES(?,?,?,?,?)
		ON CONFLICT(mac,name,medium) DO UPDATE SET code_base64=excluded.code_base64, created_at=excluded.created_at`,
		cfg.MAC, name, medium, base64.StdEncoding.EncodeToString(code), now)
	if err != nil { return SavedCommand{}, err }
	return SavedCommand{Name:name,Medium:medium,CreatedAt:now}, nil
}

func allZero(data []byte) bool {
	for _, value := range data {
		if value != 0 {
			return false
		}
	}
	return true
}

func (a *Adapter) ListCommands(ctx context.Context, target string) ([]SavedCommand, error) {
	cfg, err := a.resolve(target); if err != nil { return nil, err }
	if a.db == nil { return []SavedCommand{}, nil }
	rows, err := a.db.QueryContext(ctx, `SELECT name, medium, created_at FROM broadlink_commands WHERE mac=? ORDER BY name COLLATE NOCASE`, cfg.MAC)
	if err != nil { return nil, err }
	defer rows.Close()
	out := make([]SavedCommand, 0)
	for rows.Next() {
		var item SavedCommand
		if err := rows.Scan(&item.Name,&item.Medium,&item.CreatedAt); err != nil { return nil, err }
		out = append(out,item)
	}
	return out, rows.Err()
}

func (a *Adapter) SendStoredCommand(ctx context.Context, target, name, medium string) error {
	cfg, err := a.resolve(target); if err != nil { return err }
	if a.db == nil { return fmt.Errorf("broadlink: persistence is unavailable") }
	var encoded string
	if medium == "" {
		err = a.db.QueryRowContext(ctx, `SELECT code_base64 FROM broadlink_commands WHERE mac=? AND name=? ORDER BY CASE medium WHEN 'ir' THEN 0 ELSE 1 END LIMIT 1`, cfg.MAC, name).Scan(&encoded)
	} else {
		err = a.db.QueryRowContext(ctx, `SELECT code_base64 FROM broadlink_commands WHERE mac=? AND name=? AND medium=?`, cfg.MAC, name, medium).Scan(&encoded)
	}
	if err != nil { return fmt.Errorf("broadlink: saved command %q not found: %w", name, err) }
	code, err := base64.StdEncoding.DecodeString(encoded); if err != nil { return fmt.Errorf("broadlink: stored command data is invalid") }
	client, err := a.client(cfg); if err != nil { return err }
	err = client.SendData(ctx, code)
	a.markOnline(cfg.MAC, err == nil)
	return err
}

func (a *Adapter) SendRaw(ctx context.Context, target, code string) error {
	cfg, err := a.resolve(target); if err != nil { return err }
	data, err := base64.StdEncoding.DecodeString(code)
	if err != nil || len(data) == 0 || len(data) > 4096 { return fmt.Errorf("broadlink: code_base64 must encode 1 to 4096 bytes") }
	client, err := a.client(cfg); if err != nil { return err }
	err = client.SendData(ctx, data)
	a.markOnline(cfg.MAC, err == nil)
	return err
}

func (a *Adapter) RemoveDevice(ctx context.Context, target string) error {
	cfg, err := a.resolve(target); if err != nil { return err }
	if a.db != nil {
		if _, err := a.db.ExecContext(ctx, "DELETE FROM broadlink_commands WHERE mac=?", cfg.MAC); err != nil { return err }
		if _, err := a.db.ExecContext(ctx, "DELETE FROM broadlink_devices WHERE mac=?", cfg.MAC); err != nil { return err }
	}
	if a.entRepo != nil {
		if ent, err := a.entRepo.GetByKey(ctx, entityKey(cfg.MAC)); err == nil && ent != nil { if err := a.entRepo.Delete(ctx,ent.ID); err != nil { return err } }
	}
	if a.devRepo != nil {
		if dev, err := a.devRepo.GetByID(ctx,deviceRecordID(cfg.MAC)); err == nil && dev != nil { if err := a.devRepo.Delete(ctx,dev.ID); err != nil { return err } }
	}
	a.mu.Lock()
	delete(a.devices, cfg.MAC)
	delete(a.onlineByMAC, cfg.MAC)
	a.mu.Unlock()
	return nil
}

func (a *Adapter) handleCommandEvent(ctx context.Context, evt events.Event) error {
	raw, err := json.Marshal(evt.Payload); if err != nil { return nil }
	var payload struct {
		EntityID string `json:"entity_id"`
		EntityKey string `json:"entity_key"`
		Command string `json:"command"`
		Service string `json:"service"`
		Params map[string]any `json:"params"`
	}
	if json.Unmarshal(raw,&payload) != nil { return nil }
	if payload.Command == "" { payload.Command = payload.Service }
	if payload.EntityKey == "" && payload.EntityID != "" && a.entRepo != nil {
		if ent, e := a.entRepo.GetByID(ctx,payload.EntityID); e == nil && ent != nil { payload.EntityKey = ent.EntityKey }
	}
	if payload.EntityKey == "" { return nil }
	if payload.Command != "send" && payload.Command != "send_command" { return nil }
	name, _ := payload.Params["name"].(string)
	if name == "" { return nil }
	medium, _ := payload.Params["medium"].(string)
	return a.SendStoredCommand(ctx,payload.EntityKey,name,medium)
}
