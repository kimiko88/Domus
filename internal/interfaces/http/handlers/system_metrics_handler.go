// Package handlers provides HTTP route handlers for state history and metrics.
package handlers

import (
	"net/http"
	"time"

	"github.com/domus-hub/domus/internal/domain/device"
	"github.com/domus-hub/domus/internal/domain/entity"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
)

// WSClientCounter exposes client connection count from the WebSocket hub.
type WSClientCounter interface {
	ClientCount() int
}

// SystemMetricsHandler reports hub operational and performance metrics.
type SystemMetricsHandler struct {
	startedAt     time.Time
	repos         *sqlite.Repositories
	wsCounter     WSClientCounter
	mqttConnected bool
}

// NewSystemMetricsHandler constructs a SystemMetricsHandler.
func NewSystemMetricsHandler(
	startedAt time.Time,
	repos *sqlite.Repositories,
	wsCounter WSClientCounter,
	mqttConnected bool,
) *SystemMetricsHandler {
	return &SystemMetricsHandler{
		startedAt:     startedAt,
		repos:         repos,
		wsCounter:     wsCounter,
		mqttConnected: mqttConnected,
	}
}

// ServeHTTP responds with a JSON snapshot of system metrics.
func (h *SystemMetricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var entityCount int
	if h.repos != nil && h.repos.Entities != nil {
		if list, err := h.repos.Entities.List(ctx, entity.ListFilter{}); err == nil {
			entityCount = len(list)
		}
	}

	var deviceCount int
	if h.repos != nil && h.repos.Devices != nil {
		if list, err := h.repos.Devices.List(ctx, device.ListFilter{}); err == nil {
			deviceCount = len(list)
		}
	}

	var wsClients int
	if h.wsCounter != nil {
		wsClients = h.wsCounter.ClientCount()
	}

	uptimeSeconds := int64(time.Since(h.startedAt).Seconds())
	if uptimeSeconds < 0 {
		uptimeSeconds = 0
	}

	resp := map[string]any{
		"status":         "ok",
		"uptime_s":       uptimeSeconds,
		"entity_count":   entityCount,
		"device_count":   deviceCount,
		"mqtt_connected": h.mqttConnected,
		"ws_clients":     wsClients,
		"db_size_bytes":  0,
	}

	writeJSON(w, http.StatusOK, resp)
}
