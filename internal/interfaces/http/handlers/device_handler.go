// Package handlers provides HTTP route handlers for Devices.
package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/domus-hub/domus/internal/domain/device"
	"github.com/domus-hub/domus/internal/domain/entity"
	domusmw "github.com/domus-hub/domus/internal/interfaces/http/middleware"
	"github.com/go-chi/chi/v5"
)

// DeviceHandler handles HTTP operations for Device entities.
type DeviceHandler struct {
	repo       device.Repository
	entityRepo entity.Repository
}

// NewDeviceHandler constructs a DeviceHandler.
func NewDeviceHandler(repo device.Repository) *DeviceHandler {
	return &DeviceHandler{repo: repo}
}

func (h *DeviceHandler) WithEntities(repo entity.Repository) *DeviceHandler {
	h.entityRepo = repo
	return h
}

func (h *DeviceHandler) canViewDevice(r *http.Request, dev *device.Device) (bool, error) {
	currentUser := domusmw.GetUser(r.Context())
	if currentUser == nil {
		return true, nil
	}
	permissions := currentUser.EffectivePermissions()
	if permissions == nil || !permissions.CanAccessArea(dev.AreaID) {
		return false, nil
	}
	if permissions.AllowedEntityIDs == nil {
		return true, nil
	}
	if h.entityRepo == nil {
		return false, nil
	}
	entities, err := h.entityRepo.List(r.Context(), entity.ListFilter{DeviceID: dev.ID})
	if err != nil {
		return false, err
	}
	for _, item := range entities {
		if currentUser.CanAccessEntityInArea(item.ID, item.AreaID) {
			return true, nil
		}
	}
	return false, nil
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
	if domusmw.GetUser(r.Context()) != nil {
		visible := make([]*device.Device, 0, len(devices))
		for _, item := range devices {
			allowed, accessErr := h.canViewDevice(r, item)
			if accessErr != nil {
				writeError(w, http.StatusInternalServerError, "internal_error", "failed to verify device permissions")
				return
			}
			if allowed {
				visible = append(visible, item)
			}
		}
		devices = visible
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

	if domusmw.GetUser(r.Context()) != nil {
		allowed, accessErr := h.canViewDevice(r, dev)
		if accessErr != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to verify device permissions")
			return
		}
		if !allowed {
			writeError(w, http.StatusForbidden, "forbidden", "device is outside the member's allowed resources")
			return
		}
	}
	writeJSON(w, http.StatusOK, dev)
}
