// Package open_display exposes OpenDisplay e-paper devices to Domus through
// the authenticated py-opendisplay bridge. The bridge keeps BlueZ privileges
// outside the main Domus process.
package open_display

import (
    "bytes"
    "context"
    "encoding/base64"
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "log/slog"
    "net/http"
    "net/url"
    "regexp"
    "strings"
    "sync"
    "time"

    "github.com/domus-hub/domus/internal/domain/device"
    "github.com/domus-hub/domus/internal/domain/entity"
    "github.com/domus-hub/domus/internal/domain/events"
    "github.com/google/uuid"
)

const (
    IntegrationID = "open_display"
    maxImageBytes = 8 << 20
    maxResponseBytes = 2 << 20
)

var entityPartPattern = regexp.MustCompile(`[^a-z0-9]+`)

type Config struct {
    BridgeURL    string        `json:"bridge_url"`
    Token        string        `json:"token"`
    PollInterval time.Duration `json:"poll_interval,omitempty"`
    ScanSeconds  int           `json:"scan_seconds,omitempty"`
    HTTPTimeout  time.Duration `json:"http_timeout,omitempty"`
}

type Advertisement struct {
    BatteryMV     *int     `json:"battery_mv,omitempty"`
    TemperatureC  *float64 `json:"temperature_c,omitempty"`
    LoopCounter   *int     `json:"loop_counter,omitempty"`
    FormatVersion string   `json:"format_version,omitempty"`
}

type DeviceInfo struct {
    Name          string          `json:"name"`
    Address       string          `json:"address"`
    MAC           string          `json:"mac,omitempty"`
    Transport     string          `json:"transport"`
    Host          string          `json:"host,omitempty"`
    Port          int             `json:"port,omitempty"`
    TLS           bool            `json:"tls,omitempty"`
    Firmware      string          `json:"firmware,omitempty"`
    Online        bool            `json:"online"`
    Advertisement *Advertisement  `json:"advertisement,omitempty"`
}

type uploadPayload struct {
    ImageBase64 string `json:"image_base64"`
    Transport   string `json:"transport"`
    Host        string `json:"host,omitempty"`
    Port        int    `json:"port,omitempty"`
    TLS         bool   `json:"tls,omitempty"`
    Fit         string `json:"fit"`
    RefreshMode string `json:"refresh_mode"`
}

type Adapter struct {
    cfg      Config
    baseURL  string
    client   *http.Client
    devices  device.Repository
    entities entity.Repository
    bus      events.Bus
    logger   *slog.Logger

    mu       sync.RWMutex
    running  bool
    cancel   context.CancelFunc
    done     chan struct{}
    known    map[string]DeviceInfo
    lastScan time.Time
    lastErr  error
}

var _ interface {
    ID() string
    Type() string
    Start(context.Context) error
    Stop(context.Context) error
    HealthCheck(context.Context) error
} = (*Adapter)(nil)

