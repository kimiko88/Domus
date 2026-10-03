// Package handlers provides HTTP request handlers for automation management.
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	appauto "github.com/domus-hub/domus/internal/application/automation"
	"github.com/domus-hub/domus/internal/domain/automation"
	"github.com/go-chi/chi/v5"
)

// AutomationHandler handles HTTP endpoints for automations.
type AutomationHandler struct {
	repo   automation.Repository
	engine *appauto.Engine
}

// NewAutomationHandler constructs an AutomationHandler.
func NewAutomationHandler(repo automation.Repository, engine *appauto.Engine) *AutomationHandler {
	return &AutomationHandler{
		repo:   repo,
		engine: engine,
	}
}

// List handles GET /api/v1/automations.
func (h *AutomationHandler) List(w http.ResponseWriter, r *http.Request) {
	var filter automation.ListFilter
	if enabledStr := r.URL.Query().Get("enabled"); enabledStr != "" {
		enabled := enabledStr == "true" || enabledStr == "1"
		filter.Enabled = &enabled
	}

	automations, err := h.repo.List(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"automations": automations,
		"total":       len(automations),
	})
}

// Create handles POST /api/v1/automations.
func (h *AutomationHandler) Create(w http.ResponseWriter, r *http.Request) {
	var a automation.Automation
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "malformed request body")
		return
	}

	if err := a.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_FAILED", err.Error())
		return
	}

	if err := h.repo.Create(r.Context(), &a); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, a)
}

// Get handles GET /api/v1/automations/{id}.
func (h *AutomationHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	a, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, automation.ErrAutomationNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "automation not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, a)
}

// Update handles PUT /api/v1/automations/{id}.
func (h *AutomationHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	existing, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, automation.ErrAutomationNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "automation not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	var req automation.Automation
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "malformed request body")
		return
	}

	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Description != "" {
		existing.Description = req.Description
	}
	existing.Enabled = req.Enabled
	if len(req.Triggers) > 0 {
		existing.Triggers = req.Triggers
	}
	if len(req.Conditions) > 0 {
		existing.Conditions = req.Conditions
	}
	if len(req.Actions) > 0 {
		existing.Actions = req.Actions
	}
	if req.ExecutionMode != "" {
		existing.ExecutionMode = req.ExecutionMode
	}
	if req.CooldownSeconds >= 0 {
		existing.CooldownSeconds = req.CooldownSeconds
	}

	if err := h.repo.Update(r.Context(), existing); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, existing)
}

// Delete handles DELETE /api/v1/automations/{id}.
func (h *AutomationHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.repo.Delete(r.Context(), id); err != nil {
		if errors.Is(err, automation.ErrAutomationNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "automation not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Trigger handles POST /api/v1/automations/{id}/trigger.
func (h *AutomationHandler) Trigger(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if h.engine == nil {
		writeError(w, http.StatusInternalServerError, "ENGINE_UNAVAILABLE", "automation engine is not running")
		return
	}

	if err := h.engine.TriggerAutomation(r.Context(), id); err != nil {
		if errors.Is(err, automation.ErrAutomationNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "automation not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{
		"status":  "accepted",
		"message": "automation triggered",
	})
}

// ListRuns handles GET /api/v1/automations/{id}/runs.
func (h *AutomationHandler) ListRuns(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	limit := 20
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	}

	runs, err := h.repo.ListRuns(r.Context(), id, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"runs":  runs,
		"total": len(runs),
	})
}
