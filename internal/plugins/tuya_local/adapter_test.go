package tuya_local

import (
	"database/sql"
	"context"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
	_ "modernc.org/sqlite"
)

type mockBus struct {
	published []events.Event
}

func (m *mockBus) Publish(ctx context.Context, evt events.Event) error {
	m.published = append(m.published, evt)
	return nil
}

func (m *mockBus) Subscribe(eventType events.EventType, handler events.Handler) func() {
	return func() {}
}

func (m *mockBus) SubscribeAll(handler events.Handler) func() {
	return func() {}
}

func TestAdapter_Lifecycle(t *testing.T) {
	bus := &mockBus{}
	adapter := NewAdapter("tuya_test", bus)

	if adapter.ID() != "tuya_test" {
		t.Errorf("expected ID tuya_test, got %s", adapter.ID())
	}
	if adapter.Type() != "tuya_local" {
		t.Errorf("expected Type tuya_local, got %s", adapter.Type())
	}

	ctx := context.Background()
	if err := adapter.Start(ctx); err != nil {
		t.Fatalf("start failed: %v", err)
	}

	if err := adapter.HealthCheck(ctx); err != nil {
		t.Errorf("health check failed: %v", err)
	}

	adapter.RegisterDevice(DeviceConfig{
		DeviceID:  "dev_001",
		IP:        "127.0.0.1",
		LocalKey:  "0123456789abcdef",
		Category:  CategorySwitch,
		EntityKey: "switch.living_room_socket",
	})

	time.Sleep(50 * time.Millisecond)

	if err := adapter.Stop(ctx); err != nil {
		t.Fatalf("stop failed: %v", err)
	}

	if err := adapter.HealthCheck(ctx); err == nil {
		t.Errorf("expected health check failure after stop")
	}
}

func TestParseCategory(t *testing.T) {
	tests := []struct {
		input    string
		expected DeviceCategory
	}{
		{"kg", CategorySwitch},
		{"switch", CategorySwitch},
		{"cz", CategoryPlug},
		{"plug", CategoryPlug},
		{"socket", CategoryPlug},
		{"dj", CategoryLight},
		{"light", CategoryLight},
		{"dd", CategoryLight},
		{"wk", CategoryClimate},
		{"climate", CategoryClimate},
		{"unknown_xyz", CategorySwitch},
	}

	for _, tc := range tests {
		if got := ParseCategory(tc.input); got != tc.expected {
			t.Errorf("ParseCategory(%q) = %v, want %v", tc.input, got, tc.expected)
		}
	}
}

func TestAdapter_RegisterAndPersistDevice(t *testing.T) {
	bus := &mockBus{}
	adapter := NewAdapter("tuya_test", bus)

	// Discovery records the sender address, which is the actual LAN address.
	adapter.discovery.handleDiscovered(DiscoveredDevice{
		DeviceID: "dev_test_99",
		IP:       "192.168.1.77",
		Version:  "3.4",
	})

	ctx := context.Background()
	cfg := DeviceConfig{
		DeviceID: "dev_test_99",
		LocalKey: "localkey12345678",
		Category: CategoryLight,
	}

	err := adapter.RegisterAndPersistDevice(ctx, cfg, "Smart Ceiling Lamp")
	if err != nil {
		t.Fatalf("RegisterAndPersistDevice failed: %v", err)
	}

	if !adapter.IsDeviceConfigured("dev_test_99") {
		t.Errorf("device should be configured")
	}

	devices := adapter.GetConfiguredDevices()
	if len(devices) != 1 {
		t.Fatalf("expected 1 configured device, got %d", len(devices))
	}
	if devices[0].Name != "Smart Ceiling Lamp" || devices[0].Category != CategoryLight {
		t.Errorf("unexpected device in configs: %+v", devices[0])
	}
	if devices[0].IP != "192.168.1.77" || devices[0].Version != "3.4" {
		t.Fatalf("expected LAN discovery IP/version, got IP=%q version=%q", devices[0].IP, devices[0].Version)
	}
}

func TestAdapter_UnregisterDevice(t *testing.T) {
	adapter := NewAdapter("tuya_test", &mockBus{})
	adapter.RegisterDevice(DeviceConfig{
		DeviceID: "dev_remove_1",
		LocalKey: "0123456789abcdef",
		Category: CategoryClimate,
		EntityKey: "climate.tuya_dev_remove_1",
	})
	if !adapter.IsDeviceConfigured("dev_remove_1") {
		t.Fatal("device should be configured before removal")
	}
	if err := adapter.UnregisterDevice(context.Background(), "dev_remove_1"); err != nil {
		t.Fatalf("UnregisterDevice failed: %v", err)
	}
	if adapter.IsDeviceConfigured("dev_remove_1") {
		t.Fatal("device should no longer be configured")
	}
}

