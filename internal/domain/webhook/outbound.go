// Package webhook defines domain models for inbound and outbound HTTP webhooks.
package webhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

var (
	ErrOutboundNotFound = errors.New("webhook: outbound endpoint not found")
	ErrInvalidTargetURL = errors.New("webhook: invalid target url")
)

// OutboundWebhook defines an external webhook endpoint triggered by internal domain events.
type OutboundWebhook struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	TargetURL        string     `json:"target_url"`
	SecretToken      string     `json:"secret_token,omitempty"`
	Events           []string   `json:"events"` // e.g. ["*"], ["alarm.*"], ["entity.state_changed"]
	Enabled          bool       `json:"enabled"`
	LastDispatchedAt *time.Time `json:"last_dispatched_at,omitempty"`
	LastStatus       int        `json:"last_status"`
	FailCount        int        `json:"fail_count"`
	CreatedAt        time.Time  `json:"created_at"`
}

// ComputeSignature calculates the HMAC-SHA256 signature for outgoing payloads.
func (o *OutboundWebhook) ComputeSignature(payload []byte) string {
	if strings.TrimSpace(o.SecretToken) == "" {
		return ""
	}
	mac := hmac.New(sha256.New, []byte(o.SecretToken))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// MatchesEvent reports whether this webhook is subscribed to the specified event type.
func (o *OutboundWebhook) MatchesEvent(eventType string) bool {
	if !o.Enabled {
		return false
	}
	for _, pattern := range o.Events {
		pattern = strings.TrimSpace(pattern)
		if pattern == "*" || pattern == "" {
			return true
		}
		if strings.HasSuffix(pattern, ".*") {
			prefix := strings.TrimSuffix(pattern, ".*")
			if strings.HasPrefix(eventType, prefix+".") {
				return true
			}
		}
		if strings.EqualFold(pattern, eventType) {
			return true
		}
	}
	return false
}

// OutboundRepository defines persistence operations for outbound webhooks.
type OutboundRepository interface {
	Create(ctx context.Context, wh *OutboundWebhook) error
	List(ctx context.Context) ([]OutboundWebhook, error)
	GetByID(ctx context.Context, id string) (*OutboundWebhook, error)
	Update(ctx context.Context, wh *OutboundWebhook) error
	Delete(ctx context.Context, id string) error
	RecordDispatch(ctx context.Context, id string, statusCode int, failed bool) error
}
