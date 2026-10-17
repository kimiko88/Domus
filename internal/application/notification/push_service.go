// Package notification provides application logic for push notifications and actionable alerts.
package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/domain/notification"
	"github.com/google/uuid"
)

// PushSender transports push payloads to APNs or FCM endpoints.
type PushSender interface {
	SendAPNs(ctx context.Context, token string, payload map[string]any) error
	SendFCM(ctx context.Context, token string, payload map[string]any) error
}

// PushDispatchReport summarizes the result of a push broadcast or user dispatch.
type PushDispatchReport struct {
	MessageID    string   `json:"message_id"`
	TotalTokens  int      `json:"total_tokens"`
	Successes    int      `json:"successes"`
	Failures     int      `json:"failures"`
	FailedTokens []string `json:"failed_tokens,omitempty"`
}

// ErrInvalidToken indicates the provider rejected the device token as unregistered/expired.
var ErrInvalidToken = errors.New("device token is invalid or expired")

// PushService coordinates push token lifecycle and native APNs/FCM delivery.
type PushService struct {
	repo   notification.PushTokenRepository
	sender PushSender
	bus    events.Bus
	logger *slog.Logger
}

// NewPushService creates a new PushService.
func NewPushService(repo notification.PushTokenRepository, sender PushSender, bus events.Bus, logger *slog.Logger) *PushService {
	if logger == nil {
		logger = slog.Default()
	}
	if sender == nil {
		sender = NewInMemoryPushSender()
	}
	return &PushService{
		repo:   repo,
		sender: sender,
		bus:    bus,
		logger: logger,
	}
}

// RegisterToken records or updates an active companion device token.
func (s *PushService) RegisterToken(ctx context.Context, t *notification.DevicePushToken) error {
	if t.DeviceToken == "" {
		return errors.New("device_token cannot be empty")
	}
	if t.UserID == "" {
		return errors.New("user_id cannot be empty")
	}
	if t.Platform != notification.PlatformIOS && t.Platform != notification.PlatformAndroid {
		return fmt.Errorf("invalid platform: %s (must be ios or android)", t.Platform)
	}
	return s.repo.RegisterToken(ctx, t)
}

// RevokeToken removes a companion device push token.
func (s *PushService) RevokeToken(ctx context.Context, token string) error {
	if token == "" {
		return errors.New("token cannot be empty")
	}
	return s.repo.RevokeToken(ctx, token)
}

// GetTokensByUser retrieves active tokens registered for a specific user.
func (s *PushService) GetTokensByUser(ctx context.Context, userID string) ([]notification.DevicePushToken, error) {
	return s.repo.GetTokensByUser(ctx, userID)
}

// GetAllTokens retrieves all registered push tokens.
func (s *PushService) GetAllTokens(ctx context.Context) ([]notification.DevicePushToken, error) {
	return s.repo.GetAllTokens(ctx)
}

// Send formats payloads and delivers the push notification to recipient companion devices.
func (s *PushService) Send(ctx context.Context, msg notification.PushMessage) (*PushDispatchReport, error) {
	if msg.ID == "" {
		msg.ID = "msg_" + uuid.NewString()[:8]
	}

	var (
		tokens []notification.DevicePushToken
		err    error
	)

	if msg.UserID != "" {
		tokens, err = s.repo.GetTokensByUser(ctx, msg.UserID)
	} else {
		tokens, err = s.repo.GetAllTokens(ctx)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch push tokens: %w", err)
	}

	report := &PushDispatchReport{
		MessageID:   msg.ID,
		TotalTokens: len(tokens),
	}

	for _, t := range tokens {
		switch t.Platform {
		case notification.PlatformIOS:
			apnsPayload := BuildAPNsPayload(msg)
			if err := s.sender.SendAPNs(ctx, t.DeviceToken, apnsPayload); err != nil {
				s.logger.WarnContext(ctx, "failed to dispatch APNs push",
					"token", t.DeviceToken, "error", err)
				report.Failures++
				report.FailedTokens = append(report.FailedTokens, t.DeviceToken)
				if errors.Is(err, ErrInvalidToken) {
					_ = s.repo.DeleteToken(ctx, t.DeviceToken)
				}
			} else {
				report.Successes++
			}

		case notification.PlatformAndroid:
			fcmPayload := BuildFCMPayload(msg)
			if err := s.sender.SendFCM(ctx, t.DeviceToken, fcmPayload); err != nil {
				s.logger.WarnContext(ctx, "failed to dispatch FCM push",
					"token", t.DeviceToken, "error", err)
				report.Failures++
				report.FailedTokens = append(report.FailedTokens, t.DeviceToken)
				if errors.Is(err, ErrInvalidToken) {
					_ = s.repo.DeleteToken(ctx, t.DeviceToken)
				}
			} else {
				report.Successes++
			}
		}
	}

	// Publish push dispatched event to EventBus
	if s.bus != nil {
		_ = s.bus.Publish(ctx, events.Event{
			ID:         uuid.NewString(),
			Type:       events.EventType("notification.push_dispatched"),
			Payload:    msg,
			OccurredAt: time.Now().UTC(),
		})
	}

	return report, nil
}

