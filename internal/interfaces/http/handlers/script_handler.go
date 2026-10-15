// Package handlers provides HTTP route handlers for reusable scripts.
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	appscript "github.com/domus-hub/domus/internal/application/script"
	domscript "github.com/domus-hub/domus/internal/domain/script"
	"github.com/go-chi/chi/v5"
)

// ScriptHandler handles HTTP endpoints for scripts.
type ScriptHandler struct {
	service *appscript.Service
}

// NewScriptHandler constructs a ScriptHandler.
func NewScriptHandler(service *appscript.Service) *ScriptHandler {
	return &ScriptHandler{service: service}
}

// List handles GET /api/v1/scripts.
func (h *ScriptHandler) List(w http.ResponseWriter, r *http.Request) {
	scripts, err := h.service.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if scripts == nil {
		scripts = []*domscript.Script{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"scripts": scripts,
		"total":   len(scripts),
	})
}

// Get handles GET /api/v1/scripts/{id}.
func (h *ScriptHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sc, err := h.service.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, domscript.ErrScriptNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "script not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, sc)
}

// Create handles POST /api/v1/scripts.
func (h *ScriptHandler) Create(w http.ResponseWriter, r *http.Request) {
	var sc domscript.Script
	if err := json.NewDecoder(r.Body).Decode(&sc); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "malformed request body")
		return
	}

	if err := h.service.Create(r.Context(), &sc); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, sc)
}

// Update handles PUT /api/v1/scripts/{id}.
func (h *ScriptHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var sc domscript.Script
	if err := json.NewDecoder(r.Body).Decode(&sc); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "malformed request body")
		return
	}
	sc.ID = id

	if err := h.service.Update(r.Context(), &sc); err != nil {
		if errors.Is(err, domscript.ErrScriptNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "script not found")
			return
		}
		writeError(w, http.StatusBadRequest, "VALIDATION_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, sc)
}

// Delete handles DELETE /api/v1/scripts/{id}.
func (h *ScriptHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.service.Delete(r.Context(), id); err != nil {
		if errors.Is(err, domscript.ErrScriptNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "script not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Execute handles POST /api/v1/scripts/{id}/execute.
func (h *ScriptHandler) Execute(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var vars map[string]any
	_ = json.NewDecoder(r.Body).Decode(&vars)

	if err := h.service.Execute(r.Context(), id, vars); err != nil {
		if errors.Is(err, domscript.ErrScriptNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "script not found")
			return
		}
		if errors.Is(err, domscript.ErrScriptAlreadyRunning) {
			writeError(w, http.StatusConflict, "SCRIPT_ALREADY_RUNNING", err.Error())
			return
		}
		if errors.Is(err, domscript.ErrMaxRunsExceeded) {
			writeError(w, http.StatusTooManyRequests, "MAX_RUNS_EXCEEDED", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "EXECUTION_FAILED", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "success",
		"message": "script executed successfully",
	})
}
