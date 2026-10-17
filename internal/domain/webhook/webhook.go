// Package webhook defines domain models for inbound external HTTP webhooks.
package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

var (
	ErrWebhookNotFound  = errors.New("webhook: endpoint not found")
	ErrWebhookDisabled  = errors.New("webhook: endpoint is disabled")
	ErrMethodNotAllowed = errors.New("webhook: HTTP method not allowed")
	ErrInvalidSignature = errors.New("webhook: HMAC signature mismatch")
)

type Webhook struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	WebhookID       string     `json:"webhook_id"`
	SecretToken     string     `json:"secret_token,omitempty"`
	AllowedMethods  string     `json:"allowed_methods"` // "POST", "POST,GET"
	Enabled         bool       `json:"enabled"`
	LastTriggeredAt *time.Time `json:"last_triggered_at,omitempty"`
	TriggerCount    int        `json:"trigger_count"`
	CreatedAt       time.Time  `json:"created_at"`
}

// VerifySignature validates an incoming HMAC-SHA256 signature if SecretToken is defined.
func (w *Webhook) VerifySignature(body []byte, signatureHeader string) bool {
	if strings.TrimSpace(w.SecretToken) == "" {
		return true // No signature required
	}

	mac := hmac.New(sha256.New, []byte(w.SecretToken))
	mac.Write(body)
	expectedMAC := hex.EncodeToString(mac.Sum(nil))

	cleanHeader := strings.TrimPrefix(strings.TrimSpace(signatureHeader), "sha256=")
	return hmac.Equal([]byte(cleanHeader), []byte(expectedMAC))
}

// IsMethodAllowed checks if the HTTP method is acceptable.
func (w *Webhook) IsMethodAllowed(method string) bool {
	if w.AllowedMethods == "" {
		return method == "POST"
	}
	methods := strings.Split(strings.ToUpper(w.AllowedMethods), ",")
	for _, m := range methods {
		if strings.TrimSpace(m) == strings.ToUpper(method) {
			return true
		}
	}
	return false
}
