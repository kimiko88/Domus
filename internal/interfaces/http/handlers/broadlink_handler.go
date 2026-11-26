package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/domus-hub/domus/internal/application/plugin_runtime"
	"github.com/domus-hub/domus/internal/plugins/broadlink"
	"github.com/go-chi/chi/v5"
)

type BroadlinkHandler struct {
	adapter *broadlink.Adapter
	manager *plugin_runtime.Manager
}

func NewBroadlinkHandler(adapter *broadlink.Adapter) *BroadlinkHandler {
	return &BroadlinkHandler{adapter: adapter}
}

// WithPluginManager makes the managed adapter the sole source of truth. In
// manager mode the legacy pointer is deliberately never used as a fallback:
// disabling or deleting the instance must immediately stop HTTP commands.
func (h *BroadlinkHandler) WithPluginManager(manager *plugin_runtime.Manager) *BroadlinkHandler {
	h.manager = manager
	return h
}

func (h *BroadlinkHandler) currentAdapter(ctx context.Context) (*broadlink.Adapter, error) {
	if h.manager == nil {
		if h.adapter == nil {
			return nil, errors.New("broadlink: adapter is not configured")
		}
		return h.adapter, nil
	}

	managed, ok := h.manager.GetAdapterByType(broadlink.PluginType)
	if !ok {
		return nil, errors.New("broadlink: managed integration is disabled or not configured")
	}
	adapter, ok := managed.(*broadlink.Adapter)
	if !ok {
		return nil, errors.New("broadlink: managed integration has an unexpected adapter type")
	}
	if err := adapter.HealthCheck(ctx); err != nil {
		return nil, err
	}
	return adapter, nil
}

func (h *BroadlinkHandler) resolveAdapter(w http.ResponseWriter, r *http.Request) (*broadlink.Adapter, bool) {
	adapter, err := h.currentAdapter(r.Context())
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "BROADLINK_UNAVAILABLE", "BroadLink integration is disabled or unavailable")
		return nil, false
	}
	return adapter, true
}

func (h *BroadlinkHandler) RegisterRoutes(r chi.Router) {
	r.Get("/devices", h.ListDevices)
	r.Post("/discovery/scan", h.Scan)
	r.Post("/devices", h.PairDevice)
	r.Delete("/devices/{deviceID}", h.RemoveDevice)
	r.Get("/devices/{deviceID}/commands", h.ListCommands)
	r.Post("/devices/{deviceID}/learn/start", h.StartLearning)
	r.Post("/devices/{deviceID}/learn/result", h.SaveLearnedCommand)
	r.Post("/devices/{deviceID}/commands/send", h.SendCommand)
}

func (h *BroadlinkHandler) ListDevices(w http.ResponseWriter, r *http.Request) {
	adapter, ok := h.resolveAdapter(w, r)
	if !ok { return }
	writeJSON(w, http.StatusOK, map[string]any{"devices": adapter.ListDevices()})
}

func (h *BroadlinkHandler) Scan(w http.ResponseWriter, r *http.Request) {
	adapter, ok := h.resolveAdapter(w, r)
	if !ok { return }
	var req struct { IP string `json:"ip"` }
	if r.Body != nil { _ = json.NewDecoder(r.Body).Decode(&req) }
	ctx, cancel := withBroadlinkTimeout(r, 4*time.Second)
	defer cancel()
	devices, err := adapter.Scan(ctx, strings.TrimSpace(req.IP))
	if err != nil {
		writeError(w, http.StatusBadRequest, "BROADLINK_SCAN_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"devices": devices, "count": len(devices)})
}

func (h *BroadlinkHandler) PairDevice(w http.ResponseWriter, r *http.Request) {
	adapter, ok := h.resolveAdapter(w, r)
	if !ok { return }
	var req struct { IP string `json:"ip"`; Name string `json:"name"` }
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid pairing payload")
		return
	}
	ctx, cancel := withBroadlinkTimeout(r, 8*time.Second)
	defer cancel()
	device, err := adapter.PairDevice(ctx, req.IP, req.Name)
	if err != nil {
		writeError(w, http.StatusBadGateway, "BROADLINK_PAIR_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"device": device})
}

func (h *BroadlinkHandler) RemoveDevice(w http.ResponseWriter, r *http.Request) {
	adapter, ok := h.resolveAdapter(w, r)
	if !ok { return }
	if err := adapter.RemoveDevice(r.Context(), chi.URLParam(r, "deviceID")); err != nil {
		writeError(w, http.StatusBadRequest, "BROADLINK_REMOVE_FAILED", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *BroadlinkHandler) ListCommands(w http.ResponseWriter, r *http.Request) {
	adapter, ok := h.resolveAdapter(w, r)
	if !ok { return }
	items, err := adapter.ListCommands(r.Context(), chi.URLParam(r, "deviceID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "BROADLINK_COMMANDS_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"commands": items})
}

func (h *BroadlinkHandler) StartLearning(w http.ResponseWriter, r *http.Request) {
	adapter, ok := h.resolveAdapter(w, r)
	if !ok { return }
	var req struct { Medium string `json:"medium"` }
	if r.Body != nil { _ = json.NewDecoder(r.Body).Decode(&req) }
	if req.Medium == "" { req.Medium = "ir" }
	ctx, cancel := withBroadlinkTimeout(r, 10*time.Second)
	defer cancel()
	if err := adapter.StartLearning(ctx, chi.URLParam(r, "deviceID"), req.Medium); err != nil {
		writeError(w, http.StatusBadGateway, "BROADLINK_LEARN_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"status": "learning", "medium": req.Medium})
}

func (h *BroadlinkHandler) SaveLearnedCommand(w http.ResponseWriter, r *http.Request) {
	adapter, ok := h.resolveAdapter(w, r)
	if !ok { return }
	var req struct { Name string `json:"name"`; Medium string `json:"medium"` }
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid learning result payload")
		return
	}
	if req.Medium == "" { req.Medium = "ir" }
	ctx, cancel := withBroadlinkTimeout(r, 6*time.Second)
	defer cancel()
	item, err := adapter.SaveLearnedCommand(ctx, chi.URLParam(r, "deviceID"), req.Name, req.Medium)
	if err != nil {
		writeError(w, http.StatusBadGateway, "BROADLINK_CAPTURE_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"command": item})
}

func (h *BroadlinkHandler) SendCommand(w http.ResponseWriter, r *http.Request) {
	adapter, ok := h.resolveAdapter(w, r)
	if !ok { return }
	var req struct {
		Name string `json:"name"`
		Medium string `json:"medium"`
		CodeBase64 string `json:"code_base64"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid command payload")
		return
	}
	ctx, cancel := withBroadlinkTimeout(r, 5*time.Second)
	defer cancel()
	var err error
	if req.CodeBase64 != "" {
		err = adapter.SendRaw(ctx, chi.URLParam(r, "deviceID"), req.CodeBase64)
	} else {
		err = adapter.SendStoredCommand(ctx, chi.URLParam(r, "deviceID"), req.Name, req.Medium)
	}
	if err != nil {
		writeError(w, http.StatusBadGateway, "BROADLINK_SEND_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "transport": "lan_local"})
}

func withBroadlinkTimeout(r *http.Request, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), timeout)
}
