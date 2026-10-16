package webhook_test

import (
	"context"
	"database/sql"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	appwebhook "github.com/domus-hub/domus/internal/application/webhook"
	"github.com/domus-hub/domus/internal/domain/events"
	domainWebhook "github.com/domus-hub/domus/internal/domain/webhook"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func setupOutboundTestDB(t *testing.T) (*sqlite.OutboundWebhookRepository, *sql.DB) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	_, err = db.Exec(`
		CREATE TABLE outbound_webhooks (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			target_url TEXT NOT NULL,
			secret_token TEXT,
			events_json TEXT NOT NULL DEFAULT '["*"]',
			enabled INTEGER NOT NULL DEFAULT 1,
			last_dispatched_at TIMESTAMP,
			last_status INTEGER NOT NULL DEFAULT 0,
			fail_count INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	require.NoError(t, err)

	return sqlite.NewOutboundWebhookRepository(db), db
}

func TestOutboundDispatcher_LifecycleAndTest(t *testing.T) {
	repo, db := setupOutboundTestDB(t)
	defer db.Close()

	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 64, Workers: 2})
	dispatcher := appwebhook.NewOutboundDispatcher(repo, bus, nil)
	dispatcher.Start(context.Background())
	defer dispatcher.Stop()

	// Mock receiver server
	var receivedCount int32
	var lastReceivedSig string
	var lastReceivedEvent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&receivedCount, 1)
		lastReceivedSig = r.Header.Get("X-Domus-Signature")
		lastReceivedEvent = r.Header.Get("X-Domus-Event")
		body, _ := io.ReadAll(r.Body)
		assert.NotEmpty(t, body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// 1. Create Outbound Webhook
	wh, err := dispatcher.CreateOutbound(context.Background(), "Home Assistant Relay", server.URL, "super-secret-key", []string{"alarm.*", "entity.state_changed"})
	require.NoError(t, err)
	assert.NotEmpty(t, wh.ID)
	assert.Equal(t, "Home Assistant Relay", wh.Name)
	assert.Equal(t, server.URL, wh.TargetURL)

	// 2. List Outbound Webhooks
	list, err := dispatcher.ListOutbound(context.Background())
	require.NoError(t, err)
	assert.Len(t, list, 1)

	// 3. Test Outbound Webhook ping
	err = dispatcher.TestOutbound(context.Background(), wh.ID)
	require.NoError(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&receivedCount))
	assert.Contains(t, lastReceivedSig, "sha256=")
	assert.Equal(t, "webhook.test", lastReceivedEvent)

	// 4. Dispatch matching domain event via bus
	err = bus.Publish(context.Background(), events.Event{
		ID:         "evt_alarm_1",
		Type:       events.AlarmTriggered,
		OccurredAt: time.Now().UTC(),
		Payload:    map[string]any{"zone": "hallway"},
	})
	require.NoError(t, err)

	// Allow worker to dispatch
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, int32(2), atomic.LoadInt32(&receivedCount))
	assert.Equal(t, string(events.AlarmTriggered), lastReceivedEvent)

	// 5. Dispatch non-matching event (should not be sent)
	err = bus.Publish(context.Background(), events.Event{
		ID:         "evt_camera_1",
		Type:       events.CameraDetected,
		OccurredAt: time.Now().UTC(),
	})
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, int32(2), atomic.LoadInt32(&receivedCount))

	// 6. Delete Outbound Webhook
	err = dispatcher.DeleteOutbound(context.Background(), wh.ID)
	require.NoError(t, err)

	listAfter, err := dispatcher.ListOutbound(context.Background())
	require.NoError(t, err)
	assert.Empty(t, listAfter)
}

func TestOutboundDispatcher_InvalidURL(t *testing.T) {
	repo, db := setupOutboundTestDB(t)
	defer db.Close()

	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 64, Workers: 2})
	dispatcher := appwebhook.NewOutboundDispatcher(repo, bus, nil)

	_, err := dispatcher.CreateOutbound(context.Background(), "Invalid", "not-a-valid-url", "", []string{"*"})
	assert.ErrorIs(t, err, domainWebhook.ErrInvalidTargetURL)
}

func TestOutboundDispatcher_SignatureCalculation(t *testing.T) {
	wh := &domainWebhook.OutboundWebhook{
		SecretToken: "test-token",
	}
	payload := []byte(`{"hello":"world"}`)
	sig := wh.ComputeSignature(payload)
	assert.True(t, len(sig) > 10)
	assert.Contains(t, sig, "sha256=")

	// Empty secret token produces empty signature
	emptyWh := &domainWebhook.OutboundWebhook{}
	assert.Empty(t, emptyWh.ComputeSignature(payload))
}
