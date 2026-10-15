// Package handlers provides HTTP route handlers for multi-utility consumption meters.
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	domenergy "github.com/domus-hub/domus/internal/domain/energy"
	"github.com/domus-hub/domus/internal/domain/entity"
	"github.com/go-chi/chi/v5"
)

// UtilityMeterHandler handles HTTP endpoints for electricity, gas, and water meters.
type UtilityMeterHandler struct {
	repo       domenergy.UtilityMeterRepository
	entityRepo entity.Repository
}

// NewUtilityMeterHandler constructs a UtilityMeterHandler.
func NewUtilityMeterHandler(repo domenergy.UtilityMeterRepository, entityRepo entity.Repository) *UtilityMeterHandler {
	return &UtilityMeterHandler{
		repo:       repo,
		entityRepo: entityRepo,
	}
}

// List handles GET /api/v1/energy/utility-meters.
func (h *UtilityMeterHandler) List(w http.ResponseWriter, r *http.Request) {
	uType := r.URL.Query().Get("type")
	meters, err := h.repo.List(r.Context(), uType)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"meters": meters,
		"total":  len(meters),
	})
}

// Get handles GET /api/v1/energy/utility-meters/{id}.
func (h *UtilityMeterHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	m, err := h.repo.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, domenergy.ErrMeterNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "utility meter not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, m)
}

// Create handles POST /api/v1/energy/utility-meters.
func (h *UtilityMeterHandler) Create(w http.ResponseWriter, r *http.Request) {
	var m domenergy.UtilityMeter
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "malformed request body")
		return
	}
	if err := m.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_FAILED", err.Error())
		return
	}
	if err := h.repo.Create(r.Context(), &m); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, m)
}

// Update handles PUT /api/v1/energy/utility-meters/{id}.
func (h *UtilityMeterHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	m, err := h.repo.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, domenergy.ErrMeterNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "utility meter not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	var req domenergy.UtilityMeter
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "malformed request body")
		return
	}

	m.Name = req.Name
	m.UtilityType = req.UtilityType
	m.Unit = req.Unit
	m.CostPerUnit = req.CostPerUnit
	m.SourceEntityID = req.SourceEntityID
	if err := m.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_FAILED", err.Error())
		return
	}

	if err := h.repo.Update(r.Context(), m); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, m)
}

// Delete handles DELETE /api/v1/energy/utility-meters/{id}.
func (h *UtilityMeterHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.repo.Delete(r.Context(), id); err != nil {
		if errors.Is(err, domenergy.ErrMeterNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "utility meter not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Summary handles GET /api/v1/energy/multi-utility-summary.
func (h *UtilityMeterHandler) Summary(w http.ResponseWriter, r *http.Request) {
	meters, err := h.repo.List(r.Context(), "")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	var (
		electricityKwh  float64
		electricityCost float64
		gasM3           float64
		gasCost         float64
		waterLiters     float64
		waterCost       float64
	)

	for _, m := range meters {
		reading := m.CurrentReading
		// Try to fetch live state from entity repo if available
		if h.entityRepo != nil {
			if ent, err := h.entityRepo.GetByID(r.Context(), m.SourceEntityID); err == nil && ent != nil {
				if parsed, err := strconv.ParseFloat(ent.State, 64); err == nil {
					reading = parsed
				}
			}
		}

		cost := reading * m.CostPerUnit
		switch m.UtilityType {
		case "electricity":
			electricityKwh += reading
			electricityCost += cost
		case "gas":
			gasM3 += reading
			gasCost += cost
		case "water":
			waterLiters += reading
			waterCost += cost
		}
	}

	totalCost := electricityCost + gasCost + waterCost
	writeJSON(w, http.StatusOK, map[string]any{
		"electricity": map[string]any{
			"total_consumption": electricityKwh,
			"unit":              "kWh",
			"cost_eur":          electricityCost,
		},
		"gas": map[string]any{
			"total_consumption": gasM3,
			"unit":              "m³",
			"cost_eur":          gasCost,
		},
		"water": map[string]any{
			"total_consumption": waterLiters,
			"unit":              "L",
			"cost_eur":          waterCost,
		},
		"total_cost_eur": totalCost,
		"currency":       "EUR",
	})
}
