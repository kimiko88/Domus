// Package handlers provides HTTP route handlers for Entities.
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/domus-hub/domus/internal/domain/entity"
	"github.com/domus-hub/domus/internal/domain/events"
	domusmw "github.com/domus-hub/domus/internal/interfaces/http/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// EntityHandler handles HTTP operations for Entity objects.
type EntityHandler struct {
	repo entity.Repository
	bus  events.Bus
}

// NewEntityHandler constructs an EntityHandler.
func NewEntityHandler(repo entity.Repository, bus events.Bus) *EntityHandler {
	return &EntityHandler{
		repo: repo,
		bus:  bus,
	}
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

	writeJSON(w, http.StatusOK, entities)
}

// GetByID returns entity details by path ID parameter.
func (h *EntityHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ent, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, entity.ErrEntityNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "entity not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to get entity")
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
	ent, err := h.repo.GetByID(r.Context(), id)
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
