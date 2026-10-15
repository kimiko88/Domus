// Package handlers provides HTTP route handlers for the human-readable logbook.
package handlers

import (
	"net/http"
	"strconv"
	"time"

	applog "github.com/domus-hub/domus/internal/application/logbook"
	domlog "github.com/domus-hub/domus/internal/domain/logbook"
)

// LogbookHandler handles HTTP queries for the timeline logbook.
type LogbookHandler struct {
	service *applog.Service
}

// NewLogbookHandler constructs a LogbookHandler.
func NewLogbookHandler(service *applog.Service) *LogbookHandler {
	return &LogbookHandler{service: service}
}

// Query handles GET /api/v1/logbook.
func (h *LogbookHandler) Query(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	filter := domlog.Filter{
		EntityID: q.Get("entity_id"),
		Domain:   q.Get("domain"),
		Limit:    100,
	}

	if lStr := q.Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			filter.Limit = l
		}
	}

	if fromStr := q.Get("from"); fromStr != "" {
		if t, err := time.Parse(time.RFC3339, fromStr); err == nil {
			filter.From = t
		}
	}

	if toStr := q.Get("to"); toStr != "" {
		if t, err := time.Parse(time.RFC3339, toStr); err == nil {
			filter.To = t
		}
	}

	entries, err := h.service.Query(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	if entries == nil {
		entries = []*domlog.Entry{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"entries": entries,
		"total":   len(entries),
	})
}
