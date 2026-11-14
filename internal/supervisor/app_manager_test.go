package supervisor_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/domus-hub/domus/internal/supervisor"
)

var validZigbeeManifest = []byte(`
id: zigbee2mqtt
name: Zigbee2MQTT
version: "1.36.0"
description: "Universal local Zigbee coordinator"
image: "ghcr.io/domus-hub/apps-zigbee2mqtt:1.36.0"
permissions:
  network:
    mode: "bridge"
    inbound_ports:
      - container_port: 8080
        host_port: 8099
  devices:
    - path: "/dev/ttyUSB0"
  storage:
    - path: "/data"
      volume_name: "z2m-data"
resources:
  memory_limit_mb: 256
  cpu_limit: 1.0
`)

func TestAppManager_InstallLifecycle(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "domus-app-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	runtime := newMockRuntime()
	mgr := supervisor.NewAppManager(tempDir, runtime)

	ctx := context.Background()

	// 1. Install App
	manifest, err := mgr.Install(ctx, validZigbeeManifest)
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}
	if manifest.ID != "zigbee2mqtt" {
		t.Errorf("expected id zigbee2mqtt, got %s", manifest.ID)
	}

	// Verify container config
	runtime.mu.Lock()
	cfg, exists := runtime.configs["domus-app-zigbee2mqtt"]
	runtime.mu.Unlock()
	if !exists {
		t.Fatalf("container domus-app-zigbee2mqtt was not registered")
	}
	if !cfg.ReadOnlyRoot {
		t.Errorf("expected ReadOnlyRoot to be true for sandboxing")
	}
	if cfg.MemoryLimitMB != 256 {
		t.Errorf("expected MemoryLimitMB 256, got %d", cfg.MemoryLimitMB)
	}

	// 2. Start App
	if err := mgr.Start(ctx, "zigbee2mqtt"); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	m, st, err := mgr.GetApp(ctx, "zigbee2mqtt")
	if err != nil || m == nil {
		t.Fatalf("GetApp failed: %v", err)
	}
	if st.State != supervisor.ContainerStateRunning {
		t.Errorf("expected running state, got: %s", st.State)
	}

	// 3. Stop App
	if err := mgr.Stop(ctx, "zigbee2mqtt"); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
	_, st2, _ := mgr.GetApp(ctx, "zigbee2mqtt")
	if st2.State != supervisor.ContainerStateStopped {
		t.Errorf("expected stopped state, got: %s", st2.State)
	}

	// 4. List Apps
	list, err := mgr.ListApps(ctx)
	if err != nil {
		t.Fatalf("ListApps failed: %v", err)
	}
	if len(list) != 1 || list[0].ID != "zigbee2mqtt" {
		t.Errorf("expected 1 app with id zigbee2mqtt, got: %+v", list)
	}

	// 5. Uninstall App
	if err := mgr.Uninstall(ctx, "zigbee2mqtt", true); err != nil {
		t.Fatalf("Uninstall failed: %v", err)
	}
	listAfter, _ := mgr.ListApps(ctx)
	if len(listAfter) != 0 {
		t.Errorf("expected 0 apps after uninstall, got %d", len(listAfter))
	}
}

