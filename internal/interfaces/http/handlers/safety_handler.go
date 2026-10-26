// Package handlers provides HTTP route handlers for safety safe-stop and emergency actuators.
package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/domus-hub/domus/internal/application/safety"
	"github.com/go-chi/chi/v5"
)

// SafetyHandler manages HTTP endpoints for the sub-50ms deterministic safe-stop pipeline.
type SafetyHandler struct {
	svc *safety.Service
}

// NewSafetyHandler constructs a new SafetyHandler.
func NewSafetyHandler(svc *safety.Service) *SafetyHandler {
	return &SafetyHandler{svc: svc}
}

// RegisterRoutes registers all safety endpoints on the provided router.
func (h *SafetyHandler) RegisterRoutes(r chi.Router) {
	r.Get("/status", h.GetStatus)
	r.Get("/valves", h.ListValves)
	r.Post("/valves", h.RegisterValve)
	r.Delete("/valves/{id}", h.DeleteValve)
	r.Post("/trigger", h.TriggerSafeStop)
	r.Post("/reset", h.ResetEmergency)
	r.Get("/incidents", h.ListIncidents)
}

// GetStatus handles GET /api/v1/safety/status.
func (h *SafetyHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "safety service not configured")
		return
	}

	writeJSON(w, http.StatusOK, h.svc.Status())
}

// ListValves handles GET /api/v1/safety/valves.
func (h *SafetyHandler) ListValves(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "safety service not configured")
		return
	}

	valves, err := h.svc.ListValves(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, valves)
}

// RegisterValve handles POST /api/v1/safety/valves.
func (h *SafetyHandler) RegisterValve(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "safety service not configured")
		return
	}

	var req safety.ValveRegistration
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == "" {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "valve id and type are required")
		return
	}

	if err := h.svc.RegisterValve(req); err != nil {
		writeError(w, http.StatusBadRequest, "REGISTRATION_FAILED", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, req)
}

// DeleteValve handles DELETE /api/v1/safety/valves/{id}.
func (h *SafetyHandler) DeleteValve(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "safety service not configured")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "valve ID is required")
		return
	}

	h.svc.UnregisterValve(id)
	writeJSON(w, http.StatusOK, map[string]any{
		"id":      id,
		"deleted": true,
	})
}

// TriggerSafeStop handles POST /api/v1/safety/trigger.
func (h *SafetyHandler) TriggerSafeStop(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "safety service not configured")
		return
	}

	var alert safety.EmergencyAlert
	if err := json.NewDecoder(r.Body).Decode(&alert); err != nil || alert.Hazard == "" {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "hazard type is required")
		return
	}

	if alert.OccurredAt.IsZero() {
		alert.OccurredAt = time.Now().UTC()
	}

	res, err := h.svc.TriggerSafeStop(r.Context(), alert)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "TRIGGER_FAILED", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, res)
}

// ResetEmergency handles POST /api/v1/safety/reset.
func (h *SafetyHandler) ResetEmergency(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "safety service not configured")
		return
	}

	var req struct {
		OperatorID string `json:"operator_id"`
		Reason     string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OperatorID == "" {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "operator_id and reason are required")
		return
	}

	if err := h.svc.ResetEmergency(r.Context(), req.OperatorID, req.Reason); err != nil {
		writeError(w, http.StatusBadRequest, "RESET_FAILED", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"reset":       true,
		"operator_id": req.OperatorID,
		"reason":      req.Reason,
	})
}

// ListIncidents handles GET /api/v1/safety/incidents.
func (h *SafetyHandler) ListIncidents(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "safety service not configured")
		return
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	incidents, err := h.svc.ListIncidents(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, incidents)
}
