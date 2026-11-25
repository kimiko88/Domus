package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/integration"
	"github.com/domus-hub/domus/internal/infrastructure/crypto"
	"github.com/domus-hub/domus/internal/plugins/vesync"
	"github.com/go-chi/chi/v5"
)

const vesyncIntegrationID = "integration_vesync_cloud"

type VeSyncHandler struct {
	provider *vesync.Adapter
	repo integration.Repository
	cipher *crypto.EnvelopeCipher
	client *vesync.Client
	mu sync.Mutex
}

type vesyncStoredConfig struct {
	Session vesync.Session `json:"session"`
}

type vesyncConfigureRequest struct {
	Email string `json:"email"`
	Password string `json:"password"`
	CountryCode string `json:"country_code"`
	TimeZone string `json:"time_zone"`
}

type vesyncCommandRequest struct {
	Action string `json:"action"`
	Value any `json:"value,omitempty"`
}

func NewVeSyncHandler(repo integration.Repository, cipher *crypto.EnvelopeCipher) (*VeSyncHandler, error) {
	if repo == nil || cipher == nil {
		return nil, errors.New("vesync: integration repository and encryption are required")
	}
	return &VeSyncHandler{repo: repo, cipher: cipher, client: vesync.NewClient(8*time.Second)}, nil
}


 // WithPluginAdapter binds VeSync's API client to the supervised provider.
func (h *VeSyncHandler) WithPluginAdapter(adapter *vesync.Adapter) error {
	if adapter == nil { return errors.New("vesync: plugin runtime adapter is required") }
	client, err := adapter.Client()
	if err != nil { return err }
	h.client = client
	h.provider = adapter
	return nil
}

func (h *VeSyncHandler) RegisterRoutes(r chi.Router) {
	r.Get("/status", h.Status)
	r.Put("/config", h.Configure)
	r.Get("/devices", h.Devices)
	r.Post("/devices/{id}/commands", h.CommandDevice)
	r.Post("/disconnect", h.Disconnect)
}

func (h *VeSyncHandler) Configure(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store, private")
	var req vesyncConfigureRequest
	if !decodeLimitedJSON(w, r, &req, 16*1024) {
		return
	}
	req.Email = strings.TrimSpace(req.Email)
	req.CountryCode = strings.ToUpper(strings.TrimSpace(req.CountryCode))
	req.TimeZone = strings.TrimSpace(req.TimeZone)
	if req.Email == "" || len(req.Email) > 254 || req.Password == "" || len(req.Password) > 1024 {
		writeError(w, http.StatusBadRequest, "INVALID_CREDENTIALS", "A VeSync email address and password are required")
		return
	}
	if req.CountryCode == "" {
		req.CountryCode = "IT"
	}
	if req.TimeZone == "" {
		req.TimeZone = "Europe/Rome"
	}
	if _, err := time.LoadLocation(req.TimeZone); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_TIME_ZONE", "A valid IANA time zone is required")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	session, err := h.client.Authenticate(ctx, req.Email, req.Password, req.CountryCode, req.TimeZone)
	if err != nil {
		writeError(w, http.StatusBadGateway, "VESYNC_AUTH_FAILED", "VeSync cloud authentication failed; verify credentials, account region and internet connectivity")
		return
	}
	// Validate the token by listing the account's devices before persisting it.
	devices, err := h.client.ListDevices(ctx, session)
	if err != nil {
		writeError(w, http.StatusBadGateway, "VESYNC_DEVICE_LIST_FAILED", "VeSync login succeeded but the cloud device list could not be loaded")
		return
	}
	if err := h.saveConfig(r.Context(), vesyncStoredConfig{Session: session}, "connected"); err != nil {
		writeError(w, http.StatusInternalServerError, "STORAGE_ERROR", "Could not save the encrypted VeSync session")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"configured": true,
		"connected": true,
		"region": session.Region,
		"country_code": session.CountryCode,
		"device_count": len(devices),
	})
}

func (h *VeSyncHandler) Status(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store, private")
	config, err := h.loadConfig(r)
	if errors.Is(err, integration.ErrIntegrationNotFound) {
		writeJSON(w, http.StatusOK, map[string]any{"configured": false, "connected": false})
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "STORAGE_ERROR", "Could not load VeSync integration status")
		return
	}
	connected := config.Session.AccountID != "" && config.Session.Token != ""
	writeJSON(w, http.StatusOK, map[string]any{
		"configured": connected,
		"connected": connected,
		"region": config.Session.Region,
		"country_code": config.Session.CountryCode,
		"mode": "cloud",
	})
}

func (h *VeSyncHandler) Devices(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store, private")
	config, err := h.loadConfig(r)
	if err != nil || config.Session.Token == "" {
		writeError(w, http.StatusConflict, "VESYNC_NOT_CONNECTED", "Configure VeSync cloud credentials before listing devices")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()
	devices, err := h.client.ListDevices(ctx, config.Session)
	if err != nil {
		writeError(w, http.StatusBadGateway, "VESYNC_DEVICE_LIST_FAILED", "Could not load devices from VeSync cloud; reconnect if the session has expired")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"devices": devices, "count": len(devices), "mode": "cloud"})
}

