// Package handlers provides HTTP request handlers for automation management.
package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	appauto "github.com/domus-hub/domus/internal/application/automation"
	"github.com/domus-hub/domus/internal/domain/automation"
	"github.com/domus-hub/domus/internal/domain/entity"
	"github.com/go-chi/chi/v5"
)

// AutomationHandler handles HTTP endpoints for automations.
type AutomationHandler struct {
	repo       automation.Repository
	engine     *appauto.Engine
	entityRepo entity.Repository
}

// NewAutomationHandler constructs an AutomationHandler.
func NewAutomationHandler(repo automation.Repository, engine *appauto.Engine) *AutomationHandler {
	return &AutomationHandler{
		repo:   repo,
		engine: engine,
	}
}

// WithEntities attaches the entity repository for simulation.
func (h *AutomationHandler) WithEntities(entities entity.Repository) *AutomationHandler {
	h.entityRepo = entities
	return h
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

	// Pointer fields preserve the distinction between an omitted field and an
	// explicit zero/false/empty value. This endpoint is used for partial updates
	// by the UI, even though it uses the PUT method for compatibility.
	var req struct {
		Name            *string                  `json:"name"`
		Description     *string                  `json:"description"`
		Enabled         *bool                    `json:"enabled"`
		Triggers        *[]automation.Trigger    `json:"triggers"`
		Conditions      *[]automation.Condition  `json:"conditions"`
		Actions         *[]automation.Action     `json:"actions"`
		ExecutionMode   *string                  `json:"execution_mode"`
		CooldownSeconds *int                     `json:"cooldown_seconds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "malformed request body")
		return
	}

	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}
	if req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}
	if req.Triggers != nil {
		existing.Triggers = *req.Triggers
	}
	if req.Conditions != nil {
		existing.Conditions = *req.Conditions
	}
	if req.Actions != nil {
		existing.Actions = *req.Actions
	}
	if req.ExecutionMode != nil {
		existing.ExecutionMode = *req.ExecutionMode
	}
	if req.CooldownSeconds != nil {
		existing.CooldownSeconds = *req.CooldownSeconds
	}

	if err := existing.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_FAILED", err.Error())
		return
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
	const maxLimit = 200
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			if l > maxLimit {
				l = maxLimit
			}
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

// ListBlueprints handles GET /api/v1/automations/blueprints.
func (h *AutomationHandler) ListBlueprints(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, appauto.Catalog)
}

// InstantiateBlueprint handles POST /api/v1/automations/blueprints/{id}/instantiate.
func (h *AutomationHandler) InstantiateBlueprint(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	bp, found := appauto.GetBlueprint(id)
	if !found {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "blueprint not found")
		return
	}

	var raw map[string]any
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "malformed request body")
		return
	}

	params := make(map[string]string)
	if p, ok := raw["parameters"].(map[string]any); ok {
		for k, v := range p {
			params[k] = fmt.Sprint(v)
		}
	} else {
		for k, v := range raw {
			if str, ok := v.(string); ok {
				params[k] = str
			} else {
				params[k] = fmt.Sprint(v)
			}
		}
	}

	a, err := bp.Factory(params)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "BLUEPRINT_ERROR", err.Error())
		return
	}

	if err := h.repo.Create(r.Context(), a); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, a)
}

// DryRun handles POST /api/v1/automations/{id}/dry-run.
func (h *AutomationHandler) DryRun(w http.ResponseWriter, r *http.Request) {
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

	// A request body may provide an unsaved draft. Evaluate it in memory only:
	// dry-run must never persist changes or dispatch device commands.
	var draft automation.Automation
	if err := json.NewDecoder(r.Body).Decode(&draft); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "malformed dry-run request body")
		return
	}
	if draft.Name != "" || len(draft.Triggers) > 0 || len(draft.Actions) > 0 || draft.Conditions != nil {
		draft.ID = a.ID
		if err := draft.Validate(); err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_FAILED", err.Error())
			return
		}
		a = &draft
	}

	sim := appauto.NewSimulator(h.entityRepo, nil)
	result, err := sim.DryRun(r.Context(), a, time.Now().UTC())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "SIMULATION_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}
