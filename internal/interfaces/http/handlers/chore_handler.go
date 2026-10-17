// Package handlers provides HTTP route handlers for domestic chores.
package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	appchore "github.com/domus-hub/domus/internal/application/chore"
	"github.com/domus-hub/domus/internal/domain/calendar"
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

// ExportICal outputs all pending and recurring chores as an RFC 5545 iCalendar stream.
func (h *ChoreHandler) ExportICal(w http.ResponseWriter, r *http.Request) {
	chores, err := h.repo.List(r.Context(), domainchore.Filter{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to list chores")
		return
	}

	var sb strings.Builder
	sb.WriteString("BEGIN:VCALENDAR\r\n")
	sb.WriteString("VERSION:2.0\r\n")
	sb.WriteString("PRODID:-//Domus Hub//Domus Chores Calendar//IT\r\n")
	sb.WriteString("CALSCALE:GREGORIAN\r\n")
	sb.WriteString("METHOD:PUBLISH\r\n")
	sb.WriteString("X-WR-CALNAME:Faccende Domus\r\n")
	sb.WriteString("X-WR-TIMEZONE:Europe/Rome\r\n")

	nowStr := time.Now().UTC().Format("20060102T150405Z")

	for _, c := range chores {
		startTime := c.CreatedAt
		if c.DueDate != nil {
			startTime = *c.DueDate
		}
		endTime := startTime.Add(1 * time.Hour)

		sb.WriteString("BEGIN:VEVENT\r\n")
		sb.WriteString(fmt.Sprintf("UID:%s@domus.local\r\n", c.ID))
		sb.WriteString(fmt.Sprintf("DTSTAMP:%s\r\n", nowStr))
		sb.WriteString(fmt.Sprintf("DTSTART:%s\r\n", startTime.UTC().Format("20060102T150405Z")))
		sb.WriteString(fmt.Sprintf("DTEND:%s\r\n", endTime.UTC().Format("20060102T150405Z")))
		sb.WriteString(fmt.Sprintf("SUMMARY:%s\r\n", c.Title))
		if c.Description != "" {
			sb.WriteString(fmt.Sprintf("DESCRIPTION:%s\r\n", c.Description))
		}
		if c.AssignedUserID != "" {
			sb.WriteString(fmt.Sprintf("ATTENDEE;CN=%s:mailto:%s@domus.local\r\n", c.AssignedUserID, c.AssignedUserID))
		}
		if c.Status == domainchore.StatusCompleted {
			sb.WriteString("STATUS:COMPLETED\r\n")
		} else {
			sb.WriteString("STATUS:CONFIRMED\r\n")
		}
		switch c.RecurrenceType {
		case domainchore.RecurrenceDaily:
			sb.WriteString("RRULE:FREQ=DAILY\r\n")
		case domainchore.RecurrenceWeekly:
			sb.WriteString("RRULE:FREQ=WEEKLY\r\n")
		case domainchore.RecurrenceMonthly:
			sb.WriteString("RRULE:FREQ=MONTHLY\r\n")
		}
		sb.WriteString("END:VEVENT\r\n")
	}

	sb.WriteString("END:VCALENDAR\r\n")

	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Content-Disposition", "inline; filename=\"domus-chores.ics\"")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(sb.String()))
}

// SyncGoogleCalendar imports events from a Google Calendar iCal link or raw payload into Domus chores.
func (h *ChoreHandler) SyncGoogleCalendar(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL        string `json:"url"`
		RawContent string `json:"raw_content,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON payload")
		return
	}

	content := req.RawContent
	if req.URL != "" {
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Get(req.URL)
		if err != nil {
			writeError(w, http.StatusBadRequest, "fetch_failed", "unable to fetch Google Calendar iCal feed: "+err.Error())
			return
		}
		defer resp.Body.Close()
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			writeError(w, http.StatusBadRequest, "read_failed", "unable to read iCal feed")
			return
		}
		content = string(bodyBytes)
	}

	if content == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "empty calendar content")
		return
	}

	eventsList, err := calendar.ParseICal(content)
	if err != nil {
		writeError(w, http.StatusBadRequest, "parse_error", "failed to parse calendar iCal format: "+err.Error())
		return
	}

	imported := 0
	for _, e := range eventsList {
		if strings.TrimSpace(e.Summary) == "" {
			continue
		}
		due := e.StartTime
		choreObj := &domainchore.Chore{
			ID:             "chore_gcal_" + uuid.NewString()[:8],
			Title:          e.Summary,
			Description:    e.Description,
			Priority:       domainchore.PriorityMedium,
			DueDate:        &due,
			RecurrenceType: domainchore.RecurrenceNone,
			Status:         domainchore.StatusPending,
		}
		if err := h.service.CreateTask(r.Context(), choreObj); err == nil {
			imported++
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"synced":  imported,
		"message": fmt.Sprintf("Sincronizzate con successo %d faccende da Google Calendar", imported),
	})
}

