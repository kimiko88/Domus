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
