// Package handlers provides HTTP route handlers for Entities.
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	appauto "github.com/domus-hub/domus/internal/application/automation"
	"github.com/domus-hub/domus/internal/domain/entity"
	"github.com/domus-hub/domus/internal/domain/events"
	domusmw "github.com/domus-hub/domus/internal/interfaces/http/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// EntityHandler handles HTTP operations for Entity objects.
type EntityHandler struct {
	repo          entity.Repository
	bus           events.Bus
	climateEngine *appauto.Engine
}

// NewEntityHandler constructs an EntityHandler.
func NewEntityHandler(repo entity.Repository, bus events.Bus) *EntityHandler {
	return &EntityHandler{
		repo: repo,
		bus:  bus,
	}
}

func (h *EntityHandler) resolveEntity(ctx context.Context, idOrKey string) (*entity.Entity, error) {
	ent, err := h.repo.GetByID(ctx, idOrKey)
	if err == nil && ent != nil {
		return ent, nil
	}
	if errors.Is(err, entity.ErrEntityNotFound) {
		return h.repo.GetByKey(ctx, idOrKey)
	}
	return nil, err
}

// WithClimateEngine attaches the automation engine used for temporary
// thermostat profile overrides.
func (h *EntityHandler) WithClimateEngine(engine *appauto.Engine) *EntityHandler {
	h.climateEngine = engine
	return h
}

type climateProfileRequest struct {
	Profile         string   `json:"profile"`
	Temperature     *float64 `json:"temperature,omitempty"`
	DurationMinutes int      `json:"duration_minutes"`
}

// ApplyClimateProfile applies a thermostat profile. A positive duration restores
// the previous setpoint automatically; zero leaves the selected setpoint active.
func (h *EntityHandler) ApplyClimateProfile(w http.ResponseWriter, r *http.Request) {
	if h.climateEngine == nil {
		writeError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "thermostat profile service is unavailable")
		return
	}
	id := chi.URLParam(r, "id")
	ent, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, entity.ErrEntityNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "entity not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get thermostat")
		return
	}
	if currentUser := domusmw.GetUser(r.Context()); currentUser != nil && !currentUser.CanAccessEntityInArea(ent.ID, ent.AreaID) {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "thermostat is outside the member's allowed resources")
		return
	}
	if ent.Domain != "climate" || !ent.Writable || !ent.Available {
		writeError(w, http.StatusUnprocessableEntity, "INVALID_THERMOSTAT", "target must be an available, writable climate entity")
		return
	}
	var req climateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "malformed request body")
		return
	}
	if req.DurationMinutes < 0 || req.DurationMinutes > 1440 {
		writeError(w, http.StatusBadRequest, "INVALID_DURATION", "duration_minutes must be between 0 and 1440")
		return
	}
	if err := h.climateEngine.ApplyClimateProfile(r.Context(), ent.EntityKey, req.Profile, req.Temperature, req.DurationMinutes); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "CLIMATE_PROFILE_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{
		"status": "accepted", "entity_key": ent.EntityKey,
		"profile": req.Profile, "duration_minutes": req.DurationMinutes,
	})
}

// List returns entities matching optional query filters.
func (h *EntityHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := entity.ListFilter{
		Domain:   q.Get("domain"),
		AreaID:   q.Get("area_id"),
		DeviceID: q.Get("device_id"),
	}

	if wStr := q.Get("writable"); wStr != "" {
		if b, err := strconv.ParseBool(wStr); err == nil {
			filter.Writable = &b
		}
	}

	entities, err := h.repo.List(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to list entities")
		return
	}
	if currentUser := domusmw.GetUser(r.Context()); currentUser != nil {
		visible := make([]*entity.Entity, 0, len(entities))
		for _, item := range entities {
			if currentUser.CanAccessEntityInArea(item.ID, item.AreaID) {
				visible = append(visible, item)
			}
		}
		entities = visible
	}

	writeJSON(w, http.StatusOK, entities)
}

// GetByID returns entity details by path ID parameter (supports UUID or entity_key).
func (h *EntityHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ent, err := h.resolveEntity(r.Context(), id)
	if err != nil {
		if errors.Is(err, entity.ErrEntityNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "entity not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to get entity")
		return
	}

	if currentUser := domusmw.GetUser(r.Context()); currentUser != nil && !currentUser.CanAccessEntityInArea(ent.ID, ent.AreaID) {
		writeError(w, http.StatusForbidden, "forbidden", "entity is outside the member's allowed resources")
		return
	}
	writeJSON(w, http.StatusOK, ent)
}

type commandRequest struct {
	Service string         `json:"service"`
	Params  map[string]any `json:"params"`
}

// Command accepts and dispatches a control command for an entity.
func (h *EntityHandler) Command(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ent, err := h.resolveEntity(r.Context(), id)
	if err != nil {
		if errors.Is(err, entity.ErrEntityNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "entity not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to get entity")
		return
	}

	if !ent.Writable {
		writeError(w, http.StatusForbidden, "not_writable", "entity is read-only")
		return
	}
	if currentUser := domusmw.GetUser(r.Context()); currentUser != nil {
		permissions := currentUser.EffectivePermissions()
		if !currentUser.CanAccessEntityInArea(ent.ID, ent.AreaID) || permissions == nil || !permissions.CanControlDevices {
			writeError(w, http.StatusForbidden, "forbidden", "member is not allowed to control this entity")
			return
		}
	}

	var req commandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON payload")
		return
	}

	if req.Service == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "service is required")
		return
	}

	source := "system"
	if u := domusmw.GetUser(r.Context()); u != nil {
		source = "user:" + u.Username
	}

	// Dispatch CommandRequested domain event
	if h.bus != nil {
		_ = h.bus.Publish(r.Context(), events.Event{
			ID:         uuid.NewString(),
			Type:       events.CommandRequested,
			OccurredAt: time.Now().UTC(),
			Source:     source,
			Payload: map[string]any{
				"entity_id":  ent.ID,
				"entity_key": ent.EntityKey,
				"service":    req.Service,
				"params":     req.Params,
			},
		})
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"status":     "accepted",
		"entity_key": ent.EntityKey,
		"service":    req.Service,
	})
}

type serviceCallRequest struct {
	EntityID string         `json:"entity_id"`
	Data     map[string]any `json:"data"`
}

// CallService accepts commands in mobile/service style: POST /api/v1/services/{domain}/{service}
func (h *EntityHandler) CallService(w http.ResponseWriter, r *http.Request) {
	domain := chi.URLParam(r, "domain")
	service := chi.URLParam(r, "service")

	var req serviceCallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON payload")
		return
	}

	if req.EntityID == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "entity_id is required")
		return
	}

	ent, err := h.resolveEntity(r.Context(), req.EntityID)
	if err != nil {
		if errors.Is(err, entity.ErrEntityNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "entity not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to get entity")
		return
	}

	if !ent.Writable {
		writeError(w, http.StatusForbidden, "not_writable", "entity is read-only")
		return
	}

	source := "mobile"
	if u := domusmw.GetUser(r.Context()); u != nil {
		source = "user:" + u.Username
	}

	if h.bus != nil {
		_ = h.bus.Publish(r.Context(), events.Event{
			ID:         uuid.NewString(),
			Type:       events.CommandRequested,
			OccurredAt: time.Now().UTC(),
			Source:     source,
			Payload: map[string]any{
				"entity_id":  ent.ID,
				"entity_key": ent.EntityKey,
				"service":    service,
				"domain":     domain,
				"params":     req.Data,
			},
		})
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"status":     "accepted",
		"entity_key": ent.EntityKey,
		"service":    service,
		"domain":     domain,
	})
}
