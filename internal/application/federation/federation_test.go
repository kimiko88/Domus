package federation_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/application/federation"
	"github.com/domus-hub/domus/internal/domain/events"
)

type mockBus struct {
	mu     sync.Mutex
	events []events.Event
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

type mockProxy struct {
	mu       sync.Mutex
	commands []string
}

func (p *mockProxy) ForwardCommand(ctx context.Context, satelliteID, localEntityID, command string, payload any) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.commands = append(p.commands, satelliteID+":"+localEntityID+":"+command)
	return nil
}

func TestFederation_LifecycleAndEntityMirroring(t *testing.T) {
	bus := &mockBus{}
	proxy := &mockProxy{}

	svc := federation.NewService(bus, federation.Config{
		Role:         federation.RoleParent,
		CommandProxy: proxy,
	})

	if svc.Role() != federation.RoleParent {
		t.Errorf("expected RoleParent, got %s", svc.Role())
	}

	// 1. Register Satellite
	err := svc.RegisterSatellite(federation.SatelliteNode{
		ID:       "sat_outbuilding",
		Name:     "Garden Dependance",
		Endpoint: "https://10.0.8.2:8443",
	})
	if err != nil {
		t.Fatalf("RegisterSatellite failed: %v", err)
	}

	sats := svc.ListSatellites()
	if len(sats) != 1 || sats[0].ID != "sat_outbuilding" {
		t.Fatalf("expected 1 satellite enrolled, got %+v", sats)
	}

	// 2. Sync Satellite Entities
	remoteEntities := []federation.RemoteEntity{
		{
			LocalID: "light.patio",
			Domain:  "light",
			State:   "off",
		},
		{
			LocalID: "climate.sauna",
			Domain:  "climate",
			State:   "heat",
			Attributes: map[string]any{
				"target_temp": 75.0,
			},
		},
	}

	if err := svc.SyncEntities("sat_outbuilding", remoteEntities); err != nil {
		t.Fatalf("SyncEntities failed: %v", err)
	}

	entities := svc.ListEntities()
	if len(entities) != 2 {
		t.Fatalf("expected 2 mirrored entities, got %d", len(entities))
	}

	patio, ok := svc.GetEntity("sat_outbuilding:light.patio")
	if !ok {
		t.Fatalf("failed to find sat_outbuilding:light.patio by global ID")
	}
	if patio.State != "off" {
		t.Errorf("expected state 'off', got %q", patio.State)
	}

	// 3. Dispatch Command to Satellite Entity
	ctx := context.Background()
	err = svc.DispatchCommand(ctx, "sat_outbuilding:light.patio", "turn_on", nil)
	if err != nil {
		t.Fatalf("DispatchCommand failed: %v", err)
	}

	proxy.mu.Lock()
	if len(proxy.commands) != 1 || proxy.commands[0] != "sat_outbuilding:light.patio:turn_on" {
		t.Errorf("unexpected command proxy execution: %v", proxy.commands)
	}
	proxy.mu.Unlock()

	// 4. Ingest State Change from Satellite
	now := time.Now()
	svc.IngestStateChange("sat_outbuilding", "light.patio", "on", map[string]any{"brightness": 100}, now)

	patioAfter, _ := svc.GetEntity("sat_outbuilding:light.patio")
	if patioAfter.State != "on" {
		t.Errorf("expected state 'on' after ingestion, got %q", patioAfter.State)
	}

	// Check domain event on parent bus
	time.Sleep(50 * time.Millisecond)
	bus.mu.Lock()
	eventsCount := len(bus.events)
	bus.mu.Unlock()

	if eventsCount != 1 {
		t.Errorf("expected 1 domain event on parent bus, got %d", eventsCount)
	}
}
