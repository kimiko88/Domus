package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/domus-hub/domus/internal/supervisor"
)

// SupervisorClient abstracts communication between Domus Core and Domus Supervisor.
type SupervisorClient interface {
	GetDiagnostics(ctx context.Context) (*supervisor.SystemDiagnostics, error)
	ExportDiagnostics(ctx context.Context) (*supervisor.DiagnosticExport, error)
	GetMaintenanceStatus(ctx context.Context) (*supervisor.MaintenanceStatus, error)
	EnableMaintenance(ctx context.Context, req supervisor.EnableMaintenanceRequest) (*supervisor.MaintenanceStatus, error)
	DisableMaintenance(ctx context.Context) (*supervisor.MaintenanceStatus, error)
}

// SystemOSHandler serves OS-level appliance diagnostics and maintenance mode endpoints.
type SystemOSHandler struct {
	client SupervisorClient
}

// NewSystemOSHandler creates a new SystemOSHandler.
func NewSystemOSHandler(client SupervisorClient) *SystemOSHandler {
	return &SystemOSHandler{client: client}
}

// GetDiagnostics handles GET /api/v1/system/diagnostics
func (h *SystemOSHandler) GetDiagnostics(w http.ResponseWriter, r *http.Request) {
	if h.client == nil {
		writeError(w, http.StatusServiceUnavailable, "SUPERVISOR_UNAVAILABLE", "supervisor client not configured")
		return
	}
	diag, err := h.client.GetDiagnostics(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DIAGNOSTICS_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, diag)
}

// ExportDiagnostics handles GET /api/v1/system/diagnostics/export
func (h *SystemOSHandler) ExportDiagnostics(w http.ResponseWriter, r *http.Request) {
	if h.client == nil {
		writeError(w, http.StatusServiceUnavailable, "SUPERVISOR_UNAVAILABLE", "supervisor client not configured")
		return
	}
	export, err := h.client.ExportDiagnostics(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "EXPORT_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, export)
}

// GetMaintenanceStatus handles GET /api/v1/system/maintenance/status
func (h *SystemOSHandler) GetMaintenanceStatus(w http.ResponseWriter, r *http.Request) {
	if h.client == nil {
		writeError(w, http.StatusServiceUnavailable, "SUPERVISOR_UNAVAILABLE", "supervisor client not configured")
		return
	}
	status, err := h.client.GetMaintenanceStatus(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "MAINTENANCE_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, status)
}

// EnableMaintenance handles POST /api/v1/system/maintenance/enable
func (h *SystemOSHandler) EnableMaintenance(w http.ResponseWriter, r *http.Request) {
	if h.client == nil {
		writeError(w, http.StatusServiceUnavailable, "SUPERVISOR_UNAVAILABLE", "supervisor client not configured")
		return
	}
	var req supervisor.EnableMaintenanceRequest
	if r.Body != nil && r.ContentLength > 0 {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	status, err := h.client.EnableMaintenance(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "MAINTENANCE_ENABLE_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, status)
}

// DisableMaintenance handles POST /api/v1/system/maintenance/disable
func (h *SystemOSHandler) DisableMaintenance(w http.ResponseWriter, r *http.Request) {
	if h.client == nil {
		writeError(w, http.StatusServiceUnavailable, "SUPERVISOR_UNAVAILABLE", "supervisor client not configured")
		return
	}
	status, err := h.client.DisableMaintenance(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "MAINTENANCE_DISABLE_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, status)
}
