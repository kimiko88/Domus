// Package scene defines the repository interface port for scenes.
package scene

import "context"

// Repository defines the persistence port for scenes.
type Repository interface {
	Create(ctx context.Context, s *Scene) error
	GetByID(ctx context.Context, id string) (*Scene, error)
	List(ctx context.Context) ([]*Scene, error)
	Update(ctx context.Context, s *Scene) error
	Delete(ctx context.Context, id string) error
}
