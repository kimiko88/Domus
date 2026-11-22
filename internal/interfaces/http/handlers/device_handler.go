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
	"github.com/domus-hub/domus/internal/domain/integration"
	domusmw "github.com/domus-hub/domus/internal/interfaces/http/middleware"
	"github.com/go-chi/chi/v5"
)

// DeviceHandler handles HTTP operations for Device entities.
type DeviceHandler struct {
	repo           device.Repository
	entityRepo     entity.Repository
	areaRepo       area.Repository
	integrationRepo integration.Repository
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

func (h *DeviceHandler) WithIntegrations(repo integration.Repository) *DeviceHandler {
	h.integrationRepo = repo
	return h
}

// ensureTuyaIntegrationRecord satisfies the devices.integration_id foreign key
// for a cloud-listed device that is first registered through room assignment.
func (h *DeviceHandler) ensureTuyaIntegrationRecord(ctx context.Context, integrationID string, online bool) error {
	if h.integrationRepo == nil {
		return errors.New("integration repository is unavailable")
	}
	if _, err := h.integrationRepo.GetByID(ctx, integrationID); err == nil {
		return nil
	} else if !errors.Is(err, integration.ErrIntegrationNotFound) {
		return err
	}

	name := "Tuya Local"
	switch integrationID {
	case "tuya_cloud":
		name = "Tuya Cloud"
	case "tuya_oauth":
		name = "Smart Life OAuth"
	}
	status := "disconnected"
	if online {
		status = "connected"
	}
	now := time.Now().UTC()
	candidate := &integration.Integration{
		ID: integrationID, Type: integrationID, Name: name,
		Enabled: true, Status: status, CreatedAt: now, UpdatedAt: now,
	}
	if err := h.integrationRepo.Create(ctx, candidate); err != nil {
		// Another request may have created the shared integration row in the
		// meantime. Re-read it before returning an error.
		if _, readErr := h.integrationRepo.GetByID(ctx, integrationID); readErr == nil {
			return nil
		}
		return err
	}
	return nil
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
		AreaID       json.RawMessage `json:"area_id"`
		ExternalID   string          `json:"external_id"`
		Name         string          `json:"name"`
		IntegrationID string         `json:"integration_id"`
		Online       *bool           `json:"online"`
		Manufacturer string          `json:"manufacturer"`
		Model        string          `json:"model"`
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
	deviceCreated := false
	if !deviceExists {
		// Cloud-listed Tuya devices can be visible in the UI before the local
		// adapter has provisioned a registry row (for example, devices without
		// a Local Key). Accept only the canonical tuya_<external-id> identifier
		// and an explicit, supported Tuya integration identity from an admin.
		prefixOK := strings.HasPrefix(deviceID, "tuya_")
		externalID := strings.TrimSpace(req.ExternalID)
		integrationID := strings.TrimSpace(req.IntegrationID)
		name := strings.TrimSpace(req.Name)
		validIntegration := integrationID == "tuya_cloud" ||
			integrationID == "tuya_oauth" || integrationID == "tuya_local"
		validIdentity := prefixOK && externalID != "" &&
			len(externalID) <= 128 && externalID == strings.TrimPrefix(deviceID, "tuya_") &&
			!strings.ContainsAny(externalID, "/?#") && validIntegration &&
			name != "" && len(name) <= 256
		if validIdentity {
			now := time.Now().UTC()
			online := req.Online != nil && *req.Online
			if err := h.ensureTuyaIntegrationRecord(r.Context(), integrationID, online); err != nil {
				writeError(w, http.StatusInternalServerError, "INTEGRATION_UNAVAILABLE", "could not register the Tuya integration")
				return
			}
			metadata := map[string]any{"protocol": integrationID, "cloud_managed": true}
			metadataJSON, _ := json.Marshal(metadata)
			dev = &device.Device{
				ID: deviceID, IntegrationID: integrationID, ExternalID: externalID,
				Name: name, Manufacturer: strings.TrimSpace(req.Manufacturer),
				Model: strings.TrimSpace(req.Model), AreaID: areaID, Online: online,
				Metadata: string(metadataJSON), CreatedAt: now, UpdatedAt: now,
			}
			if err := h.repo.Create(r.Context(), dev); err != nil {
				writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to register cloud device for room assignment")
				return
			}
			deviceExists = true
			deviceCreated = true
		} else if len(linkedEntities) == 0 {
			writeError(w, http.StatusNotFound, "DEVICE_NOT_FOUND", "device not found")
			return
		}
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
		"device_created": deviceCreated,
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
