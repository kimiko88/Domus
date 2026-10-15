// Package logbook defines domain models for the user-facing chronological activity log.
package logbook

import (
	"context"
	"time"
)

// Entry represents a human-readable event occurrence in the smart home timeline.
type Entry struct {
	ID          string    `json:"id"`
	EntityID    string    `json:"entity_id"`
	EntityName  string    `json:"entity_name"`
	Domain      string    `json:"domain"`
	State       string    `json:"state"`
	Message     string    `json:"message"`
	ContextType string    `json:"context_type"` // "user", "automation", "script", "system"
	ContextID   string    `json:"context_id,omitempty"`
	ContextName string    `json:"context_name,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// Filter specifies query criteria for logbook entries.
type Filter struct {
	EntityID string
	Domain   string
	From     time.Time
	To       time.Time
	Limit    int
}

// Repository persists and queries logbook entries.
type Repository interface {
	Save(ctx context.Context, entry *Entry) error
	List(ctx context.Context, filter Filter) ([]*Entry, error)
	PurgeOlderThan(ctx context.Context, cutoff time.Time) (int64, error)
}
