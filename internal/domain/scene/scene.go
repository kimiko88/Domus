// Package scene defines the scene domain model and validation rules.
package scene

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrSceneNotFound   = errors.New("scene: not found")
	ErrInvalidName     = errors.New("scene: name cannot be empty")
	ErrMissingActions  = errors.New("scene: at least one action is required")
)

// Action defines an action executed when a scene is activated.
type Action struct {
	EntityKey string         `json:"entity_key"`
	Service   string         `json:"service"` // e.g. "turn_on", "turn_off"
	Params    map[string]any `json:"params,omitempty"`
}

// Scene represents a predefined group state of multiple entities.
type Scene struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Icon      string    `json:"icon,omitempty"`
	Actions   []Action  `json:"actions"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Validate checks scene invariants.
func (s *Scene) Validate() error {
	if strings.TrimSpace(s.Name) == "" {
		return ErrInvalidName
	}
	if len(s.Actions) == 0 {
		return ErrMissingActions
	}
	return nil
}
