// Package entity defines the Entity repository interface port.
package entity

import (
	"context"
	"time"
)

// ListFilter specifies filtering criteria for querying entities.
type ListFilter struct {
	Domain   string
	AreaID   string
	DeviceID string
	Writable *bool
}

// Repository is the domain port for Entity persistence.
type Repository interface {
	Create(ctx context.Context, e *Entity) error
	GetByID(ctx context.Context, id string) (*Entity, error)
	GetByKey(ctx context.Context, key string) (*Entity, error)
	List(ctx context.Context, filter ListFilter) ([]*Entity, error)
	Update(ctx context.Context, e *Entity) error
	UpdateState(ctx context.Context, id string, stateJSON, attributesJSON string, available bool, at time.Time) error
	Delete(ctx context.Context, id string) error
}
