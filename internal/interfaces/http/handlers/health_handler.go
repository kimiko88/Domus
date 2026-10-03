// Package handlers contains HTTP route handlers for the Domus hub.
package handlers

import (
	"encoding/json"
	"net/http"
	"time"
)

// Pinger defines an interface for checking connectivity to dependencies such as the database.
type Pinger interface {
	Ping() error
}

// HealthHandler serves health check requests for Domus.
type HealthHandler struct {
	version   string
	startedAt time.Time
	pinger    Pinger
}

// NewHealthHandler constructs a HealthHandler.
func NewHealthHandler(version string, startedAt time.Time, pinger Pinger) *HealthHandler {
	return &HealthHandler{
		version:   version,
		startedAt: startedAt,
		pinger:    pinger,
	}
}

// ServeHTTP implements http.Handler.
func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	uptimeSeconds := int64(time.Since(h.startedAt).Seconds())

	dbStatus := "healthy"
	overallStatus := "ok"
	statusCode := http.StatusOK

	if h.pinger != nil {
		if err := h.pinger.Ping(); err != nil {
			dbStatus = "unhealthy"
			overallStatus = "degraded"
		}
	}

	resp := map[string]any{
		"status":   overallStatus,
		"version":  h.version,
		"uptime_s": uptimeSeconds,
		"database": dbStatus,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(resp)
}
