package os_test

import (
	"bufio"
	"context"
	"encoding/json"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/supervisor"
)

func getRepoRoot(t *testing.T) string {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working dir: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			t.Fatal("could not find repository root containing go.mod")
		}
		wd = parent
	}
}

func TestAvahiDaemonConfig(t *testing.T) {
	root := getRepoRoot(t)
	confPath := filepath.Join(root, "os", "configs", "avahi-daemon.conf")

	data, err := os.ReadFile(confPath)
	if err != nil {
		t.Fatalf("avahi-daemon.conf missing: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "host-name=domus") {
		t.Errorf("avahi-daemon.conf must set host-name=domus")
	}
	if !strings.Contains(content, "domain-name=local") {
		t.Errorf("avahi-daemon.conf must set domain-name=local")
	}
	if !strings.Contains(content, "use-ipv4=yes") {
		t.Errorf("avahi-daemon.conf must enable IPv4")
	}
	if !strings.Contains(content, "enable-dbus=yes") {
		t.Errorf("avahi-daemon.conf must enable D-Bus")
	}
}

type ServiceGroupXML struct {
	XMLName  xml.Name     `xml:"service-group"`
	Name     string       `xml:"name"`
	Services []ServiceXML `xml:"service"`
}

type ServiceXML struct {
	Type       string   `xml:"type"`
	Port       int      `xml:"port"`
	TxtRecords []string `xml:"txt-record"`
}

func TestAvahiServiceDefinition(t *testing.T) {
	root := getRepoRoot(t)
	servicePath := filepath.Join(root, "os", "configs", "domus-http.service")

	data, err := os.ReadFile(servicePath)
	if err != nil {
		t.Fatalf("domus-http.service missing: %v", err)
	}

	var sg ServiceGroupXML
	if err := xml.Unmarshal(data, &sg); err != nil {
		t.Fatalf("domus-http.service XML unmarshal failed: %v", err)
	}

	if len(sg.Services) < 2 {
		t.Fatalf("expected at least 2 services (_http and _domus), found %d", len(sg.Services))
	}

	hasHTTP := false
	hasDomus := false
	for _, s := range sg.Services {
		if s.Type == "_http._tcp" && s.Port == 80 {
			hasHTTP = true
		}
		if s.Type == "_domus._tcp" && s.Port == 80 {
			hasDomus = true
			if len(s.TxtRecords) == 0 {
				t.Error("expected TXT records on _domus._tcp service")
			}
		}
	}

	if !hasHTTP {
		t.Error("missing _http._tcp service advertisement on port 80")
	}
	if !hasDomus {
		t.Error("missing _domus._tcp appliance discovery service on port 80")
	}
}

func TestBuildrootDefconfig_IncludesAvahiAndMDNS(t *testing.T) {
	root := getRepoRoot(t)
	defconfigPath := filepath.Join(root, "os", "configs", "domus_rpi4_defconfig")

	data, err := os.ReadFile(defconfigPath)
	if err != nil {
		t.Fatalf("defconfig missing: %v", err)
	}

	content := string(data)
	requiredSymbols := []string{
		"BR2_PACKAGE_AVAHI=y",
		"BR2_PACKAGE_AVAHI_DAEMON=y",
		"BR2_PACKAGE_NSS_MDNS=y",
		"BR2_PACKAGE_DBUS=y",
	}

	for _, sym := range requiredSymbols {
		if !strings.Contains(content, sym) {
			t.Errorf("domus_rpi4_defconfig missing required configuration: %s", sym)
		}
	}
}

func TestPostBuildHook_ProvisionsAvahiAndNSS(t *testing.T) {
	root := getRepoRoot(t)
	postBuildPath := filepath.Join(root, "os", "board", "raspberrypi4-64", "post-build.sh")

	file, err := os.Open(postBuildPath)
	if err != nil {
		t.Fatalf("post-build.sh missing: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	hasAvahiConf := false
	hasAvahiService := false
	hasNSSMDNS := false

	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "avahi-daemon.conf") {
			hasAvahiConf = true
		}
		if strings.Contains(line, "domus-http.service") {
			hasAvahiService = true
		}
		if strings.Contains(line, "mdns4_minimal") {
			hasNSSMDNS = true
		}
	}

	if !hasAvahiConf {
		t.Error("post-build.sh does not install avahi-daemon.conf")
	}
	if !hasAvahiService {
		t.Error("post-build.sh does not install domus-http.service")
	}
	if !hasNSSMDNS {
		t.Error("post-build.sh does not configure nsswitch.conf for mdns4_minimal")
	}
}

func TestEndToEnd_SupervisorDiagnosticsAndMaintenanceAPI(t *testing.T) {
	tempDir := t.TempDir()

	cfg := supervisor.DefaultConfig()
	cfg.DataDir = tempDir
	cfg.HealthCheckInterval = 5 * time.Millisecond
	cfg.MaxHealthRetries = 2

	// Setup mock engine
	engine := supervisor.NewEngine(cfg, &mockRuntime{}, &mockRauc{}, &mockHealthChecker{})
	ctx := context.Background()
	_ = engine.Start(ctx)

	// Keep this test hermetic: do not attempt to start a real host SSH daemon
	// on the CI runner.
	engine.GetMaintenanceManager().SetServiceManager(&mockMaintenanceSSH{})

	apiServer := supervisor.NewAPIServer(engine)
	handler := apiServer.Handler()

	// 1. GET /api/v1/system/diagnostics
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/diagnostics", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for diagnostics, got %d", rec.Code)
	}
	var diag supervisor.SystemDiagnostics
	if err := json.Unmarshal(rec.Body.Bytes(), &diag); err != nil {
		t.Fatalf("failed to decode diagnostics JSON: %v", err)
	}
	if diag.Architecture == "" {
		t.Error("empty architecture in diagnostics")
	}

	// 2. GET /api/v1/system/diagnostics/export
	recExport := httptest.NewRecorder()
	reqExport := httptest.NewRequest(http.MethodGet, "/api/v1/system/diagnostics/export", nil)
	handler.ServeHTTP(recExport, reqExport)

	if recExport.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for export, got %d", recExport.Code)
	}
	var export supervisor.DiagnosticExport
	if err := json.Unmarshal(recExport.Body.Bytes(), &export); err != nil {
		t.Fatalf("failed to decode export JSON: %v", err)
	}
	if export.ExportedAt.IsZero() {
		t.Error("export missing timestamp")
	}

	// 3. Maintenance lifecycle
	recMaint := httptest.NewRecorder()
	reqMaint := httptest.NewRequest(http.MethodGet, "/api/v1/system/maintenance/status", nil)
	handler.ServeHTTP(recMaint, reqMaint)

	var mStatus supervisor.MaintenanceStatus
	_ = json.Unmarshal(recMaint.Body.Bytes(), &mStatus)
	if mStatus.Active {
		t.Error("maintenance mode must be inactive by default")
	}

	// Enable maintenance
	body := `{"duration_minutes": 20, "authorized_key": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIH... test@domus"}`
	recEnable := httptest.NewRecorder()
	reqEnable := httptest.NewRequest(http.MethodPost, "/api/v1/system/maintenance/enable", strings.NewReader(body))
	handler.ServeHTTP(recEnable, reqEnable)

	if recEnable.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for enable maintenance, got %d: %s", recEnable.Code, recEnable.Body.String())
	}
	var mActive supervisor.MaintenanceStatus
	_ = json.Unmarshal(recEnable.Body.Bytes(), &mActive)
	if !mActive.Active {
		t.Error("maintenance mode should be active")
	}
	if mActive.RemainingSeconds <= 0 {
		t.Error("expected remaining seconds > 0")
	}

	// Disable maintenance
	recDisable := httptest.NewRecorder()
	reqDisable := httptest.NewRequest(http.MethodPost, "/api/v1/system/maintenance/disable", nil)
	handler.ServeHTTP(recDisable, reqDisable)

	if recDisable.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for disable maintenance, got %d", recDisable.Code)
	}
	var mDisabled supervisor.MaintenanceStatus
	_ = json.Unmarshal(recDisable.Body.Bytes(), &mDisabled)
	if mDisabled.Active {
		t.Error("maintenance mode should be inactive after disable")
	}
}

