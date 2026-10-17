// Package dashboard defines domain models and storage ports for customizable widget dashboards and kiosk mode.
package dashboard

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrDashboardNotFound = errors.New("dashboard: not found")
	ErrInvalidTitle      = errors.New("dashboard: title cannot be empty")
	ErrInvalidShareToken = errors.New("dashboard: invalid or expired share token")
)

// KioskConfig configures wall-mounted tablet behavior.
type KioskConfig struct {
	Enabled               bool `json:"enabled"`
	ScreensaverTimeoutSec int  `json:"screensaver_timeout_sec"` // e.g. 120s
	WakeOnPresence        bool `json:"wake_on_presence"`        // wake display when radar/BLE detects presence
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
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	Icon            string    `json:"icon"`
	IsDefault       bool      `json:"is_default"`
	SortOrder       int       `json:"sort_order"`
	LayoutJSON      string    `json:"layout_json"`
	KioskConfigJSON string    `json:"kiosk_config_json"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
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

// ShareTokenPayload defines the claims carried by a temporary guest share token.
type ShareTokenPayload struct {
	DashboardID string    `json:"dashboard_id"`
	Title       string    `json:"title"`
	ReadOnly    bool      `json:"read_only"`
	ExpiresAt   time.Time `json:"expires_at"`
	CreatedAt   time.Time `json:"created_at"`
}

// GenerateShareToken creates a signed, tamper-proof URL-safe share token.
func GenerateShareToken(p ShareTokenPayload, secretKey []byte) (string, error) {
	if len(secretKey) == 0 {
		return "", errors.New("empty secret key")
	}
	payloadBytes, err := json.Marshal(p)
	if err != nil {
		return "", fmt.Errorf("marshal share payload: %w", err)
	}

	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadBytes)

	mac := hmac.New(sha256.New, secretKey)
	mac.Write([]byte(payloadB64))
	sigHex := hex.EncodeToString(mac.Sum(nil))

	return payloadB64 + "." + sigHex, nil
}

// ValidateShareToken verifies the HMAC signature and expiration of a share token.
func ValidateShareToken(token string, secretKey []byte) (*ShareTokenPayload, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return nil, ErrInvalidShareToken
	}

	payloadB64, sigHex := parts[0], parts[1]

	mac := hmac.New(sha256.New, secretKey)
	mac.Write([]byte(payloadB64))
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(sigHex), []byte(expectedSig)) {
		return nil, ErrInvalidShareToken
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return nil, ErrInvalidShareToken
	}

	var p ShareTokenPayload
	if err := json.Unmarshal(payloadBytes, &p); err != nil {
		return nil, ErrInvalidShareToken
	}

	if time.Now().UTC().After(p.ExpiresAt) {
		return nil, ErrInvalidShareToken
	}

	return &p, nil
}
