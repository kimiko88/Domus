// Package notification defines persistent browser push subscriptions and user preferences.
package notification

import (
	"context"
	"time"
)

// WebPushSubscription represents a browser Push API subscription. Endpoint and
// authentication material are supplied by the browser and must be treated as secrets.
type WebPushSubscription struct {
	ID             string     `json:"id"`
	UserID         string     `json:"user_id"`
	Endpoint       string     `json:"endpoint"`
	P256DH         string     `json:"p256dh"`
	Auth           string     `json:"auth"`
	Locale         string     `json:"locale"`
	ExpirationTime *int64     `json:"expiration_time,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// WebPushPreferences are per-user event opt-ins. Subscription permission alone
// does not silently enable every category; defaults enable safety-critical alerts.
type WebPushPreferences struct {
	AlarmAlerts              bool `json:"alarm_alerts"`
	EmergencyAlerts          bool `json:"emergency_alerts"`
	DeviceOfflineAlerts      bool `json:"device_offline_alerts"`
	AutomationFailureAlerts  bool `json:"automation_failure_alerts"`
	CommandFailureAlerts     bool `json:"command_failure_alerts"`
	NetworkAlerts            bool `json:"network_alerts"`
	CameraAlerts             bool `json:"camera_alerts"`
	DoorbellAlerts           bool `json:"doorbell_alerts"`
	PantryAlerts             bool `json:"pantry_alerts"`
	FinanceAlerts            bool `json:"finance_alerts"`
}

func DefaultWebPushPreferences() WebPushPreferences {
	return WebPushPreferences{
		AlarmAlerts:             true,
		EmergencyAlerts:         true,
		DeviceOfflineAlerts:     true,
		AutomationFailureAlerts: true,
		CommandFailureAlerts:    false,
		NetworkAlerts:           true,
		CameraAlerts:            false,
		DoorbellAlerts:          true,
		PantryAlerts:            false,
		FinanceAlerts:           false,
	}
}

// WebPushRepository persists browser subscriptions and per-user preferences.
type WebPushRepository interface {
	RegisterWebPushSubscription(ctx context.Context, subscription *WebPushSubscription) error
	DeleteWebPushSubscription(ctx context.Context, userID, endpoint string) error
	ListWebPushSubscriptions(ctx context.Context, userID string) ([]WebPushSubscription, error)
	GetWebPushPreferences(ctx context.Context, userID string) (WebPushPreferences, error)
	SaveWebPushPreferences(ctx context.Context, userID string, preferences WebPushPreferences) error
	ListWebPushRecipients(ctx context.Context, category string) ([]WebPushSubscription, error)
}