// BuildAPNsPayload constructs Apple Push Notification dictionary with critical alert support.
func BuildAPNsPayload(msg notification.PushMessage) map[string]any {
	aps := map[string]any{
		"alert": map[string]string{
			"title": msg.Title,
			"body":  msg.Body,
		},
	}

	if msg.Category != "" {
		aps["category"] = msg.Category
	}

	if msg.Critical {
		vol := msg.Volume
		if vol <= 0 {
			vol = 1.0
		}
		aps["sound"] = map[string]any{
			"critical": 1,
			"name":     "alarm.caf",
			"volume":   vol,
		}
		aps["interruption-level"] = "critical"
	} else {
		aps["sound"] = "default"
		aps["interruption-level"] = "active"
	}

	payload := map[string]any{
		"aps":     aps,
		"msg_id":  msg.ID,
		"user_id": msg.UserID,
	}

	if len(msg.Data) > 0 {
		for k, v := range msg.Data {
			payload[k] = v
		}
	}

	return payload
}

// BuildFCMPayload constructs Firebase Cloud Messaging v1 payload with high-priority support.
func BuildFCMPayload(msg notification.PushMessage) map[string]any {
	priority := "normal"
	channelID := "default_notifications"
	if msg.Critical {
		priority = "high"
		channelID = "emergency_alerts"
	}

	dataMap := map[string]any{
		"id":       msg.ID,
		"title":    msg.Title,
		"body":     msg.Body,
		"critical": msg.Critical,
		"channel":  channelID,
	}

	if len(msg.Data) > 0 {
		for k, v := range msg.Data {
			dataMap[k] = v
		}
	}

	return map[string]any{
		"priority": priority,
		"data":     dataMap,
	}
}

// InMemoryPushSender is a thread-safe in-memory recorder for push deliveries.
type InMemoryPushSender struct {
	mu           sync.RWMutex
	Dispatches   []map[string]any
	FailOnTokens map[string]error
}

// NewInMemoryPushSender creates a new InMemoryPushSender.
func NewInMemoryPushSender() *InMemoryPushSender {
	return &InMemoryPushSender{
		FailOnTokens: make(map[string]error),
	}
}

func (s *InMemoryPushSender) SendAPNs(ctx context.Context, token string, payload map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err, ok := s.FailOnTokens[token]; ok {
		return err
	}
	s.Dispatches = append(s.Dispatches, map[string]any{
		"platform": "ios",
		"token":    token,
		"payload":  payload,
	})
	return nil
}

func (s *InMemoryPushSender) SendFCM(ctx context.Context, token string, payload map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err, ok := s.FailOnTokens[token]; ok {
		return err
	}
	s.Dispatches = append(s.Dispatches, map[string]any{
		"platform": "android",
		"token":    token,
		"payload":  payload,
	})
	return nil
}

// HTTPPushSender implements PushSender over live APNs and FCM HTTP endpoints.
type HTTPPushSender struct {
	APNsBaseURL string
	FCMBaseURL  string
	HTTPClient  *http.Client
}

// NewHTTPPushSender creates an HTTPPushSender configured for remote push delivery.
func NewHTTPPushSender(apnsURL, fcmURL string) *HTTPPushSender {
	if apnsURL == "" {
		apnsURL = "https://api.push.apple.com/3/device"
	}
	if fcmURL == "" {
		fcmURL = "https://fcm.googleapis.com/fcm/send"
	}
	return &HTTPPushSender{
		APNsBaseURL: apnsURL,
		FCMBaseURL:  fcmURL,
		HTTPClient:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (h *HTTPPushSender) SendAPNs(ctx context.Context, token string, payload map[string]any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/%s", h.APNsBaseURL, token)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusGone || resp.StatusCode == http.StatusNotFound {
		return ErrInvalidToken
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("APNs error: status code %d", resp.StatusCode)
	}
	return nil
}

func (h *HTTPPushSender) SendFCM(ctx context.Context, token string, payload map[string]any) error {
	payload["to"] = token
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.FCMBaseURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return ErrInvalidToken
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("FCM error: status code %d", resp.StatusCode)
	}
	return nil
}
