// Package energy defines models and contracts for multi-utility consumption and load shedding.
package energy

import (
	"context"
	"errors"
	"strings"
	"time"
)

var (
	ErrMeterNotFound = errors.New("utility_meter: not found")
	ErrInvalidMeter = errors.New("utility_meter: name, utility_type, and source_entity_id are required")
)

// UtilityMeter represents a physical or pulse meter tracking electricity, natural gas, or water.
type UtilityMeter struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	UtilityType    string     `json:"utility_type"` // "electricity", "gas", "water"
	Unit           string     `json:"unit"`         // "kWh", "m³", "L"
	CostPerUnit    float64    `json:"cost_per_unit"`
	SourceEntityID string     `json:"source_entity_id"`
	CurrentReading float64    `json:"current_reading"`
	LastReset      *time.Time `json:"last_reset,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// Validate checks meter structural invariants.
func (m *UtilityMeter) Validate() error {
	if strings.TrimSpace(m.Name) == "" || strings.TrimSpace(m.SourceEntityID) == "" {
		return ErrInvalidMeter
	}
	m.UtilityType = strings.ToLower(strings.TrimSpace(m.UtilityType))
	if m.UtilityType != "electricity" && m.UtilityType != "gas" && m.UtilityType != "water" {
		m.UtilityType = "electricity"
	}
	if m.Unit == "" {
		switch m.UtilityType {
		case "gas":
			m.Unit = "m³"
		case "water":
			m.Unit = "L"
		default:
			m.Unit = "kWh"
		}
	}
	return nil
}

// UtilityMeterRepository persists multi-utility meter configurations in the database.
type UtilityMeterRepository interface {
	Create(ctx context.Context, m *UtilityMeter) error
	Get(ctx context.Context, id string) (*UtilityMeter, error)
	List(ctx context.Context, utilityType string) ([]*UtilityMeter, error)
	Update(ctx context.Context, m *UtilityMeter) error
	Delete(ctx context.Context, id string) error
}
