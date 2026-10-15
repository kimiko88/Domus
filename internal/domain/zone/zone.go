// Package zone defines domain models and repository contracts for multi-zone spatial geofencing and person tracking.
package zone

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"
)

var (
	ErrZoneNotFound   = errors.New("zone: not found")
	ErrPersonNotFound = errors.New("person: not found")
	ErrInvalidZone    = errors.New("zone: name, latitude, and longitude are required")
)

// Zone represents a geographical boundary (e.g. Home, Work, School).
type Zone struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	Radius    float64   `json:"radius"` // meters
	Icon      string    `json:"icon"`
	Passive   bool      `json:"passive"` // if passive, won't trigger presence
	CreatedAt time.Time `json:"created_at"`
}

// DistanceTo calculates distance in meters to a given lat/lon using the Haversine formula.
func (z *Zone) DistanceTo(lat, lon float64) float64 {
	const earthRadius = 6371000.0 // meters
	dLat := (lat - z.Latitude) * (math.Pi / 180.0)
	dLon := (lon - z.Longitude) * (math.Pi / 180.0)

	lat1 := z.Latitude * (math.Pi / 180.0)
	lat2 := lat * (math.Pi / 180.0)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Sin(dLon/2)*math.Sin(dLon/2)*math.Cos(lat1)*math.Cos(lat2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadius * c
}

// Contains checks if the specified coordinates fall within the zone's radius.
func (z *Zone) Contains(lat, lon float64) bool {
	return z.DistanceTo(lat, lon) <= z.Radius
}

// HaversineDistance calculates the geodesic distance in meters between two coordinate pairs.
func HaversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	z := &Zone{Latitude: lat1, Longitude: lon1}
	return z.DistanceTo(lat2, lon2)
}

// Person represents a family member or user tracked across multiple presence sensors and devices.
type Person struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	UserID         string    `json:"user_id,omitempty"`
	State          string    `json:"state"` // "home", "not_home", or zone name (e.g. "Ufficio", "Scuola")
	Latitude       *float64  `json:"latitude,omitempty"`
	Longitude      *float64  `json:"longitude,omitempty"`
	DeviceTrackers []string  `json:"device_trackers"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Validate checks zone invariants.
func (z *Zone) Validate() error {
	if strings.TrimSpace(z.Name) == "" {
		return ErrInvalidZone
	}
	if z.Radius <= 0 {
		z.Radius = 100.0
	}
	if z.Icon == "" {
		z.Icon = "map-pin"
	}
	return nil
}

// Repository persists spatial zones and person tracker profiles.
type Repository interface {
	CreateZone(ctx context.Context, z *Zone) error
	GetZone(ctx context.Context, id string) (*Zone, error)
	ListZones(ctx context.Context) ([]*Zone, error)
	UpdateZone(ctx context.Context, z *Zone) error
	DeleteZone(ctx context.Context, id string) error

	CreatePerson(ctx context.Context, p *Person) error
	GetPerson(ctx context.Context, id string) (*Person, error)
	ListPersons(ctx context.Context) ([]*Person, error)
	UpdatePerson(ctx context.Context, p *Person) error
	DeletePerson(ctx context.Context, id string) error
}
