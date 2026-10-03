// Package device defines the physical/logical Device entity in Domus.
package device

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrDeviceNotFound    = errors.New("device: not found")
	ErrInvalidDeviceName = errors.New("device: name cannot be empty")
)

// Device represents a physical piece of hardware (e.g. Sonoff switch, Zigbee motion sensor, ESP32 board).
type Device struct {
	ID            string    `json:"id"`
	IntegrationID string    `json:"integration_id"`
	ExternalID    string    `json:"external_id"`
	Name          string    `json:"name"`
	Manufacturer  string    `json:"manufacturer,omitempty"`
	Model         string    `json:"model,omitempty"`
	AreaID        string    `json:"area_id,omitempty"`
	Online        bool      `json:"online"`
	Metadata      string    `json:"metadata,omitempty"` // JSON string
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Validate checks basic device properties.
func (d *Device) Validate() error {
	if strings.TrimSpace(d.Name) == "" {
		return ErrInvalidDeviceName
	}
	if strings.TrimSpace(d.IntegrationID) == "" {
		return errors.New("device: integration_id is required")
	}
	if strings.TrimSpace(d.ExternalID) == "" {
		return errors.New("device: external_id is required")
	}
	return nil
}