func TestTrustedLANIP(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"192.168.1.42", "192.168.1.42"},
		{"10.0.0.8", "10.0.0.8"},
		{"172.16.0.9", "172.16.0.9"},
		{"87.123.45.67", ""},
		{"8.8.8.8", ""},
		{"", ""},
		{"not-an-ip", ""},
	}
	for _, tt := range tests {
		if got := trustedLANIP(tt.input); got != tt.want {
			t.Errorf("trustedLANIP(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestAdapter_RejectsPublicIPOnRegistrationAndDiscovery(t *testing.T) {
	adapter := NewAdapter("tuya_test", &mockBus{})
	adapter.RegisterDevice(DeviceConfig{
		DeviceID: "dev_public_ip",
		IP: "87.123.45.67",
		LocalKey: "0123456789abcdef",
		Category: CategorySwitch,
	})
	if got := adapter.GetDeviceIP("dev_public_ip"); got != "" {
		t.Fatalf("public IP must not be registered as LAN address, got %q", got)
	}

	adapter.handleDiscoveredDevice(DiscoveredDevice{
		DeviceID: "dev_public_ip",
		IP: "87.123.45.67",
		Version: "3.5",
	})
	if got := adapter.GetDeviceIP("dev_public_ip"); got != "" {
		t.Fatalf("public IP must not be accepted from discovery, got %q", got)
	}
}

func TestAdapter_LoadStoredDevicesClearsUntrustedIPWithSingleConnection(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)

	_, err = db.Exec(`
		CREATE TABLE tuya_local_devices (
			device_id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			ip TEXT NOT NULL,
			local_key_enc TEXT NOT NULL,
			version TEXT NOT NULL,
			category TEXT NOT NULL,
			entity_key TEXT NOT NULL,
			updated_at INTEGER NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("create Tuya table: %v", err)
	}
	_, err = db.Exec(`
		INSERT INTO tuya_local_devices
			(device_id, name, ip, local_key_enc, version, category, entity_key, updated_at)
		VALUES
			('dev_legacy_wan', 'Legacy Tuya device', '87.123.45.67',
			 '0123456789abcdef', '3.3', 'switch', 'switch.tuya_legacy', 1)
	`)
	if err != nil {
		t.Fatalf("seed Tuya device: %v", err)
	}

	adapter := NewAdapter("tuya_test", &mockBus{}).WithPersistence(db, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := adapter.loadStoredDevices(ctx); err != nil {
		t.Fatalf("load stored devices: %v", err)
	}

	var storedIP string
	if err := db.QueryRowContext(context.Background(),
		"SELECT ip FROM tuya_local_devices WHERE device_id = ?", "dev_legacy_wan",
	).Scan(&storedIP); err != nil {
		t.Fatalf("read sanitized device IP: %v", err)
	}
	if storedIP != "" {
		t.Fatalf("expected persisted public IP to be cleared, got %q", storedIP)
	}

	if !adapter.IsDeviceConfigured("dev_legacy_wan") {
		t.Fatal("device configuration should be retained after IP cleanup")
	}
	deviceIP := adapter.GetDeviceIP("dev_legacy_wan")
	if deviceIP != "" {
		t.Fatalf("expected configured IP to be empty until LAN discovery, got %q", deviceIP)
	}

	devices := adapter.GetConfiguredDevices()
	if len(devices) != 1 || devices[0].LocalKey != "0123456789abcdef" {
		t.Fatalf("device credentials should be retained after IP cleanup, got %+v", devices)
	}
}


func TestAdapter_WaitForDiscoveredLANIPUsesFreshAnnouncement(t *testing.T) {
	adapter := NewAdapter("tuya_test", &mockBus{})
	adapter.RegisterDevice(DeviceConfig{
		DeviceID: "dev_lan_refresh",
		LocalKey: "0123456789abcdef",
		Version:  "3.3",
		Category: CategoryClimate,
	})

	// Simulate the UDP listener receiving a packet after LocalTuya sync.
	adapter.handleDiscoveredDevice(DiscoveredDevice{
		DeviceID: "dev_lan_refresh",
		IP:       "192.168.1.88",
		Version:  "3.5",
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	cfg, found := adapter.waitForDiscoveredLANIP(ctx, "dev_lan_refresh")
	if !found {
		t.Fatal("expected a fresh private LAN discovery address")
	}
	if cfg.IP != "192.168.1.88" {
		t.Fatalf("resolved IP = %q, want 192.168.1.88", cfg.IP)
	}
	if cfg.Version != "3.5" {
		t.Fatalf("resolved version = %q, want 3.5", cfg.Version)
	}
}
