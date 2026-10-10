package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	appHelper "github.com/domus-hub/domus/internal/application/helper"
	domainHelper "github.com/domus-hub/domus/internal/domain/helper"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// HelperHandler exposes REST endpoints for User Helpers.
type HelperHandler struct {
	svc *appHelper.Service
}

// NewHelperHandler constructs a new HelperHandler.
func NewHelperHandler(svc *appHelper.Service) *HelperHandler {
	return &HelperHandler{svc: svc}
}

// List handles GET /api/v1/helpers.
func (h *HelperHandler) List(w http.ResponseWriter, r *http.Request) {
	domainFilter := r.URL.Query().Get("domain")
	helpers, err := h.svc.List(r.Context(), domainFilter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if helpers == nil {
		helpers = []*domainHelper.Helper{}
	}
	writeJSON(w, http.StatusOK, helpers)
}

// CreateHelperRequest defines payload for helper creation.
type CreateHelperRequest struct {
	Name        string  `json:"name"`
	Domain      string  `json:"domain"` // "input_boolean", "input_number", "timer"
	EntityKey   string  `json:"entity_key,omitempty"`
	Icon        string  `json:"icon,omitempty"`
	Min         float64 `json:"min,omitempty"`
	Max         float64 `json:"max,omitempty"`
	Step        float64 `json:"step,omitempty"`
	InitialVal  float64 `json:"initial_val,omitempty"`
	Unit        string  `json:"unit,omitempty"`
	DurationSec int     `json:"duration_sec,omitempty"`
}

// Create handles POST /api/v1/helpers.
func (h *HelperHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateHelperRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "malformed request payload")
		return
	}

	id := uuid.NewString()
	var newHelper *domainHelper.Helper

	switch req.Domain {
	case domainHelper.DomainInputBoolean:
		newHelper = domainHelper.NewInputBoolean(id, req.Name, req.EntityKey, req.Icon, false)
	case domainHelper.DomainInputNumber:
		newHelper = domainHelper.NewInputNumber(id, req.Name, req.EntityKey, req.Min, req.Max, req.Step, req.InitialVal, req.Unit)
	case domainHelper.DomainTimer:
		newHelper = domainHelper.NewTimer(id, req.Name, req.EntityKey, req.DurationSec)
	default:
		writeError(w, http.StatusBadRequest, "INVALID_DOMAIN", "domain must be input_boolean, input_number, or timer")
		return
	}

	if err := h.svc.Create(r.Context(), newHelper); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, newHelper)
}

// Get handles GET /api/v1/helpers/{id}.
func (h *HelperHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	hlp, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, domainHelper.ErrHelperNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "helper not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, hlp)
}

// Delete handles DELETE /api/v1/helpers/{id}.
func (h *HelperHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.Delete(r.Context(), id); err != nil {
		if errors.Is(err, domainHelper.ErrHelperNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "helper not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
}

// ActionRequest defines an operation on a helper.
type ActionRequest struct {
	Action string         `json:"action"` // "turn_on", "turn_off", "toggle", "set_value", "start", "pause", "cancel"
	Params map[string]any `json:"params,omitempty"`
}

// Action handles POST /api/v1/helpers/{id}/action.
func (h *HelperHandler) Action(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	hlp, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "helper not found")
		return
	}

	var req ActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "malformed request payload")
		return
	}

	if err := h.svc.ExecuteAction(r.Context(), hlp.EntityKey, req.Action, req.Params); err != nil {
		writeError(w, http.StatusBadRequest, "ACTION_FAILED", err.Error())
		return
	}

	updated, _ := h.svc.GetByID(r.Context(), id)
	writeJSON(w, http.StatusOK, updated)
}
