package handlers

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/device"
	"github.com/domus-hub/domus/internal/domain/entity"
	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/domain/integration"
	"github.com/domus-hub/domus/internal/infrastructure/crypto"
	"github.com/domus-hub/domus/internal/plugins/midea"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

const mideaIntegrationID = "integration_midea_smarthome"

type MideaHandler struct {
	integrations integration.Repository
	cipher       *crypto.EnvelopeCipher
	devices      device.Repository
	entities     entity.Repository
	bus          events.Bus
	adapter      *midea.Adapter
	driverV2     *midea.A1LANDriver
	driverV3     *midea.A1LANDriver
	mu           sync.Mutex
	config       mideaStoredConfig
	discovered   map[string]midea.DiscoveredAppliance
	discover     func(context.Context) ([]midea.DiscoveredAppliance, error)
	unsubscribe  func()
}

type mideaStoredConfig struct {
	Devices []midea.A1LANConfig `json:"devices"`
}

type mideaConfigureRequest struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Brand          string `json:"brand,omitempty"`
	Model          string `json:"model,omitempty"`
	TypeCode       string `json:"type_code"`
	Host           string `json:"host"`
	Port           int    `json:"port"`
	Protocol       string `json:"protocol"`
	MessageVersion uint8  `json:"message_version,omitempty"`
	ApplianceID    uint64 `json:"appliance_id,omitempty"`
	Token          string `json:"token,omitempty"`
	Key            string `json:"key,omitempty"`
}

type mideaCommandRequest struct {
	Values map[string]any `json:"values,omitempty"`
	Action string         `json:"action,omitempty"`
}

func NewMideaHandler(
	integrations integration.Repository,
	cipher *crypto.EnvelopeCipher,
	devices device.Repository,
	entities entity.Repository,
) (*MideaHandler, error) {
	if integrations == nil || cipher == nil || devices == nil || entities == nil {
		return nil, errors.New("midea: integration repository, encryption, device and entity repositories are required")
	}
	driverV2, err := midea.NewA1LANDriver(midea.ProtocolMSmartV2)
	if err != nil {
		return nil, err
	}
	driverV3, err := midea.NewA1LANDriver(midea.ProtocolMSmartV3)
	if err != nil {
		return nil, err
	}
	handler := &MideaHandler{
		integrations: integrations,
		cipher: cipher,
		devices: devices,
		entities: entities,
		driverV2: driverV2,
		driverV3: driverV3,
		adapter: midea.NewAdapter(driverV2, driverV3),
		discovered: make(map[string]midea.DiscoveredAppliance),
		discover: midea.DiscoverLAN,
	}
	if err := handler.loadStoredConfig(context.Background()); err != nil {
		return nil, err
	}
	return handler, nil
}


 // WithRuntimeAdapter replaces the handler's private protocol engine with the
 // manager-owned plugin engine and restores the decrypted device configuration.
func (h *MideaHandler) WithRuntimeAdapter(runtime *midea.RuntimeAdapter) error {
	if runtime == nil { return errors.New("midea: plugin runtime adapter is required") }
	core, err := runtime.Core()
	if err != nil { return err }
	driverV2, driverV3 := runtime.Drivers()
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, stored := range h.config.Devices {
		var driver *midea.A1LANDriver
		switch stored.Device.Protocol {
		case midea.ProtocolMSmartV2: driver = driverV2
		case midea.ProtocolMSmartV3: driver = driverV3
		default: return fmt.Errorf("midea: unsupported stored protocol %q", stored.Device.Protocol)
		}
		if err := driver.Register(stored); err != nil { return fmt.Errorf("midea: restore plugin device %q: %w", stored.Device.ID, err) }
		if err := core.Register(stored.Device); err != nil { return fmt.Errorf("midea: restore plugin adapter device %q: %w", stored.Device.ID, err) }
	}
	h.adapter, h.driverV2, h.driverV3 = core, driverV2, driverV3
	return nil
}

