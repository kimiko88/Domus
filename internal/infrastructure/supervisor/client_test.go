package supervisor_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	infra_sup "github.com/domus-hub/domus/internal/infrastructure/supervisor"
	domussup "github.com/domus-hub/domus/internal/supervisor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type unixToTestServerTransport struct {
	serverURL *url.URL
	base      http.RoundTripper
}

func (t *unixToTestServerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = t.serverURL.Scheme
	req.URL.Host = t.serverURL.Host
	return t.base.RoundTrip(req)
}

func TestClient_SocketPresent_AllAPIs(t *testing.T) {
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
			_ = json.NewEncoder(w).Encode(domussup.DiagnosticExport{
				Diagnostics: *mockDiag,
				ExportedAt:  time.Now().UTC(),
			})
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
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		}
	}))
	defer server.Close()

	srvURL, _ := url.Parse(server.URL)
	httpClient := &http.Client{
		Transport: &unixToTestServerTransport{
			serverURL: srvURL,
			base:      server.Client().Transport,
		},
	}

	// Create dummy socket file so isSocketAvailable() returns true
	tmpDir := t.TempDir()
	dummySock := filepath.Join(tmpDir, "supervisor.sock")
	require.NoError(t, os.WriteFile(dummySock, []byte(""), 0600))

	client := infra_sup.NewClient(dummySock, tmpDir)
	client.SetHTTPClient(httpClient)

	ctx := context.Background()

	// 1. GetDiagnostics via API
	diag, err := client.GetDiagnostics(ctx)
	require.NoError(t, err)
	assert.Equal(t, "domus-hub", diag.Hostname)
	assert.Equal(t, 48.5, diag.SoCTempC)

	// 2. ExportDiagnostics via API
	export, err := client.ExportDiagnostics(ctx)
	require.NoError(t, err)
	assert.Equal(t, "domus-hub", export.Diagnostics.Hostname)

	// 3. GetMaintenanceStatus via API
	st, err := client.GetMaintenanceStatus(ctx)
	require.NoError(t, err)
	assert.False(t, st.Active)

	// 4. EnableMaintenance via API
	enabledSt, err := client.EnableMaintenance(ctx, domussup.EnableMaintenanceRequest{DurationMinutes: 30})
	require.NoError(t, err)
	assert.True(t, enabledSt.Active)
	assert.Equal(t, int64(1800), enabledSt.RemainingSeconds)

	// 5. DisableMaintenance via API
	disabledSt, err := client.DisableMaintenance(ctx)
	require.NoError(t, err)
	assert.False(t, disabledSt.Active)
}

func TestClient_SocketUnavailable_Fallbacks(t *testing.T) {
	client := infra_sup.NewClient("/non/existent/path/supervisor.sock", t.TempDir())
	ctx := context.Background()

	// 1. Diagnostics uses fallback collector
	diag, err := client.GetDiagnostics(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, diag.Hostname)

	// 2. Export uses fallback collector
	export, err := client.ExportDiagnostics(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, export.Diagnostics.Hostname)

	// 3. Maintenance status returns default inactive
	st, err := client.GetMaintenanceStatus(ctx)
	require.NoError(t, err)
	assert.False(t, st.Active)

	// 4. EnableMaintenance returns error when socket missing
	_, err = client.EnableMaintenance(ctx, domussup.EnableMaintenanceRequest{DurationMinutes: 5})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not available")

	// 5. DisableMaintenance returns default inactive
	disSt, err := client.DisableMaintenance(ctx)
	require.NoError(t, err)
	assert.False(t, disSt.Active)
}

func TestClient_ServerHTTPErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server crash", http.StatusInternalServerError)
	}))
	defer server.Close()

	srvURL, _ := url.Parse(server.URL)
	httpClient := &http.Client{
		Transport: &unixToTestServerTransport{
			serverURL: srvURL,
			base:      server.Client().Transport,
		},
	}

	tmpDir := t.TempDir()
	dummySock := filepath.Join(tmpDir, "supervisor.sock")
	require.NoError(t, os.WriteFile(dummySock, []byte(""), 0600))

	client := infra_sup.NewClient(dummySock, tmpDir)
	client.SetHTTPClient(httpClient)

	ctx := context.Background()

	// When API fails with 500, GetDiagnostics falls back to collector
	diag, err := client.GetDiagnostics(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, diag.Hostname)

	// EnableMaintenance fails with error message
	_, err = client.EnableMaintenance(ctx, domussup.EnableMaintenanceRequest{DurationMinutes: 5})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status 500")
}
