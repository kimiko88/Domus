// Package integration defines the Integration entity representing hardware or cloud integrations.
package integration

import (
	"errors"
	"time"
)

var (
	ErrIntegrationNotFound = errors.New("integration: not found")
)

// Integration represents an active plugin or protocol adapter instance.
type Integration struct {
	ID              string    `json:"id"`
	Type            string    `json:"type"` // "mqtt_generic", "zigbee2mqtt", "esphome_mqtt", etc.
	Name            string    `json:"name"`
	Enabled         bool      `json:"enabled"`
	ConfigEncrypted []byte    `json:"-"`
	Status          string    `json:"status"` // "connected", "disconnected", "error"
	LastError       string    `json:"last_error,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
