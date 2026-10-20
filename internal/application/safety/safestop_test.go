package safety_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/application/safety"
	"github.com/domus-hub/domus/internal/domain/events"
)

type mockActuator struct {
	mu           sync.Mutex
	delay        time.Duration
	failValves   map[string]bool
	closedValves map[string]bool
	openedValves map[string]bool
	closeCount   int64
}

func newMockActuator(delay time.Duration) *mockActuator {
	return &mockActuator{
		delay:        delay,
		failValves:   make(map[string]bool),
		closedValves: make(map[string]bool),
		openedValves: make(map[string]bool),
	}
}

func (m *mockActuator) CloseValve(ctx context.Context, valveID string) error {
	if m.delay > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(m.delay):
		}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	atomic.AddInt64(&m.closeCount, 1)

	if m.failValves[valveID] {
		return errors.New("hardware bus communication timeout")
	}
	m.closedValves[valveID] = true
	return nil
}

func (m *mockActuator) OpenValve(ctx context.Context, valveID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.openedValves[valveID] = true
	return nil
}

type mockBus struct {
	events []events.Event
	mu     sync.Mutex
}

func (b *mockBus) Publish(ctx context.Context, e events.Event) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = append(b.events, e)
	return nil
}

func (b *mockBus) Subscribe(eventType events.EventType, handler events.Handler) func() {
	return func() {}
}

func (b *mockBus) SubscribeAll(handler events.Handler) func() {
	return func() {}
}

func (b *mockBus) Close() error {
	return nil
}

func TestSafeStop_Sub50MsLatencyAndSelectiveShutoff(t *testing.T) {
	bus := &mockBus{}
	svc := safety.NewService(bus)
	defer svc.Stop()

	// Hardware actuators simulating real relays responding in 8ms
	actuator := newMockActuator(8 * time.Millisecond)

	// Register 2 water valves and 1 gas valve
	_ = svc.RegisterValve(safety.ValveRegistration{
		ID:        "valve.water_main",
		Type:      safety.ValveWater,
		Name:      "Main Water Supply",
		Location:  "Basement",
		Actuator:  actuator,
		TimeoutMs: 35,
	})
	_ = svc.RegisterValve(safety.ValveRegistration{
		ID:        "valve.water_secondary",
		Type:      safety.ValveWater,
		Name:      "Garden Line",
		Location:  "Outdoors",
		Actuator:  actuator,
		TimeoutMs: 35,
	})
	_ = svc.RegisterValve(safety.ValveRegistration{
		ID:        "valve.gas_main",
		Type:      safety.ValveGas,
		Name:      "Main Methane Meter",
		Location:  "Utility Closet",
		Actuator:  actuator,
		TimeoutMs: 35,
	})

	// 1. Water leak detected
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	alert := safety.EmergencyAlert{
		ID:           "alert-water-01",
		Hazard:       safety.HazardWaterLeak,
		SourceSensor: "sensor.kitchen_flood",
		Location:     "Kitchen",
		Severity:     "critical",
	}

	result, err := svc.TriggerSafeStop(ctx, alert)
	if err != nil {
		t.Fatalf("unexpected TriggerSafeStop error: %v", err)
	}

	// Verify ultra-low latency requirement (MUST BE < 50ms)
	if result.DurationMs >= 50.0 {
		t.Errorf("expected reaction latency < 50ms, got %.2fms", result.DurationMs)
	}
	if !result.Success {
		t.Errorf("expected success, got errors: %v", result.ValvesFailed)
	}
	if len(result.ValvesClosed) != 2 {
		t.Fatalf("expected 2 water valves closed, got %d", len(result.ValvesClosed))
	}

	// Gas valve must NOT have been actuated
	actuator.mu.Lock()
	if actuator.closedValves["valve.gas_main"] {
		t.Errorf("gas valve should not close on water leak")
	}
	if !actuator.closedValves["valve.water_main"] || !actuator.closedValves["valve.water_secondary"] {
		t.Errorf("water valves must be closed")
	}
	actuator.mu.Unlock()

	// Verify latch state
	status := svc.Status()
	if !status.IsLatched {
		t.Errorf("expected safe-stop to be latched")
	}
	if status.ActiveAlert == nil || status.ActiveAlert.ID != "alert-water-01" {
		t.Errorf("expected active alert in status")
	}

	// Reset latch
	if err := svc.ResetEmergency(ctx, "operator-alice", "leak fixed and confirmed dry"); err != nil {
		t.Fatalf("failed to reset emergency: %v", err)
	}

	if svc.Status().IsLatched {
		t.Errorf("expected latch to be cleared after reset")
	}
}

