// Package integration defines the Integration repository interface port.
package integration

import "context"

// Repository is the domain port for Integration persistence.
type Repository interface {
	Create(ctx context.Context, in *Integration) error
	GetByID(ctx context.Context, id string) (*Integration, error)
	List(ctx context.Context) ([]*Integration, error)
	Update(ctx context.Context, in *Integration) error
	SetStatus(ctx context.Context, id string, status, lastError string) error
	Delete(ctx context.Context, id string) error
}
