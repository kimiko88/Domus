// Package area defines the Area domain entity representing physical spaces (rooms, floors).
package area

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidAreaName = errors.New("area: name cannot be empty and must be under 100 characters")
	ErrAreaNotFound    = errors.New("area: not found")
)

// Area represents a logical or physical room or zone in the home.
type Area struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Icon         string    `json:"icon,omitempty"`
	ParentAreaID string    `json:"parent_area_id,omitempty"`
	SortOrder    int       `json:"sort_order"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Validate checks the area properties.
func (a *Area) Validate() error {
	trimmed := strings.TrimSpace(a.Name)
	if len(trimmed) == 0 || len(trimmed) > 100 {
		return ErrInvalidAreaName
	}
	return nil
}