func NewAdapter(cfg Config, devices device.Repository, entities entity.Repository, bus events.Bus, logger *slog.Logger) (*Adapter, error) {
    cfg.BridgeURL = strings.TrimRight(strings.TrimSpace(cfg.BridgeURL), "/")
    parsed, err := url.Parse(cfg.BridgeURL)
    if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" ||
        parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
        return nil, errors.New("open_display: bridge_url must be an absolute HTTP(S) URL without credentials, query or fragment")
    }
    cfg.Token = strings.TrimSpace(cfg.Token)
    if len(cfg.Token) < 32 || strings.ContainsAny(cfg.Token, "\r\n") {
        return nil, errors.New("open_display: bridge token must contain at least 32 characters")
    }
    if devices == nil || entities == nil {
        return nil, errors.New("open_display: device and entity repositories are required")
    }
    if cfg.PollInterval == 0 {
        cfg.PollInterval = 60 * time.Second
    }
    if cfg.PollInterval < 15*time.Second || cfg.PollInterval > 30*time.Minute {
        return nil, errors.New("open_display: poll_interval must be between 15s and 30m")
    }
    if cfg.ScanSeconds == 0 {
        cfg.ScanSeconds = 6
    }
    if cfg.ScanSeconds < 2 || cfg.ScanSeconds > 20 {
        return nil, errors.New("open_display: scan_seconds must be between 2 and 20")
    }
    if cfg.HTTPTimeout == 0 {
        cfg.HTTPTimeout = 45 * time.Second
    }
    if cfg.HTTPTimeout < time.Second || cfg.HTTPTimeout > 2*time.Minute {
        return nil, errors.New("open_display: http_timeout must be between 1s and 2m")
    }
    if logger == nil {
        logger = slog.Default()
    }
    return &Adapter{
        cfg: cfg, baseURL: cfg.BridgeURL, client: &http.Client{Timeout: cfg.HTTPTimeout},
        devices: devices, entities: entities, bus: bus, logger: logger,
        known: make(map[string]DeviceInfo),
    }, nil
}

func (a *Adapter) ID() string   { return IntegrationID }
func (a *Adapter) Type() string { return IntegrationID }

func (a *Adapter) Start(parent context.Context) error {
    a.mu.Lock()
    defer a.mu.Unlock()
    if a.running {
        return nil
    }
    ctx, cancel := context.WithCancel(parent)
    a.cancel = cancel
    a.done = make(chan struct{})
    a.running = true
    go a.run(ctx, a.done)
    a.logger.Info("OpenDisplay integration started", "integration_id", a.ID(), "bridge", a.baseURL)
    return nil
}

func (a *Adapter) Stop(ctx context.Context) error {
    a.mu.RLock()
    cancel, done := a.cancel, a.done
    a.mu.RUnlock()
    if cancel == nil {
        return nil
    }
    cancel()
    select {
    case <-done:
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}

func (a *Adapter) HealthCheck(ctx context.Context) error {
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.baseURL+"/health", nil)
    if err != nil {
        return err
    }
    resp, err := a.client.Do(req)
    if err != nil {
        return fmt.Errorf("open_display: bridge health check: %w", err)
    }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("open_display: bridge health returned HTTP %d", resp.StatusCode)
    }
    return nil
}

func (a *Adapter) run(ctx context.Context, done chan struct{}) {
    defer close(done)
    defer func() {
        a.mu.Lock()
        a.running = false
        a.cancel = nil
        a.mu.Unlock()
    }()
    ticker := time.NewTicker(a.cfg.PollInterval)
    defer ticker.Stop()
    for {
        scanCtx, cancel := context.WithTimeout(ctx, a.cfg.HTTPTimeout)
        err := a.DiscoverNow(scanCtx)
        cancel()
        if err != nil && ctx.Err() == nil {
            a.mu.Lock()
            a.lastErr = err
            a.mu.Unlock()
            a.logger.Warn("OpenDisplay discovery failed", "error", err, "retry_in", a.cfg.PollInterval)
        }
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
        }
    }
}

func (a *Adapter) request(ctx context.Context, method, path string, body []byte) (*http.Response, error) {
    req, err := http.NewRequestWithContext(ctx, method, a.baseURL+path, bytes.NewReader(body))
    if err != nil {
        return nil, err
    }
    req.Header.Set("Authorization", "Bearer "+a.cfg.Token)
    req.Header.Set("Accept", "application/json")
    if body != nil {
        req.Header.Set("Content-Type", "application/json")
    }
    resp, err := a.client.Do(req)
    if err != nil {
        return nil, err
    }
    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        message, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
        resp.Body.Close()
        return nil, fmt.Errorf("bridge returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(message)))
    }
    return resp, nil
}

