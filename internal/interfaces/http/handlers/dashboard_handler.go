package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

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

type shareDashboardRequest struct {
	DurationMinutes int  `json:"duration_minutes"` // default 1440 (24h)
	ReadOnly        bool `json:"read_only"`
}

// Share handles POST /api/v1/dashboards/{id}/share.
func (h *DashboardHandler) Share(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req shareDashboardRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	duration := 24 * time.Hour
	if req.DurationMinutes > 0 {
		duration = time.Duration(req.DurationMinutes) * time.Minute
	}

	token, payload, err := h.svc.CreateShareToken(r.Context(), id, duration, req.ReadOnly)
	if err != nil {
		if errors.Is(err, domainDash.ErrDashboardNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "dashboard not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"token":        token,
		"share_url":    "/share/" + token,
		"expires_at":   payload.ExpiresAt,
		"read_only":    payload.ReadOnly,
		"dashboard_id": id,
	})
}

// GetShared handles GET /api/v1/dashboards/shared/{token} (public/guest endpoint).
func (h *DashboardHandler) GetShared(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if token == "" {
		writeError(w, http.StatusBadRequest, "INVALID_TOKEN", "token cannot be empty")
		return
	}

	d, payload, err := h.svc.GetShared(r.Context(), token)
	if err != nil {
		if errors.Is(err, domainDash.ErrInvalidShareToken) {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "share token is invalid or expired")
			return
		}
		if errors.Is(err, domainDash.ErrDashboardNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "dashboard no longer exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"dashboard":  d,
		"share_info": payload,
	})
}
