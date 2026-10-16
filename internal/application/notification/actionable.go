package notification

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/google/uuid"
)

type NotificationAction struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Destructive bool   `json:"destructive,omitempty"`
}

type ActionableNotification struct {
	ID         string               `json:"id"`
	Title      string               `json:"title"`
	Message    string               `json:"message"`
	ImageURL   string               `json:"image_url,omitempty"`
	Critical   bool                 `json:"critical"` // Overrides Do Not Disturb (APNs critical / Android MAX importance)
	Volume     float64              `json:"volume"`   // 0.0 to 1.0 sound level
	Actions    []NotificationAction `json:"actions"`
	CreatedAt  time.Time            `json:"created_at"`
	ResolvedAt *time.Time           `json:"resolved_at,omitempty"`
}

type MobilePushPayload struct {
	APNsPayload map[string]any `json:"apns"`
	FCMPayload  map[string]any `json:"fcm"`
}

type Manager struct {
	mu            sync.RWMutex
	notifications map[string]*ActionableNotification
	bus           events.Bus
}

func NewManager(bus events.Bus) *Manager {
	return &Manager{
		notifications: make(map[string]*ActionableNotification),
		bus:           bus,
	}
}

func (m *Manager) ListNotifications(ctx context.Context) []*ActionableNotification {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]*ActionableNotification, 0, len(m.notifications))
	for _, n := range m.notifications {
		res = append(res, n)
	}
	return res
}

func (m *Manager) CreateNotification(ctx context.Context, notif *ActionableNotification) (*MobilePushPayload, error) {
	if notif.ID == "" {
		notif.ID = "notif_" + uuid.NewString()[:8]
	}
	notif.CreatedAt = time.Now().UTC()

	m.mu.Lock()
	m.notifications[notif.ID] = notif
	m.mu.Unlock()

	// Format Apple Push Notification Service (APNs) payload
	aps := map[string]any{
		"alert": map[string]string{
			"title": notif.Title,
			"body":  notif.Message,
		},
		"category": "ACTIONABLE_" + notif.ID,
	}
	if notif.Critical {
		soundLevel := notif.Volume
		if soundLevel <= 0 {
			soundLevel = 1.0
		}
		aps["sound"] = map[string]any{
			"critical": 1,
			"name":     "alarm.caf",
			"volume":   soundLevel,
		}
	} else {
		aps["sound"] = "default"
	}

	apns := map[string]any{
		"aps":      aps,
		"image":    notif.ImageURL,
		"actions":  notif.Actions,
		"notif_id": notif.ID,
	}

	// Format Firebase Cloud Messaging (FCM) Android payload
	fcmPriority := "normal"
	if notif.Critical {
		fcmPriority = "high"
	}
	fcm := map[string]any{
		"priority": fcmPriority,
		"data": map[string]any{
			"id":       notif.ID,
			"title":    notif.Title,
			"body":     notif.Message,
			"image":    notif.ImageURL,
			"critical": notif.Critical,
			"channel":  "emergency_alerts",
			"actions":  notif.Actions,
		},
	}

	// Emit NotificationCreated event
	if m.bus != nil {
		_ = m.bus.Publish(ctx, events.Event{
			ID:         uuid.NewString(),
			Type:       events.NotificationCreated,
			Payload:    notif,
			OccurredAt: time.Now().UTC(),
		})
	}

	return &MobilePushPayload{
		APNsPayload: apns,
		FCMPayload:  fcm,
	}, nil
}

type ActionResult struct {
	NotificationID string    `json:"notification_id"`
	ActionID       string    `json:"action_id"`
	User           string    `json:"user"`
	HandledAt      time.Time `json:"handled_at"`
}

func (m *Manager) HandleAction(ctx context.Context, notifID, actionID, user string) (*ActionResult, error) {
	m.mu.Lock()
	notif, ok := m.notifications[notifID]
	if !ok {
		m.mu.Unlock()
		return nil, fmt.Errorf("notification not found: %s", notifID)
	}
	now := time.Now().UTC()
	notif.ResolvedAt = &now
	m.mu.Unlock()

	result := &ActionResult{
		NotificationID: notifID,
		ActionID:       actionID,
		User:           user,
		HandledAt:      now,
	}

	if m.bus != nil {
		_ = m.bus.Publish(ctx, events.Event{
			ID:         uuid.NewString(),
			Type:       events.EventType("notification.action_received"),
			Payload:    result,
			OccurredAt: now,
			Source:     user,
		})
	}

	return result, nil
}
