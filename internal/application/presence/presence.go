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
type roomPresenceState struct {
	SubState  string
	LastSeenAt time.Time
}

type PersonLocation struct {
	UserID        string        `json:"user_id"`
	State         PresenceState `json:"state"`
	GPSLat        float64       `json:"gps_lat,omitempty"`
	GPSLon        float64       `json:"gps_lon,omitempty"`
	HasGPS        bool          `json:"has_gps"`
	GPSLastSeenAt time.Time     `json:"gps_last_seen_at,omitempty"`
	WiFiConnected  bool      `json:"wifi_connected"`
	WiFiLastSeenAt time.Time `json:"wifi_last_seen_at,omitempty"`
	WiFiMAC       string        `json:"wifi_mac,omitempty"`
	RoomZone      string        `json:"room_zone,omitempty"` // e.g. "bedroom", "living_room"
	SubState      string        `json:"sub_state,omitempty"` // "moving", "stationary", "sleeping"
	RoomOccupied  bool          `json:"room_occupied"`
	RoomLastSeenAt time.Time    `json:"room_last_seen_at,omitempty"`
	roomPresence  map[string]roomPresenceState
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
	p.GPSLastSeenAt = now
	p.LastSeenAt = now

	geofenceEnabled := s.geofence.Enabled || s.geofence.Latitude != 0 || s.geofence.Longitude != 0
	inGPSZone := false
	if geofenceEnabled {
		distMeters := haversineDistance(lat, lon, s.geofence.Latitude, s.geofence.Longitude)
		inGPSZone = distMeters <= s.geofence.RadiusM
	}
	gpsFresh := p.HasGPS && !p.GPSLastSeenAt.IsZero() && now.Sub(p.GPSLastSeenAt) <= 10*time.Minute
	if !gpsFresh {
		inGPSZone = false
	}
	wifiFresh := p.WiFiConnected && !p.WiFiLastSeenAt.IsZero() && now.Sub(p.WiFiLastSeenAt) <= 2*time.Minute
	roomPresent := p.RoomOccupied && !p.RoomLastSeenAt.IsZero() && now.Sub(p.RoomLastSeenAt) <= 5*time.Minute
	return s.evaluateStateLocked(ctx, p, inGPSZone, wifiFresh, roomPresent, now)
}

// UpdateWiFi handles local network ARP / Wi-Fi association updates.
func (s *Service) UpdateWiFi(ctx context.Context, userID string, mac string, connected bool, now time.Time) PresenceState {
	s.mu.Lock()
	defer s.mu.Unlock()

	p := s.getOrCreate(userID)
	p.WiFiMAC = mac
	p.WiFiConnected = connected
	p.WiFiLastSeenAt = now
	p.LastSeenAt = now

	// Wi-Fi connected to local LAN is high-confidence proof that user is at home.
	geofenceEnabled := s.geofence.Enabled || s.geofence.Latitude != 0 || s.geofence.Longitude != 0
	inGPSZone := false
	if p.HasGPS && geofenceEnabled {
		inGPSZone = haversineDistance(p.GPSLat, p.GPSLon, s.geofence.Latitude, s.geofence.Longitude) <= s.geofence.RadiusM
	}
	gpsFresh := p.HasGPS && !p.GPSLastSeenAt.IsZero() && now.Sub(p.GPSLastSeenAt) <= 10*time.Minute
	if !gpsFresh {
		inGPSZone = false
	}
	roomPresent := p.RoomOccupied && !p.RoomLastSeenAt.IsZero() && now.Sub(p.RoomLastSeenAt) <= 5*time.Minute
	return s.evaluateStateLocked(ctx, p, inGPSZone, connected, roomPresent, now)
}

// UpdateRoomSensor updates room-level mmWave radar presence (stationary, moving, sleeping).
func (s *Service) UpdateRoomSensor(ctx context.Context, userID string, room string, subState string, now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	p := s.getOrCreate(userID)
	p.LastSeenAt = now
	room = strings.TrimSpace(room)
	normalized := strings.ToLower(strings.TrimSpace(subState))
	if p.roomPresence == nil {
		p.roomPresence = make(map[string]roomPresenceState)
	}
	roomActive := room != "" &&
		normalized != "clear" && normalized != "cleared" &&
		normalized != "away" && normalized != "vacant" &&
		normalized != "none" && normalized != "off"
	if roomActive {
		p.roomPresence[room] = roomPresenceState{SubState: normalized, LastSeenAt: now}
	} else {
		delete(p.roomPresence, room)
	}

	// Keep presence from every room independently: a clear event in one room
	// must not cancel a fresh occupied signal from another room.
	p.RoomOccupied = false
	p.RoomLastSeenAt = time.Time{}
	latestRoom := ""
	latestSubState := ""
	for roomName, signal := range p.roomPresence {
		age := now.Sub(signal.LastSeenAt)
		if age < -30*time.Second || age > 5*time.Minute {
			delete(p.roomPresence, roomName)
			continue
		}
		if p.RoomLastSeenAt.IsZero() || signal.LastSeenAt.After(p.RoomLastSeenAt) {
			p.RoomOccupied = true
			p.RoomLastSeenAt = signal.LastSeenAt
			latestRoom = roomName
			latestSubState = signal.SubState
		}
	}
	if p.RoomOccupied {
		p.RoomZone = latestRoom
		p.SubState = latestSubState
	} else {
		p.RoomZone = room
		p.SubState = normalized
	}
	geofenceEnabled := s.geofence.Enabled || s.geofence.Latitude != 0 || s.geofence.Longitude != 0
	inGPSZone := p.HasGPS && geofenceEnabled &&
		haversineDistance(p.GPSLat, p.GPSLon, s.geofence.Latitude, s.geofence.Longitude) <= s.geofence.RadiusM
	gpsFresh := p.HasGPS && !p.GPSLastSeenAt.IsZero() && now.Sub(p.GPSLastSeenAt) <= 10*time.Minute
	if !gpsFresh {
		inGPSZone = false
	}
	wifiFresh := p.WiFiConnected && !p.WiFiLastSeenAt.IsZero() && now.Sub(p.WiFiLastSeenAt) <= 2*time.Minute
	roomPresent := p.RoomOccupied && !p.RoomLastSeenAt.IsZero()
	s.evaluateStateLocked(ctx, p, inGPSZone, wifiFresh, roomPresent, now)
}

