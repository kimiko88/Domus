// Package handlers provides HTTP route handlers for Areas.
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/domus-hub/domus/internal/domain/area"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// AreaHandler handles HTTP operations for Area entities.
type AreaHandler struct {
	repo area.Repository
}

// NewAreaHandler constructs an AreaHandler.
func NewAreaHandler(repo area.Repository) *AreaHandler {
	return &AreaHandler{repo: repo}
}

// List returns all areas.
func (h *AreaHandler) List(w http.ResponseWriter, r *http.Request) {
	areas, err := h.repo.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to list areas")
		return
	}
	writeJSON(w, http.StatusOK, areas)
}

type createAreaRequest struct {
	ID           string `json:"id,omitempty"`
	Name         string `json:"name"`
	Icon         string `json:"icon,omitempty"`
	ParentAreaID string `json:"parent_area_id,omitempty"`
	SortOrder    int    `json:"sort_order"`
}

// Create handles creating a new area.
func (h *AreaHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createAreaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON payload")
		return
	}

	id := req.ID
	if id == "" {
		id = uuid.NewString()
	}

	now := time.Now().UTC()
	a := &area.Area{
		ID:           id,
		Name:         req.Name,
		Icon:         req.Icon,
		ParentAreaID: req.ParentAreaID,
		SortOrder:    req.SortOrder,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := a.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	if err := h.repo.Create(r.Context(), a); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to create area")
		return
	}

	writeJSON(w, http.StatusCreated, a)
}

// GetByID returns an area by its path ID parameter.
func (h *AreaHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	a, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, area.ErrAreaNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "area not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to get area")
		return
	}

	writeJSON(w, http.StatusOK, a)
}
