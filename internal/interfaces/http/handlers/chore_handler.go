// Package handlers provides HTTP route handlers for domestic chores.
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	appchore "github.com/domus-hub/domus/internal/application/chore"
	domainchore "github.com/domus-hub/domus/internal/domain/chore"
	domusmw "github.com/domus-hub/domus/internal/interfaces/http/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// ChoreHandler handles HTTP operations for chores and task management.
type ChoreHandler struct {
	repo    domainchore.Repository
	service *appchore.Service
}

// NewChoreHandler constructs a new ChoreHandler.
func NewChoreHandler(repo domainchore.Repository, service *appchore.Service) *ChoreHandler {
	return &ChoreHandler{
		repo:    repo,
		service: service,
	}
}

// List returns chores matching optional filters.
func (h *ChoreHandler) List(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	assignedUser := r.URL.Query().Get("user_id")
	areaID := r.URL.Query().Get("area_id")

	filter := domainchore.Filter{
		Status:         domainchore.Status(status),
		AssignedUserID: assignedUser,
		AreaID:         areaID,
	}

	chores, err := h.repo.List(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to list chores")
		return
	}
	if chores == nil {
		chores = []*domainchore.Chore{}
	}
	writeJSON(w, http.StatusOK, chores)
}

type choreRequest struct {
	ID             string                    `json:"id,omitempty"`
	Title          string                    `json:"title"`
	Description    string                    `json:"description,omitempty"`
	AreaID         string                    `json:"area_id,omitempty"`
	AssignedUserID string                    `json:"assigned_user_id,omitempty"`
	Priority       domainchore.Priority      `json:"priority,omitempty"`
	DueDate        *string                   `json:"due_date,omitempty"`
	RecurrenceType domainchore.RecurrenceType `json:"recurrence_type,omitempty"`
	RecurrenceRule string                    `json:"recurrence_rule,omitempty"`
}

// Create registers a new domestic chore.
func (h *ChoreHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req choreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON payload")
		return
	}

	id := req.ID
	if id == "" {
		id = uuid.NewString()
	}

	var due *time.Time
	if req.DueDate != nil && *req.DueDate != "" {
		t, err := time.Parse(time.RFC3339, *req.DueDate)
		if err == nil {
			due = &t
		}
	}

	c := &domainchore.Chore{
		ID:             id,
		Title:          req.Title,
		Description:    req.Description,
		AreaID:         req.AreaID,
		AssignedUserID: req.AssignedUserID,
		Priority:       req.Priority,
		DueDate:        due,
		RecurrenceType: req.RecurrenceType,
		RecurrenceRule: req.RecurrenceRule,
		Status:         domainchore.StatusPending,
	}

	if err := c.Validate(); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "validation_error", err.Error())
		return
	}

	if err := h.service.CreateTask(r.Context(), c); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to create chore")
		return
	}

	writeJSON(w, http.StatusCreated, c)
}

// GetByID returns a single chore.
func (h *ChoreHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	c, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, domainchore.ErrChoreNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "chore not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to get chore")
		return
	}
	writeJSON(w, http.StatusOK, c)
}

// Update modifies an existing chore.
func (h *ChoreHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req choreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON payload")
		return
	}

	existing, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, domainchore.ErrChoreNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "chore not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to get chore")
		return
	}

	existing.Title = req.Title
	existing.Description = req.Description
	existing.AreaID = req.AreaID
	existing.AssignedUserID = req.AssignedUserID
	if req.Priority != "" {
		existing.Priority = req.Priority
	}
	if req.RecurrenceType != "" {
		existing.RecurrenceType = req.RecurrenceType
	}
	existing.RecurrenceRule = req.RecurrenceRule

	if req.DueDate != nil && *req.DueDate != "" {
		if t, err := time.Parse(time.RFC3339, *req.DueDate); err == nil {
			existing.DueDate = &t
		}
	} else if req.DueDate != nil && *req.DueDate == "" {
		existing.DueDate = nil
	}

	if err := existing.Validate(); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "validation_error", err.Error())
		return
	}

	if err := h.repo.Update(r.Context(), existing); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to update chore")
		return
	}

	writeJSON(w, http.StatusOK, existing)
}

// Complete marks a chore as completed and reschedules if recurring.
func (h *ChoreHandler) Complete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	completedBy := ""
	if u := domusmw.GetUser(r.Context()); u != nil {
		completedBy = u.ID
	}

	if err := h.service.CompleteTask(r.Context(), id, completedBy); err != nil {
		if errors.Is(err, domainchore.ErrChoreNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "chore not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to complete chore")
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Delete removes a chore.
func (h *ChoreHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.repo.Delete(r.Context(), id); err != nil {
		if errors.Is(err, domainchore.ErrChoreNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "chore not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to delete chore")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
