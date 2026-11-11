package supervisor_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/supervisor"
)

// MockContainerRuntime implements supervisor.ContainerRuntime for deterministic tests.
type MockContainerRuntime struct {
	mu         sync.Mutex
	containers map[string]supervisor.ContainerStatus
	configs    map[string]supervisor.ContainerConfig
	failStart  bool
}

func newMockRuntime() *MockContainerRuntime {
	return &MockContainerRuntime{
		containers: make(map[string]supervisor.ContainerStatus),
		configs:    make(map[string]supervisor.ContainerConfig),
	}
}

func (m *MockContainerRuntime) EnsureContainer(ctx context.Context, name string, cfg supervisor.ContainerConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.configs[name] = cfg
	if _, exists := m.containers[name]; !exists {
		m.containers[name] = supervisor.ContainerStatus{
			Name:  name,
			Image: cfg.Image,
			State: supervisor.ContainerStateStopped,
		}
	}
	return nil
}

func (m *MockContainerRuntime) StartContainer(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failStart {
		return errors.New("failed to start container")
	}
	c, ok := m.containers[name]
	if !ok {
		return errors.New("container not found")
	}
	c.State = supervisor.ContainerStateRunning
	c.StartedAt = time.Now()
	m.containers[name] = c
	return nil
}

func (m *MockContainerRuntime) StopContainer(ctx context.Context, name string, timeout time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.containers[name]
	if !ok {
		return errors.New("container not found")
	}
	c.State = supervisor.ContainerStateStopped
	m.containers[name] = c
	return nil
}

func (m *MockContainerRuntime) GetContainerStatus(ctx context.Context, name string) (supervisor.ContainerStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.containers[name]
	if !ok {
		return supervisor.ContainerStatus{Name: name, State: supervisor.ContainerStateUnknown}, errors.New("not found")
	}
	return c, nil
}

func (m *MockContainerRuntime) ListContainers(ctx context.Context) ([]supervisor.ContainerStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var list []supervisor.ContainerStatus
	for _, c := range m.containers {
		list = append(list, c)
	}
	return list, nil
}

// MockRaucController implements supervisor.RaucController for tests.
type MockRaucController struct {
	mu           sync.Mutex
	markGoodHits int
	rollbackHits int
	activeSlot   string
}

func newMockRauc() *MockRaucController {
	return &MockRaucController{
		activeSlot: "a",
	}
}

func (m *MockRaucController) MarkGood(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.markGoodHits++
	return nil
}

func (m *MockRaucController) GetStatus(ctx context.Context) (supervisor.RaucStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return supervisor.RaucStatus{
		Compatible: "domus-os-rpi4",
		ActiveSlot: m.activeSlot,
		BootedSlot: m.activeSlot,
		Status:     "ok",
	}, nil
}

func (m *MockRaucController) Rollback(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rollbackHits++
	return nil
}

// MockHealthChecker implements supervisor.HealthChecker for tests.
type MockHealthChecker struct {
	mu            sync.Mutex
	returnHealthy bool
	attempts      int
}

func (m *MockHealthChecker) Check(ctx context.Context, url string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.attempts++
	return m.returnHealthy, nil
}

func TestSupervisor_StartAndHealthcheckSuccess(t *testing.T) {
	runtime := newMockRuntime()
	rauc := newMockRauc()
	health := &MockHealthChecker{returnHealthy: true}

	cfg := supervisor.DefaultConfig()
	cfg.HealthCheckInterval = 10 * time.Millisecond
	cfg.MaxHealthRetries = 5

	engine := supervisor.NewEngine(cfg, runtime, rauc, health)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := engine.Start(ctx)
	if err != nil {
		t.Fatalf("engine.Start failed: %v", err)
	}

	// Verify Core container started
	coreStatus, err := runtime.GetContainerStatus(ctx, "domus-core")
	if err != nil {
		t.Fatalf("failed to get core status: %v", err)
	}
	if coreStatus.State != supervisor.ContainerStateRunning {
		t.Errorf("expected core to be running, got %s", coreStatus.State)
	}

	// Wait briefly for healthcheck routine
	time.Sleep(50 * time.Millisecond)

	status := engine.GetStatus()
	if !status.CoreHealthy {
		t.Errorf("expected core to be healthy")
	}
	if !status.SlotConfirmed {
		t.Errorf("expected slot to be confirmed")
	}

	rauc.mu.Lock()
	hits := rauc.markGoodHits
	rauc.mu.Unlock()
	if hits != 1 {
		t.Errorf("expected 1 MarkGood call, got %d", hits)
	}
}

func TestSupervisor_HealthcheckFailure_DoesNotConfirmSlot(t *testing.T) {
	runtime := newMockRuntime()
	rauc := newMockRauc()
	health := &MockHealthChecker{returnHealthy: false}

	cfg := supervisor.DefaultConfig()
	cfg.HealthCheckInterval = 5 * time.Millisecond
	cfg.MaxHealthRetries = 3

	engine := supervisor.NewEngine(cfg, runtime, rauc, health)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := engine.Start(ctx)
	if err != nil {
		t.Fatalf("engine.Start failed: %v", err)
	}

	// Wait for retries to exhaust
	time.Sleep(50 * time.Millisecond)

	status := engine.GetStatus()
	if status.CoreHealthy {
		t.Errorf("core should be unhealthy")
	}
	if status.SlotConfirmed {
		t.Errorf("slot must NOT be confirmed when healthcheck fails")
	}

	rauc.mu.Lock()
	hits := rauc.markGoodHits
	rauc.mu.Unlock()
	if hits != 0 {
		t.Errorf("expected 0 MarkGood calls on failure, got %d", hits)
	}
}

