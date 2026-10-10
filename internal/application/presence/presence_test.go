package presence

import (
	"context"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
)

type mockPresenceBus struct {
	events []events.Event
}

func (m *mockPresenceBus) Publish(ctx context.Context, evt events.Event) error {
	m.events = append(m.events, evt)
	return nil
}
func (m *mockPresenceBus) Subscribe(eventType events.EventType, handler events.Handler) func() {
	return func() {}
}
func (m *mockPresenceBus) SubscribeAll(handler events.Handler) func() { return func() {} }

func TestPresenceService_Fusion(t *testing.T) {
	bus := &mockPresenceBus{}
	// Home at Colosseum, Rome: 41.8902, 12.4922
	homeGeofence := HomeGeofence{
		Latitude:  41.8902,
		Longitude: 12.4922,
		RadiusM:   200.0,
	}

	svc := NewService(homeGeofence, bus)
	ctx := context.Background()
	now := time.Now().UTC()

	// 1. Initially far away (Milan: 45.4642, 9.1900)
	st := svc.UpdateGPS(ctx, "user_marco", 45.4642, 9.1900, now)
	if st != StateAway {
		t.Errorf("expected away, got %s", st)
	}
	if svc.IsAnyoneHome() {
		t.Errorf("expected nobody home")
	}

	// 2. Wi-Fi connects on local network (e.g. phone enters driveway)
	st = svc.UpdateWiFi(ctx, "user_marco", "AA:BB:CC:DD:EE:FF", true, now.Add(1*time.Minute))
	if st != StateJustArrived {
		t.Errorf("expected just_arrived, got %s", st)
	}
	if !svc.IsAnyoneHome() {
		t.Errorf("expected someone home")
	}

	// 3. mmWave radar detects movement in living room
	svc.UpdateRoomSensor(ctx, "user_marco", "living_room", "stationary", now.Add(2*time.Minute))
	p, ok := svc.GetPersonState("user_marco")
	if !ok || p.RoomZone != "living_room" || p.SubState != "stationary" {
		t.Errorf("expected living_room stationary, got %+v", p)
	}
}