func (a *Adapter) DiscoverNow(ctx context.Context) error {
    resp, err := a.request(ctx, http.MethodGet, "/api/v1/devices", nil)
    if err != nil {
        return fmt.Errorf("open_display: list devices: %w", err)
    }
    defer resp.Body.Close()

    var result struct {
        Devices []DeviceInfo `json:"devices"`
    }
    decoder := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes))
    if err := decoder.Decode(&result); err != nil {
        return fmt.Errorf("open_display: decode discovery response: %w", err)
    }
    seen := make(map[string]DeviceInfo, len(result.Devices))
    for _, info := range result.Devices {
        if err := validateDeviceInfo(info); err != nil {
            a.logger.Warn("ignoring invalid OpenDisplay discovery item", "error", err)
            continue
        }
        info.Name = strings.TrimSpace(info.Name)
        info.Address = strings.TrimSpace(info.Address)
        info.Online = true
        seen[info.Address] = info
        if err := a.upsertDevice(ctx, info); err != nil {
            a.logger.Warn("failed persisting OpenDisplay device", "address", info.Address, "error", err)
        }
    }

    a.mu.Lock()
    previous := make([]DeviceInfo, 0)
    for address, old := range a.known {
        if _, found := seen[address]; !found && old.Online {
            old.Online = false
            a.known[address] = old
            previous = append(previous, old)
        }
    }
    for address, info := range seen {
        a.known[address] = info
    }
    a.lastScan = time.Now().UTC()
    a.lastErr = nil
    a.mu.Unlock()

    for _, old := range previous {
        if err := a.markOffline(ctx, old); err != nil {
            a.logger.Warn("failed marking OpenDisplay device offline", "address", old.Address, "error", err)
        }
    }
    a.logger.Debug("OpenDisplay discovery completed", "devices", len(seen))
    return nil
}

func validateDeviceInfo(info DeviceInfo) error {
    if strings.TrimSpace(info.Name) == "" || len(info.Name) > 128 {
        return errors.New("invalid OpenDisplay name")
    }
    if strings.TrimSpace(info.Address) == "" || len(info.Address) > 128 {
        return errors.New("invalid OpenDisplay address")
    }
    if info.Transport != "ble" && info.Transport != "wifi" {
        return errors.New("invalid OpenDisplay transport")
    }
    if info.Transport == "wifi" && (info.Host == "" || info.Port < 1 || info.Port > 65535) {
        return errors.New("invalid OpenDisplay Wi-Fi address")
    }
    return nil
}

func safeEntitySuffix(address string) string {
    suffix := strings.ToLower(entityPartPattern.ReplaceAllString(address, "_"))
    suffix = strings.Trim(suffix, "_")
    if len(suffix) > 48 {
        suffix = suffix[len(suffix)-48:]
    }
    if suffix == "" {
        return "display"
    }
    return suffix
}

func (a *Adapter) upsertDevice(ctx context.Context, info DeviceInfo) error {
    now := time.Now().UTC()
    metadataBytes, _ := json.Marshal(info)
    dev, err := a.devices.GetByExternalID(ctx, a.ID(), info.Address)
    if err != nil {
        if !errors.Is(err, device.ErrDeviceNotFound) {
            return err
        }
        dev = &device.Device{
            ID: uuid.NewString(), IntegrationID: a.ID(), ExternalID: info.Address,
            Name: info.Name, Manufacturer: "OpenDisplay", Model: "E-paper display",
            Online: true, Metadata: string(metadataBytes), CreatedAt: now, UpdatedAt: now,
        }
        if err := a.devices.Create(ctx, dev); err != nil {
            return err
        }
    } else {
        dev.Name = info.Name
        dev.Manufacturer = "OpenDisplay"
        dev.Model = "E-paper display"
        dev.Metadata = string(metadataBytes)
        dev.UpdatedAt = now
        if err := a.devices.Update(ctx, dev); err != nil {
            return err
        }
        if !dev.Online {
            if err := a.devices.SetOnline(ctx, dev.ID, true); err != nil {
                return err
            }
        }
    }

    suffix := safeEntitySuffix(info.Address)
    if err := a.upsertConnectivity(ctx, dev.ID, suffix, true, info); err != nil {
        return err
    }
    if info.Advertisement != nil {
        if info.Advertisement.BatteryMV != nil {
            if err := a.upsertSensor(ctx, dev.ID, "sensor.opend_"+suffix+"_battery_voltage", "Battery voltage", fmt.Sprint(*info.Advertisement.BatteryMV), "mV", info); err != nil {
                return err
            }
        }
        if info.Advertisement.TemperatureC != nil {
            if err := a.upsertSensor(ctx, dev.ID, "sensor.opend_"+suffix+"_chip_temperature", "Chip temperature", fmt.Sprintf("%.1f", *info.Advertisement.TemperatureC), "°C", info); err != nil {
                return err
            }
        }
    }
    return nil
}

