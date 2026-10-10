package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/domus-hub/domus/internal/application/assistant"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/domus-hub/domus/internal/interfaces/http/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssistantHandler_Chat(t *testing.T) {
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 64, Workers: 2})
	cfg := assistant.Config{Enabled: false}
	svc := assistant.NewService(cfg, bus, nil)
	h := handlers.NewAssistantHandler(svc)

	r := chi.NewRouter()
	h.RegisterRoutes(r)

	// 1. POST /assistant/chat
	body, _ := json.Marshal(map[string]string{
		"prompt": "accendi le luci in cucina",
	})
	req := httptest.NewRequest(http.MethodPost, "/assistant/chat", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp assistant.ChatResponse
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp.ResponseText, "cucina")
	assert.Equal(t, "local_heuristic", resp.EngineUsed)

	// 2. GET /assistant/config
	req = httptest.NewRequest(http.MethodGet, "/assistant/config", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}
