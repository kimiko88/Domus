package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/domus-hub/domus/internal/plugins/tuya_cloud"
	"github.com/domus-hub/domus/internal/plugins/tuya_local"
	"github.com/go-chi/chi/v5"
)

// TuyaCloudHandler exposes server-side Tuya OpenAPI integration, device discovery,
// and automatic provisioning into the local-first LAN adapter.
type TuyaCloudHandler struct {
	client    *tuya_cloud.Client
	tuyaLocal *tuya_local.Adapter
}

// NewTuyaCloudHandler creates a new TuyaCloudHandler instance.
func NewTuyaCloudHandler(client *tuya_cloud.Client) *TuyaCloudHandler {
	return &TuyaCloudHandler{client: client}
}

// WithLocalAdapter attaches the local LAN adapter for zero-cloud device handover.
func (h *TuyaCloudHandler) WithLocalAdapter(adapter *tuya_local.Adapter) *TuyaCloudHandler {
	h.tuyaLocal = adapter
	return h
}

func (h *TuyaCloudHandler) RegisterRoutes(r chi.Router) {
	r.Get("/devices", h.ListDevices)
	r.Get("/devices/{deviceID}/functions", h.ListFunctions)
	r.Post("/devices/{deviceID}/commands", h.SendCommands)
	r.Post("/sync-local", h.SyncLocal)
}

func (h *TuyaCloudHandler) ListDevices(w http.ResponseWriter, r *http.Request) {
	devices, err := h.client.Devices(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, "TUYA_CLOUD_ERROR", err.Error())
		return
	}

	type enrichedDevice struct {
		tuya_cloud.Device
		LocalConfigured bool   `json:"local_configured"`
		LocalActive     bool   `json:"local_active"`
		LocalIP         string `json:"local_ip,omitempty"`
	}

	enriched := make([]enrichedDevice, 0, len(devices))
	for _, dev := range devices {
		ed := enrichedDevice{Device: dev}
		if h.tuyaLocal != nil {
			ed.LocalConfigured = h.tuyaLocal.IsDeviceConfigured(dev.ID)
			ed.LocalActive = h.tuyaLocal.IsDeviceConnected(dev.ID)
			ed.LocalIP = h.tuyaLocal.GetDeviceIP(dev.ID)
		}
		enriched = append(enriched, ed)
	}

	writeJSON(w, http.StatusOK, map[string]any{"devices": enriched})
}

func (h *TuyaCloudHandler) ListFunctions(w http.ResponseWriter, r *http.Request) {
	functions, err := h.client.Functions(r.Context(), chi.URLParam(r, "deviceID"))
	if err != nil {
		writeError(w, http.StatusBadGateway, "TUYA_CLOUD_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"functions": functions})
}

// SendCommands routes commands locally over LAN whenever the device has a local_key,
// completely avoiding the cloud for day-to-day operation.
func (h *TuyaCloudHandler) SendCommands(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	var req struct {
		Commands []tuya_cloud.Command `json:"commands"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "invalid command payload")
		return
	}
	if len(req.Commands) == 0 || len(req.Commands) > 32 {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "between 1 and 32 commands are required")
		return
	}

	deviceID := chi.URLParam(r, "deviceID")

	// If device is configured in Tuya Local, dispatch locally via LAN!
	if h.tuyaLocal != nil && h.tuyaLocal.IsDeviceConfigured(deviceID) {
		for _, cmd := range req.Commands {
			codeLower := strings.ToLower(cmd.Code)
			var localCmd string
			params := make(map[string]any)

			if valBool, ok := cmd.Value.(bool); ok {
				if valBool {
					localCmd = "turn_on"
				} else {
					localCmd = "turn_off"
				}
			} else if strings.Contains(codeLower, "bright") {
				localCmd = "set_brightness"
				params["brightness"] = cmd.Value
			} else if strings.Contains(codeLower, "temp") {
				localCmd = "set_temperature"
				params["temperature"] = cmd.Value
			} else {
				localCmd = "turn_on"
			}

			if err := h.tuyaLocal.SendCommand(r.Context(), deviceID, localCmd, params); err != nil {
				writeError(w, http.StatusBadGateway, "TUYA_LOCAL_ERROR", fmt.Sprintf("local LAN command failed: %v", err))
				return
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"success": true, "transport": "lan_local"})
		return
	}

	// Fallback to cloud only if not registered locally
	if err := h.client.SendCommands(r.Context(), deviceID, req.Commands); err != nil {
		writeError(w, http.StatusBadGateway, "TUYA_CLOUD_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "transport": "cloud"})
}

// SyncLocal fetches all devices and their local_key credentials from the cloud API
// and transfers them to the Tuya Local adapter for exclusive offline LAN control.
func (h *TuyaCloudHandler) SyncLocal(w http.ResponseWriter, r *http.Request) {
	if h.tuyaLocal == nil {
		writeError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "tuya local adapter not initialized")
		return
	}

	devices, err := h.client.Devices(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, "TUYA_CLOUD_ERROR", fmt.Sprintf("failed to list cloud devices: %v", err))
		return
	}

	synced := make([]tuya_local.DeviceConfig, 0, len(devices))
	for _, dev := range devices {
		localKey := dev.LocalKey
		ip := dev.IP

		// If local_key was not returned in list endpoint, query single-device endpoint
		if localKey == "" {
			if detailed, err := h.client.GetDevice(r.Context(), dev.ID); err == nil && detailed != nil {
				localKey = detailed.LocalKey
				if ip == "" {
					ip = detailed.IP
				}
			}
		}

		if localKey == "" {
			continue // cannot pair locally without encryption key
		}

		cat := tuya_local.ParseCategory(dev.Category)
		cfg := tuya_local.DeviceConfig{
			DeviceID:  dev.ID,
			Name:      dev.Name,
			IP:        ip,
			LocalKey:  localKey,
			Version:   "3.3",
			Category:  cat,
			EntityKey: fmt.Sprintf("switch.tuya_%s", dev.ID),
		}
		switch cat {
		case tuya_local.CategoryLight:
			cfg.EntityKey = fmt.Sprintf("light.tuya_%s", dev.ID)
		case tuya_local.CategoryClimate:
			cfg.EntityKey = fmt.Sprintf("climate.tuya_%s", dev.ID)
		}

		if err := h.tuyaLocal.RegisterAndPersistDevice(r.Context(), cfg, dev.Name); err != nil {
			continue
		}
		synced = append(synced, cfg)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success":      true,
		"synced_count": len(synced),
		"message":      fmt.Sprintf("%d dispositivi trasferiti al controllo locale LAN con Local Key", len(synced)),
		"devices":      synced,
	})
}
