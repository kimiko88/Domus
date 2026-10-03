// Package device defines the Device repository interface port.
package device

import "context"

// ListFilter specifies optional filtering criteria for listing devices.
type ListFilter struct {
	IntegrationID string
	AreaID        string
	Online        *bool
}

// Repository is the domain port for Device persistence.
type Repository interface {
	Create(ctx context.Context, d *Device) error
	GetByID(ctx context.Context, id string) (*Device, error)
	GetByExternalID(ctx context.Context, integrationID, externalID string) (*Device, error)
	List(ctx context.Context, filter ListFilter) ([]*Device, error)
	Update(ctx context.Context, d *Device) error
	SetOnline(ctx context.Context, id string, online bool) error
	Delete(ctx context.Context, id string) error
}
