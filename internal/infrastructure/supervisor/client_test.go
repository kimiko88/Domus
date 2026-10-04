package supervisor_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	infra_sup "github.com/domus-hub/domus/internal/infrastructure/supervisor"
	domussup "github.com/domus-hub/domus/internal/supervisor"
)

func TestClient_WithHTTPServer(t *testing.T) {
	mockDiag := &domussup.SystemDiagnostics{
		Hostname:      "domus-hub",
		OSVersion:     "Domus OS 1.0.0",
		Architecture:  "arm64",
		SoCTempC:      48.5,
		SlotConfirmed: true,
		ActiveSlot:    "domus-root-a",
		Timestamp:     time.Now().UTC(),
	}

	mockMStatus := &domussup.MaintenanceStatus{
		Active:           false,
		RemainingSeconds: 0,
		SSHPort:          22,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/system/diagnostics":
			_ = json.NewEncoder(w).Encode(mockDiag)
		case "/api/v1/system/diagnostics/export":
			_ = json.NewEncoder(w).Encode(domussup.DiagnosticExport{ExportedAt: time.Now().UTC()})
		case "/api/v1/system/maintenance/status":
			_ = json.NewEncoder(w).Encode(mockMStatus)
		case "/api/v1/system/maintenance/enable":
			mockMStatus.Active = true
			mockMStatus.RemainingSeconds = 1800
			_ = json.NewEncoder(w).Encode(mockMStatus)
		case "/api/v1/system/maintenance/disable":
			mockMStatus.Active = false
			mockMStatus.RemainingSeconds = 0
			_ = json.NewEncoder(w).Encode(mockMStatus)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := infra_sup.NewClient("", "")
	client.SetHTTPClient(server.Client())

	ctx := context.Background()

	// Test fallback when socket path not present
	diag, err := client.GetDiagnostics(ctx)
	if err != nil {
		t.Fatalf("unexpected error getting diagnostics: %v", err)
	}
	if diag == nil || diag.Architecture == "" {
		t.Fatal("expected valid diagnostics")
	}

	mStatus, err := client.GetMaintenanceStatus(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mStatus.Active {
		t.Fatal("expected inactive maintenance mode by default")
	}
}