// WithEventBus publishes verified device state to the shared automation and WebSocket event pipeline.
func (h *MideaHandler) WithEventBus(bus events.Bus) *MideaHandler {
	h.mu.Lock()
	if h.unsubscribe != nil {
		h.unsubscribe()
		h.unsubscribe = nil
	}
	h.bus = bus
	if bus != nil {
		h.unsubscribe = bus.Subscribe(events.CommandRequested, h.handleCommandEvent)
	}
	h.mu.Unlock()
	return h
}

// handleCommandEvent routes standard Domus entity commands to the owned
// dehumidifier entity. Other integrations and telemetry entities are ignored.
func (h *MideaHandler) handleCommandEvent(ctx context.Context, event events.Event) error {
	payload, err := json.Marshal(event.Payload)
	if err != nil {
		return nil
	}
	var request struct {
		EntityID  string         `json:"entity_id"`
		EntityKey string         `json:"entity_key"`
		Service   string         `json:"service"`
		Params    map[string]any `json:"params"`
	}
	if err := json.Unmarshal(payload, &request); err != nil {
		return nil
	}
	if request.EntityKey == "" && request.EntityID != "" {
		item, lookupErr := h.entities.GetByID(ctx, request.EntityID)
		if lookupErr != nil || item == nil {
			return nil
		}
		request.EntityKey = item.EntityKey
	}
	if request.EntityKey == "" {
		return nil
	}
	entityItem, err := h.entities.GetByKey(ctx, request.EntityKey)
	if err != nil || entityItem == nil || entityItem.Domain != "humidifier" || entityItem.DeviceID == "" {
		return nil
	}
	ownedDevice, err := h.devices.GetByID(ctx, entityItem.DeviceID)
	if err != nil || ownedDevice == nil || ownedDevice.IntegrationID != mideaIntegrationID {
		return nil
	}

	params := request.Params
	if params == nil {
		params = make(map[string]any)
	}
	command := midea.Command{Values: make(map[string]any)}
	switch request.Service {
	case "turn_on":
		command.Values["power"] = true
	case "turn_off":
		command.Values["power"] = false
	case "set_humidity":
		value, ok := params["humidity"]
		if !ok { value = params["target_humidity"] }
		if value == nil {
			return errors.New("midea: set_humidity requires humidity or target_humidity")
		}
		command.Values["target_humidity"] = value
	case "set_mode":
		value, ok := params["mode"]
		if !ok || value == nil {
			return errors.New("midea: set_mode requires mode")
		}
		command.Values["mode"] = value
	case "set_fan_mode", "set_fan_speed":
		value, ok := params["fan_mode"]
		if !ok { value = params["fan_speed"] }
		if value == nil {
			return errors.New("midea: set_fan_mode requires fan_mode or fan_speed")
		}
		command.Values["fan_speed"] = value
	case "reset_filter":
		command.Action = "reset_filter"
	default:
		return fmt.Errorf("midea: unsupported Domus humidifier service %q", request.Service)
	}
	if len(command.Values) == 0 && command.Action == "" {
		return errors.New("midea: empty Domus command")
	}
	state, commandErr := h.adapter.Command(ctx, ownedDevice.ExternalID, command)
	if commandErr != nil {
		if h.bus != nil {
			_ = h.bus.Publish(ctx, events.Event{
				ID: uuid.NewString(), Type: events.CommandFailed, OccurredAt: time.Now().UTC(),
				Source: "midea_smarthome",
				Payload: map[string]any{"entity_key": request.EntityKey, "service": request.Service, "error": safeMideaError(commandErr)},
			})
		}
		return commandErr
	}
	if err := h.syncEntities(ctx, ownedDevice.ExternalID, state); err != nil {
		return err
	}
	if h.bus != nil {
		_ = h.bus.Publish(ctx, events.Event{
			ID: uuid.NewString(), Type: events.CommandSucceeded, OccurredAt: time.Now().UTC(),
			Source: "midea_smarthome",
			Payload: map[string]any{"entity_key": request.EntityKey, "service": request.Service},
		})
	}
	return nil
}

