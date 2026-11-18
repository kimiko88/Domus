package notification

import (
	"context"
	"time"
)

type DevicePlatform string

const (
	PlatformIOS     DevicePlatform = "ios"
	PlatformAndroid DevicePlatform = "android"
)

// DevicePushToken stores APNs or FCM push registration tokens for companion apps
type DevicePushToken struct {
	ID          string         `json:"id"`
	UserID      string         `json:"user_id"`
	Platform    DevicePlatform `json:"platform"`
	DeviceToken string         `json:"device_token"`
	DeviceName  string         `json:"device_name"`
	AppVersion  string         `json:"app_version"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// PushMessage represents an outgoing push notification
type PushMessage struct {
	ID       string         `json:"id,omitempty"`
	UserID   string         `json:"user_id,omitempty"` // empty means broadcast to all users
	Title    string         `json:"title"`
	Body     string         `json:"body"`
	Data     map[string]any `json:"data,omitempty"`
	Critical bool           `json:"critical,omitempty"` // APNs critical alert / FCM high priority
	Volume   float64        `json:"volume,omitempty"`   // 0.0 to 1.0 for critical alert volume
	Category string         `json:"category,omitempty"`
}

// PushTokenRepository defines persistence contract for companion push tokens
type PushTokenRepository interface {
	RegisterToken(ctx context.Context, token *DevicePushToken) error
	RevokeToken(ctx context.Context, token string) error
	GetTokensByUser(ctx context.Context, userID string) ([]DevicePushToken, error)
	GetAllTokens(ctx context.Context) ([]DevicePushToken, error)
	DeleteToken(ctx context.Context, token string) error
}


// WebPushSubscription stores a browser Push API subscription. The endpoint and
// encryption keys are private user data and must never be returned to the client
// after registration.
type WebPushSubscription struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	Endpoint       string    `json:"endpoint"`
	P256DH         string    `json:"p256dh"`
	Auth           string    `json:"auth"`
	Locale         string    `json:"locale"`
	ExpirationTime *int64    `json:"expiration_time,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// WebPushPreferences are per-user controls. Expiry windows are expressed in
// calendar days; zero disables the corresponding expiry reminder.
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
	TaskAlerts               bool `json:"task_alerts"`
	MedicineExpiryAlerts     bool `json:"medicine_expiry_alerts"`
	MedicineLowStockAlerts   bool `json:"medicine_low_stock_alerts"`
	MedicineExpiryDays       int  `json:"medicine_expiry_days"`
	FoodExpiryAlerts         bool `json:"food_expiry_alerts"`
	FoodLowStockAlerts       bool `json:"food_low_stock_alerts"`
	FoodExpiryDays           int  `json:"food_expiry_days"`
}

// DefaultWebPushPreferences enables household reminders after explicit browser
// subscription consent. Other high-volume categories remain individually opted in.
func DefaultWebPushPreferences() WebPushPreferences {
	return WebPushPreferences{
		AlarmAlerts:             true,
		EmergencyAlerts:         true,
		DeviceOfflineAlerts:     true,
		AutomationFailureAlerts: true,
		NetworkAlerts:           true,
		DoorbellAlerts:          true,
		TaskAlerts:              true,
		MedicineExpiryAlerts:    true,
		MedicineLowStockAlerts:  true,
		MedicineExpiryDays:      7,
		FoodExpiryAlerts:        true,
		FoodLowStockAlerts:      true,
		FoodExpiryDays:          3,
	}
}

// WebPushRepository persists browser subscriptions, alert preferences, and
// idempotency records used by the scheduled reminder dispatcher.
type WebPushRepository interface {
	RegisterWebPushSubscription(ctx context.Context, subscription *WebPushSubscription) error
	DeleteWebPushSubscription(ctx context.Context, userID, endpoint string) error
	ListWebPushSubscriptions(ctx context.Context, userID string) ([]WebPushSubscription, error)
	GetWebPushPreferences(ctx context.Context, userID string) (WebPushPreferences, error)
	SaveWebPushPreferences(ctx context.Context, userID string, preferences WebPushPreferences) error
	ListWebPushRecipients(ctx context.Context, category string) ([]WebPushSubscription, error)
	ClaimWebPushDelivery(ctx context.Context, userID, dedupeKey string) (bool, error)
	ReleaseWebPushDelivery(ctx context.Context, userID, dedupeKey string) error
}
