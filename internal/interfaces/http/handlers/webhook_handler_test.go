package handlers_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/domus-hub/domus/internal/application/webhook"
	domainWebhook "github.com/domus-hub/domus/internal/domain/webhook"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	"github.com/domus-hub/domus/internal/interfaces/http/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func setupWebhookHandlerTest(t *testing.T) (*handlers.WebhookHandler, *sql.DB) {
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

	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 64, Workers: 2})
	svc := webhook.NewService(db, bus)
	obRepo := sqlite.NewOutboundWebhookRepository(db)
	dispatcher := webhook.NewOutboundDispatcher(obRepo, bus, nil)
	return handlers.NewWebhookHandler(svc).WithOutbound(dispatcher), db
}

func TestWebhookHandler_Routes(t *testing.T) {
	h, db := setupWebhookHandlerTest(t)
	defer db.Close()

	r := chi.NewRouter()
	h.RegisterRoutes(r)

	// 1. Create a webhook via POST /webhook-configs
	createBody, _ := json.Marshal(map[string]string{
		"name":            "SmartThings Relay",
		"secret_token":    "",
		"allowed_methods": "POST,GET",
	})
	req := httptest.NewRequest(http.MethodPost, "/webhook-configs", bytes.NewReader(createBody))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusCreated, rr.Code)

	var created domainWebhook.Webhook
	err := json.NewDecoder(rr.Body).Decode(&created)
	require.NoError(t, err)
	assert.NotEmpty(t, created.WebhookID)

	// 2. Trigger public webhook via POST /webhooks/{webhook_id}
	trigReq := httptest.NewRequest(http.MethodPost, "/webhooks/"+created.WebhookID, bytes.NewReader([]byte(`{"temp": 24.5}`)))
	trigRr := httptest.NewRecorder()
	r.ServeHTTP(trigRr, trigReq)
	assert.Equal(t, http.StatusOK, trigRr.Code)

	// 3. List webhooks
	listReq := httptest.NewRequest(http.MethodGet, "/webhook-configs", nil)
	listRr := httptest.NewRecorder()
	r.ServeHTTP(listRr, listReq)
	assert.Equal(t, http.StatusOK, listRr.Code)

	var list []domainWebhook.Webhook
	err = json.NewDecoder(listRr.Body).Decode(&list)
	require.NoError(t, err)
	assert.Len(t, list, 1)
	assert.Equal(t, 1, list[0].TriggerCount)

	// 4. Delete webhook
	delReq := httptest.NewRequest(http.MethodDelete, "/webhook-configs/"+created.ID, nil)
	delRr := httptest.NewRecorder()
	r.ServeHTTP(delRr, delReq)
	assert.Equal(t, http.StatusNoContent, delRr.Code)
}

func TestWebhookHandler_OutboundRoutes(t *testing.T) {
	h, db := setupWebhookHandlerTest(t)
	defer db.Close()

	r := chi.NewRouter()
	h.RegisterRoutes(r)

	// Mock receiver for test
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer receiver.Close()

	// 1. Create Outbound Webhook via POST /webhook-configs/outbound
	createBody, _ := json.Marshal(map[string]any{
		"name":         "Pushbullet Relay",
		"target_url":   receiver.URL,
		"secret_token": "secret123",
		"events":       []string{"alarm.*"},
	})
	req := httptest.NewRequest(http.MethodPost, "/webhook-configs/outbound", bytes.NewReader(createBody))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusCreated, rr.Code)

	var created domainWebhook.OutboundWebhook
	err := json.NewDecoder(rr.Body).Decode(&created)
	require.NoError(t, err)
	assert.NotEmpty(t, created.ID)
	assert.Equal(t, "Pushbullet Relay", created.Name)

	// 2. List Outbound Webhooks via GET /webhook-configs/outbound
	listReq := httptest.NewRequest(http.MethodGet, "/webhook-configs/outbound", nil)
	listRr := httptest.NewRecorder()
	r.ServeHTTP(listRr, listReq)
	assert.Equal(t, http.StatusOK, listRr.Code)

	var list []domainWebhook.OutboundWebhook
	err = json.NewDecoder(listRr.Body).Decode(&list)
	require.NoError(t, err)
	assert.Len(t, list, 1)

	// 3. Test Outbound Webhook via POST /webhook-configs/outbound/{id}/test
	testReq := httptest.NewRequest(http.MethodPost, "/webhook-configs/outbound/"+created.ID+"/test", nil)
	testRr := httptest.NewRecorder()
	r.ServeHTTP(testRr, testReq)
	assert.Equal(t, http.StatusOK, testRr.Code)

	// 4. Delete Outbound Webhook via DELETE /webhook-configs/outbound/{id}
	delReq := httptest.NewRequest(http.MethodDelete, "/webhook-configs/outbound/"+created.ID, nil)
	delRr := httptest.NewRecorder()
	r.ServeHTTP(delRr, delReq)
	assert.Equal(t, http.StatusNoContent, delRr.Code)
}
