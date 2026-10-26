// Package handlers provides HTTP route handlers for household anomaly detection.
package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/domus-hub/domus/internal/application/anomaly"
	"github.com/go-chi/chi/v5"
)

// AnomalyHandler provides HTTP endpoints for household anomaly detection,
// water micro-leak monitoring, appliance predictive health, and condensation/mold analysis.
type AnomalyHandler struct {
	svc *anomaly.Service
}

// NewAnomalyHandler constructs a new AnomalyHandler.
func NewAnomalyHandler(svc *anomaly.Service) *AnomalyHandler {
	return &AnomalyHandler{svc: svc}
}

// RegisterRoutes registers all anomaly endpoints on the provided router.
func (h *AnomalyHandler) RegisterRoutes(r chi.Router) {
	r.Get("/", h.ListAlerts)
	r.Post("/{id}/acknowledge", h.AcknowledgeAlert)
	r.Get("/water", h.GetWaterStatus)
	r.Post("/water/ingest", h.IngestWaterFlow)
	r.Get("/mold", h.AssessMold)
	r.Get("/appliances", h.ListAppliances)
}

// ListAlerts handles GET /api/v1/anomalies.
func (h *AnomalyHandler) ListAlerts(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "anomaly service not configured")
		return
	}

	var ackPtr *bool
	ackQuery := r.URL.Query().Get("acknowledged")
	if ackQuery != "" {
		ackVal := ackQuery == "true" || ackQuery == "1"
		ackPtr = &ackVal
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	alerts, err := h.svc.ListAlerts(r.Context(), ackPtr, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, alerts)
}

// AcknowledgeAlert handles POST /api/v1/anomalies/{id}/acknowledge.
func (h *AnomalyHandler) AcknowledgeAlert(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "anomaly service not configured")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "alert ID is required")
		return
	}

	if err := h.svc.AcknowledgeAlert(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":           id,
		"acknowledged": true,
	})
}

// GetWaterStatus handles GET /api/v1/anomalies/water.
func (h *AnomalyHandler) GetWaterStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":    "active",
		"timestamp": time.Now().UTC(),
	})
}

// IngestWaterFlow handles POST /api/v1/anomalies/water/ingest.
func (h *AnomalyHandler) IngestWaterFlow(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "anomaly service not configured")
		return
	}

	var req struct {
		MeterID   string  `json:"meter_id"`
		FlowLMin  float64 `json:"flow_l_min"`
		Timestamp string  `json:"timestamp,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.MeterID == "" {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "meter_id and flow_l_min are required")
		return
	}

	now := time.Now().UTC()
	if req.Timestamp != "" {
		if t, err := time.Parse(time.RFC3339, req.Timestamp); err == nil {
			now = t
		}
	}

	alert := h.svc.CheckWaterFlow(req.MeterID, req.FlowLMin, now)
	resp := map[string]any{
		"meter_id":   req.MeterID,
		"flow_l_min": req.FlowLMin,
		"alert":      alert,
	}
	if alert != nil {
		writeJSON(w, http.StatusCreated, resp)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// AssessMold handles GET /api/v1/anomalies/mold.
func (h *AnomalyHandler) AssessMold(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "anomaly service not configured")
		return
	}

	ambient, err1 := strconv.ParseFloat(r.URL.Query().Get("ambient_c"), 64)
	rh, err2 := strconv.ParseFloat(r.URL.Query().Get("rh_pct"), 64)
	surface, err3 := strconv.ParseFloat(r.URL.Query().Get("surface_c"), 64)

	if err1 != nil || err2 != nil || err3 != nil {
		writeError(w, http.StatusBadRequest, "INVALID_PARAMS", "ambient_c, rh_pct, and surface_c query parameters are required")
		return
	}

	res := h.svc.AssessSurfaceMold(ambient, rh, surface, time.Now().UTC())
	writeJSON(w, http.StatusOK, res)
}

// ListAppliances handles GET /api/v1/anomalies/appliances.
func (h *AnomalyHandler) ListAppliances(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "anomaly service not configured")
		return
	}

	appliances, err := h.svc.ListAppliances(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, appliances)
}
