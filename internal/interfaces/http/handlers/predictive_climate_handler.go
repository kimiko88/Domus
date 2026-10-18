// Package handlers provides HTTP route handlers for predictive climate control.
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	appclimate "github.com/domus-hub/domus/internal/application/climate"
	domclimate "github.com/domus-hub/domus/internal/domain/climate"
	"github.com/go-chi/chi/v5"
)

// PredictiveClimateHandler manages HTTP requests for multi-zone predictive climate control.
type PredictiveClimateHandler struct {
	service *appclimate.PredictiveService
}

// NewPredictiveClimateHandler creates a new PredictiveClimateHandler.
func NewPredictiveClimateHandler(service *appclimate.PredictiveService) *PredictiveClimateHandler {
	return &PredictiveClimateHandler{service: service}
}

// RegisterRoutes mounts predictive climate endpoints onto a chi router.
func (h *PredictiveClimateHandler) RegisterRoutes(r chi.Router) {
	r.Get("/status", h.GetStatus)
	r.Get("/zones", h.ListZones)
	r.Post("/zones", h.CreateZone)
	r.Get("/zones/{id}", h.GetZone)
	r.Put("/zones/{id}", h.UpdateZone)
	r.Delete("/zones/{id}", h.DeleteZone)
	r.Post("/zones/{id}/evaluate", h.EvaluateZone)
	r.Get("/zones/{id}/evaluations", h.GetZoneEvaluations)
	r.Post("/evaluate-all", h.EvaluateAll)
}

// GetStatus handles GET /api/v1/climate/predictive/status.
func (h *PredictiveClimateHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	status, err := h.service.GetDashboardStatus(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, status)
}

// ListZones handles GET /api/v1/climate/predictive/zones.
func (h *PredictiveClimateHandler) ListZones(w http.ResponseWriter, r *http.Request) {
	zones, err := h.service.ListZones(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if zones == nil {
		zones = []*domclimate.PredictiveZoneConfig{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"zones": zones,
		"total": len(zones),
	})
}

// GetZone handles GET /api/v1/climate/predictive/zones/{id}.
func (h *PredictiveClimateHandler) GetZone(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	zone, err := h.service.GetZone(r.Context(), id)
	if err != nil {
		if errors.Is(err, domclimate.ErrPredictiveZoneNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "predictive zone not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, zone)
}

// CreateZone handles POST /api/v1/climate/predictive/zones.
func (h *PredictiveClimateHandler) CreateZone(w http.ResponseWriter, r *http.Request) {
	var cfg domclimate.PredictiveZoneConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "malformed request body")
		return
	}

	if err := h.service.SaveZone(r.Context(), &cfg); err != nil {
		writeError(w, http.StatusBadRequest, "SAVE_FAILED", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, cfg)
}

// UpdateZone handles PUT /api/v1/climate/predictive/zones/{id}.
func (h *PredictiveClimateHandler) UpdateZone(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var cfg domclimate.PredictiveZoneConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "malformed request body")
		return
	}
	cfg.ID = id

	if err := h.service.SaveZone(r.Context(), &cfg); err != nil {
		writeError(w, http.StatusBadRequest, "UPDATE_FAILED", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, cfg)
}

// DeleteZone handles DELETE /api/v1/climate/predictive/zones/{id}.
func (h *PredictiveClimateHandler) DeleteZone(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.service.DeleteZone(r.Context(), id); err != nil {
		if errors.Is(err, domclimate.ErrPredictiveZoneNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "predictive zone not found")
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

// EvaluateZone handles POST /api/v1/climate/predictive/zones/{id}/evaluate.
func (h *PredictiveClimateHandler) EvaluateZone(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	eval, err := h.service.EvaluateZone(r.Context(), id)
	if err != nil {
		if errors.Is(err, domclimate.ErrPredictiveZoneNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "predictive zone not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "EVALUATION_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, eval)
}

// EvaluateAll handles POST /api/v1/climate/predictive/evaluate-all.
func (h *PredictiveClimateHandler) EvaluateAll(w http.ResponseWriter, r *http.Request) {
	results, err := h.service.EvaluateAll(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "EVALUATION_FAILED", err.Error())
		return
	}
	if results == nil {
		results = []*domclimate.PredictiveEvaluation{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"evaluations": results,
		"count":       len(results),
	})
}

// GetZoneEvaluations handles GET /api/v1/climate/predictive/zones/{id}/evaluations.
func (h *PredictiveClimateHandler) GetZoneEvaluations(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	evals, err := h.service.GetZoneEvaluations(r.Context(), id, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if evals == nil {
		evals = []*domclimate.PredictiveEvaluation{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"evaluations": evals,
		"count":       len(evals),
	})
}
