package presence

import (
	"context"
	"math"
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
	st = svc.UpdateWiFi(ctx, "user_marco", "AA:BB:CC:DD:EE:FF", true, now.Add(1*time.Second))
	if st != StateJustArrived {
		t.Errorf("expected just_arrived, got %s", st)
	}
	if !svc.IsAnyoneHome() {
		t.Errorf("expected someone home")
	}

	// 3. mmWave radar detects movement in living room
	svc.UpdateRoomSensor(ctx, "user_marco", "living_room", "stationary", now.Add(2*time.Second))
	p, ok := svc.GetPersonState("user_marco")
	if !ok || p.RoomZone != "living_room" || p.SubState != "stationary" {
		t.Errorf("expected living_room stationary, got %+v", p)
	}
}


func TestPresenceService_RoomSensorAndSignalExpiry(t *testing.T) {
	svc := NewService(HomeGeofence{Latitude: 45.0, Longitude: 9.0, RadiusM: 100}, nil)
	ctx := context.Background()
	now := time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC)

	// A room sensor can establish presence even when GPS is outside the geofence.
	svc.UpdateRoomSensor(ctx, "user-1", "living_room", "stationary", now)
	person, ok := svc.GetPersonState("user-1")
	if !ok || person.State != StateJustArrived || !person.RoomOccupied {
		t.Fatalf("room sensor should establish presence, got %+v", person)
	}

	// A second room sensor independently reports presence.
	svc.UpdateRoomSensor(ctx, "user-1", "kitchen", "occupied", now.Add(10*time.Second))
	// Clearing the living room must not cancel the kitchen's fresh occupancy.
	svc.UpdateRoomSensor(ctx, "user-1", "living_room", "clear", now.Add(20*time.Second))
	person, _ = svc.GetPersonState("user-1")
	if !person.RoomOccupied || person.RoomZone != "kitchen" {
		t.Fatalf("clearing one room must preserve another active room, got %+v", person)
	}
	// Clearing the remaining occupied room removes the room-based evidence.
	svc.UpdateRoomSensor(ctx, "user-1", "kitchen", "clear", now.Add(time.Minute))
	person, _ = svc.GetPersonState("user-1")
	if person.RoomOccupied || person.State != StateJustLeft {
		t.Fatalf("clearing all room sensors should remove presence, got %+v", person)
	}

	// A stale GPS fix must not keep a person home indefinitely.
	svc.UpdateGPS(ctx, "user-2", 45.0, 9.0, now)
	person, _ = svc.GetPersonState("user-2")
	if person.State != StateJustArrived {
		t.Fatalf("expected geofence arrival, got %s", person.State)
	}
	svc.UpdateWiFi(ctx, "user-2", "AA:BB:CC:DD:EE:FF", false, now.Add(11*time.Minute))
	person, _ = svc.GetPersonState("user-2")
	if person.State != StateJustLeft {
		t.Fatalf("stale GPS should expire, expected just_left, got %s", person.State)
	}
}

func TestValidCoordinates(t *testing.T) {
	valid := [][2]float64{{0, 0}, {90, 180}, {-90, -180}, {45.4642, 9.19}}
	for _, point := range valid {
		if !ValidCoordinates(point[0], point[1]) {
			t.Errorf("expected valid coordinates %v", point)
		}
	}
	invalid := [][2]float64{{91, 0}, {0, 181}, {-91, 0}, {0, -181}, {math.NaN(), 0}, {0, math.Inf(1)}}
	for _, point := range invalid {
		if ValidCoordinates(point[0], point[1]) {
			t.Errorf("expected invalid coordinates %v", point)
		}
	}
}
