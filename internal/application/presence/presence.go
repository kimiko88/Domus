// Package presence implements multi-tier presence detection and sensor fusion (GPS, Wi-Fi ARP, mmWave).
package presence

import (
	"context"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/google/uuid"
)

type PresenceState string

const (
	StateHome        PresenceState = "home"
	StateAway        PresenceState = "away"
	StateJustArrived PresenceState = "just_arrived"
	StateJustLeft    PresenceState = "just_left"
)

// PersonLocation holds raw multi-sensor telemetry for a person.
type PersonLocation struct {
	UserID        string        `json:"user_id"`
	State         PresenceState `json:"state"`
	GPSLat        float64       `json:"gps_lat,omitempty"`
	GPSLon        float64       `json:"gps_lon,omitempty"`
	HasGPS        bool          `json:"has_gps"`
	WiFiConnected bool          `json:"wifi_connected"`
	WiFiMAC       string        `json:"wifi_mac,omitempty"`
	RoomZone      string        `json:"room_zone,omitempty"` // e.g. "bedroom", "living_room"
	SubState      string        `json:"sub_state,omitempty"` // "moving", "stationary", "sleeping"
	RoomOccupied  bool          `json:"room_occupied"`
	RoomLastSeenAt time.Time    `json:"room_last_seen_at,omitempty"`
	LastSeenAt    time.Time     `json:"last_seen_at"`
}

// HomeGeofence defines home coordinates and radius.
type HomeGeofence struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	RadiusM   float64 `json:"radius_m"` // default 150m
	Enabled   bool    `json:"enabled"`
}

// Service manages presence evaluation and multi-sensor fusion.
type Service struct {
	bus      events.Bus
	geofence HomeGeofence

	mu     sync.RWMutex
	people map[string]*PersonLocation
}

// NewService creates a new presence tracking service.
func NewService(geofence HomeGeofence, bus events.Bus) *Service {
	if geofence.RadiusM <= 0 {
		geofence.RadiusM = 150.0
	}
	return &Service{
		bus:      bus,
		geofence: geofence,
		people:   make(map[string]*PersonLocation),
	}
}

// UpdateGPS handles GPS telemetry updates from the mobile app.
func (s *Service) UpdateGPS(ctx context.Context, userID string, lat, lon float64, now time.Time) PresenceState {
	s.mu.Lock()
	defer s.mu.Unlock()

	p := s.getOrCreate(userID)
	if !ValidCoordinates(lat, lon) {
		return p.State
	}
	p.GPSLat = lat
	p.GPSLon = lon
	p.HasGPS = true
	p.LastSeenAt = now

	geofenceEnabled := s.geofence.Enabled || s.geofence.Latitude != 0 || s.geofence.Longitude != 0
	inGPSZone := false
	if geofenceEnabled {
		distMeters := haversineDistance(lat, lon, s.geofence.Latitude, s.geofence.Longitude)
		inGPSZone = distMeters <= s.geofence.RadiusM
	}
	roomPresent := p.RoomOccupied && !p.RoomLastSeenAt.IsZero() && now.Sub(p.RoomLastSeenAt) <= 5*time.Minute
	return s.evaluateStateLocked(ctx, p, inGPSZone, p.WiFiConnected, roomPresent, now)
}

// UpdateWiFi handles local network ARP / Wi-Fi association updates.
func (s *Service) UpdateWiFi(ctx context.Context, userID string, mac string, connected bool, now time.Time) PresenceState {
	s.mu.Lock()
	defer s.mu.Unlock()

	p := s.getOrCreate(userID)
	p.WiFiMAC = mac
	p.WiFiConnected = connected
	p.LastSeenAt = now

	// Wi-Fi connected to local LAN is high-confidence proof that user is at home.
	geofenceEnabled := s.geofence.Enabled || s.geofence.Latitude != 0 || s.geofence.Longitude != 0
	inGPSZone := false
	if p.HasGPS && geofenceEnabled {
		inGPSZone = haversineDistance(p.GPSLat, p.GPSLon, s.geofence.Latitude, s.geofence.Longitude) <= s.geofence.RadiusM
	}
	roomPresent := p.RoomOccupied && !p.RoomLastSeenAt.IsZero() && now.Sub(p.RoomLastSeenAt) <= 5*time.Minute
	return s.evaluateStateLocked(ctx, p, inGPSZone, connected, roomPresent, now)
}

