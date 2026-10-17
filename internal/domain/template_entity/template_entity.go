// Package template_entity defines the domain models and repository interface for computed template entities.
package template_entity

import (
	"context"
	"errors"
	"strings"
	"time"
)

var (
	ErrNotFound        = errors.New("template_entity: not found")
	ErrInvalidID       = errors.New("template_entity: id cannot be empty")
	ErrInvalidEntityID = errors.New("template_entity: entity_id must follow domain.name format")
	ErrInvalidTemplate = errors.New("template_entity: state_template cannot be empty")
)

// TemplateEntity represents a dynamically calculated virtual entity whose state is derived
// from expressions referencing other entity states.
type TemplateEntity struct {
	ID                 string    `json:"id"`
	EntityID           string    `json:"entity_id"` // e.g. "sensor.potenza_totale"
	Domain             string    `json:"domain"`    // e.g. "sensor", "binary_sensor", "switch"
	Name               string    `json:"name"`
	StateTemplate      string    `json:"state_template"`
	AttributesTemplate string    `json:"attributes_template"`
	UnitOfMeasurement  string    `json:"unit_of_measurement,omitempty"`
	DeviceClass        string    `json:"device_class,omitempty"`
	Icon               string    `json:"icon,omitempty"`
	SourceEntityIDs    []string  `json:"source_entity_ids"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// Validate checks entity structural invariants.
func (e *TemplateEntity) Validate() error {
	if strings.TrimSpace(e.ID) == "" {
		return ErrInvalidID
	}
	if !strings.Contains(e.EntityID, ".") {
		return ErrInvalidEntityID
	}
	if strings.TrimSpace(e.StateTemplate) == "" {
		return ErrInvalidTemplate
	}
	if e.Domain == "" {
		parts := strings.Split(e.EntityID, ".")
		e.Domain = parts[0]
	}
	return nil
}

// Repository persists template entity configurations.
type Repository interface {
	Create(ctx context.Context, e *TemplateEntity) error
	GetByID(ctx context.Context, id string) (*TemplateEntity, error)
	GetByEntityID(ctx context.Context, entityID string) (*TemplateEntity, error)
	List(ctx context.Context) ([]*TemplateEntity, error)
	Update(ctx context.Context, e *TemplateEntity) error
	Delete(ctx context.Context, id string) error
}
