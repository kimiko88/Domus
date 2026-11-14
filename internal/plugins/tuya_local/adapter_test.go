package tuya_local

import (
	"context"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
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
