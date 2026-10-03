// Package handlers provides HTTP request handlers for scene preset management.
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	appscene "github.com/domus-hub/domus/internal/application/scene"
	"github.com/domus-hub/domus/internal/domain/scene"
	"github.com/go-chi/chi/v5"
)

// SceneHandler handles HTTP endpoints for scenes.
type SceneHandler struct {
	repo    scene.Repository
	service *appscene.Service
}

// NewSceneHandler constructs a SceneHandler.
func NewSceneHandler(repo scene.Repository, service *appscene.Service) *SceneHandler {
	return &SceneHandler{
		repo:    repo,
		service: service,
	}
}

// List handles GET /api/v1/scenes.
func (h *SceneHandler) List(w http.ResponseWriter, r *http.Request) {
	scenes, err := h.repo.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"scenes": scenes,
		"total":  len(scenes),
	})
}

// Create handles POST /api/v1/scenes.
func (h *SceneHandler) Create(w http.ResponseWriter, r *http.Request) {
	var s scene.Scene
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "malformed request body")
		return
	}

	if err := s.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_FAILED", err.Error())
		return
	}

	if err := h.repo.Create(r.Context(), &s); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, s)
}

// Get handles GET /api/v1/scenes/{id}.
func (h *SceneHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	s, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, scene.ErrSceneNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "scene not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, s)
}

// Update handles PUT /api/v1/scenes/{id}.
func (h *SceneHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	existing, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, scene.ErrSceneNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "scene not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	var req scene.Scene
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "malformed request body")
		return
	}

	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Icon != "" {
		existing.Icon = req.Icon
	}
	if len(req.Actions) > 0 {
		existing.Actions = req.Actions
	}

	if err := h.repo.Update(r.Context(), existing); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, existing)
}

// Delete handles DELETE /api/v1/scenes/{id}.
func (h *SceneHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.repo.Delete(r.Context(), id); err != nil {
		if errors.Is(err, scene.ErrSceneNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "scene not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Apply handles POST /api/v1/scenes/{id}/apply.
func (h *SceneHandler) Apply(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if h.service == nil {
		writeError(w, http.StatusInternalServerError, "SERVICE_UNAVAILABLE", "scene service is not configured")
		return
	}

	if err := h.service.ApplyScene(r.Context(), id); err != nil {
		if errors.Is(err, scene.ErrSceneNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "scene not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "applied",
		"message": "scene applied successfully",
	})
}
