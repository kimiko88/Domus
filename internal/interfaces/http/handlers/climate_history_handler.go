package handlers

import (
	"encoding/csv"
	"encoding/json"
	"errors"

	"github.com/domus-hub/domus/internal/domain/entity"
	domusmw "github.com/domus-hub/domus/internal/interfaces/http/middleware"
	"net/http"
	"strconv"
	"time"

	appclimatehistory "github.com/domus-hub/domus/internal/application/climate_history"
	domclimatehistory "github.com/domus-hub/domus/internal/domain/climate_history"
)

// ClimateHistoryHandler exposes climate telemetry, summaries, settings and CSV export.
type ClimateHistoryHandler struct {
	service *appclimatehistory.Service
	entities entity.Repository
}

func NewClimateHistoryHandler(service *appclimatehistory.Service, entities ...entity.Repository) *ClimateHistoryHandler {
	h := &ClimateHistoryHandler{service: service}
	if len(entities) > 0 { h.entities = entities[0] }
	return h
}

func (h *ClimateHistoryHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := h.service.GetSettings(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "CLIMATE_SETTINGS_FAILED", "failed to load climate history settings")
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

func (h *ClimateHistoryHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var settings domclimatehistory.Settings
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&settings); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid climate history settings payload")
		return
	}
	if err := h.service.UpdateSettings(r.Context(), settings); err != nil {
		if errors.Is(err, domclimatehistory.ErrInvalidSettings) {
			writeError(w, http.StatusBadRequest, "INVALID_CLIMATE_SETTINGS", "one or more climate history settings are outside the supported range")
			return
		}
		writeError(w, http.StatusInternalServerError, "CLIMATE_SETTINGS_SAVE_FAILED", "failed to save climate history settings")
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

func (h *ClimateHistoryHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	filter, ok := parseClimateFilter(w, r)
	if !ok || !h.applyAccessScope(w, r, &filter) {
		return
	}
	points, err := h.service.GetHistory(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "CLIMATE_HISTORY_FAILED", "failed to load climate history")
		return
	}
	if points == nil {
		points = []domclimatehistory.Point{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"points": points, "total": len(points)})
}

func (h *ClimateHistoryHandler) GetSummary(w http.ResponseWriter, r *http.Request) {
	filter, ok := parseClimateFilter(w, r)
	if !ok {
		return
	}
	if filter.From.IsZero() {
		filter.From = time.Now().UTC().Add(-24 * time.Hour)
	}
	if filter.To.IsZero() {
		filter.To = time.Now().UTC()
	}
	summary, err := h.service.Summarize(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "CLIMATE_SUMMARY_FAILED", "failed to calculate climate summary")
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (h *ClimateHistoryHandler) ExportCSV(w http.ResponseWriter, r *http.Request) {
	filter, ok := parseClimateFilter(w, r)
	if !ok {
		return
	}
	points, err := h.service.GetHistory(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "CLIMATE_EXPORT_FAILED", "failed to export climate history")
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=domus-climate-history.csv")
	writer := csv.NewWriter(w)
	_ = writer.Write([]string{"recorded_at", "area_id", "entity_id", "entity_name", "period", "temperature_avg_c", "temperature_min_c", "temperature_max_c", "humidity_avg_percent", "humidity_min_percent", "humidity_max_percent", "target_temperature_c", "hvac_mode", "hvac_action", "sample_count"})
	for _, p := range points {
		_ = writer.Write([]string{
			p.RecordedAt.UTC().Format(time.RFC3339), p.AreaID, p.EntityID, p.EntityName, p.Period,
			formatOptionalFloat(p.TemperatureAverage), formatOptionalFloat(p.TemperatureMinimum), formatOptionalFloat(p.TemperatureMaximum),
			formatOptionalFloat(p.HumidityAverage), formatOptionalFloat(p.HumidityMinimum), formatOptionalFloat(p.HumidityMaximum),
			formatOptionalFloat(p.TargetTemperatureAvg), p.HVACMode, p.HVACAction, strconv.Itoa(p.SampleCount),
		})
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		// Headers have already been sent; only the server can log the streaming error.
		return
	}
}

func (h *ClimateHistoryHandler) applyAccessScope(w http.ResponseWriter, r *http.Request, filter *domclimatehistory.Filter) bool {
	currentUser := domusmw.GetUser(r.Context())
	if currentUser == nil { return true }
	permissions := currentUser.EffectivePermissions()
	if permissions == nil || (permissions.AllowedAreaIDs == nil && permissions.AllowedEntityIDs == nil) { return true }
	if h.entities == nil {
		writeError(w, http.StatusServiceUnavailable, "CLIMATE_ACCESS_UNAVAILABLE", "cannot validate climate history access")
		return false
	}
	items, err := h.entities.List(r.Context(), entity.ListFilter{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "CLIMATE_ACCESS_FAILED", "failed to resolve permitted climate entities")
		return false
	}
	allowed := make([]string, 0, len(items))
	for _, item := range items {
		if item != nil && currentUser.CanAccessEntityInArea(item.ID, item.AreaID) &&
			(filter.AreaID == "" || filter.AreaID == item.AreaID) &&
			(filter.EntityID == "" || filter.EntityID == item.ID) {
			allowed = append(allowed, item.ID)
		}
	}
	filter.EntityIDs = allowed
	return true
}

func parseClimateFilter(w http.ResponseWriter, r *http.Request) (domclimatehistory.Filter, bool) {
	q := r.URL.Query()
	filter := domclimatehistory.Filter{EntityID: q.Get("entity_id"), AreaID: q.Get("area_id"), Limit: 2000}
	for key, target := range map[string]*time.Time{"from": &filter.From, "to": &filter.To} {
		if value := q.Get(key); value != "" {
			parsed, err := time.Parse(time.RFC3339, value)
			if err != nil {
				writeError(w, http.StatusBadRequest, "INVALID_TIME_RANGE", key+" must be an RFC3339 timestamp")
				return filter, false
			}
			*target = parsed.UTC()
		}
	}
	if !filter.From.IsZero() && !filter.To.IsZero() && filter.From.After(filter.To) {
		writeError(w, http.StatusBadRequest, "INVALID_TIME_RANGE", "from must not be later than to")
		return filter, false
	}
	if value := q.Get("limit"); value != "" {
		limit, err := strconv.Atoi(value)
		if err != nil || limit < 1 || limit > 10000 {
			writeError(w, http.StatusBadRequest, "INVALID_LIMIT", "limit must be between 1 and 10000")
			return filter, false
		}
		filter.Limit = limit
	}
	return filter, true
}

func formatOptionalFloat(value *float64) string {
	if value == nil {
		return ""
	}
	return strconv.FormatFloat(*value, 'f', 3, 64)
}

