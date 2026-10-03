// Package handlers_test provides unit tests for the HTTP handlers (TDD red-to-green phase).
package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/interfaces/http/handlers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockPinger struct {
	err error
}

func (m *mockPinger) Ping() error {
	return m.err
}

func TestHealthHandler_ReturnsOK(t *testing.T) {
	startedAt := time.Now().Add(-10 * time.Second)
	pinger := &mockPinger{err: nil}

	handler := handlers.NewHealthHandler("1.0.0", startedAt, pinger)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var body map[string]any
	err := json.Unmarshal(rr.Body.Bytes(), &body)
	require.NoError(t, err)

	assert.Equal(t, "ok", body["status"])
	assert.Equal(t, "1.0.0", body["version"])
	assert.GreaterOrEqual(t, body["uptime_s"].(float64), float64(10))
	assert.Equal(t, "healthy", body["database"])
}