func (a *Adapter) upsertSensor(ctx context.Context, deviceID, key, name, state, unit string, info DeviceInfo) error {
    attrs, _ := json.Marshal(map[string]any{"transport": info.Transport, "firmware": info.Firmware, "manufacturer": "OpenDisplay"})
    return a.upsertEntity(ctx, &entity.Entity{
        ID: uuid.NewString(), DeviceID: deviceID, EntityKey: key, Domain: "sensor",
        Name: name, Icon: "mdi:display", UnitOfMeasurement: unit, Writable: false,
        State: state, Attributes: string(attrs), Available: true,
    })
}

func (a *Adapter) upsertConnectivity(ctx context.Context, deviceID, suffix string, online bool, info DeviceInfo) error {
    state := "off"
    if online {
        state = "on"
    }
    attrs, _ := json.Marshal(map[string]any{"transport": info.Transport, "address": info.Address})
    return a.upsertEntity(ctx, &entity.Entity{
        ID: uuid.NewString(), DeviceID: deviceID, EntityKey: "binary_sensor.opend_"+suffix+"_connectivity",
        Domain: "binary_sensor", Name: "OpenDisplay connectivity", Icon: "mdi:bluetooth-connect",
        Writable: false, State: state, Attributes: string(attrs), Available: true,
    })
}

func (a *Adapter) upsertEntity(ctx context.Context, next *entity.Entity) error {
    now := time.Now().UTC()
    existing, err := a.entities.GetByKey(ctx, next.EntityKey)
    if err != nil {
        if !errors.Is(err, entity.ErrEntityNotFound) {
            return err
        }
        next.LastChangedAt, next.LastUpdatedAt = now, now
        if err := a.entities.Create(ctx, next); err != nil {
            return err
        }
        return a.publishState(ctx, next)
    }
    existing.Attributes = next.Attributes
    existing.Available = next.Available
    existing.Name = next.Name
    existing.Icon = next.Icon
    existing.UnitOfMeasurement = next.UnitOfMeasurement
    if err := a.entities.UpdateState(ctx, existing.ID, next.State, next.Attributes, next.Available, now); err != nil {
        return err
    }
    if existing.State != next.State {
        a.logger.Debug("OpenDisplay entity state changed", "entity_key", next.EntityKey, "from", existing.State, "to", next.State)
    }
    existing.State = next.State
    existing.LastUpdatedAt = now
    if existing.State != next.State {
        existing.LastChangedAt = now
    }
    return a.publishState(ctx, existing)
}

func (a *Adapter) publishState(ctx context.Context, ent *entity.Entity) error {
    if a.bus == nil {
        return nil
    }
    return a.bus.Publish(ctx, events.Event{
        ID: uuid.NewString(), Type: events.EntityStateChanged, OccurredAt: time.Now().UTC(), Source: IntegrationID,
        Payload: map[string]any{
            "entity_id": ent.ID, "entity_key": ent.EntityKey, "state": ent.State,
            "attributes": ent.Attributes, "available": ent.Available,
        },
    })
}

