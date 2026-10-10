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
	`)
	require.NoError(t, err)

	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 64, Workers: 2})
	svc := webhook.NewService(db, bus)
	return handlers.NewWebhookHandler(svc), db
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
