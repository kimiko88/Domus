// Package dashboard defines domain models and storage ports for customizable widget dashboards and kiosk mode.
package dashboard

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrDashboardNotFound = errors.New("dashboard: not found")
	ErrInvalidTitle      = errors.New("dashboard: title cannot be empty")
)

// KioskConfig configures wall-mounted tablet behavior.
type KioskConfig struct {
	Enabled               bool `json:"enabled"`
	ScreensaverTimeoutSec int  `json:"screensaver_timeout_sec"` // e.g. 120s
	WakeOnPresence        bool `json:"wake_on_presence"`         // wake display when radar/BLE detects presence
	ShowClock             bool `json:"show_clock"`
	ShowWeather           bool `json:"show_weather"`
}

// Widget defines a modular block in the grid dashboard.
type Widget struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"` // "thermostat", "sparkline", "sankey", "entity_card", "light_dimmer", "camera_stream"
	Title     string         `json:"title"`
	EntityKey string         `json:"entity_key,omitempty"`
	ColSpan   int            `json:"col_span"` // 1, 2, 3, 4
	RowSpan   int            `json:"row_span"` // 1, 2, 3
	Config    map[string]any `json:"config,omitempty"`
}

// Dashboard represents a customizable UI layout.
type Dashboard struct {
	ID              string      `json:"id"`
	Title           string      `json:"title"`
	Icon            string      `json:"icon"`
	IsDefault       bool        `json:"is_default"`
	SortOrder       int         `json:"sort_order"`
	LayoutJSON      string      `json:"layout_json"`
	KioskConfigJSON string      `json:"kiosk_config_json"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
}

// DefaultKioskConfig returns default wall tablet parameters.
func DefaultKioskConfig() KioskConfig {
	return KioskConfig{
		Enabled:               false,
		ScreensaverTimeoutSec: 120,
		WakeOnPresence:        true,
		ShowClock:             true,
		ShowWeather:           true,
	}
}

// SetKioskConfig serializes and sets the kiosk config JSON.
func (d *Dashboard) SetKioskConfig(cfg KioskConfig) {
	b, _ := json.Marshal(cfg)
	d.KioskConfigJSON = string(b)
}

// GetKioskConfig deserializes the kiosk configuration.
func (d *Dashboard) GetKioskConfig() KioskConfig {
	var cfg KioskConfig
	if d.KioskConfigJSON == "" {
		return DefaultKioskConfig()
	}
	_ = json.Unmarshal([]byte(d.KioskConfigJSON), &cfg)
	return cfg
}

// Repository defines CRUD operations for dashboards.
type Repository interface {
	Create(ctx context.Context, d *Dashboard) error
	GetByID(ctx context.Context, id string) (*Dashboard, error)
	List(ctx context.Context) ([]*Dashboard, error)
	Update(ctx context.Context, d *Dashboard) error
	Delete(ctx context.Context, id string) error
	SetDefault(ctx context.Context, id string) error
}
