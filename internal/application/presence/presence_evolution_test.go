package presence_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/application/presence"
	"github.com/domus-hub/domus/internal/domain/events"
)

type mockPresenceBus struct {
	mu     sync.Mutex
	events []events.Event
}

func (b *mockPresenceBus) Publish(ctx context.Context, e events.Event) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = append(b.events, e)
	return nil
}

func (b *mockPresenceBus) Subscribe(eventType events.EventType, handler events.Handler) func() {
	return func() {}
}

func (b *mockPresenceBus) SubscribeAll(handler events.Handler) func() {
	return func() {}
}

func TestVacationSimulator_ReplayWithJitter(t *testing.T) {
	bus := &mockPresenceBus{}
	sim := presence.NewVacationSimulator(bus, presence.VacationSimulatorConfig{
		Enabled:       true,
		JitterMinutes: 5,
	})

	var executedCommands []string
	var cmdMu sync.Mutex

	sim.WithCommandSink(func(ctx context.Context, entityID, cmd string) error {
		cmdMu.Lock()
		defer cmdMu.Unlock()
		executedCommands = append(executedCommands, entityID+":"+cmd)
		return nil
	})

	// Habit: Turn on living room light at 20:00 (72000s)
	sim.RegisterAction(presence.SimulatedAction{
		ID:           "act-light",
		EntityID:     "light.living_room",
		Command:      "turn_on",
		TimeOfDaySec: 20 * 3600, // 20:00:00
		JitterMin:    0,         // Deterministic for this test
	})

	ctx := context.Background()

	// 1. Evaluate at 18:00 (too early)
	t1 := time.Date(2026, 10, 20, 18, 0, 0, 0, time.UTC)
	acts := sim.EvaluateNow(ctx, t1)
	if len(acts) != 0 {
		t.Errorf("expected 0 actions at 18:00, got %d", len(acts))
	}

	// 2. Evaluate at 20:00 (target time)
	t2 := time.Date(2026, 10, 20, 20, 0, 0, 0, time.UTC)
	acts = sim.EvaluateNow(ctx, t2)
	if len(acts) != 1 {
		t.Fatalf("expected 1 action executed at 20:00, got %d", len(acts))
	}

	cmdMu.Lock()
	if len(executedCommands) != 1 || executedCommands[0] != "light.living_room:turn_on" {
		t.Errorf("expected command sink invoked, got %v", executedCommands)
	}
	cmdMu.Unlock()

	// 3. Evaluate 10 minutes later (same day) should not re-trigger
	t3 := time.Date(2026, 10, 20, 20, 10, 0, 0, time.UTC)
	acts = sim.EvaluateNow(ctx, t3)
	if len(acts) != 0 {
		t.Errorf("action should not re-trigger on same day")
	}

	// 4. Test disable
	sim.SetVacationMode(false)
	if sim.IsEnabled() {
		t.Errorf("expected IsEnabled=false after disable")
	}
}

func TestCommuteTracker_CalculatesETAAndTriggersPreConditioning(t *testing.T) {
	bus := &mockPresenceBus{}

	// Home at Rome (41.9028, 12.4964)
	tracker := presence.NewCommuteTracker(bus, presence.CommuteConfig{
		HomeLatitude:            41.9028,
		HomeLongitude:           12.4964,
		PreHeatMinutesThreshold: 30.0,
		EVChargeMinutesThreshold: 15.0,
		DefaultSpeedKmh:         50.0,
	})

	ctx := context.Background()
	now := time.Now()

	// Point 1: 50 km away (far, e.g. at work ~ 41.5, 12.4)
	s1 := tracker.UpdateLocation(ctx, "user-bob", 41.5, 12.4964, 50.0, now)
	if s1.IsApproaching {
		t.Errorf("initial location should not be marked approaching yet")
	}
	if s1.PreHeatTriggered {
		t.Errorf("50km away should not trigger preheat (ETA > 50 min)")
	}

	// Point 2: ~13.6 km away, moving towards home (speed 60 km/h -> ETA ~13.6 min)
	now = now.Add(20 * time.Minute)
	s2 := tracker.UpdateLocation(ctx, "user-bob", 41.78, 12.4964, 60.0, now)

	if !s2.IsApproaching {
		t.Errorf("user should be marked as approaching home")
	}
	if !s2.PreHeatTriggered {
		t.Errorf("expected pre-heat to be triggered within 30 min window (ETA is ~15 min)")
	}
	if !s2.EVChargeTriggered {
		t.Errorf("expected EV charge to be triggered within 15 min window")
	}

	// Point 3: Arrived home (< 100m)
	now = now.Add(15 * time.Minute)
	s3 := tracker.UpdateLocation(ctx, "user-bob", 41.9028, 12.4964, 0.0, now)
	if s3.PreHeatTriggered || s3.EVChargeTriggered {
		t.Errorf("triggers should reset once home")
	}
	if s3.MinutesToHome != 0.0 {
		t.Errorf("expected 0 minutes to home on arrival")
	}
}
