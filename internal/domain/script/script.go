// Package script defines domain models and contracts for reusable action sequences.
package script

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/domus-hub/domus/internal/domain/automation"
)

var (
	ErrScriptNotFound       = errors.New("script: not found")
	ErrInvalidScriptName    = errors.New("script: name cannot be empty")
	ErrMissingSequence      = errors.New("script: at least one action is required in sequence")
	ErrScriptAlreadyRunning  = errors.New("script: script is already running (single mode)")
	ErrMaxRunsExceeded      = errors.New("script: maximum concurrent runs exceeded")
)

const (
	ModeSingle   = "single"
	ModeRestart  = "restart"
	ModeQueued   = "queued"
	ModeParallel = "parallel"
)

// FieldDef defines a typed input parameter schema for the script.
type FieldDef struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required"`
	Default     any    `json:"default,omitempty"`
	Type        string `json:"type,omitempty"` // "string", "number", "boolean"
}

// Script represents a reusable macro sequence of actions.
type Script struct {
	ID          string                       `json:"id"`
	Name        string                       `json:"name"`
	Description string                       `json:"description,omitempty"`
	Icon        string                       `json:"icon,omitempty"`
	Mode        string                       `json:"mode"`     // "single", "restart", "queued", "parallel"
	MaxRuns     int                          `json:"max_runs"` // max concurrent executions (for queued/parallel)
	Fields      map[string]FieldDef          `json:"fields,omitempty"`
	Sequence    []automation.Action          `json:"sequence"`
	CreatedAt   time.Time                    `json:"created_at"`
	UpdatedAt   time.Time                    `json:"updated_at"`
}

// Validate checks script invariants.
func (s *Script) Validate() error {
	if strings.TrimSpace(s.Name) == "" {
		return ErrInvalidScriptName
	}
	if len(s.Sequence) == 0 {
		return ErrMissingSequence
	}
	if s.Mode == "" {
		s.Mode = ModeSingle
	}
	if s.MaxRuns <= 0 {
		s.MaxRuns = 10
	}
	if s.Icon == "" {
		s.Icon = "script"
	}
	return nil
}

// Repository persists and retrieves scripts.
type Repository interface {
	Create(ctx context.Context, s *Script) error
	Get(ctx context.Context, id string) (*Script, error)
	List(ctx context.Context) ([]*Script, error)
	Update(ctx context.Context, s *Script) error
	Delete(ctx context.Context, id string) error
}
