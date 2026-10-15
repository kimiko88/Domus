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