func TestAppManager_SecuritySandboxing_Rejections(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "domus-sandbox-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	runtime := newMockRuntime()
	mgr := supervisor.NewAppManager(tempDir, runtime)
	ctx := context.Background()

	testCases := []struct {
		name     string
		manifest string
		errSub   string
	}{
		{
			name: "Forbidden disk access /dev/sda",
			manifest: `
id: evil-app
name: Evil App
version: "1.0.0"
image: "evil/app:latest"
permissions:
  network: { mode: "bridge" }
  devices: [{ path: "/dev/sda" }]
resources: { memory_limit_mb: 128 }
`,
			errSub: "forbidden device path",
		},
		{
			name: "Forbidden memory device /dev/mem",
			manifest: `
id: evil-app-mem
name: Evil Mem
version: "1.0.0"
image: "evil/app:latest"
permissions:
  network: { mode: "bridge" }
  devices: [{ path: "/dev/mem" }]
resources: { memory_limit_mb: 128 }
`,
			errSub: "forbidden device path",
		},
		{
			name: "Forbidden root mount path /",
			manifest: `
id: evil-mount
name: Evil Mount
version: "1.0.0"
image: "evil/app:latest"
permissions:
  network: { mode: "bridge" }
  storage: [{ path: "/", volume_name: "root" }]
resources: { memory_limit_mb: 128 }
`,
			errSub: "forbidden storage mount path",
		},
		{
			name: "Storage volume path traversal",
			manifest: `
id: evil-volume
name: Evil Volume
version: "1.0.0"
image: "evil/app:latest"
permissions:
  network: { mode: "bridge" }
  storage: [{ path: "/data", volume_name: "../../outside" }]
resources: { memory_limit_mb: 128 }
`,
			errSub: "invalid storage volume_name",
		},
		{
			name: "Excessive memory limit exceeding 3GB limit",
			manifest: `
id: greedy-app
name: Greedy App
version: "1.0.0"
image: "greedy/app:latest"
permissions: { network: { mode: "bridge" } }
resources: { memory_limit_mb: 8192 }
`,
			errSub: "memory_limit_mb (8192) must be between 32 and 3072 MB",
		},
		{
			name: "Invalid uppercase app ID",
			manifest: `
id: INVALID_ID
name: Bad ID
version: "1.0.0"
image: "test/app:latest"
permissions: { network: { mode: "bridge" } }
resources: { memory_limit_mb: 128 }
`,
			errSub: "invalid app ID",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := mgr.Install(ctx, []byte(tc.manifest))
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.errSub)
			}
			if !bytes.Contains([]byte(err.Error()), []byte(tc.errSub)) {
				t.Errorf("expected error %q to contain %q", err.Error(), tc.errSub)
			}
		})
	}
}

func TestAppManager_API_Endpoints(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "domus-app-api-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	runtime := newMockRuntime()
	rauc := newMockRauc()
	health := &MockHealthChecker{returnHealthy: true}

	cfg := supervisor.DefaultConfig()
	cfg.DataDir = tempDir
	engine := supervisor.NewEngine(cfg, runtime, rauc, health)
	apiHandler := supervisor.NewAPIServer(engine).Handler()

	// 1. Install app via POST /api/v1/apps/install
	t.Run("POST /api/v1/apps/install", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/apps/install", bytes.NewReader(validZigbeeManifest))
		rec := httptest.NewRecorder()
		apiHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d (body: %s)", rec.Code, rec.Body.String())
		}
		var m supervisor.AppManifest
		_ = json.Unmarshal(rec.Body.Bytes(), &m)
		if m.ID != "zigbee2mqtt" {
			t.Errorf("expected zigbee2mqtt, got %s", m.ID)
		}
	})

	// 2. List apps via GET /api/v1/apps
	t.Run("GET /api/v1/apps", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/apps", nil)
		rec := httptest.NewRecorder()
		apiHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}
		var list []supervisor.AppSummary
		_ = json.Unmarshal(rec.Body.Bytes(), &list)
		if len(list) != 1 {
			t.Fatalf("expected 1 app, got %d", len(list))
		}
	})

	// 3. Get app via GET /api/v1/apps/zigbee2mqtt
	t.Run("GET /api/v1/apps/zigbee2mqtt", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/apps/zigbee2mqtt", nil)
		rec := httptest.NewRecorder()
		apiHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}
	})

	// 4. Start app via POST /api/v1/apps/zigbee2mqtt/start
	t.Run("POST /api/v1/apps/zigbee2mqtt/start", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/apps/zigbee2mqtt/start", nil)
		rec := httptest.NewRecorder()
		apiHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}
	})

	// 5. Stop app via POST /api/v1/apps/zigbee2mqtt/stop
	t.Run("POST /api/v1/apps/zigbee2mqtt/stop", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/apps/zigbee2mqtt/stop", nil)
		rec := httptest.NewRecorder()
		apiHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}
	})

	// 6. Uninstall app via DELETE /api/v1/apps/zigbee2mqtt
	t.Run("DELETE /api/v1/apps/zigbee2mqtt", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/apps/zigbee2mqtt?purge=true", nil)
		rec := httptest.NewRecorder()
		apiHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}
	})
}