// Mocks for test
type mockRuntime struct{}

func (m *mockRuntime) EnsureContainer(ctx context.Context, name string, cfg supervisor.ContainerConfig) error {
	return nil
}
func (m *mockRuntime) StartContainer(ctx context.Context, name string) error {
	return nil
}
func (m *mockRuntime) StopContainer(ctx context.Context, name string, timeout time.Duration) error {
	return nil
}
func (m *mockRuntime) GetContainerStatus(ctx context.Context, name string) (supervisor.ContainerStatus, error) {
	return supervisor.ContainerStatus{Name: name, State: supervisor.ContainerStateRunning}, nil
}
func (m *mockRuntime) ListContainers(ctx context.Context) ([]supervisor.ContainerStatus, error) {
	return []supervisor.ContainerStatus{
		{Name: "domus-core", State: supervisor.ContainerStateRunning},
	}, nil
}

type mockRauc struct{}

func (m *mockRauc) MarkGood(ctx context.Context) error { return nil }
func (m *mockRauc) GetStatus(ctx context.Context) (supervisor.RaucStatus, error) {
	return supervisor.RaucStatus{
		ActiveSlot: "domus-root-a",
		BootedSlot: "domus-root-a",
		Status:     "good",
	}, nil
}
func (m *mockRauc) Rollback(ctx context.Context) error { return nil }

type mockHealthChecker struct{}

func (m *mockHealthChecker) Check(ctx context.Context, url string) (bool, error) {
	return true, nil
}


type mockMaintenanceSSH struct{}

func (*mockMaintenanceSSH) StartSSH(context.Context) error { return nil }
func (*mockMaintenanceSSH) StopSSH(context.Context) error { return nil }
func (*mockMaintenanceSSH) IsSSHRunning(context.Context) (bool, error) { return true, nil }
