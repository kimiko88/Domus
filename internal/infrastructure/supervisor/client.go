package supervisor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	domussup "github.com/domus-hub/domus/internal/supervisor"
)

// Client talks to the Domus Supervisor REST API via Unix socket or fallback.
type Client struct {
	socketPath string
	httpClient *http.Client
	fallback   *domussup.DiagnosticsCollector
}

// NewClient creates a new Client configured to use the specified Unix socket path.
func NewClient(socketPath string, dataDir string) *Client {
	if socketPath == "" {
		socketPath = "/run/domus/supervisor.sock"
	}

	httpClient := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, "unix", socketPath)
			},
		},
	}

	cfg := domussup.DefaultConfig()
	if dataDir != "" {
		cfg.DataDir = dataDir
	}
	fallback := domussup.NewDiagnosticsCollector(cfg, nil, nil, nil)

	return &Client{
		socketPath: socketPath,
		httpClient: httpClient,
		fallback:   fallback,
	}
}

// SetHTTPClient overrides the internal http.Client (useful for testing with httptest).
func (c *Client) SetHTTPClient(client *http.Client) {
	c.httpClient = client
}

// GetDiagnostics retrieves system telemetry either from the supervisor socket or fallback.
func (c *Client) GetDiagnostics(ctx context.Context) (*domussup.SystemDiagnostics, error) {
	// If socket exists, call API
	if c.isSocketAvailable() {
		var diag domussup.SystemDiagnostics
		if err := c.getJSON(ctx, "/api/v1/system/diagnostics", &diag); err == nil {
			return &diag, nil
		}
	}

	// Fallback to local collector
	return c.fallback.Collect(ctx)
}

// ExportDiagnostics retrieves sanitized debug bundle.
func (c *Client) ExportDiagnostics(ctx context.Context) (*domussup.DiagnosticExport, error) {
	if c.isSocketAvailable() {
		var export domussup.DiagnosticExport
		if err := c.getJSON(ctx, "/api/v1/system/diagnostics/export", &export); err == nil {
			return &export, nil
		}
	}

	return c.fallback.Export(ctx)
}

// GetMaintenanceStatus checks current on-demand SSH maintenance mode state.
func (c *Client) GetMaintenanceStatus(ctx context.Context) (*domussup.MaintenanceStatus, error) {
	if c.isSocketAvailable() {
		var status domussup.MaintenanceStatus
		if err := c.getJSON(ctx, "/api/v1/system/maintenance/status", &status); err == nil {
			return &status, nil
		}
	}

	return &domussup.MaintenanceStatus{
		Active:           false,
		RemainingSeconds: 0,
		SSHPort:          22,
	}, nil
}

// EnableMaintenance requests activation of maintenance mode.
func (c *Client) EnableMaintenance(ctx context.Context, req domussup.EnableMaintenanceRequest) (*domussup.MaintenanceStatus, error) {
	if !c.isSocketAvailable() {
		return nil, fmt.Errorf("supervisor socket %s is not available", c.socketPath)
	}

	var status domussup.MaintenanceStatus
	if err := c.postJSON(ctx, "/api/v1/system/maintenance/enable", req, &status); err != nil {
		return nil, err
	}
	return &status, nil
}

// DisableMaintenance immediately terminates maintenance mode.
func (c *Client) DisableMaintenance(ctx context.Context) (*domussup.MaintenanceStatus, error) {
	if !c.isSocketAvailable() {
		return &domussup.MaintenanceStatus{Active: false, RemainingSeconds: 0, SSHPort: 22}, nil
	}

	var status domussup.MaintenanceStatus
	if err := c.postJSON(ctx, "/api/v1/system/maintenance/disable", nil, &status); err != nil {
		return nil, err
	}
	return &status, nil
}

func (c *Client) isSocketAvailable() bool {
	if c.socketPath == "" {
		return false
	}
	info, err := os.Stat(c.socketPath)
	if err != nil {
		return false
	}
	// On Unix, socket mode is os.ModeSocket
	return (info.Mode() & os.ModeSocket) != 0 || !info.IsDir()
}

func (c *Client) getJSON(ctx context.Context, path string, target interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://unix"+path, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("supervisor returned status %d: %s", resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(target)
}

func (c *Client) postJSON(ctx context.Context, path string, payload interface{}, target interface{}) error {
	var bodyReader io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://unix"+path, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("supervisor returned status %d: %s", resp.StatusCode, string(body))
	}

	if target != nil {
		return json.NewDecoder(resp.Body).Decode(target)
	}
	return nil
}