func (h *MideaHandler) RegisterRoutes(r chi.Router) {
	r.Get("/status", h.Status)
	r.Post("/discovery/scan", h.Scan)
	r.Get("/devices", h.Devices)
	r.Post("/devices", h.ConfigureDevice)
	r.Post("/devices/{id}/refresh", h.RefreshDevice)
	r.Post("/devices/{id}/commands", h.CommandDevice)
	r.Delete("/devices/{id}", h.DeleteDevice)
}

func (h *MideaHandler) Status(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store, private")
	h.mu.Lock()
	count := len(h.config.Devices)
	h.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{
		"configured": count > 0,
		"mode": "local",
		"device_count": count,
		"supported_types": []string{"A1"},
		"pending_drivers": []string{"AC", "AD", "B0", "B1", "B3", "B4", "B6", "B8", "BF", "C2", "C3", "CA", "CC", "CD", "CE", "CF", "DA", "DB", "DC", "E1", "E2", "E3", "E6", "E8", "EA", "EC", "ED", "FA", "FB", "FC", "FD"},
	})
}

func (h *MideaHandler) Scan(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store, private")
	ctx, cancel := context.WithTimeout(r.Context(), 7*time.Second)
	defer cancel()
	started := time.Now()
	devices, err := h.discover(ctx)
	if err != nil && len(devices) == 0 {
		writeError(w, http.StatusBadGateway, "MIDEA_DISCOVERY_FAILED", "M-Smart LAN discovery failed; check that Domus and the appliance are on the same local network")
		return
	}
	configured := make(map[string]struct{})
	h.mu.Lock()
	h.discovered = make(map[string]midea.DiscoveredAppliance, len(devices))
	for _, item := range h.config.Devices {
		configured[item.Device.ID] = struct{}{}
	}
	for _, item := range devices {
		h.discovered[item.ID] = item
	}
	h.mu.Unlock()
	for i := range devices {
		if _, ok := configured[devices[i].ID]; ok {
			devices[i].Reason = "Already configured in Domus"
			devices[i].Supported = true
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"devices": devices,
		"count": len(devices),
		"supported_count": countSupportedMideaDevices(devices),
		"duration_ms": time.Since(started).Milliseconds(),
	})
}

func countSupportedMideaDevices(devices []midea.DiscoveredAppliance) int {
	count := 0
	for _, item := range devices {
		if item.Supported {
			count++
		}
	}
	return count
}

func (h *MideaHandler) Devices(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store, private")
	configured := h.adapter.ListDevices()
	writeJSON(w, http.StatusOK, map[string]any{
		"devices": configured,
		"count": len(configured),
		"mode": "local",
	})
}