func (s *Service) evaluateStateLocked(ctx context.Context, p *PersonLocation, inGPSZone bool, wifiConnected bool, roomPresent bool, now time.Time) PresenceState {
	oldState := p.State
	var newState PresenceState

	// Multi-tier fusion:
	// If Wi-Fi is connected OR GPS is inside geofence -> user is HOME
	if wifiConnected || inGPSZone || roomPresent {
		if oldState == StateAway || oldState == StateJustLeft || oldState == "" {
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
	if p.roomPresence != nil {
		cp.roomPresence = make(map[string]roomPresenceState, len(p.roomPresence))
		for room, signal := range p.roomPresence {
			cp.roomPresence[room] = signal
		}
	}
	return &cp, true
}

// Start periodically expires stale presence signals so automations receive
// departure events even when a device stops reporting without a final "away".
func (s *Service) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				s.ExpireStaleSignals(now.UTC())
			}
		}
	}()
}

// ExpireStaleSignals recomputes presence from recent GPS, Wi-Fi and room sensor
// evidence. It is exported so deterministic tests and maintenance tasks can
// trigger expiry without waiting for the periodic worker.
func (s *Service) ExpireStaleSignals(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, p := range s.people {
		p.RoomOccupied = false
		p.RoomLastSeenAt = time.Time{}
		latestRoom := ""
		latestSubState := ""
		for room, signal := range p.roomPresence {
			age := now.Sub(signal.LastSeenAt)
			if age < -30*time.Second || age > 5*time.Minute {
				delete(p.roomPresence, room)
				continue
			}
			if p.RoomLastSeenAt.IsZero() || signal.LastSeenAt.After(p.RoomLastSeenAt) {
				p.RoomOccupied = true
				p.RoomLastSeenAt = signal.LastSeenAt
				latestRoom = room
				latestSubState = signal.SubState
			}
		}
		if p.RoomOccupied {
			p.RoomZone = latestRoom
			p.SubState = latestSubState
		}

		gpsAge := now.Sub(p.GPSLastSeenAt)
		wifiAge := now.Sub(p.WiFiLastSeenAt)
		roomAge := now.Sub(p.RoomLastSeenAt)
		gpsFresh := p.HasGPS && !p.GPSLastSeenAt.IsZero() && gpsAge >= -30*time.Second && gpsAge <= 10*time.Minute
		wifiFresh := p.WiFiConnected && !p.WiFiLastSeenAt.IsZero() && wifiAge >= -30*time.Second && wifiAge <= 2*time.Minute
		roomFresh := p.RoomOccupied && !p.RoomLastSeenAt.IsZero() && roomAge >= -30*time.Second && roomAge <= 5*time.Minute

		geofenceEnabled := s.geofence.Enabled || s.geofence.Latitude != 0 || s.geofence.Longitude != 0
		gpsHome := gpsFresh && geofenceEnabled &&
			haversineDistance(p.GPSLat, p.GPSLon, s.geofence.Latitude, s.geofence.Longitude) <= s.geofence.RadiusM
		evidence := wifiFresh || roomFresh || gpsHome

		oldState := p.State
		if evidence {
			if oldState == StateAway || oldState == StateJustLeft || oldState == "" {
				p.State = StateJustArrived
			} else {
				p.State = StateHome
			}
		} else if oldState == StateHome || oldState == StateJustArrived {
			p.State = StateJustLeft
		} else {
			p.State = StateAway
		}

		if oldState != p.State {
			s.publishPresenceChanged(context.Background(), p, now)
		}
	}
}

// IsAnyoneHome returns true if at least one tracked user is home.
func (s *Service) IsAnyoneHome() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now().UTC()
	for _, p := range s.people {
		gpsAge := now.Sub(p.GPSLastSeenAt)
		wifiAge := now.Sub(p.WiFiLastSeenAt)
		roomAge := now.Sub(p.RoomLastSeenAt)
		gpsFresh := p.HasGPS && !p.GPSLastSeenAt.IsZero() && gpsAge >= -30*time.Second && gpsAge <= 10*time.Minute
		wifiFresh := p.WiFiConnected && !p.WiFiLastSeenAt.IsZero() && wifiAge >= -30*time.Second && wifiAge <= 2*time.Minute
		roomFresh := p.RoomOccupied && !p.RoomLastSeenAt.IsZero() && roomAge >= -30*time.Second && roomAge <= 5*time.Minute

		geofenceEnabled := s.geofence.Enabled || s.geofence.Latitude != 0 || s.geofence.Longitude != 0
		gpsHome := gpsFresh && geofenceEnabled &&
			haversineDistance(p.GPSLat, p.GPSLon, s.geofence.Latitude, s.geofence.Longitude) <= s.geofence.RadiusM
		if wifiFresh || roomFresh || gpsHome {
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
	// Floating-point rounding near antipodal points can move a just outside [0, 1].
	a = math.Max(0, math.Min(1, a))

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadius * c
}