func TestSafeStop_GasLeakClosesGasValves(t *testing.T) {
	bus := &mockBus{}
	svc := safety.NewService(bus)
	defer svc.Stop()

	actuator := newMockActuator(5 * time.Millisecond)

	_ = svc.RegisterValve(safety.ValveRegistration{
		ID:       "valve.water_main",
		Type:     safety.ValveWater,
		Actuator: actuator,
	})
	_ = svc.RegisterValve(safety.ValveRegistration{
		ID:       "valve.gas_main",
		Type:     safety.ValveGas,
		Actuator: actuator,
	})
	_ = svc.RegisterValve(safety.ValveRegistration{
		ID:       "valve.master_safety",
		Type:     safety.ValveGeneral,
		Actuator: actuator,
	})

	ctx := context.Background()
	alert := safety.EmergencyAlert{
		Hazard:       safety.HazardGasLeak,
		SourceSensor: "sensor.boiler_ch4",
	}

	res, err := svc.TriggerSafeStop(ctx, alert)
	if err != nil {
		t.Fatalf("TriggerSafeStop failed: %v", err)
	}

	if len(res.ValvesClosed) != 2 {
		t.Fatalf("expected 2 valves closed (gas + general), got %d: %v", len(res.ValvesClosed), res.ValvesClosed)
	}

	actuator.mu.Lock()
	if actuator.closedValves["valve.water_main"] {
		t.Errorf("water valve should not be closed on gas leak")
	}
	if !actuator.closedValves["valve.gas_main"] {
		t.Errorf("gas valve must be closed on gas leak")
	}
	if !actuator.closedValves["valve.master_safety"] {
		t.Errorf("general safety valve must be closed on gas leak")
	}
	actuator.mu.Unlock()
}

func TestSafeStop_PartialHardwareFailureIsolation(t *testing.T) {
	bus := &mockBus{}
	svc := safety.NewService(bus)
	defer svc.Stop()

	actuator := newMockActuator(5 * time.Millisecond)
	actuator.failValves["valve.broken"] = true

	_ = svc.RegisterValve(safety.ValveRegistration{
		ID:       "valve.healthy",
		Type:     safety.ValveWater,
		Actuator: actuator,
	})
	_ = svc.RegisterValve(safety.ValveRegistration{
		ID:       "valve.broken",
		Type:     safety.ValveWater,
		Actuator: actuator,
	})

	ctx := context.Background()
	res, err := svc.TriggerSafeStop(ctx, safety.EmergencyAlert{
		Hazard: safety.HazardWaterLeak,
	})
	if err != nil {
		t.Fatalf("TriggerSafeStop failed: %v", err)
	}

	if res.Success {
		t.Errorf("expected failure when broken valve fails")
	}
	if len(res.ValvesClosed) != 1 || res.ValvesClosed[0] != "valve.healthy" {
		t.Errorf("healthy valve should have closed regardless: %v", res.ValvesClosed)
	}
	if _, ok := res.ValvesFailed["valve.broken"]; !ok {
		t.Errorf("broken valve should be reported in ValvesFailed")
	}
}

func TestSafeStop_FastChannel(t *testing.T) {
	bus := &mockBus{}
	svc := safety.NewService(bus)
	defer svc.Stop()

	actuator := newMockActuator(2 * time.Millisecond)
	_ = svc.RegisterValve(safety.ValveRegistration{
		ID:       "valve.water_fast",
		Type:     safety.ValveWater,
		Actuator: actuator,
	})

	fastCh := svc.FastChannel()
	fastCh <- safety.EmergencyAlert{
		ID:     "fast-alert-01",
		Hazard: safety.HazardWaterLeak,
	}

	// Allow fast channel worker to process
	time.Sleep(50 * time.Millisecond)

	status := svc.Status()
	if !status.IsLatched {
		t.Errorf("expected system to be latched after fast channel alert")
	}
	if status.ActiveAlert == nil || status.ActiveAlert.ID != "fast-alert-01" {
		t.Errorf("expected fast-alert-01 to be active")
	}
}
