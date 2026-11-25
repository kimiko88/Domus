package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/domus-hub/domus/internal/application/plugin_runtime"
	"github.com/domus-hub/domus/internal/plugins"
	"github.com/go-chi/chi/v5"
)

// PluginRuntimeHandler exposes the local built-in adapter registry and its
// persisted instances. Secret configuration is never returned.
type PluginRuntimeHandler struct {
	manager *plugin_runtime.Manager
}

func NewPluginRuntimeHandler(manager *plugin_runtime.Manager) *PluginRuntimeHandler {
	return &PluginRuntimeHandler{manager: manager}
}

func (h *PluginRuntimeHandler) ListTypes(w http.ResponseWriter, r *http.Request) {
	if h.manager == nil {
		writeError(w, http.StatusServiceUnavailable, "PLUGIN_RUNTIME_UNAVAILABLE", "plugin runtime is not configured")
		return
	}
	writeJSON(w, http.StatusOK, h.manager.ListTypes())
}

func (h *PluginRuntimeHandler) ListInstances(w http.ResponseWriter, r *http.Request) {
	if h.manager == nil {
		writeError(w, http.StatusServiceUnavailable, "PLUGIN_RUNTIME_UNAVAILABLE", "plugin runtime is not configured")
		return
	}
	instances, err := h.manager.ListInstances(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "PLUGIN_LIST_FAILED", "could not list plugin instances")
		return
	}
	writeJSON(w, http.StatusOK, instances)
}

func (h *PluginRuntimeHandler) Create(w http.ResponseWriter, r *http.Request) {
	if h.manager == nil {
		writeError(w, http.StatusServiceUnavailable, "PLUGIN_RUNTIME_UNAVAILABLE", "plugin runtime is not configured")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, plugins.MaxPluginConfigBytes+8<<10)
	var request struct {
		ID      string          `json:"id"`
		Type    string          `json:"type"`
		Name    string          `json:"name"`
		Enabled *bool           `json:"enabled"`
		Config  json.RawMessage `json:"config"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_PLUGIN_INSTANCE", "provide id, type, name and config")
		return
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		writeError(w, http.StatusBadRequest, "INVALID_PLUGIN_INSTANCE", "unexpected trailing JSON content")
		return
	}
	enabled := true
	if request.Enabled != nil {
		enabled = *request.Enabled
	}
	instance, err := h.manager.Create(r.Context(), request.ID, request.Type, request.Name, request.Config, enabled)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, plugins.ErrPluginNotFound) {
			status = http.StatusNotFound
		}
		if strings.Contains(err.Error(), "already exists") {
			status = http.StatusConflict
		}
		writeError(w, status, "PLUGIN_CREATE_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, instance)
}

func (h *PluginRuntimeHandler) Enable(w http.ResponseWriter, r *http.Request) {
	h.setEnabled(w, r, true)
}

func (h *PluginRuntimeHandler) Disable(w http.ResponseWriter, r *http.Request) {
	h.setEnabled(w, r, false)
}

func (h *PluginRuntimeHandler) setEnabled(w http.ResponseWriter, r *http.Request, enabled bool) {
	if h.manager == nil {
		writeError(w, http.StatusServiceUnavailable, "PLUGIN_RUNTIME_UNAVAILABLE", "plugin runtime is not configured")
		return
	}
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if err := h.manager.SetEnabled(r.Context(), id, enabled); err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		writeError(w, status, "PLUGIN_STATE_CHANGE_FAILED", "plugin instance could not change state")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "enabled": enabled})
}

func (h *PluginRuntimeHandler) Restart(w http.ResponseWriter, r *http.Request) {
	if h.manager == nil {
		writeError(w, http.StatusServiceUnavailable, "PLUGIN_RUNTIME_UNAVAILABLE", "plugin runtime is not configured")
		return
	}
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if err := h.manager.Restart(r.Context(), id); err != nil {
		status := http.StatusBadGateway
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		writeError(w, status, "PLUGIN_RESTART_FAILED", "plugin instance could not be restarted")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "restarted": true})
}

func (h *PluginRuntimeHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if h.manager == nil {
		writeError(w, http.StatusServiceUnavailable, "PLUGIN_RUNTIME_UNAVAILABLE", "plugin runtime is not configured")
		return
	}
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if err := h.manager.Delete(r.Context(), id); err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		writeError(w, status, "PLUGIN_DELETE_FAILED", "plugin instance could not be deleted")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "deleted": true})
}