func (h *MideaHandler) ConfigureDevice(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store, private")
	var req mideaConfigureRequest
	if !decodeLimitedJSON(w, r, &req, 16*1024) {
		return
	}
	req.ID = strings.TrimSpace(req.ID)
	req.Name = strings.TrimSpace(req.Name)
	req.Brand = strings.TrimSpace(req.Brand)
	req.Model = strings.TrimSpace(req.Model)
	req.TypeCode = strings.ToUpper(strings.TrimSpace(req.TypeCode))
	req.Host = strings.TrimSpace(req.Host)
	req.Protocol = strings.TrimSpace(req.Protocol)
	if req.ID == "" || req.Name == "" || req.TypeCode == "" {
		writeError(w, http.StatusBadRequest, "INVALID_DEVICE", "A discovered device ID, name and type code are required")
		return
	}
	h.mu.Lock()
	discovered, wasDiscovered := h.discovered[req.ID]
	h.mu.Unlock()
	if !wasDiscovered {
		writeError(w, http.StatusConflict, "DEVICE_NOT_DISCOVERED", "Run the local M-Smart discovery scan before registering this appliance")
		return
	}
	if req.TypeCode == "" {
		req.TypeCode = discovered.TypeCode
	}
	if req.Host == "" {
		req.Host = discovered.Host
	}
	if req.Port == 0 {
		req.Port = discovered.Port
	}
	if req.Protocol == "" {
		req.Protocol = string(discovered.Protocol)
	}
	if discovered.TypeCode != req.TypeCode || discovered.Host != req.Host ||
		discovered.Port != req.Port || discovered.Protocol != midea.Protocol(req.Protocol) {
		writeError(w, http.StatusConflict, "DISCOVERY_MISMATCH", "The device ID, local address, port, model type and protocol must match the last local discovery result")
		return
	}
	if req.TypeCode != "A1" {
		writeError(w, http.StatusUnprocessableEntity, "UNSUPPORTED_DEVICE_TYPE", "This Domus release can control M-Smart A1 dehumidifiers. Other discovered models remain visible until their protocol driver is implemented.")
		return
	}
	if req.Port < 1 || req.Port > 65535 || !isPrivateMideaHost(req.Host) {
		writeError(w, http.StatusBadRequest, "INVALID_LAN_ENDPOINT", "A private local IP address and a valid TCP port are required")
		return
	}
	protocol := midea.Protocol(req.Protocol)
	if protocol != midea.ProtocolMSmartV2 && protocol != midea.ProtocolMSmartV3 {
		writeError(w, http.StatusBadRequest, "INVALID_PROTOCOL", "Choose M-Smart V2 or V3 as reported by local discovery")
		return
	}
	if protocol == midea.ProtocolMSmartV3 {
		if len(req.Token) != 128 || len(req.Key) != 64 {
			writeError(w, http.StatusBadRequest, "INVALID_MIDEA_CREDENTIALS", "M-Smart V3 requires a 64-byte token and 32-byte key, both encoded as hexadecimal")
			return
		}
		if _, err := hex.DecodeString(req.Token); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_MIDEA_CREDENTIALS", "The M-Smart V3 token must be hexadecimal")
			return
		}
		if _, err := hex.DecodeString(req.Key); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_MIDEA_CREDENTIALS", "The M-Smart V3 key must be hexadecimal")
			return
		}
	}
	applianceID, err := strconv.ParseUint(req.ID, 10, 64)
	if err != nil || applianceID == 0 {
		writeError(w, http.StatusBadRequest, "INVALID_DEVICE_ID", "The discovered Midea appliance ID must be a non-zero numeric value")
		return
	}
	if req.ApplianceID != 0 && req.ApplianceID != applianceID {
		writeError(w, http.StatusBadRequest, "INVALID_DEVICE_ID", "The appliance ID must match the discovered device ID")
		return
	}
	device := midea.Device{
		ID: req.ID, Name: req.Name, Brand: req.Brand, Model: req.Model,
		TypeCode: req.TypeCode, Kind: midea.KindDehumidifier, Protocol: protocol,
		ProtocolVersion: strconv.Itoa(int(req.MessageVersion)),
		Host: req.Host, Port: req.Port,
	}
	profile, ok := midea.NewRegistry().Resolve("A1")
	if !ok {
		writeError(w, http.StatusInternalServerError, "PROFILE_UNAVAILABLE", "The A1 dehumidifier profile is not available")
		return
	}
	device.Capabilities = append([]midea.Capability(nil), profile.Capabilities...)
	device.Writable = append([]midea.Capability(nil), profile.Writable...)
	device.Modes = append([]string(nil), profile.Modes...)
	device.FanSpeeds = append([]string(nil), profile.FanSpeeds...)
	device.Actions = append([]string(nil), profile.Actions...)
	device.MinHumidity, device.MaxHumidity = profile.MinHumidity, profile.MaxHumidity
	config := midea.A1LANConfig{
		Device: device, ApplianceID: applianceID, MessageVersion: req.MessageVersion,
		TokenHex: strings.ToLower(req.Token), KeyHex: strings.ToLower(req.Key),
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, exists := h.findStoredConfigLocked(req.ID); exists {
		writeError(w, http.StatusConflict, "DEVICE_ALREADY_CONFIGURED", "This Midea device is already configured in Domus")
		return
	}
	driver := h.driverFor(protocol)
	if driver == nil {
		writeError(w, http.StatusBadRequest, "UNSUPPORTED_PROTOCOL", "No driver is registered for this M-Smart protocol")
		return
	}
	if err := driver.Register(config); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_DEVICE_CONFIGURATION", safeMideaError(err))
		return
	}
	if err := h.adapter.Register(device); err != nil {
		driver.Remove(req.ID)
		writeError(w, http.StatusBadRequest, "INVALID_DEVICE_CONFIGURATION", safeMideaError(err))
		return
	}
	h.config.Devices = append(h.config.Devices, config)
	if err := h.saveConfigLocked(r.Context(), "connected"); err != nil {
		h.config.Devices = removeStoredMideaConfig(h.config.Devices, req.ID)
		driver.Remove(req.ID)
		h.adapter.Unregister(req.ID)
		writeError(w, http.StatusInternalServerError, "STORAGE_ERROR", "Could not securely save the Midea device credentials")
		return
	}
	domusDevice, err := h.provisionDevice(r.Context(), config)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DEVICE_PROVISION_FAILED", "The Midea connection was saved but the Domus device registry could not be updated")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"device": device, "domus_device": domusDevice, "configured": true, "mode": "local",
	})
}

