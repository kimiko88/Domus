// Package area defines the Area repository interface port.
package area

import "context"

// Repository is the domain port for Area persistence.
type Repository interface {
	Create(ctx context.Context, a *Area) error
	GetByID(ctx context.Context, id string) (*Area, error)
	List(ctx context.Context) ([]*Area, error)
	Update(ctx context.Context, a *Area) error
	Delete(ctx context.Context, id string) error
}