// UpdateRoomSensor updates room-level mmWave radar presence (stationary, moving, sleeping).
func (s *Service) UpdateRoomSensor(ctx context.Context, userID string, room string, subState string, now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	p := s.getOrCreate(userID)
	p.RoomZone = room
	p.SubState = subState
	p.LastSeenAt = now
	normalized := strings.ToLower(strings.TrimSpace(subState))
	p.RoomOccupied = strings.TrimSpace(room) != "" &&
		normalized != "clear" && normalized != "cleared" &&
		normalized != "away" && normalized != "vacant" &&
		normalized != "none" && normalized != "off"
	if p.RoomOccupied {
		p.RoomLastSeenAt = now
	} else {
		p.RoomLastSeenAt = time.Time{}
	}
	geofenceEnabled := s.geofence.Enabled || s.geofence.Latitude != 0 || s.geofence.Longitude != 0
	inGPSZone := p.HasGPS && geofenceEnabled &&
		haversineDistance(p.GPSLat, p.GPSLon, s.geofence.Latitude, s.geofence.Longitude) <= s.geofence.RadiusM
	roomPresent := p.RoomOccupied && !p.RoomLastSeenAt.IsZero()
	s.evaluateStateLocked(ctx, p, inGPSZone, p.WiFiConnected, roomPresent, now)
}

func (s *Service) evaluateStateLocked(ctx context.Context, p *PersonLocation, inGPSZone bool, wifiConnected bool, roomPresent bool, now time.Time) PresenceState {
	oldState := p.State
	var newState PresenceState

	// Multi-tier fusion:
	// If Wi-Fi is connected OR GPS is inside geofence -> user is HOME
	if wifiConnected || inGPSZone || roomPresent {
		if oldState == StateAway || oldState == "" {
			newState = StateJustArrived
		} else {
			newState = StateHome
		}
	} else {
		if oldState == StateHome || oldState == StateJustArrived {
			newState = StateJustLeft
		} else {
			newState = StateAway
		}
	}

	p.State = newState

	if oldState != newState {
		s.publishPresenceChanged(ctx, p, now)
	}

	return newState
}

func (s *Service) publishPresenceChanged(ctx context.Context, p *PersonLocation, now time.Time) {
	if s.bus == nil {
		return
	}

	_ = s.bus.Publish(ctx, events.Event{
		ID:         uuid.NewString(),
		Type:       events.EventType("presence.state_changed"),
		OccurredAt: now,
		Source:     "presence.service",
		Payload: map[string]any{
			"user_id":   p.UserID,
			"state":     string(p.State),
			"room_zone": p.RoomZone,
			"sub_state": p.SubState,
		},
	})
}

// GetPersonState returns the current state of a user.
func (s *Service) GetPersonState(userID string) (*PersonLocation, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.people[userID]
	if !ok {
		return nil, false
	}
	cp := *p
	return &cp, true
}

// IsAnyoneHome returns true if at least one tracked user is home.
func (s *Service) IsAnyoneHome() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, p := range s.people {
		if p.State == StateHome || p.State == StateJustArrived {
			return true
		}
	}
	return false
}

func (s *Service) getOrCreate(userID string) *PersonLocation {
	if p, ok := s.people[userID]; ok {
		return p
	}
	p := &PersonLocation{
		UserID: userID,
		State:  StateAway,
	}
	s.people[userID] = p
	return p
}

// ValidCoordinates reports whether latitude and longitude are finite WGS84 coordinates.
func ValidCoordinates(lat, lon float64) bool {
	return !math.IsNaN(lat) && !math.IsNaN(lon) &&
		!math.IsInf(lat, 0) && !math.IsInf(lon, 0) &&
		lat >= -90 && lat <= 90 && lon >= -180 && lon <= 180
}

// haversineDistance calculates meters between two WGS84 geographic coordinates.
func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371000.0 // meters
	dLat := (lat2 - lat1) * (math.Pi / 180.0)
	dLon := (lon2 - lon1) * (math.Pi / 180.0)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*(math.Pi/180.0))*math.Cos(lat2*(math.Pi/180.0))*
			math.Sin(dLon/2)*math.Sin(dLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadius * c
}
