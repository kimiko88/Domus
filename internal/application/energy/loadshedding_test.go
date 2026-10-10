package energy

import (
	"context"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
)

type mockBus struct {
	events []events.Event
}

func (b *mockBus) Publish(ctx context.Context, evt events.Event) error {
	b.events = append(b.events, evt)
	return nil
}

func (b *mockBus) Subscribe(eventType events.EventType, handler events.Handler) func() {
	return func() {}
}

func (b *mockBus) SubscribeAll(handler events.Handler) func() {
	return func() {}
}

func TestLoadShedding_OverloadAndRecovery(t *testing.T) {
	bus := &mockBus{}
	cfg := LoadSheddingConfig{
		MeterEntityKey:   "sensor.home_power",
		MaxPowerWatts:    3300.0,
		OverloadDuration: 60,
		RecoveryWatts:    2500.0,
		RecoveryDelaySec: 120,
		PriorityLoads:    []string{"switch.ev_charger", "switch.water_heater"},
		Enabled:          true,
	}

	ctrl := NewController(cfg, bus)
	ctx := context.Background()
	now := time.Now().UTC()

	// 1. Initial reading under limit
	_ = ctrl.ProcessPowerReading(ctx, 2800.0, now)
	if len(bus.events) != 0 {
		t.Errorf("expected no events, got %d", len(bus.events))
	}

	// 2. Overload starts at now+1s (3800W > 3300W)
	t1 := now.Add(1 * time.Second)
	_ = ctrl.ProcessPowerReading(ctx, 3800.0, t1)
	if len(bus.events) != 0 {
		t.Errorf("no shed should occur before 60s overload duration")
	}

	// 3. Overload persists past 60s (at now+62s)
	t2 := now.Add(62 * time.Second)
	_ = ctrl.ProcessPowerReading(ctx, 3900.0, t2)
	// Should shed first priority load: switch.ev_charger
	if len(bus.events) != 2 { // 1 command + 1 notification
		t.Fatalf("expected 2 events dispatched for shedding, got %d", len(bus.events))
	}

	cmdEvt := bus.events[0]
	if cmdEvt.Type != events.CommandRequested {
		t.Errorf("expected CommandRequested event, got %v", cmdEvt.Type)
	}
	payload := cmdEvt.Payload.(map[string]any)
	if payload["entity_key"] != "switch.ev_charger" || payload["command"] != "turn_off" {
		t.Errorf("expected turn_off on switch.ev_charger, got %v", payload)
	}

	status := ctrl.GetStatus()
	shedded := status["shedded_loads"].([]string)
	if len(shedded) != 1 || shedded[0] != "switch.ev_charger" {
		t.Errorf("expected shedded loads to contain switch.ev_charger, got %v", shedded)
	}

	// 4. Power drops to 2200W (below RecoveryWatts 2500W)
	t3 := now.Add(70 * time.Second)
	_ = ctrl.ProcessPowerReading(ctx, 2200.0, t3)
	// Recovery timer starts, no immediate reconnection
	if len(bus.events) != 2 {
		t.Errorf("expected no restoration before recovery delay")
	}

	// 5. Recovery delay passed (120s later -> t3 + 125s)
	t4 := t3.Add(125 * time.Second)
	_ = ctrl.ProcessPowerReading(ctx, 2100.0, t4)
	if len(bus.events) != 4 { // +1 restore command + 1 info notification
		t.Fatalf("expected 4 total events after restoration, got %d", len(bus.events))
	}

	restoreCmd := bus.events[2]
	payloadRestore := restoreCmd.Payload.(map[string]any)
	if payloadRestore["entity_key"] != "switch.ev_charger" || payloadRestore["command"] != "turn_on" {
		t.Errorf("expected turn_on on switch.ev_charger, got %v", payloadRestore)
	}
}
