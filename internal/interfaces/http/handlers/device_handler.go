// Package handlers provides HTTP route handlers for Devices.
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/domus-hub/domus/internal/domain/area"
	"github.com/domus-hub/domus/internal/domain/device"
	"github.com/domus-hub/domus/internal/domain/entity"
	domusmw "github.com/domus-hub/domus/internal/interfaces/http/middleware"
	"github.com/go-chi/chi/v5"
)

// DeviceHandler handles HTTP operations for Device entities.
type DeviceHandler struct {
	repo       device.Repository
	entityRepo entity.Repository
	areaRepo   area.Repository
}

// NewDeviceHandler constructs a DeviceHandler.
func NewDeviceHandler(repo device.Repository) *DeviceHandler {
	return &DeviceHandler{repo: repo}
}

func (h *DeviceHandler) WithEntities(repo entity.Repository) *DeviceHandler {
	h.entityRepo = repo
	return h
}

func (h *DeviceHandler) WithAreas(repo area.Repository) *DeviceHandler {
	h.areaRepo = repo
	return h
}

// UpdateArea assigns a physical device and all of its entities to a room.
// Some integrations expose a device only through its entities; those devices
// are still assignable by their shared device_id.
func (h *DeviceHandler) UpdateArea(w http.ResponseWriter, r *http.Request) {
	deviceID := strings.TrimSpace(chi.URLParam(r, "id"))
	if deviceID == "" || len(deviceID) > 256 {
		writeError(w, http.StatusBadRequest, "INVALID_DEVICE_ID", "invalid device ID")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 4*1024)
	var req struct {
		AreaID json.RawMessage `json:"area_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "invalid room assignment payload")
		return
	}
	if len(req.AreaID) == 0 {
		writeError(w, http.StatusBadRequest, "MISSING_AREA_ID", "area_id is required; use null to remove the room")
		return
	}
	var areaID string
	if string(req.AreaID) != "null" {
		if err := json.Unmarshal(req.AreaID, &areaID); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_AREA_ID", "area_id must be a room ID or null")
			return
		}
		areaID = strings.TrimSpace(areaID)
	}

	if areaID != "" {
		if h.areaRepo == nil {
			writeError(w, http.StatusServiceUnavailable, "AREAS_UNAVAILABLE", "room validation service is unavailable")
			return
		}
		if _, err := h.areaRepo.GetByID(r.Context(), areaID); err != nil {
			if errors.Is(err, area.ErrAreaNotFound) {
				writeError(w, http.StatusNotFound, "AREA_NOT_FOUND", "the selected room does not exist")
				return
			}
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to validate the selected room")
			return
		}
	}

	dev, deviceErr := h.repo.GetByID(r.Context(), deviceID)
	deviceExists := deviceErr == nil
	if deviceErr != nil && !errors.Is(deviceErr, device.ErrDeviceNotFound) {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to load device")
		return
	}

	var linkedEntities []*entity.Entity
	if h.entityRepo != nil {
		// Include legacy Tuya rows which predate device_id persistence but still
		// carry the stable tuya_<external-id> suffix in their entity key.
		items, err := h.entityRepo.List(r.Context(), entity.ListFilter{})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to load device entities")
			return
		}
		for _, item := range items {
			if item != nil && entityDeviceIDForAreaAssignment(item) == deviceID {
				linkedEntities = append(linkedEntities, item)
			}
		}
	}
	if !deviceExists && len(linkedEntities) == 0 {
		writeError(w, http.StatusNotFound, "DEVICE_NOT_FOUND", "device not found")
		return
	}

	oldDeviceArea := ""
	if deviceExists {
		oldDeviceArea = dev.AreaID
		dev.AreaID = areaID
		dev.UpdatedAt = time.Now().UTC()
		if err := h.repo.Update(r.Context(), dev); err != nil {
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to assign device to the selected room")
			return
		}
	}

	updatedEntities := make([]*entity.Entity, 0, len(linkedEntities))
	updatedEntityAreas := make(map[string]string, len(linkedEntities))
	for _, linked := range linkedEntities {
		oldArea := linked.AreaID
		linked.AreaID = areaID
		if err := h.entityRepo.Update(r.Context(), linked); err != nil {
			// Best-effort compensation avoids leaving a half-moved device when
			// one of its entity metadata updates fails.
			for _, previous := range updatedEntities {
				previous.AreaID = updatedEntityAreas[previous.ID]
				_ = h.entityRepo.Update(r.Context(), previous)
			}
			if deviceExists {
				dev.AreaID = oldDeviceArea
				_ = h.repo.Update(r.Context(), dev)
			}
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update all entities for this device")
			return
		}
		updatedEntities = append(updatedEntities, linked)
		updatedEntityAreas[linked.ID] = oldArea
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"device_id": deviceID,
		"area_id": areaID,
		"device_updated": deviceExists,
		"updated_entities": len(updatedEntities),
	})
}

// entityDeviceIDForAreaAssignment resolves the device key stored on an entity.
// Current entities use device_id; legacy Tuya entities can be linked by their
// deterministic entity-key suffix until the next integration sync fills it in.
func entityDeviceIDForAreaAssignment(item *entity.Entity) string {
	if item == nil {
		return ""
	}
	if id := strings.TrimSpace(item.DeviceID); id != "" {
		return id
	}
	for _, candidate := range []string{item.EntityKey, item.ID} {
		parts := strings.Split(strings.TrimSpace(candidate), ".")
		if len(parts) == 0 {
			continue
		}
		last := parts[len(parts)-1]
		if strings.HasPrefix(last, "tuya_") && len(last) > len("tuya_") {
			return last
		}
	}
	return ""
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