func (h *MideaHandler) RefreshDevice(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store, private")
	id := chi.URLParam(r, "id")
	state, err := h.adapter.Refresh(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusBadGateway, "MIDEA_REFRESH_FAILED", safeMideaError(err))
		return
	}
	if err := h.syncEntities(r.Context(), id, state); err != nil {
		writeError(w, http.StatusInternalServerError, "MIDEA_STATE_SYNC_FAILED", "The device answered but its state could not be synchronized with Domus entities")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"device_id": id, "state": state, "refreshed_at": time.Now().UTC()})
}

func (h *MideaHandler) CommandDevice(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store, private")
	var req mideaCommandRequest
	if !decodeLimitedJSON(w, r, &req, 8*1024) {
		return
	}
	id := chi.URLParam(r, "id")
	state, err := h.adapter.Command(r.Context(), id, midea.Command{Values: req.Values, Action: req.Action})
	if err != nil {
		status := http.StatusUnprocessableEntity
		if errors.Is(err, midea.ErrDeviceNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, "MIDEA_COMMAND_FAILED", safeMideaError(err))
		return
	}
	if err := h.syncEntities(r.Context(), id, state); err != nil {
		writeError(w, http.StatusInternalServerError, "MIDEA_STATE_SYNC_FAILED", "The command was verified but its state could not be synchronized with Domus entities")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"device_id": id, "state": state, "executed_at": time.Now().UTC(), "verified": true,
	})
}

func (h *MideaHandler) DeleteDevice(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store, private")
	id := chi.URLParam(r, "id")
	h.mu.Lock()
	defer h.mu.Unlock()
	stored, exists := h.findStoredConfigLocked(id)
	if !exists {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Midea device was not found")
		return
	}
	next := removeStoredMideaConfig(h.config.Devices, id)
	previous := h.config.Devices
	h.config.Devices = next
	status := "connected"
	if len(next) == 0 {
		status = "disconnected"
	}
	if err := h.saveConfigLocked(r.Context(), status); err != nil {
		h.config.Devices = previous
		writeError(w, http.StatusInternalServerError, "STORAGE_ERROR", "Could not update encrypted Midea configuration")
		return
	}
	if driver := h.driverFor(stored.Device.Protocol); driver != nil {
		driver.Remove(id)
	}
	h.adapter.Unregister(id)
	domusDevice, lookupErr := h.devices.GetByExternalID(r.Context(), mideaIntegrationID, id)
	if lookupErr != nil && !errors.Is(lookupErr, device.ErrDeviceNotFound) {
		writeError(w, http.StatusInternalServerError, "DEVICE_LOOKUP_FAILED", "Could not locate the Domus device record for cleanup")
		return
	}
	if domusDevice != nil {
		for _, key := range mideaEntityKeys(id) {
			item, getErr := h.entities.GetByKey(r.Context(), key)
			if errors.Is(getErr, entity.ErrEntityNotFound) {
				continue
			}
			if getErr != nil {
				writeError(w, http.StatusInternalServerError, "ENTITY_LOOKUP_FAILED", "Could not locate Midea entities for cleanup")
				return
			}
			if err := h.entities.Delete(r.Context(), item.ID); err != nil {
				writeError(w, http.StatusInternalServerError, "ENTITY_DELETE_FAILED", "Could not remove Midea entities from Domus")
				return
			}
		}
		if err := h.devices.Delete(r.Context(), domusDevice.ID); err != nil {
			writeError(w, http.StatusInternalServerError, "DEVICE_DELETE_FAILED", "Could not remove the Midea device from the Domus registry")
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true, "device_id": id})
}

