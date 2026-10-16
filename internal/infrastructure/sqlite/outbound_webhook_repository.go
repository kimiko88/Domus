package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/domus-hub/domus/internal/domain/webhook"
)

type OutboundWebhookRepository struct {
	db *sql.DB
}

func NewOutboundWebhookRepository(db *sql.DB) *OutboundWebhookRepository {
	return &OutboundWebhookRepository{db: db}
}

func (r *OutboundWebhookRepository) Create(ctx context.Context, wh *webhook.OutboundWebhook) error {
	eventsJSON, err := json.Marshal(wh.Events)
	if err != nil {
		eventsJSON = []byte(`["*"]`)
	}

	enabledInt := 0
	if wh.Enabled {
		enabledInt = 1
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO outbound_webhooks (
			id, name, target_url, secret_token, events_json, enabled, last_dispatched_at, last_status, fail_count, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, wh.ID, wh.Name, wh.TargetURL, wh.SecretToken, string(eventsJSON), enabledInt, wh.LastDispatchedAt, wh.LastStatus, wh.FailCount, wh.CreatedAt)
	return err
}

func (r *OutboundWebhookRepository) List(ctx context.Context) ([]webhook.OutboundWebhook, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, target_url, secret_token, events_json, enabled, last_dispatched_at, last_status, fail_count, created_at
		FROM outbound_webhooks
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []webhook.OutboundWebhook
	for rows.Next() {
		var (
			wh         webhook.OutboundWebhook
			eventsStr  string
			enabledInt int
			lastDisp   sql.NullTime
		)

		if err := rows.Scan(
			&wh.ID, &wh.Name, &wh.TargetURL, &wh.SecretToken, &eventsStr, &enabledInt,
			&lastDisp, &wh.LastStatus, &wh.FailCount, &wh.CreatedAt,
		); err != nil {
			return nil, err
		}

		wh.Enabled = enabledInt == 1
		if lastDisp.Valid {
			t := lastDisp.Time
			wh.LastDispatchedAt = &t
		}
		_ = json.Unmarshal([]byte(eventsStr), &wh.Events)
		if wh.Events == nil {
			wh.Events = []string{"*"}
		}

		list = append(list, wh)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	if list == nil {
		list = []webhook.OutboundWebhook{}
	}
	return list, nil
}

func (r *OutboundWebhookRepository) GetByID(ctx context.Context, id string) (*webhook.OutboundWebhook, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, name, target_url, secret_token, events_json, enabled, last_dispatched_at, last_status, fail_count, created_at
		FROM outbound_webhooks
		WHERE id = ?
	`, id)

	var (
		wh         webhook.OutboundWebhook
		eventsStr  string
		enabledInt int
		lastDisp   sql.NullTime
	)

	err := row.Scan(
		&wh.ID, &wh.Name, &wh.TargetURL, &wh.SecretToken, &eventsStr, &enabledInt,
		&lastDisp, &wh.LastStatus, &wh.FailCount, &wh.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, webhook.ErrOutboundNotFound
	} else if err != nil {
		return nil, err
	}

	wh.Enabled = enabledInt == 1
	if lastDisp.Valid {
		t := lastDisp.Time
		wh.LastDispatchedAt = &t
	}
	_ = json.Unmarshal([]byte(eventsStr), &wh.Events)
	if wh.Events == nil {
		wh.Events = []string{"*"}
	}

	return &wh, nil
}

func (r *OutboundWebhookRepository) Update(ctx context.Context, wh *webhook.OutboundWebhook) error {
	eventsJSON, err := json.Marshal(wh.Events)
	if err != nil {
		eventsJSON = []byte(`["*"]`)
	}

	enabledInt := 0
	if wh.Enabled {
		enabledInt = 1
	}

	res, err := r.db.ExecContext(ctx, `
		UPDATE outbound_webhooks
		SET name = ?, target_url = ?, secret_token = ?, events_json = ?, enabled = ?
		WHERE id = ?
	`, wh.Name, wh.TargetURL, wh.SecretToken, string(eventsJSON), enabledInt, wh.ID)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return webhook.ErrOutboundNotFound
	}
	return nil
}

func (r *OutboundWebhookRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, "DELETE FROM outbound_webhooks WHERE id = ?", id)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return webhook.ErrOutboundNotFound
	}
	return nil
}

func (r *OutboundWebhookRepository) RecordDispatch(ctx context.Context, id string, statusCode int, failed bool) error {
	now := time.Now().UTC()
	query := `
		UPDATE outbound_webhooks
		SET last_dispatched_at = ?, last_status = ?,
		    fail_count = CASE WHEN ? THEN fail_count + 1 ELSE 0 END
		WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, query, now, statusCode, failed, id)
	return err
}
