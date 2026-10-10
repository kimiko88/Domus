package brand

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/device"
	"github.com/domus-hub/domus/internal/domain/entity"
	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/domain/integration"
	"github.com/google/uuid"
)

var keySanitizer = regexp.MustCompile(`[^a-zA-Z0-9_]`)

// Service manages smart home brand catalog, auto-discovery, zero-cloud pairing, and direct HTTP/RPC command execution.
type Service struct {
	deviceRepo      device.Repository
	entityRepo      entity.Repository
	integrationRepo integration.Repository
	bus             events.Bus
	scanner         *Scanner
	httpClient      *http.Client

	mu                sync.RWMutex
	discoveredDevices map[string]*DiscoveredDevice
	pairedDevices     map[string]*PairedDeviceRuntime // Keyed by entity_key
}

// NewService constructs a brand management and auto-discovery service.
func NewService(
	devRepo device.Repository,
	entRepo entity.Repository,
	intRepo integration.Repository,
	bus events.Bus,
	client *http.Client,
) *Service {
	if client == nil {
		client = &http.Client{
			Timeout: 3 * time.Second,
		}
	}

	s := &Service{
		deviceRepo:        devRepo,
		entityRepo:        entRepo,
		integrationRepo:   intRepo,
		bus:               bus,
		scanner:           NewScanner(client),
		httpClient:        client,
		discoveredDevices: make(map[string]*DiscoveredDevice),
		pairedDevices:     make(map[string]*PairedDeviceRuntime),
	}

	if bus != nil {
		bus.Subscribe(events.CommandRequested, s.handleCommandEvent)
	}

	return s
}

// GetCatalog returns the full brand ecosystem catalog.
func (s *Service) GetCatalog() []BrandCatalogEntry {
	return DefaultCatalog()
}

// ListDiscovered returns all currently discovered brand devices.
func (s *Service) ListDiscovered() []*DiscoveredDevice {
	s.mu.RLock()
	defer s.mu.RUnlock()

	res := make([]*DiscoveredDevice, 0, len(s.discoveredDevices))
	for _, d := range s.discoveredDevices {
		res = append(res, d)
	}
	return res
}

// ProbeIP probes a specific IP address and registers the discovered device.
func (s *Service) ProbeIP(ctx context.Context, ip string, brandHint BrandIdentifier) (*DiscoveredDevice, error) {
	dev, err := s.scanner.ProbeIP(ctx, ip, brandHint)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.discoveredDevices[dev.ID] = dev
	s.mu.Unlock()

	return dev, nil
}

// Scan probes target addresses or local subnet IPs.
func (s *Service) Scan(ctx context.Context, target string) ([]*DiscoveredDevice, error) {
	target = strings.TrimSpace(target)
	if target != "" {
		dev, err := s.ProbeIP(ctx, target, "")
		if err != nil {
			return nil, err
		}
		return []*DiscoveredDevice{dev}, nil
	}

	s.mu.RLock()
	current := s.ListDiscovered()
	s.mu.RUnlock()
	return current, nil
}

