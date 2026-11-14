package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/domus-hub/domus/internal/plugins/tuya_local"
	"github.com/go-chi/chi/v5"
)

// TuyaLocalHandler exposes local LAN-first Tuya device inspection and direct local dispatch.
type TuyaLocalHandler struct {
	adapter *tuya_local.Adapter
}

// NewTuyaLocalHandler creates a new TuyaLocalHandler instance.
func NewTuyaLocalHandler(adapter *tuya_local.Adapter) *TuyaLocalHandler {
	return &TuyaLocalHandler{adapter: adapter}
}

// RegisterRoutes registers endpoints for Tuya Local LAN management and commands.
func (h *TuyaLocalHandler) RegisterRoutes(r chi.Router) {
	r.Get("/devices", h.ListDevices)
	r.Post("/devices/{deviceID}/command", h.SendCommand)
}

func (h *TuyaLocalHandler) ListDevices(w http.ResponseWriter, r *http.Request) {
	if h.adapter == nil {
		writeJSON(w, http.StatusOK, map[string]any{"devices": []any{}})
		return
	}

	configs := h.adapter.GetConfiguredDevices()
	type item struct {
		DeviceID  string `json:"device_id"`
		Name      string `json:"name"`
		IP        string `json:"ip"`
		Version   string `json:"version"`
		Category  string `json:"category"`
		EntityKey string `json:"entity_key"`
		Connected bool   `json:"connected"`
	}
	devices := make([]item, 0, len(configs))
	for _, c := range configs {
		devices = append(devices, item{
			DeviceID:  c.DeviceID,
			Name:      c.Name,
			IP:        c.IP,
			Version:   c.Version,
			Category:  string(c.Category),
			EntityKey: c.EntityKey,
			Connected: h.adapter.IsDeviceConnected(c.DeviceID),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"devices": devices})
}

func (h *TuyaLocalHandler) SendCommand(w http.ResponseWriter, r *http.Request) {
	if h.adapter == nil {
		writeError(w, http.StatusServiceUnavailable, "ADAPTER_NOT_INITIALIZED", "tuya local adapter is not available")
		return
	}

	deviceID := chi.URLParam(r, "deviceID")
	var req struct {
		Command string         `json:"command"`
		Params  map[string]any `json:"params"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid command payload")
		return
	}
	if req.Command == "" {
		req.Command = "turn_on"
	}

	if err := h.adapter.SendCommand(r.Context(), deviceID, req.Command, req.Params); err != nil {
		writeError(w, http.StatusBadGateway, "TUYA_LOCAL_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true, "transport": "lan_local"})
}
