// Package handlers provides HTTP route handlers for Devices.
package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/domus-hub/domus/internal/domain/device"
	domusmw "github.com/domus-hub/domus/internal/interfaces/http/middleware"
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
	if currentUser := domusmw.GetUser(r.Context()); currentUser != nil {
		permissions := currentUser.EffectivePermissions()
		if permissions != nil && permissions.AllowedAreaIDs != nil {
			visible := make([]*device.Device, 0, len(devices))
			for _, item := range devices {
				if permissions.CanAccessArea(item.AreaID) {
					visible = append(visible, item)
				}
			}
			devices = visible
		}
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

	if currentUser := domusmw.GetUser(r.Context()); currentUser != nil {
		permissions := currentUser.EffectivePermissions()
		if permissions == nil || !permissions.CanAccessArea(dev.AreaID) {
			writeError(w, http.StatusForbidden, "forbidden", "device is outside the member's allowed areas")
			return
		}
	}
	writeJSON(w, http.StatusOK, dev)
}
