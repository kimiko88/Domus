// Package handlers provides HTTP route handlers for weekly schedule matrix.
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	appschedule "github.com/domus-hub/domus/internal/application/schedule"
	domschedule "github.com/domus-hub/domus/internal/domain/schedule"
	"github.com/go-chi/chi/v5"
)

// WeeklyScheduleHandler manages HTTP endpoints for the 7-day x 24h cronotermostato visual matrix.
type WeeklyScheduleHandler struct {
	service *appschedule.MatrixService
}

// NewWeeklyScheduleHandler creates a new WeeklyScheduleHandler.
func NewWeeklyScheduleHandler(service *appschedule.MatrixService) *WeeklyScheduleHandler {
	return &WeeklyScheduleHandler{service: service}
}

// List handles GET /api/v1/schedules/weekly.
func (h *WeeklyScheduleHandler) List(w http.ResponseWriter, r *http.Request) {
	schedules, err := h.service.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if schedules == nil {
		schedules = []*domschedule.WeeklySchedule{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"schedules": schedules,
		"total":     len(schedules),
	})
}

// Get handles GET /api/v1/schedules/weekly/{id}.
func (h *WeeklyScheduleHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ws, err := h.service.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, domschedule.ErrScheduleNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "weekly schedule not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, ws)
}

// Create handles POST /api/v1/schedules/weekly.
func (h *WeeklyScheduleHandler) Create(w http.ResponseWriter, r *http.Request) {
	var ws domschedule.WeeklySchedule
	if err := json.NewDecoder(r.Body).Decode(&ws); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "malformed request body")
		return
	}

	if err := h.service.Create(r.Context(), &ws); err != nil {
		writeError(w, http.StatusBadRequest, "CREATE_FAILED", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, ws)
}

// Update handles PUT /api/v1/schedules/weekly/{id}.
func (h *WeeklyScheduleHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var ws domschedule.WeeklySchedule
	if err := json.NewDecoder(r.Body).Decode(&ws); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "malformed request body")
		return
	}
	ws.ID = id

	if err := h.service.Update(r.Context(), &ws); err != nil {
		writeError(w, http.StatusBadRequest, "UPDATE_FAILED", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, ws)
}

// Delete handles DELETE /api/v1/schedules/weekly/{id}.
func (h *WeeklyScheduleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.service.Delete(r.Context(), id); err != nil {
		if errors.Is(err, domschedule.ErrScheduleNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "weekly schedule not found")
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

// Evaluate handles POST /api/v1/schedules/weekly/evaluate.
func (h *WeeklyScheduleHandler) Evaluate(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()
	if err := h.service.EvaluateAll(r.Context(), now); err != nil {
		writeError(w, http.StatusInternalServerError, "EVALUATION_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":       "evaluated",
		"evaluated_at": now.Format(time.RFC3339),
	})
}
