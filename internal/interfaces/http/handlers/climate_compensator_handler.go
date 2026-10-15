// Package handlers provides HTTP route handlers for intelligent climate compensation.
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	appclimate "github.com/domus-hub/domus/internal/application/climate"
	domclimate "github.com/domus-hub/domus/internal/domain/climate"
	"github.com/go-chi/chi/v5"
)

// ClimateCompensatorHandler manages HTTP requests for Better Thermostat TRV compensations.
type ClimateCompensatorHandler struct {
	service *appclimate.Service
}

// NewClimateCompensatorHandler creates a new ClimateCompensatorHandler.
func NewClimateCompensatorHandler(service *appclimate.Service) *ClimateCompensatorHandler {
	return &ClimateCompensatorHandler{service: service}
}

// List handles GET /api/v1/climate/compensations.
func (h *ClimateCompensatorHandler) List(w http.ResponseWriter, r *http.Request) {
	comps, err := h.service.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if comps == nil {
		comps = []*domclimate.ClimateCompensation{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"compensations": comps,
		"total":         len(comps),
	})
}

// Get handles GET /api/v1/climate/compensations/{id}.
func (h *ClimateCompensatorHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	comp, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, domclimate.ErrCompensationNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "climate compensation not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, comp)
}

// Create handles POST /api/v1/climate/compensations.
func (h *ClimateCompensatorHandler) Create(w http.ResponseWriter, r *http.Request) {
	var comp domclimate.ClimateCompensation
	if err := json.NewDecoder(r.Body).Decode(&comp); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "malformed request body")
		return
	}

	if err := h.service.Save(r.Context(), &comp); err != nil {
		writeError(w, http.StatusBadRequest, "SAVE_FAILED", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, comp)
}

// Update handles PUT /api/v1/climate/compensations/{id}.
func (h *ClimateCompensatorHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var comp domclimate.ClimateCompensation
	if err := json.NewDecoder(r.Body).Decode(&comp); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "malformed request body")
		return
	}
	comp.ID = id

	if err := h.service.Save(r.Context(), &comp); err != nil {
		writeError(w, http.StatusBadRequest, "UPDATE_FAILED", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, comp)
}

// Delete handles DELETE /api/v1/climate/compensations/{id}.
func (h *ClimateCompensatorHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.service.Delete(r.Context(), id); err != nil {
		if errors.Is(err, domclimate.ErrCompensationNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "climate compensation not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "deleted",
		"id":     id,
	})
}
