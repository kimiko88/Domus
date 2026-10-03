// Package handlers provides HTTP route handlers for Devices.
package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/domus-hub/domus/internal/domain/device"
	"github.com/go-chi/chi/v5"
)

// DeviceHandler handles HTTP operations for Device entities.
type DeviceHandler struct {
	repo device.Repository
}

// NewDeviceHandler constructs a DeviceHandler.
func NewDeviceHandler(repo device.Repository) *DeviceHandler {
	return &DeviceHandler{repo: repo}
}

// List returns devices matching query parameters.
func (h *DeviceHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := device.ListFilter{
		IntegrationID: q.Get("integration_id"),
		AreaID:        q.Get("area_id"),
	}

	if onlineStr := q.Get("online"); onlineStr != "" {
		if b, err := strconv.ParseBool(onlineStr); err == nil {
			filter.Online = &b
		}
	}

	devices, err := h.repo.List(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to list devices")
		return
	}

	writeJSON(w, http.StatusOK, devices)
}

// GetByID returns device details by path ID parameter.
func (h *DeviceHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	dev, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, device.ErrDeviceNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "device not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to get device")
		return
	}

	writeJSON(w, http.StatusOK, dev)
}