// PairDevice provisions a discovered brand device into Domus Device and Entity repositories with 1-click.
func (s *Service) PairDevice(ctx context.Context, req PairRequest) (*device.Device, []*entity.Entity, error) {
	if req.IP == "" {
		return nil, nil, fmt.Errorf("brand: target ip is required")
	}

	// 1. Identify device if not already discovered
	s.mu.RLock()
	var disc *DiscoveredDevice
	for _, d := range s.discoveredDevices {
		if d.IP == req.IP {
			disc = d
			break
		}
	}
	s.mu.RUnlock()

	if disc == nil {
		var err error
		disc, err = s.scanner.ProbeIP(ctx, req.IP, req.Brand)
		if err != nil {
			return nil, nil, fmt.Errorf("brand: failed to probe device at %s: %w", req.IP, err)
		}
		s.mu.Lock()
		s.discoveredDevices[disc.ID] = disc
		s.mu.Unlock()
	}

	name := req.Name
	if name == "" {
		name = disc.Name
	}

	// 2. Ensure brand integration record exists in integration repository
	intID := fmt.Sprintf("integration_%s_direct", disc.Brand)
	if s.integrationRepo != nil {
		_, err := s.integrationRepo.GetByID(ctx, intID)
		if err != nil {
			// Create brand integration
			now := time.Now().UTC()
			_ = s.integrationRepo.Create(ctx, &integration.Integration{
				ID:        intID,
				Type:      string(disc.Brand),
				Name:      fmt.Sprintf("%s Direct Local", strings.ToUpper(string(disc.Brand))),
				Enabled:   true,
				Status:    "connected",
				CreatedAt: now,
				UpdatedAt: now,
			})
		}
	}

	// 3. Create Device record
	devID := uuid.NewString()
	extID := disc.MAC
	if extID == "" {
		extID = fmt.Sprintf("%s_%s", disc.Brand, disc.IP)
	}

	metaJSON, _ := json.Marshal(map[string]any{
		"ip":         disc.IP,
		"port":       disc.Port,
		"brand":      disc.Brand,
		"model":      disc.Model,
		"firmware":   disc.Firmware,
		"auth_token": req.AuthToken,
	})

	now := time.Now().UTC()
	dev := &device.Device{
		ID:            devID,
		IntegrationID: intID,
		ExternalID:    extID,
		Name:          name,
		Manufacturer:  string(disc.Brand),
		Model:         disc.Model,
		AreaID:        req.AreaID,
		Online:        true,
		Metadata:      string(metaJSON),
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if s.deviceRepo != nil {
		if err := s.deviceRepo.Create(ctx, dev); err != nil {
			return nil, nil, fmt.Errorf("brand: failed to persist device: %w", err)
		}
	}

	// 4. Create Entities based on suggested entities
	cleanBaseName := keySanitizer.ReplaceAllString(strings.ToLower(name), "_")
	var createdEntities []*entity.Entity
	var runtimeKeys []string

	for _, sug := range disc.SuggestedEntities {
		entID := uuid.NewString()
		entKey := fmt.Sprintf("%s.%s_%s", sug.Domain, cleanBaseName, sug.KeySuffix)

		initState := `{"state":false}`
		if sug.Domain == "sensor" {
			initState = `{"value":0.0}`
		}

		ent := &entity.Entity{
			ID:                entID,
			DeviceID:          devID,
			EntityKey:         entKey,
			Domain:            sug.Domain,
			Name:              fmt.Sprintf("%s %s", name, sug.Name),
			Icon:              sug.Icon,
			UnitOfMeasurement: sug.UnitOfMeasurement,
			Writable:          sug.Writable,
			State:             initState,
			Attributes:        `{}`,
			Available:         true,
			AreaID:            req.AreaID,
			LastChangedAt:     now,
			LastUpdatedAt:     now,
		}

		if s.entityRepo != nil {
			if err := s.entityRepo.Create(ctx, ent); err == nil {
				createdEntities = append(createdEntities, ent)
				runtimeKeys = append(runtimeKeys, entKey)
			}
		}
	}

	// 5. Store in runtime map for command routing
	runtime := &PairedDeviceRuntime{
		DeviceID:   devID,
		Brand:      disc.Brand,
		IP:         disc.IP,
		Port:       disc.Port,
		Model:      disc.Model,
		AuthToken:  req.AuthToken,
		EntityKeys: runtimeKeys,
	}

	s.mu.Lock()
	for _, k := range runtimeKeys {
		s.pairedDevices[k] = runtime
	}
	disc.Status = "paired"
	s.mu.Unlock()

	return dev, createdEntities, nil
}

// SendCommand executes direct HTTP/RPC command against paired smart home brand hardware.
func (s *Service) SendCommand(ctx context.Context, entityKey string, service string, params map[string]any) error {
	s.mu.RLock()
	runtime, ok := s.pairedDevices[entityKey]
	s.mu.RUnlock()

	if !ok || runtime == nil {
		return fmt.Errorf("brand: no paired runtime device found for entity %s", entityKey)
	}

	turnOn := service == "turn_on"
	if service == "toggle" {
		turnOn = true
	}

	switch runtime.Brand {
	case BrandShelly:
		return s.sendShellyCommand(ctx, runtime, turnOn)
	case BrandWLED:
		return s.sendWLEDCommand(ctx, runtime, turnOn, params)
	case BrandTasmota:
		return s.sendTasmotaCommand(ctx, runtime, turnOn)
	case BrandSonoff:
		return s.sendSonoffCommand(ctx, runtime, turnOn)
	case BrandHue:
		return s.sendHueCommand(ctx, runtime, turnOn, params)
	default:
		return fmt.Errorf("brand: unsupported brand command %s", runtime.Brand)
	}
}

func (s *Service) sendShellyCommand(ctx context.Context, runtime *PairedDeviceRuntime, turnOn bool) error {
	// Try Gen2/Gen3 RPC Switch.Set first
	rpcURL := fmt.Sprintf("http://%s/rpc/Switch.Set", runtime.IP)
	rpcPayload, _ := json.Marshal(map[string]any{
		"id": 0,
		"on": turnOn,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rpcURL, bytes.NewReader(rpcPayload))
	if err == nil {
		req.Header.Set("Content-Type", "application/json")
		resp, err := s.httpClient.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			_ = resp.Body.Close()
			return nil
		}
		if resp != nil {
			_ = resp.Body.Close()
		}
	}

	// Fallback Gen1 HTTP relay/0
	stateStr := "off"
	if turnOn {
		stateStr = "on"
	}
	gen1URL := fmt.Sprintf("http://%s/relay/0?turn=%s", runtime.IP, stateStr)
	reqGen1, err := http.NewRequestWithContext(ctx, http.MethodGet, gen1URL, nil)
	if err != nil {
		return err
	}
	respGen1, err := s.httpClient.Do(reqGen1)
	if err != nil {
		return err
	}
	defer respGen1.Body.Close()

	if respGen1.StatusCode != http.StatusOK {
		return fmt.Errorf("shelly responded with status %d", respGen1.StatusCode)
	}
	return nil
}

func (s *Service) sendWLEDCommand(ctx context.Context, runtime *PairedDeviceRuntime, turnOn bool, params map[string]any) error {
	url := fmt.Sprintf("http://%s/json/state", runtime.IP)
	body := map[string]any{
		"on": turnOn,
	}
	if b, ok := params["brightness"]; ok {
		body["bri"] = b
	}

	payload, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (s *Service) sendTasmotaCommand(ctx context.Context, runtime *PairedDeviceRuntime, turnOn bool) error {
	cmd := "Power%20OFF"
	if turnOn {
		cmd = "Power%20ON"
	}
	url := fmt.Sprintf("http://%s/cm?cmnd=%s", runtime.IP, cmd)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (s *Service) sendSonoffCommand(ctx context.Context, runtime *PairedDeviceRuntime, turnOn bool) error {
	sw := "off"
	if turnOn {
		sw = "on"
	}
	url := fmt.Sprintf("http://%s:8081/zeroconf/switch", runtime.IP)
	payload, _ := json.Marshal(map[string]any{
		"data": map[string]any{
			"switch": sw,
		},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (s *Service) sendHueCommand(ctx context.Context, runtime *PairedDeviceRuntime, turnOn bool, params map[string]any) error {
	url := fmt.Sprintf("http://%s/api/%s/groups/0/action", runtime.IP, runtime.AuthToken)
	body := map[string]any{
		"on": turnOn,
	}
	if b, ok := params["brightness"]; ok {
		body["bri"] = b
	}
	payload, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (s *Service) handleCommandEvent(ctx context.Context, evt events.Event) error {
	payload, ok := evt.Payload.(map[string]any)
	if !ok {
		return nil
	}

	entityKey, _ := payload["entity_key"].(string)
	service, _ := payload["service"].(string)
	params, _ := payload["params"].(map[string]any)

	if entityKey == "" || service == "" {
		return nil
	}

	s.mu.RLock()
	_, isPaired := s.pairedDevices[entityKey]
	s.mu.RUnlock()

	if !isPaired {
		return nil
	}

	slog.Info("brand: executing command for device entity", "entity_key", entityKey, "service", service)
	if err := s.SendCommand(ctx, entityKey, service, params); err != nil {
		slog.Warn("brand: failed to execute hardware command", "entity_key", entityKey, "error", err)
		return err
	}

	// Update entity state in repository
	if s.entityRepo != nil {
		ent, err := s.entityRepo.GetByKey(ctx, entityKey)
		if err == nil && ent != nil {
			newState := `{"state":true}`
			if service == "turn_off" {
				newState = `{"state":false}`
			}
			now := time.Now().UTC()
			_ = s.entityRepo.UpdateState(ctx, ent.ID, newState, ent.Attributes, true, now)

			if s.bus != nil {
				_ = s.bus.Publish(ctx, events.Event{
					ID:         uuid.NewString(),
					Type:       events.EntityStateChanged,
					OccurredAt: now,
					Source:     "brand.service",
					Payload: map[string]any{
						"entity_id":  ent.ID,
						"entity_key": ent.EntityKey,
						"state":      newState,
					},
				})
			}
		}
	}

	return nil
}
