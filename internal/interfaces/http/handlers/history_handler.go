// Package handlers provides HTTP route handlers for state history and metrics.
package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	apphist "github.com/domus-hub/domus/internal/application/history"
	"github.com/domus-hub/domus/internal/domain/entity"
	"github.com/domus-hub/domus/internal/domain/history"
	"github.com/go-chi/chi/v5"
)

// HistoryHandler handles HTTP requests for historical entity states and aggregated statistics.
type HistoryHandler struct {
	historyRepo history.Repository
	statsRepo   history.StatisticsRepository
	rollupSvc   *apphist.RollupService
	entityRepo  entity.Repository
}

// NewHistoryHandler constructs a HistoryHandler.
func NewHistoryHandler(historyRepo history.Repository, entityRepo entity.Repository) *HistoryHandler {
	return &HistoryHandler{
		historyRepo: historyRepo,
		entityRepo:  entityRepo,
	}
}

// WithStatistics attaches long-term statistics repository and rollup service.
func (h *HistoryHandler) WithStatistics(statsRepo history.StatisticsRepository, rollupSvc *apphist.RollupService) *HistoryHandler {
	h.statsRepo = statsRepo
	h.rollupSvc = rollupSvc
	return h
}

// GetEntityHistory returns historical state records for a given entity ID or key.
func (h *HistoryHandler) GetEntityHistory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "entity id or key is required")
		return
	}

	// 1. Resolve entity either by UUID ID or by entity_key
	ent, err := h.entityRepo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, entity.ErrEntityNotFound) {
			ent, err = h.entityRepo.GetByKey(r.Context(), id)
		}
		if err != nil {
			if errors.Is(err, entity.ErrEntityNotFound) {
				writeError(w, http.StatusNotFound, "not_found", "entity not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to lookup entity")
			return
		}
	}

	// 2. Parse query parameters
	filter := history.HistoryFilter{
		EntityID: ent.ID,
		Limit:    100,
	}

	q := r.URL.Query()
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
	if limitStr := q.Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			filter.Limit = l
		}
	}

	// 3. Query repository
	records, err := h.historyRepo.ListByEntity(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to fetch state history")
		return
	}

	if records == nil {
		records = []*history.StateHistory{}
	}

	writeJSON(w, http.StatusOK, records)
}

// GetStatistics handles GET /api/v1/history/statistics.
func (h *HistoryHandler) GetStatistics(w http.ResponseWriter, r *http.Request) {
	if h.statsRepo == nil {
		writeError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "statistics repository not configured")
		return
	}

	q := r.URL.Query()
	var entityIDs []string
	if rawIDs := q.Get("entity_ids"); rawIDs != "" {
		entityIDs = strings.Split(rawIDs, ",")
	}

	period := q.Get("period")
	if period == "" {
		period = "hourly"
	}

	filter := history.StatisticsFilter{
		EntityIDs: entityIDs,
		Period:    period,
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

	stats, err := h.statsRepo.GetStatistics(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	if stats == nil {
		stats = []*history.StatisticsRecord{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"statistics": stats,
		"total":      len(stats),
		"period":     period,
	})
}

// TriggerRollup handles POST /api/v1/history/statistics/rollup.
func (h *HistoryHandler) TriggerRollup(w http.ResponseWriter, r *http.Request) {
	if h.rollupSvc == nil {
		writeError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "rollup service not configured")
		return
	}

	lookback := 24
	if lStr := r.URL.Query().Get("hours"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			lookback = l
		}
	}

	recordsCount, err := h.rollupSvc.ExecuteHourlyRollup(r.Context(), lookback)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "ROLLUP_FAILED", err.Error())
		return
	}

	purgedRows, _ := h.rollupSvc.PurgeOldRawHistory(r.Context(), 10)

	writeJSON(w, http.StatusOK, map[string]any{
		"status":          "success",
		"records_created": recordsCount,
		"raw_rows_purged": purgedRows,
	})
}
