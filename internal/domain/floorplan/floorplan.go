// Package floorplan defines the domain models and repository interface for floor plans and entity markers.
package floorplan

import (
	"context"
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidFloorName = errors.New("floorplan: name cannot be empty and must be under 100 characters")
	ErrFloorNotFound    = errors.New("floorplan: floor not found")
	ErrMarkerNotFound   = errors.New("floorplan: marker not found")
	ErrInvalidMarkerPos = errors.New("floorplan: coordinates must be between 0.0 and 100.0 percent")
	ErrInvalidEntityKey = errors.New("floorplan: entity key cannot be empty")
)

// Floor represents a architectural level/floor of the home.
type Floor struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Level     int       `json:"level"` // e.g. -1 for basement, 0 for ground floor, 1 for first floor
	PlanSVG   string    `json:"plan_svg,omitempty"`
	PlanImage string    `json:"plan_image,omitempty"`
	SortOrder int       `json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Validate checks the floor properties.
func (f *Floor) Validate() error {
	trimmed := strings.TrimSpace(f.Name)
	if len(trimmed) == 0 || len(trimmed) > 100 {
		return ErrInvalidFloorName
	}
	return nil
}

// Marker represents the 2D visual position of an entity on a floor plan.
type Marker struct {
	ID           string    `json:"id"`
	FloorID      string    `json:"floor_id"`
	EntityKey    string    `json:"entity_key"`
	XPercent     float64   `json:"x_percent"` // 0.0 to 100.0%
	YPercent     float64   `json:"y_percent"` // 0.0 to 100.0%
	IconOverride string    `json:"icon_override,omitempty"`
	ShowBadge    bool      `json:"show_badge"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Validate checks marker position and properties.
func (m *Marker) Validate() error {
	if strings.TrimSpace(m.FloorID) == "" {
		return errors.New("floorplan: floor_id is required")
	}
	if strings.TrimSpace(m.EntityKey) == "" {
		return ErrInvalidEntityKey
	}
	if m.XPercent < 0.0 || m.XPercent > 100.0 || m.YPercent < 0.0 || m.YPercent > 100.0 {
		return ErrInvalidMarkerPos
	}
	return nil
}

// Repository defines storage operations for floors and markers.
type Repository interface {
	CreateFloor(ctx context.Context, floor *Floor) error
	GetFloor(ctx context.Context, id string) (*Floor, error)
	ListFloors(ctx context.Context) ([]*Floor, error)
	UpdateFloor(ctx context.Context, floor *Floor) error
	DeleteFloor(ctx context.Context, id string) error

	ListMarkers(ctx context.Context, floorID string) ([]*Marker, error)
	SaveMarker(ctx context.Context, marker *Marker) error
	DeleteMarker(ctx context.Context, id string) error
}
