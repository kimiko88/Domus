// Package handlers provides HTTP route handlers for Floors and Floor Markers.
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/domus-hub/domus/internal/domain/floorplan"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// FloorplanHandler handles HTTP operations for floors and entity marker positions.
type FloorplanHandler struct {
	repo floorplan.Repository
}

// NewFloorplanHandler constructs a new FloorplanHandler.
func NewFloorplanHandler(repo floorplan.Repository) *FloorplanHandler {
	return &FloorplanHandler{repo: repo}
}

// List returns all floors ordered by level.
func (h *FloorplanHandler) List(w http.ResponseWriter, r *http.Request) {
	floors, err := h.repo.ListFloors(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to list floors")
		return
	}
	if floors == nil {
		floors = []*floorplan.Floor{}
	}
	writeJSON(w, http.StatusOK, floors)
}

type floorRequest struct {
	ID        string `json:"id,omitempty"`
	Name      string `json:"name"`
	Level     int    `json:"level"`
	PlanSVG   string `json:"plan_svg,omitempty"`
	PlanImage string `json:"plan_image,omitempty"`
	SortOrder int    `json:"sort_order"`
}

// Create registers a new floor plan level.
func (h *FloorplanHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req floorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON payload")
		return
	}

	id := req.ID
	if id == "" {
		id = uuid.NewString()
	}

	f := &floorplan.Floor{
		ID:        id,
		Name:      req.Name,
		Level:     req.Level,
		PlanSVG:   req.PlanSVG,
		PlanImage: req.PlanImage,
		SortOrder: req.SortOrder,
	}

	if err := f.Validate(); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "validation_error", err.Error())
		return
	}

	if err := h.repo.CreateFloor(r.Context(), f); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to create floor")
		return
	}

	writeJSON(w, http.StatusCreated, f)
}

// GetByID returns a single floor by ID.
func (h *FloorplanHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	f, err := h.repo.GetFloor(r.Context(), id)
	if err != nil {
		if errors.Is(err, floorplan.ErrFloorNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "floor not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to get floor")
		return
	}
	writeJSON(w, http.StatusOK, f)
}

// Update updates an existing floor.
func (h *FloorplanHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req floorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON payload")
		return
	}

	f := &floorplan.Floor{
		ID:        id,
		Name:      req.Name,
		Level:     req.Level,
		PlanSVG:   req.PlanSVG,
		PlanImage: req.PlanImage,
		SortOrder: req.SortOrder,
	}

	if err := f.Validate(); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "validation_error", err.Error())
		return
	}

	if err := h.repo.UpdateFloor(r.Context(), f); err != nil {
		if errors.Is(err, floorplan.ErrFloorNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "floor not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to update floor")
		return
	}

	writeJSON(w, http.StatusOK, f)
}

// Delete removes a floor and all attached markers.
func (h *FloorplanHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.repo.DeleteFloor(r.Context(), id); err != nil {
		if errors.Is(err, floorplan.ErrFloorNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "floor not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to delete floor")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListMarkers returns all placed markers on a floor.
func (h *FloorplanHandler) ListMarkers(w http.ResponseWriter, r *http.Request) {
	floorID := chi.URLParam(r, "id")
	markers, err := h.repo.ListMarkers(r.Context(), floorID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to list markers")
		return
	}
	if markers == nil {
		markers = []*floorplan.Marker{}
	}
	writeJSON(w, http.StatusOK, markers)
}

type saveMarkerRequest struct {
	ID           string  `json:"id,omitempty"`
	EntityKey    string  `json:"entity_key"`
	XPercent     float64 `json:"x_percent"`
	YPercent     float64 `json:"y_percent"`
	IconOverride string  `json:"icon_override,omitempty"`
	ShowBadge    bool    `json:"show_badge"`
}

// SaveMarker saves or updates an entity position marker.
func (h *FloorplanHandler) SaveMarker(w http.ResponseWriter, r *http.Request) {
	floorID := chi.URLParam(r, "id")
	var req saveMarkerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON payload")
		return
	}

	id := req.ID
	if id == "" {
		id = uuid.NewString()
	}

	m := &floorplan.Marker{
		ID:           id,
		FloorID:      floorID,
		EntityKey:    req.EntityKey,
		XPercent:     req.XPercent,
		YPercent:     req.YPercent,
		IconOverride: req.IconOverride,
		ShowBadge:    req.ShowBadge,
	}

	if err := m.Validate(); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "validation_error", err.Error())
		return
	}

	if err := h.repo.SaveMarker(r.Context(), m); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to save marker")
		return
	}

	writeJSON(w, http.StatusOK, m)
}

// DeleteMarker removes a marker by ID.
func (h *FloorplanHandler) DeleteMarker(w http.ResponseWriter, r *http.Request) {
	markerID := chi.URLParam(r, "markerId")
	if err := h.repo.DeleteMarker(r.Context(), markerID); err != nil {
		if errors.Is(err, floorplan.ErrMarkerNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "marker not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to delete marker")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
