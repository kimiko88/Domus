// Package history defines the StateHistory domain model and repository port.
package history

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrInvalidEntityID is returned when an entity ID is missing.
	ErrInvalidEntityID = errors.New("history: entity_id is required")
)

// StateHistory records an immutable snapshot of an entity's state at a point in time.
type StateHistory struct {
	ID         int64     `json:"id,omitempty"`
	EntityID   string    `json:"entity_id"`
	State      string    `json:"state"` // JSON string of state value or object
	RecordedAt time.Time `json:"recorded_at"`
}

// Validate checks that the StateHistory record meets domain invariants.
func (h *StateHistory) Validate() error {
	if h.EntityID == "" {
		return ErrInvalidEntityID
	}
	if h.RecordedAt.IsZero() {
		h.RecordedAt = time.Now().UTC()
	}
	return nil
}

// HistoryFilter encapsulates query criteria for querying historical states.
type HistoryFilter struct {
	EntityID string
	From     time.Time
	To       time.Time
	Limit    int
}

// Repository defines persistence operations for entity state history.
type Repository interface {
	SaveBatch(ctx context.Context, entries []*StateHistory) error
	ListByEntity(ctx context.Context, filter HistoryFilter) ([]*StateHistory, error)
	PurgeOlderThan(ctx context.Context, cutoff time.Time) (int64, error)
}
