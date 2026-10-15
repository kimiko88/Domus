// Package sqlite provides SQLite repository implementation for device push tokens.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/domus-hub/domus/internal/domain/notification"
	"github.com/google/uuid"
)

// PushTokenRepository persists device push tokens in SQLite.
type PushTokenRepository struct {
	db *sql.DB
}

// NewPushTokenRepository creates a new PushTokenRepository.
func NewPushTokenRepository(db *sql.DB) *PushTokenRepository {
	return &PushTokenRepository{db: db}
}

// RegisterToken inserts or updates a device push token based on unique device_token.
func (r *PushTokenRepository) RegisterToken(ctx context.Context, t *notification.DevicePushToken) error {
	if t.ID == "" {
		t.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now
	}
	t.UpdatedAt = now

	query := `
		INSERT INTO device_push_tokens (
			id, user_id, platform, device_token, device_name, app_version, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(device_token) DO UPDATE SET
			user_id = excluded.user_id,
			platform = excluded.platform,
			device_name = excluded.device_name,
			app_version = excluded.app_version,
			updated_at = excluded.updated_at
	`
	_, err := r.db.ExecContext(ctx, query,
		t.ID,
		t.UserID,
		string(t.Platform),
		t.DeviceToken,
		t.DeviceName,
		t.AppVersion,
		t.CreatedAt.Format(time.RFC3339),
		t.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("failed to register push token: %w", err)
	}
	return nil
}

// RevokeToken / DeleteToken deletes a device token from SQLite.
func (r *PushTokenRepository) RevokeToken(ctx context.Context, token string) error {
	return r.DeleteToken(ctx, token)
}

// DeleteToken removes a device push token by token string.
func (r *PushTokenRepository) DeleteToken(ctx context.Context, token string) error {
	query := `DELETE FROM device_push_tokens WHERE device_token = ?`
	_, err := r.db.ExecContext(ctx, query, token)
	if err != nil {
		return fmt.Errorf("failed to delete push token: %w", err)
	}
	return nil
}

// GetTokensByUser returns all push tokens associated with a given user ID.
func (r *PushTokenRepository) GetTokensByUser(ctx context.Context, userID string) ([]notification.DevicePushToken, error) {
	query := `
		SELECT id, user_id, platform, device_token, device_name, app_version, created_at, updated_at
		FROM device_push_tokens
		WHERE user_id = ?
		ORDER BY updated_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query tokens by user: %w", err)
	}
	defer rows.Close()

	return scanPushTokens(rows)
}

// GetAllTokens returns all registered push tokens in the system.
func (r *PushTokenRepository) GetAllTokens(ctx context.Context) ([]notification.DevicePushToken, error) {
	query := `
		SELECT id, user_id, platform, device_token, device_name, app_version, created_at, updated_at
		FROM device_push_tokens
		ORDER BY updated_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query all tokens: %w", err)
	}
	defer rows.Close()

	return scanPushTokens(rows)
}

func scanPushTokens(rows *sql.Rows) ([]notification.DevicePushToken, error) {
	var tokens []notification.DevicePushToken
	for rows.Next() {
		var (
			t            notification.DevicePushToken
			platform     string
			createdAtStr string
			updatedAtStr string
		)
		err := rows.Scan(
			&t.ID,
			&t.UserID,
			&platform,
			&t.DeviceToken,
			&t.DeviceName,
			&t.AppVersion,
			&createdAtStr,
			&updatedAtStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan push token: %w", err)
		}
		t.Platform = notification.DevicePlatform(platform)
		if parsed, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			t.CreatedAt = parsed
		}
		if parsed, err := time.Parse(time.RFC3339, updatedAtStr); err == nil {
			t.UpdatedAt = parsed
		}
		tokens = append(tokens, t)
	}
	return tokens, nil
}
