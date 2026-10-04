package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/interfaces/http/handlers"
	"github.com/domus-hub/domus/internal/supervisor"
)

type mockSupervisorClient struct {
	diag      *supervisor.SystemDiagnostics
	export    *supervisor.DiagnosticExport
	mStatus   *supervisor.MaintenanceStatus
	enableErr error
}

func (m *mockSupervisorClient) GetDiagnostics(ctx context.Context) (*supervisor.SystemDiagnostics, error) {
	return m.diag, nil
}

func (m *mockSupervisorClient) ExportDiagnostics(ctx context.Context) (*supervisor.DiagnosticExport, error) {
	return m.export, nil
}

func (m *mockSupervisorClient) GetMaintenanceStatus(ctx context.Context) (*supervisor.MaintenanceStatus, error) {
	return m.mStatus, nil
}

func (m *mockSupervisorClient) EnableMaintenance(ctx context.Context, req supervisor.EnableMaintenanceRequest) (*supervisor.MaintenanceStatus, error) {
	if m.enableErr != nil {
		return nil, m.enableErr
	}
	m.mStatus.Active = true
	m.mStatus.RemainingSeconds = int64(req.DurationMinutes * 60)
	return m.mStatus, nil
}

func (m *mockSupervisorClient) DisableMaintenance(ctx context.Context) (*supervisor.MaintenanceStatus, error) {
	m.mStatus.Active = false
	m.mStatus.RemainingSeconds = 0
	return m.mStatus, nil
}

func TestSystemOSHandler(t *testing.T) {
	mockClient := &mockSupervisorClient{
		diag: &supervisor.SystemDiagnostics{
			Hostname:      "domus",
			OSVersion:     "Domus OS 1.0.0",
			Architecture:  "arm64",
			SoCTempC:      45.2,
			ActiveSlot:    "domus-root-a",
			SlotConfirmed: true,
			Timestamp:     time.Now().UTC(),
		},
		export: &supervisor.DiagnosticExport{
			ExportedAt: time.Now().UTC(),
		},
		mStatus: &supervisor.MaintenanceStatus{
			Active:           false,
			RemainingSeconds: 0,
			SSHPort:          22,
		},
	}

	h := handlers.NewSystemOSHandler(mockClient)

	// 1. Diagnostics
	t.Run("GetDiagnostics", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/system/diagnostics", nil)
		rec := httptest.NewRecorder()
		h.GetDiagnostics(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}
		var diag supervisor.SystemDiagnostics
		_ = json.Unmarshal(rec.Body.Bytes(), &diag)
		if diag.SoCTempC != 45.2 {
			t.Errorf("expected temp 45.2, got %v", diag.SoCTempC)
		}
	})

	// 2. Export
	t.Run("ExportDiagnostics", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/system/diagnostics/export", nil)
		rec := httptest.NewRecorder()
		h.ExportDiagnostics(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}
	})

	// 3. Maintenance Status
	t.Run("GetMaintenanceStatus", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/system/maintenance/status", nil)
		rec := httptest.NewRecorder()
		h.GetMaintenanceStatus(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}
	})

	// 4. Enable Maintenance
	t.Run("EnableMaintenance", func(t *testing.T) {
		body := `{"duration_minutes": 25, "authorized_key": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5..."}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/system/maintenance/enable", strings.NewReader(body))
		rec := httptest.NewRecorder()
		h.EnableMaintenance(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}
		var status supervisor.MaintenanceStatus
		_ = json.Unmarshal(rec.Body.Bytes(), &status)
		if !status.Active || status.RemainingSeconds != 25*60 {
			t.Errorf("expected active with 1500s remaining, got active=%v, rem=%v", status.Active, status.RemainingSeconds)
		}
	})

	// 5. Disable Maintenance
	t.Run("DisableMaintenance", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/system/maintenance/disable", nil)
		rec := httptest.NewRecorder()
		h.DisableMaintenance(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}
		var status supervisor.MaintenanceStatus
		_ = json.Unmarshal(rec.Body.Bytes(), &status)
		if status.Active {
			t.Errorf("expected inactive status, got active=%v", status.Active)
		}
	})
}