func (h *VeSyncHandler) CommandDevice(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store, private")
	var req vesyncCommandRequest
	if !decodeLimitedJSON(w, r, &req, 8*1024) {
		return
	}
	config, err := h.loadConfig(r)
	if err != nil || config.Session.Token == "" {
		writeError(w, http.StatusConflict, "VESYNC_NOT_CONNECTED", "Configure VeSync cloud credentials before controlling devices")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()
	devices, err := h.client.ListDevices(ctx, config.Session)
	if err != nil {
		writeError(w, http.StatusBadGateway, "VESYNC_DEVICE_LIST_FAILED", "Could not verify the device in the connected VeSync account")
		return
	}
	id := chi.URLParam(r, "id")
	var selected *vesync.DeviceInfo
	for i := range devices {
		if devices[i].ID == id {
			selected = &devices[i]
			break
		}
	}
	if selected == nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "VeSync device was not found in the connected account")
		return
	}

	var result any
	switch req.Action {
	case "refresh":
		result, err = h.client.GetPurifierStatus(ctx, config.Session, *selected)
	case "set_power":
		value, ok := req.Value.(bool)
		if !ok {
			writeError(w, http.StatusBadRequest, "INVALID_COMMAND", "Power value must be boolean")
			return
		}
		result, err = h.client.SetPurifierPower(ctx, config.Session, *selected, value)
	case "set_mode":
		value, ok := req.Value.(string)
		if !ok {
			writeError(w, http.StatusBadRequest, "INVALID_COMMAND", "Mode value must be a string")
			return
		}
		result, err = h.client.SetPurifierMode(ctx, config.Session, *selected, value)
	case "set_fan_level":
		var level int
		switch v := req.Value.(type) {
		case float64:
			if v != float64(int(v)) {
				writeError(w, http.StatusBadRequest, "INVALID_COMMAND", "Fan level must be an integer")
				return
			}
			level = int(v)
		case json.Number:
			n, parseErr := strconv.Atoi(v.String())
			if parseErr != nil {
				writeError(w, http.StatusBadRequest, "INVALID_COMMAND", "Fan level must be an integer")
				return
			}
			level = n
		default:
			writeError(w, http.StatusBadRequest, "INVALID_COMMAND", "Fan level must be an integer")
			return
		}
		result, err = h.client.SetPurifierFanLevel(ctx, config.Session, *selected, level)
	default:
		writeError(w, http.StatusBadRequest, "INVALID_COMMAND", "Unsupported VeSync command")
		return
	}
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "VESYNC_COMMAND_FAILED", safeVeSyncError(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"device_id": selected.ID,
		"action": req.Action,
		"result": result,
		"executed_at": time.Now().UTC(),
	})
}

func safeVeSyncError(err error) string {
	if err == nil {
		return "VeSync command failed"
	}
	message := err.Error()
	if strings.Contains(strings.ToLower(message), "token") || strings.Contains(strings.ToLower(message), "password") || strings.Contains(strings.ToLower(message), "accountid") {
		return "VeSync rejected the command or the cloud session has expired"
	}
	if len(message) > 180 {
		message = message[:180]
	}
	return message
}

func (h *VeSyncHandler) Disconnect(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store, private")
	if _, err := h.loadConfig(r); err != nil {
		writeError(w, http.StatusNotFound, "NOT_CONFIGURED", "VeSync integration is not configured")
		return
	}
	if err := h.saveConfig(r.Context(), vesyncStoredConfig{}, "disconnected"); err != nil {
		writeError(w, http.StatusInternalServerError, "STORAGE_ERROR", "Could not clear the encrypted VeSync session")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"connected": false})
}

func (h *VeSyncHandler) loadConfig(r *http.Request) (*vesyncStoredConfig, error) {
	record, err := h.repo.GetByID(r.Context(), vesyncIntegrationID)
	if err != nil {
		return nil, err
	}
	plain, err := h.cipher.Decrypt(string(record.ConfigEncrypted))
	if err != nil {
		return nil, fmt.Errorf("vesync: decrypt stored configuration: %w", err)
	}
	var config vesyncStoredConfig
	if err := json.Unmarshal([]byte(plain), &config); err != nil {
		return nil, errors.New("vesync: invalid stored configuration")
	}
	return &config, nil
}

func (h *VeSyncHandler) saveConfig(ctx context.Context, config vesyncStoredConfig, status string) error {
	encoded, err := json.Marshal(config)
	if err != nil {
		return err
	}
	encrypted, err := h.cipher.Encrypt(string(encoded))
	if err != nil {
		return err
	}
	record, getErr := h.repo.GetByID(ctx, vesyncIntegrationID)
	now := time.Now().UTC()
	if errors.Is(getErr, integration.ErrIntegrationNotFound) {
		record = &integration.Integration{
			ID: vesyncIntegrationID, Type: "vesync_cloud", Name: "VeSync Cloud",
			Enabled: true, Status: status, CreatedAt: now, UpdatedAt: now,
		}
	} else if getErr != nil {
		return getErr
	} else {
		record.ConfigEncrypted = []byte(encrypted)
		record.Name = "VeSync Cloud"
		record.Enabled = status != "disconnected"
		record.UpdatedAt = now
		if err := h.repo.Update(ctx, record); err != nil {
			return err
		}
		return h.repo.SetStatus(ctx, vesyncIntegrationID, status, "")
	}
	record.ConfigEncrypted = []byte(encrypted)
	if err := h.repo.Create(ctx, record); err != nil {
		return err
	}
	return h.repo.SetStatus(ctx, vesyncIntegrationID, status, "")
}