func (h *MideaHandler) loadStoredConfig(ctx context.Context) error {
	record, err := h.integrations.GetByID(ctx, mideaIntegrationID)
	if errors.Is(err, integration.ErrIntegrationNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("midea: load integration record: %w", err)
	}
	plain, err := h.cipher.Decrypt(string(record.ConfigEncrypted))
	if err != nil {
		return fmt.Errorf("midea: decrypt stored integration configuration: %w", err)
	}
	var config mideaStoredConfig
	if err := json.Unmarshal([]byte(plain), &config); err != nil {
		return errors.New("midea: stored integration configuration is invalid")
	}
	for _, stored := range config.Devices {
		driver := h.driverFor(stored.Device.Protocol)
		if driver == nil {
			return fmt.Errorf("midea: no driver available for stored protocol %q", stored.Device.Protocol)
		}
		if err := driver.Register(stored); err != nil {
			return fmt.Errorf("midea: restore configured device %q: %w", stored.Device.ID, err)
		}
		if err := h.adapter.Register(stored.Device); err != nil {
			return fmt.Errorf("midea: restore device adapter %q: %w", stored.Device.ID, err)
		}
		if _, err := h.provisionDevice(ctx, stored); err != nil {
			return fmt.Errorf("midea: restore Domus device registry %q: %w", stored.Device.ID, err)
		}
	}
	h.config = config
	return nil
}

func (h *MideaHandler) saveConfigLocked(ctx context.Context, status string) error {
	encoded, err := json.Marshal(h.config)
	if err != nil {
		return err
	}
	encrypted, err := h.cipher.Encrypt(string(encoded))
	if err != nil {
		return err
	}
	record, getErr := h.integrations.GetByID(ctx, mideaIntegrationID)
	now := time.Now().UTC()
	if errors.Is(getErr, integration.ErrIntegrationNotFound) {
		record = &integration.Integration{
			ID: mideaIntegrationID, Type: "midea_smarthome_local",
			Name: "Midea SmartHome (local)", Enabled: status == "connected",
			ConfigEncrypted: []byte(encrypted), Status: status, CreatedAt: now, UpdatedAt: now,
		}
		if err := h.integrations.Create(ctx, record); err != nil {
			return err
		}
		return h.integrations.SetStatus(ctx, mideaIntegrationID, status, "")
	}
	if getErr != nil {
		return getErr
	}
	record.Type = "midea_smarthome_local"
	record.Name = "Midea SmartHome (local)"
	record.Enabled = status == "connected"
	record.ConfigEncrypted = []byte(encrypted)
	record.UpdatedAt = now
	if err := h.integrations.Update(ctx, record); err != nil {
		return err
	}
	return h.integrations.SetStatus(ctx, mideaIntegrationID, status, "")
}

