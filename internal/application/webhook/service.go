// Package webhook provides application services for managing and processing inbound webhooks.
package webhook

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/domain/webhook"
	"github.com/google/uuid"
)

type Service struct {
	db  *sql.DB
	bus events.Bus
	mu  sync.RWMutex
}

func NewService(db *sql.DB, bus events.Bus) *Service {
	return &Service{
		db:  db,
		bus: bus,
	}
}

// CreateWebhook registers a new inbound webhook endpoint.
func (s *Service) CreateWebhook(ctx context.Context, name, secretToken, allowedMethods string) (*webhook.Webhook, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return nil, fmt.Errorf("webhook: name cannot be empty")
	}

	whID := "wh_" + strings.ReplaceAll(uuid.NewString()[:8], "-", "")
	if allowedMethods == "" {
		allowedMethods = "POST"
	}

	now := time.Now().UTC()
	wh := &webhook.Webhook{
		ID:             uuid.NewString(),
		Name:           trimmedName,
		WebhookID:      whID,
		SecretToken:    strings.TrimSpace(secretToken),
		AllowedMethods: allowedMethods,
		Enabled:        true,
		TriggerCount:   0,
		CreatedAt:      now,
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO webhooks (
			id, name, webhook_id, secret_token, allowed_methods, enabled, trigger_count, created_at
		) VALUES (?, ?, ?, ?, ?, 1, 0, ?)
	`, wh.ID, wh.Name, wh.WebhookID, wh.SecretToken, wh.AllowedMethods, wh.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("webhook: failed to insert: %w", err)
	}

	return wh, nil
}

// ListWebhooks returns all configured webhook endpoints.
func (s *Service) ListWebhooks(ctx context.Context) ([]webhook.Webhook, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, webhook_id, secret_token, allowed_methods, enabled, last_triggered_at, trigger_count, created_at
		FROM webhooks
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []webhook.Webhook
	for rows.Next() {
		var wh webhook.Webhook
		var lastTrig sql.NullTime
		var secToken sql.NullString

		if err := rows.Scan(
			&wh.ID, &wh.Name, &wh.WebhookID, &secToken, &wh.AllowedMethods, &wh.Enabled,
			&lastTrig, &wh.TriggerCount, &wh.CreatedAt,
		); err != nil {
			return nil, err
		}

		if secToken.Valid {
			wh.SecretToken = secToken.String
		}
		if lastTrig.Valid {
			t := lastTrig.Time
			wh.LastTriggeredAt = &t
		}
		list = append(list, wh)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if list == nil {
		list = []webhook.Webhook{}
	}
	return list, nil
}

// DeleteWebhook removes a webhook endpoint.
func (s *Service) DeleteWebhook(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	res, err := s.db.ExecContext(ctx, "DELETE FROM webhooks WHERE id = ? OR webhook_id = ?", id, id)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return webhook.ErrWebhookNotFound
	}
	return nil
}

// ExecuteWebhook processes an incoming HTTP webhook request and fires domain events.
func (s *Service) ExecuteWebhook(ctx context.Context, webhookID, method string, headers http.Header, query map[string]string, body []byte) (*webhook.Webhook, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	row := s.db.QueryRowContext(ctx, `
		SELECT id, name, webhook_id, secret_token, allowed_methods, enabled, last_triggered_at, trigger_count, created_at
		FROM webhooks
		WHERE webhook_id = ?
	`, webhookID)

	var wh webhook.Webhook
	var secToken sql.NullString
	var lastTrig sql.NullTime

	err := row.Scan(
		&wh.ID, &wh.Name, &wh.WebhookID, &secToken, &wh.AllowedMethods, &wh.Enabled,
		&lastTrig, &wh.TriggerCount, &wh.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, webhook.ErrWebhookNotFound
	} else if err != nil {
		return nil, err
	}

	if secToken.Valid {
		wh.SecretToken = secToken.String
	}
	if lastTrig.Valid {
		t := lastTrig.Time
		wh.LastTriggeredAt = &t
	}

	if !wh.Enabled {
		return nil, webhook.ErrWebhookDisabled
	}

	if !wh.IsMethodAllowed(method) {
		return nil, webhook.ErrMethodNotAllowed
	}

	// Validate signature if secret token is present
	sigHeader := headers.Get("X-Domus-Signature")
	if sigHeader == "" {
		sigHeader = headers.Get("X-Hub-Signature-256")
	}
	if wh.SecretToken != "" {
		// check query token as alternative
		if tokenQuery, ok := query["token"]; ok && tokenQuery == wh.SecretToken {
			// Query token matches
		} else if !wh.VerifySignature(body, sigHeader) {
			return nil, webhook.ErrInvalidSignature
		}
	}

	// Increment metrics
	now := time.Now().UTC()
	wh.TriggerCount++
	wh.LastTriggeredAt = &now

	_, _ = s.db.ExecContext(ctx, `
		UPDATE webhooks SET trigger_count = ?, last_triggered_at = ? WHERE id = ?
	`, wh.TriggerCount, now, wh.ID)

	// Publish domain event
	if s.bus != nil {
		_ = s.bus.Publish(ctx, events.Event{
			ID:         uuid.NewString(),
			Type:       events.WebhookReceived,
			OccurredAt: now,
			Source:     "webhook:" + wh.WebhookID,
			Payload: map[string]any{
				"webhook_id": wh.WebhookID,
				"name":       wh.Name,
				"method":     method,
				"headers":    headers,
				"query":      query,
				"body":       string(body),
				"timestamp":  now.Format(time.RFC3339),
			},
		})
	}

	return &wh, nil
}
