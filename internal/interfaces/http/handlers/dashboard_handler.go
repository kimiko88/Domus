package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	appDash "github.com/domus-hub/domus/internal/application/dashboard"
	domainDash "github.com/domus-hub/domus/internal/domain/dashboard"
	"github.com/go-chi/chi/v5"
)

// DashboardHandler exposes REST endpoints for custom widget dashboards.
type DashboardHandler struct {
	svc *appDash.Service
}

// NewDashboardHandler constructs a new DashboardHandler.
func NewDashboardHandler(svc *appDash.Service) *DashboardHandler {
	return &DashboardHandler{svc: svc}
}

// List handles GET /api/v1/dashboards.
func (h *DashboardHandler) List(w http.ResponseWriter, r *http.Request) {
	dashboards, err := h.svc.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if dashboards == nil {
		dashboards = []*domainDash.Dashboard{}
	}
	writeJSON(w, http.StatusOK, dashboards)
}

// Create handles POST /api/v1/dashboards.
func (h *DashboardHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req domainDash.Dashboard
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "malformed request payload")
		return
	}

	if err := h.svc.Create(r.Context(), &req); err != nil {
		if errors.Is(err, domainDash.ErrInvalidTitle) {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, req)
}

// Get handles GET /api/v1/dashboards/{id}.
func (h *DashboardHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	d, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, domainDash.ErrDashboardNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "dashboard not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, d)
}

// Update handles PUT /api/v1/dashboards/{id}.
func (h *DashboardHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req domainDash.Dashboard
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "malformed request payload")
		return
	}
	req.ID = id

	if err := h.svc.Update(r.Context(), &req); err != nil {
		if errors.Is(err, domainDash.ErrDashboardNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "dashboard not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, req)
}

// Delete handles DELETE /api/v1/dashboards/{id}.
func (h *DashboardHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.Delete(r.Context(), id); err != nil {
		if errors.Is(err, domainDash.ErrDashboardNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "dashboard not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
}

// SetDefault handles POST /api/v1/dashboards/{id}/default.
func (h *DashboardHandler) SetDefault(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.SetDefault(r.Context(), id); err != nil {
		if errors.Is(err, domainDash.ErrDashboardNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "dashboard not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"default": true, "id": id})
}
