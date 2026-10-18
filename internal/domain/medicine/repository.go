package medicine

import (
	"context"
)

// Filter specifies query constraints when retrieving medicine records.
type Filter struct {
	Location     Location
	Search       string
	ExpiringDays *int  // Items expiring within N days
	IsExpired    *bool // True to get only expired items
	ColdChain    *bool // Filter by requires cold chain
	LowStock     *bool // Filter by quantity <= min_threshold
}

// Repository defines persistence operations for medicine inventory.
type Repository interface {
	Create(ctx context.Context, item *Medicine) error
	GetByID(ctx context.Context, id string) (*Medicine, error)
	Update(ctx context.Context, item *Medicine) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filter Filter) ([]*Medicine, error)
}