func (h *MideaHandler) provisionDevice(ctx context.Context, config midea.A1LANConfig) (*device.Device, error) {
	externalID := config.Device.ID
	now := time.Now().UTC()
	metadata, err := json.Marshal(map[string]any{
		"protocol": config.Device.Protocol, "type_code": config.Device.TypeCode,
		"host": config.Device.Host, "port": config.Device.Port, "kind": config.Device.Kind,
	})
	if err != nil {
		return nil, err
	}
	domusDevice, err := h.devices.GetByExternalID(ctx, mideaIntegrationID, externalID)
	if errors.Is(err, device.ErrDeviceNotFound) {
		domusDevice = &device.Device{
			ID: uuid.NewString(), IntegrationID: mideaIntegrationID, ExternalID: externalID,
			Name: config.Device.Name, Manufacturer: defaultMideaBrand(config.Device.Brand),
			Model: config.Device.Model, Online: false, Metadata: string(metadata),
			CreatedAt: now, UpdatedAt: now,
		}
		if err := h.devices.Create(ctx, domusDevice); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	} else {
		domusDevice.Name = config.Device.Name
		domusDevice.Manufacturer = defaultMideaBrand(config.Device.Brand)
		domusDevice.Model = config.Device.Model
		domusDevice.Metadata = string(metadata)
		domusDevice.UpdatedAt = now
		if err := h.devices.Update(ctx, domusDevice); err != nil {
			return nil, err
		}
	}
	for _, key := range mideaEntityKeys(externalID) {
		if err := h.ensureMideaEntity(ctx, domusDevice, config.Device, key); err != nil {
			return nil, err
		}
	}
	return domusDevice, nil
}

func defaultMideaBrand(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "Midea"
	}
	return value
}

func mideaEntitySuffix(id string) string {
	return strings.NewReplacer("-", "_", ":", "_", ".", "_").Replace(id)
}

func mideaEntityKeys(id string) []string {
	suffix := "midea_" + mideaEntitySuffix(id)
	return []string{
		"humidifier." + suffix,
		"sensor." + suffix + "_humidity",
		"sensor." + suffix + "_temperature",
		"binary_sensor." + suffix + "_tank_full",
	}
}

func (h *MideaHandler) ensureMideaEntity(ctx context.Context, domusDevice *device.Device, source midea.Device, key string) error {
	now := time.Now().UTC()
	domain := strings.SplitN(key, ".", 2)[0]
	name, unit, icon, writable := source.Name, "", "mdi:dehumidifier", false
	state := "null"
	attributes := map[string]any{"device_class": "humidity", "manufacturer": defaultMideaBrand(source.Brand), "model": source.Model}
	switch {
	case domain == "humidifier":
		writable = true
		state = "{\"state\":\"off\"}"
		attributes = map[string]any{
			"supported_modes": source.Modes, "fan_modes": source.FanSpeeds,
			"capabilities": source.Capabilities, "writable_capabilities": source.Writable,
			"min_humidity": source.MinHumidity, "max_humidity": source.MaxHumidity,
		}
	case strings.HasSuffix(key, "_humidity"):
		name += " · umidità"
		unit, icon = "%", "mdi:water-percent"
	case strings.HasSuffix(key, "_temperature"):
		name += " · temperatura"
		unit, icon = "°C", "mdi:thermometer"
		attributes["device_class"] = "temperature"
	case strings.HasSuffix(key, "_tank_full"):
		name += " · serbatoio pieno"
		icon = "mdi:water-alert"
		attributes["device_class"] = "moisture"
	}
	attributesJSON, err := json.Marshal(attributes)
	if err != nil {
		return err
	}
	item, getErr := h.entities.GetByKey(ctx, key)
	if errors.Is(getErr, entity.ErrEntityNotFound) {
		item = &entity.Entity{
			ID: uuid.NewString(), DeviceID: domusDevice.ID, EntityKey: key,
			Domain: domain, Name: name, Icon: icon, UnitOfMeasurement: unit,
			Writable: writable, State: state, Attributes: string(attributesJSON),
			Available: false, AreaID: domusDevice.AreaID,
			LastChangedAt: now, LastUpdatedAt: now,
		}
		return h.entities.Create(ctx, item)
	}
	if getErr != nil {
		return getErr
	}
	item.DeviceID, item.Domain, item.Name, item.Icon = domusDevice.ID, domain, name, icon
	item.UnitOfMeasurement, item.Writable = unit, writable
	item.Attributes, item.AreaID, item.LastUpdatedAt = string(attributesJSON), domusDevice.AreaID, now
	return h.entities.Update(ctx, item)
}