func TestSupervisor_CoreRestart(t *testing.T) {
	runtime := newMockRuntime()
	rauc := newMockRauc()
	health := &MockHealthChecker{returnHealthy: true}

	cfg := supervisor.DefaultConfig()
	engine := supervisor.NewEngine(cfg, runtime, rauc, health)

	ctx := context.Background()
	_ = engine.Start(ctx)

	err := engine.RestartCore(ctx)
	if err != nil {
		t.Fatalf("RestartCore failed: %v", err)
	}

	status, _ := runtime.GetContainerStatus(ctx, "domus-core")
	if status.State != supervisor.ContainerStateRunning {
		t.Errorf("expected core to be running after restart, got: %s", status.State)
	}
}

func TestSupervisor_API_Endpoints(t *testing.T) {
	runtime := newMockRuntime()
	rauc := newMockRauc()
	health := &MockHealthChecker{returnHealthy: true}

	cfg := supervisor.DefaultConfig()
	cfg.DataDir = t.TempDir()
	cfg.HealthCheckInterval = 5 * time.Millisecond
	cfg.MaxHealthRetries = 3
	engine := supervisor.NewEngine(cfg, runtime, rauc, health)
	ctx := context.Background()
	_ = engine.Start(ctx)

	time.Sleep(30 * time.Millisecond)

	apiHandler := supervisor.NewAPIServer(engine).Handler()

	// 1. Test /api/v1/supervisor/ping
	t.Run("GET /api/v1/supervisor/ping", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/supervisor/ping", nil)
		rec := httptest.NewRecorder()
		apiHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}
		var resp map[string]interface{}
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		if resp["status"] != "pong" {
			t.Errorf("expected pong, got %v", resp["status"])
		}
	})

	// 2. Test /api/v1/supervisor/status
	t.Run("GET /api/v1/supervisor/status", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/supervisor/status", nil)
		rec := httptest.NewRecorder()
		apiHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}
		var st supervisor.SupervisorStatus
		if err := json.Unmarshal(rec.Body.Bytes(), &st); err != nil {
			t.Fatalf("failed to decode status: %v", err)
		}
		if st.Version == "" {
			t.Errorf("empty version in status")
		}
	})

	// 3. Test /api/v1/supervisor/health
	t.Run("GET /api/v1/supervisor/health", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/supervisor/health", nil)
		rec := httptest.NewRecorder()
		apiHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}
	})

	// 4. Test /api/v1/supervisor/core/restart
	t.Run("POST /api/v1/supervisor/core/restart", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/supervisor/core/restart", nil)
		rec := httptest.NewRecorder()
		apiHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}
	})

	// 5. Test /api/v1/system/diagnostics
	t.Run("GET /api/v1/system/diagnostics", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/system/diagnostics", nil)
		rec := httptest.NewRecorder()
		apiHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}
		var diag supervisor.SystemDiagnostics
		if err := json.Unmarshal(rec.Body.Bytes(), &diag); err != nil {
			t.Fatalf("failed to decode diagnostics JSON: %v", err)
		}
		if diag.Architecture == "" {
			t.Error("expected non-empty architecture in diagnostics")
		}
	})

	// 6. Test /api/v1/system/diagnostics/export
	t.Run("GET /api/v1/system/diagnostics/export", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/system/diagnostics/export", nil)
		rec := httptest.NewRecorder()
		apiHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}
		var export supervisor.DiagnosticExport
		if err := json.Unmarshal(rec.Body.Bytes(), &export); err != nil {
			t.Fatalf("failed to decode diagnostics export JSON: %v", err)
		}
		if export.ExportedAt.IsZero() {
			t.Error("expected non-zero ExportedAt")
		}
	})

	// 7. Test /api/v1/system/maintenance/status
	t.Run("GET /api/v1/system/maintenance/status", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/system/maintenance/status", nil)
		rec := httptest.NewRecorder()
		apiHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}
		var status supervisor.MaintenanceStatus
		if err := json.Unmarshal(rec.Body.Bytes(), &status); err != nil {
			t.Fatalf("failed to decode maintenance status: %v", err)
		}
		if status.Active {
			t.Error("expected maintenance mode to be disabled by default")
		}
	})

	// 8. Test /api/v1/system/maintenance/enable & disable
	t.Run("POST /api/v1/system/maintenance/enable and disable", func(t *testing.T) {
		reqBody := `{"duration_minutes": 15, "authorized_key": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIG... user@host"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/system/maintenance/enable", strings.NewReader(reqBody))
		rec := httptest.NewRecorder()
		apiHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}
		var status supervisor.MaintenanceStatus
		if err := json.Unmarshal(rec.Body.Bytes(), &status); err != nil {
			t.Fatalf("failed to decode status: %v", err)
		}
		if !status.Active {
			t.Error("expected maintenance mode active after enable")
		}

		// Disable
		reqDis := httptest.NewRequest(http.MethodPost, "/api/v1/system/maintenance/disable", nil)
		recDis := httptest.NewRecorder()
		apiHandler.ServeHTTP(recDis, reqDis)

		if recDis.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", recDis.Code)
		}
		var disStatus supervisor.MaintenanceStatus
		_ = json.Unmarshal(recDis.Body.Bytes(), &disStatus)
		if disStatus.Active {
			t.Error("expected maintenance mode inactive after disable")
		}
	})
}
