// Package automation defines the repository interface port for automations.
package automation

import (
	"context"
	"time"
)

// ListFilter specifies optional filtering criteria for listing automations.
type ListFilter struct {
	Enabled *bool
}

// Repository defines the persistence port for automations and run logs.
type Repository interface {
	Create(ctx context.Context, a *Automation) error
	GetByID(ctx context.Context, id string) (*Automation, error)
	List(ctx context.Context, filter ListFilter) ([]*Automation, error)
	Update(ctx context.Context, a *Automation) error
	UpdateLastRun(ctx context.Context, id string, status string) error
	Delete(ctx context.Context, id string) error

	// Run history
	CreateRun(ctx context.Context, run *AutomationRun) error
	UpdateRun(ctx context.Context, run *AutomationRun) error
	ListRuns(ctx context.Context, automationID string, limit int) ([]*AutomationRun, error)
	PurgeRunsOlderThan(ctx context.Context, cutoff time.Time) (int64, error)
}
