package webhook_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"net/http"
	"testing"

	"github.com/domus-hub/domus/internal/application/webhook"
	domainWebhook "github.com/domus-hub/domus/internal/domain/webhook"
	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func setupWebhookTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	_, err = db.Exec(`
		CREATE TABLE webhooks (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			webhook_id TEXT NOT NULL UNIQUE,
			secret_token TEXT,
			allowed_methods TEXT NOT NULL DEFAULT 'POST',
			enabled INTEGER NOT NULL DEFAULT 1,
			last_triggered_at TIMESTAMP,
			trigger_count INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	require.NoError(t, err)
	return db
}

func TestWebhookService_Lifecycle(t *testing.T) {
	db := setupWebhookTestDB(t)
	defer db.Close()

	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 64, Workers: 2})
	svc := webhook.NewService(db, bus)
	ctx := context.Background()

	// 1. Create Webhook
	wh, err := svc.CreateWebhook(ctx, "IFTTT Consegne Pacchi", "my-secret-key", "POST")
	require.NoError(t, err)
	assert.NotEmpty(t, wh.WebhookID)
	assert.Equal(t, "IFTTT Consegne Pacchi", wh.Name)

	// Listen for WebhookReceived on bus
	receivedCh := make(chan map[string]any, 10)
	bus.Subscribe(events.WebhookReceived, func(c context.Context, e events.Event) error {
		if payload, ok := e.Payload.(map[string]any); ok {
			receivedCh <- payload
		}
		return nil
	})

	// 2. Execute with invalid signature should fail
	body := []byte(`{"parcel_status": "delivered", "box": 1}`)
	headers := http.Header{}
	headers.Set("X-Domus-Signature", "invalid-signature")

	_, err = svc.ExecuteWebhook(ctx, wh.WebhookID, "POST", headers, nil, body)
	assert.ErrorIs(t, err, domainWebhook.ErrInvalidSignature)

	// 3. Execute with valid HMAC signature
	mac := hmac.New(sha256.New, []byte("my-secret-key"))
	mac.Write(body)
	validSig := hex.EncodeToString(mac.Sum(nil))
	headers.Set("X-Domus-Signature", validSig)

	executed, err := svc.ExecuteWebhook(ctx, wh.WebhookID, "POST", headers, nil, body)
	require.NoError(t, err)
	assert.Equal(t, 1, executed.TriggerCount)

	ev := <-receivedCh
	assert.Equal(t, wh.WebhookID, ev["webhook_id"])
	assert.Contains(t, ev["body"], "delivered")

	// 4. List and Delete
	list, err := svc.ListWebhooks(ctx)
	require.NoError(t, err)
	assert.Len(t, list, 1)

	err = svc.DeleteWebhook(ctx, wh.ID)
	require.NoError(t, err)

	list, err = svc.ListWebhooks(ctx)
	require.NoError(t, err)
	assert.Empty(t, list)
}