func (a *Adapter) markOffline(ctx context.Context, info DeviceInfo) error {
    dev, err := a.devices.GetByExternalID(ctx, a.ID(), info.Address)
    if err != nil {
        if errors.Is(err, device.ErrDeviceNotFound) {
            return nil
        }
        return err
    }
    if err := a.devices.SetOnline(ctx, dev.ID, false); err != nil {
        return err
    }
    return a.upsertConnectivity(ctx, dev.ID, safeEntitySuffix(info.Address), false, info)
}

func (a *Adapter) ListDevices() []DeviceInfo {
    a.mu.RLock()
    defer a.mu.RUnlock()
    list := make([]DeviceInfo, 0, len(a.known))
    for _, info := range a.known {
        list = append(list, info)
    }
    return list
}

func (a *Adapter) Device(address string) (DeviceInfo, bool) {
    a.mu.RLock()
    defer a.mu.RUnlock()
    info, ok := a.known[address]
    return info, ok
}

func (a *Adapter) UploadImage(ctx context.Context, address string, image []byte, fit, refreshMode string) error {
    if len(image) == 0 || len(image) > maxImageBytes {
        return fmt.Errorf("open_display: image must be between 1 byte and %d bytes", maxImageBytes)
    }
    info, ok := a.Device(address)
    if !ok || !info.Online {
        return fmt.Errorf("open_display: device is not currently discovered")
    }
    if fit == "" {
        fit = "contain"
    }
    if refreshMode == "" {
        refreshMode = "full"
    }
    switch fit {
    case "contain", "cover", "stretch", "crop":
    default:
        return errors.New("open_display: invalid image fit mode")
    }
    switch refreshMode {
    case "full", "fast":
    default:
        return errors.New("open_display: invalid refresh mode")
    }
    payload := uploadPayload{
        ImageBase64: base64.StdEncoding.EncodeToString(image),
        Transport: info.Transport, Host: info.Host, Port: info.Port, TLS: info.TLS,
        Fit: fit, RefreshMode: refreshMode,
    }
    body, err := json.Marshal(payload)
    if err != nil {
        return err
    }
    path := "/api/v1/devices/" + url.PathEscape(address) + "/image"
    resp, err := a.request(ctx, http.MethodPost, path, body)
    if err != nil {
        return fmt.Errorf("open_display: upload failed: %w", err)
    }
    defer resp.Body.Close()
    return nil
}

func RegisterPlugin(registry interface {
    Register(plugins.Manifest, plugins.Factory) error
}) error {
    return registry.Register(plugins.Manifest{
        APIVersion: plugins.ManifestAPIVersion,
        ID: IntegrationID, Name: "OpenDisplay e-paper displays", Version: "1.0.0",
        Description: "Bluetooth LE and optional Wi-Fi e-paper displays through the isolated py-opendisplay companion.",
        Capabilities: []plugins.Capability{
            plugins.CapabilityDevicesRead, plugins.CapabilityDevicesWrite, plugins.CapabilityNetworkLAN,
        },
        Config: []plugins.ConfigField{
            {Name: "bridge_url", Type: "url", Required: true, Description: "Local OpenDisplay bridge URL"},
            {Name: "token", Type: "string", Required: true, Secret: true, Description: "Bearer token for the bridge"},
            {Name: "poll_interval", Type: "duration", Description: "Discovery interval"},
            {Name: "scan_seconds", Type: "integer", Description: "BLE scan duration in seconds"},
        },
    }, func(ctx context.Context, instanceID string, raw json.RawMessage, deps plugins.Dependencies) (plugins.Adapter, error) {
        cfg, err := plugins.DecodeConfig[Config](raw)
        if err != nil {
            return nil, err
        }
        if instanceID != IntegrationID {
            return nil, fmt.Errorf("open_display: instance ID must be %q", IntegrationID)
        }
        return NewAdapter(cfg, deps.Devices, deps.Entities, deps.Bus, deps.Logger)
    })
}
