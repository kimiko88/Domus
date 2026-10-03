// Package entity defines the core Entity domain model in Domus.
package entity

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrEntityNotFound    = errors.New("entity: not found")
	ErrInvalidEntityKey  = errors.New("entity: key must be in format 'domain.name' (e.g. light.living_room)")
	ErrEntityNotWritable = errors.New("entity: entity is read-only and does not accept commands")
)

// Entity represents an observable or controllable telemetry point.
type Entity struct {
	ID                string    `json:"id"`
	DeviceID          string    `json:"device_id,omitempty"`
	EntityKey         string    `json:"entity_key"`
	Domain            string    `json:"domain"` // "light", "switch", "sensor", "binary_sensor"
	Name              string    `json:"name"`
	Icon              string    `json:"icon,omitempty"`
	UnitOfMeasurement string    `json:"unit_of_measurement,omitempty"`
	Writable          bool      `json:"writable"`
	State             string    `json:"state"`      // JSON string representation of current state
	Attributes        string    `json:"attributes"` // JSON string representation of extra attributes
	Available         bool      `json:"available"`
	AreaID            string    `json:"area_id,omitempty"`
	LastChangedAt     time.Time `json:"last_changed_at"`
	LastUpdatedAt     time.Time `json:"last_updated_at"`
}

// Validate checks the entity invariants.
func (e *Entity) Validate() error {
	if !strings.Contains(e.EntityKey, ".") {
		return ErrInvalidEntityKey
	}
	parts := strings.SplitN(e.EntityKey, ".", 2)
	if parts[0] == "" || parts[1] == "" {
		return ErrInvalidEntityKey
	}
	if e.Domain == "" {
		e.Domain = parts[0]
	}
	if strings.TrimSpace(e.Name) == "" {
		return errors.New("entity: name cannot be empty")
	}
	if e.State == "" {
		e.State = "null"
	}
	if e.Attributes == "" {
		e.Attributes = "{}"
	}
	return nil
}