func (h *MideaHandler) syncEntities(ctx context.Context, externalID string, state map[string]any) error {
	domusDevice, err := h.devices.GetByExternalID(ctx, mideaIntegrationID, externalID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	for _, key := range mideaEntityKeys(externalID) {
		item, err := h.entities.GetByKey(ctx, key)
		if err != nil {
			return err
		}
		var rawState any
		var attrs map[string]any
		switch {
		case strings.HasPrefix(key, "humidifier."):
			rawState = "off"
			if state["power"] == true { rawState = "on" }
			attrs = cloneMideaState(state)
		case strings.HasSuffix(key, "_humidity"):
			rawState = state["current_humidity"]
			attrs = map[string]any{"device_class": "humidity", "current_humidity": state["current_humidity"], "unit_of_measurement": "%"}
		case strings.HasSuffix(key, "_temperature"):
			rawState = state["current_temperature"]
			attrs = map[string]any{"device_class": "temperature", "current_temperature": state["current_temperature"], "unit_of_measurement": "°C"}
		case strings.HasSuffix(key, "_tank_full"):
			rawState = "off"
			if state["tank_full"] == true { rawState = "on" }
			attrs = map[string]any{"device_class": "moisture", "tank_full": state["tank_full"]}
		}
		stateJSON, marshalErr := json.Marshal(rawState)
		if marshalErr != nil { return marshalErr }
		attributesJSON, marshalErr := json.Marshal(attrs)
		if marshalErr != nil { return marshalErr }
		if err := h.entities.UpdateState(ctx, item.ID, string(stateJSON), string(attributesJSON), true, now); err != nil {
			return err
		}
		if h.bus != nil {
			_ = h.bus.Publish(ctx, events.Event{
				ID: uuid.NewString(), Type: events.EntityStateChanged, OccurredAt: now,
				Source: "midea_smarthome",
				Payload: map[string]any{"entity_key": key, "state": rawState, "attributes": attrs},
			})
		}
	}
	if err := h.devices.SetOnline(ctx, domusDevice.ID, true); err != nil {
		return err
	}
	if h.bus != nil {
		_ = h.bus.Publish(ctx, events.Event{
			ID: uuid.NewString(), Type: events.DeviceOnline, OccurredAt: now,
			Source: "midea_smarthome",
			Payload: map[string]any{"device_id": domusDevice.ID, "external_id": externalID},
		})
	}
	return nil
}

func cloneMideaState(state map[string]any) map[string]any {
	result := make(map[string]any, len(state)+2)
	for key, value := range state { result[key] = value }
	result["device_class"] = "dehumidifier"
	result["unit_of_measurement"] = "%"
	return result
}

func (h *MideaHandler) findStoredConfigLocked(id string) (midea.A1LANConfig, bool) {
	for _, config := range h.config.Devices {
		if config.Device.ID == id { return config, true }
	}
	return midea.A1LANConfig{}, false
}

func removeStoredMideaConfig(configs []midea.A1LANConfig, id string) []midea.A1LANConfig {
	result := make([]midea.A1LANConfig, 0, len(configs))
	for _, config := range configs {
		if config.Device.ID != id { result = append(result, config) }
	}
	return result
}

func (h *MideaHandler) driverFor(protocol midea.Protocol) *midea.A1LANDriver {
	switch protocol {
	case midea.ProtocolMSmartV2:
		return h.driverV2
	case midea.ProtocolMSmartV3:
		return h.driverV3
	default:
		return nil
	}
}

func isPrivateMideaHost(raw string) bool {
	ip := net.ParseIP(strings.TrimSpace(raw))
	return ip != nil && ip.IsPrivate() && !ip.IsLoopback() && !ip.IsUnspecified() && !ip.IsMulticast()
}

func safeMideaError(err error) string {
	if err == nil { return "Midea operation failed" }
	message := err.Error()
	lower := strings.ToLower(message)
	for _, sensitive := range []string{"token", "password", "secret", "local_key", "key_hex"} {
		if strings.Contains(lower, sensitive) {
			return "Midea rejected the operation or the local connection credentials are invalid"
		}
	}
	if len(message) > 180 { message = message[:180] }
	return message
}
