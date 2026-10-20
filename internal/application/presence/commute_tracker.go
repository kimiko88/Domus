// Package presence implements dynamic commute and ETA tracking for smart climate & EV charging pre-conditioning.
package presence

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/google/uuid"
)

// CommuteStatus captures dynamic transit telemetry towards home.
type CommuteStatus struct {
	UserID            string    `json:"user_id"`
	CurrentLat        float64   `json:"current_lat"`
	CurrentLon        float64   `json:"current_lon"`
	DistanceMeters    float64   `json:"distance_meters"`
	EstimatedSpeedKmh float64   `json:"estimated_speed_kmh"`
	MinutesToHome     float64   `json:"minutes_to_home"`
	IsApproaching     bool      `json:"is_approaching"`
	PreHeatTriggered  bool      `json:"preheat_triggered"`
	EVChargeTriggered bool      `json:"ev_charge_triggered"`
	LastUpdate        time.Time `json:"last_update"`
}

// CommuteConfig configures home coordinates and automation thresholds.
type CommuteConfig struct {
	HomeLatitude            float64 `json:"home_latitude"`
	HomeLongitude           float64 `json:"home_longitude"`
	PreHeatMinutesThreshold float64 `json:"preheat_minutes"` // e.g. 30 min before arrival
	EVChargeMinutesThreshold float64 `json:"ev_charge_minutes"` // e.g. 15 min before arrival
	DefaultSpeedKmh         float64 `json:"default_speed_kmh"` // e.g. 45 km/h for ETA fallback
}

// CommuteTracker maintains ETA calculations and initiates early pre-conditioning.
type CommuteTracker struct {
	mu       sync.RWMutex
	cfg      CommuteConfig
	bus      events.Bus
	statuses map[string]*CommuteStatus
}

// NewCommuteTracker creates a dynamic commute and ETA tracking service.
func NewCommuteTracker(bus events.Bus, cfg CommuteConfig) *CommuteTracker {
	if cfg.DefaultSpeedKmh <= 0 {
		cfg.DefaultSpeedKmh = 40.0
	}
	if cfg.PreHeatMinutesThreshold <= 0 {
		cfg.PreHeatMinutesThreshold = 30.0
	}
	if cfg.EVChargeMinutesThreshold <= 0 {
		cfg.EVChargeMinutesThreshold = 15.0
	}

	return &CommuteTracker{
		cfg:      cfg,
		bus:      bus,
		statuses: make(map[string]*CommuteStatus),
	}
}

// UpdateLocation recalculates distance, speed, ETA, and checks pre-conditioning triggers.
func (t *CommuteTracker) UpdateLocation(ctx context.Context, userID string, lat, lon, speedKmh float64, now time.Time) CommuteStatus {
	t.mu.Lock()
	defer t.mu.Unlock()

	distMeters := haversineDistance(lat, lon, t.cfg.HomeLatitude, t.cfg.HomeLongitude)

	// Effective speed for ETA estimation
	effectiveSpeed := speedKmh
	if effectiveSpeed < 10.0 {
		effectiveSpeed = t.cfg.DefaultSpeedKmh
	}

	// ETA in minutes: (dist in km / speed in km/h) * 60
	distKm := distMeters / 1000.0
	minutesToHome := (distKm / effectiveSpeed) * 60.0

	prev, exists := t.statuses[userID]
	isApproaching := false
	preheatTriggered := false
	evTriggered := false

	if exists {
		// Approaching if current distance is closer than previous
		isApproaching = distMeters < prev.DistanceMeters
		preheatTriggered = prev.PreHeatTriggered
		evTriggered = prev.EVChargeTriggered
	}

	// Threshold evaluation (only when actively approaching and outside home)
	if isApproaching && distMeters > 300.0 {
		if !preheatTriggered && minutesToHome <= t.cfg.PreHeatMinutesThreshold {
			preheatTriggered = true
			slog.Info("commute tracker triggering pre-heat climate conditioning",
				"user_id", userID,
				"eta_min", fmt.Sprintf("%.1f", minutesToHome),
			)
		}
		if !evTriggered && minutesToHome <= t.cfg.EVChargeMinutesThreshold {
			evTriggered = true
			slog.Info("commute tracker triggering EV charger pre-conditioning",
				"user_id", userID,
				"eta_min", fmt.Sprintf("%.1f", minutesToHome),
			)
		}
	}

	// Reset triggers when user reaches home (< 100m)
	if distMeters < 100.0 {
		preheatTriggered = false
		evTriggered = false
		isApproaching = false
		minutesToHome = 0.0
	}

	status := &CommuteStatus{
		UserID:            userID,
		CurrentLat:        lat,
		CurrentLon:        lon,
		DistanceMeters:    distMeters,
		EstimatedSpeedKmh: effectiveSpeed,
		MinutesToHome:     minutesToHome,
		IsApproaching:     isApproaching,
		PreHeatTriggered:  preheatTriggered,
		EVChargeTriggered: evTriggered,
		LastUpdate:        now,
	}

	t.statuses[userID] = status

	// Emit domain event for automation rules
	if t.bus != nil {
		go func(s CommuteStatus) {
			pubCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = t.bus.Publish(pubCtx, events.Event{
				ID:         uuid.NewString(),
				Type:       events.PresenceCommuteUpdated,
				OccurredAt: now,
				Source:     "presence.commute_tracker",
				Payload:    s,
			})
		}(*status)
	}

	return *status
}

// GetStatus returns the current transit state of a user.
func (t *CommuteTracker) GetStatus(userID string) (CommuteStatus, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	s, ok := t.statuses[userID]
	if !ok {
		return CommuteStatus{}, false
	}
	return *s, true
}
