package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/interfaces/http/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIdempotency_Middleware(t *testing.T) {
	store := middleware.NewIdempotencyStore(1 * time.Second)
	defer store.Close()

	var counter int64

	handler := middleware.Idempotency(store)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		val := atomic.AddInt64(&counter, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "ok",
			"count":  val,
		})
	}))

	// 1. Without Idempotency Key -> counter increments on each POST
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/commands", nil)
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	assert.Equal(t, http.StatusCreated, rec1.Code)
	assert.Equal(t, int64(1), atomic.LoadInt64(&counter))

	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/commands", nil)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusCreated, rec2.Code)
	assert.Equal(t, int64(2), atomic.LoadInt64(&counter))

	// 2. With X-Idempotency-Key -> first executes, second returns cached response
	idempotencyKey := "key-uuid-12345"

	req3 := httptest.NewRequest(http.MethodPost, "/api/v1/commands", nil)
	req3.Header.Set("X-Idempotency-Key", idempotencyKey)
	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, req3)
	assert.Equal(t, http.StatusCreated, rec3.Code)
	assert.Equal(t, int64(3), atomic.LoadInt64(&counter))

	var body3 map[string]any
	require.NoError(t, json.NewDecoder(rec3.Body).Decode(&body3))
	assert.Equal(t, float64(3), body3["count"])

	// Repeat identical request with same key
	req4 := httptest.NewRequest(http.MethodPost, "/api/v1/commands", nil)
	req4.Header.Set("X-Idempotency-Key", idempotencyKey)
	rec4 := httptest.NewRecorder()
	handler.ServeHTTP(rec4, req4)

	assert.Equal(t, http.StatusCreated, rec4.Code)
	assert.Equal(t, "HIT-IDEMPOTENT", rec4.Header().Get("X-Cache-Lookup"))
	// Counter should NOT have incremented!
	assert.Equal(t, int64(3), atomic.LoadInt64(&counter))

	var body4 map[string]any
	require.NoError(t, json.NewDecoder(rec4.Body).Decode(&body4))
	assert.Equal(t, float64(3), body4["count"])

	// 3. Different key -> executes and increments counter
	req5 := httptest.NewRequest(http.MethodPost, "/api/v1/commands", nil)
	req5.Header.Set("X-Idempotency-Key", "key-uuid-67890")
	rec5 := httptest.NewRecorder()
	handler.ServeHTTP(rec5, req5)
	assert.Equal(t, http.StatusCreated, rec5.Code)
	assert.Equal(t, int64(4), atomic.LoadInt64(&counter))

	// 4. GET requests should not be cached by idempotency
	reqGet := httptest.NewRequest(http.MethodGet, "/api/v1/entities", nil)
	reqGet.Header.Set("X-Idempotency-Key", "get-key-1")
	recGet := httptest.NewRecorder()
	handler.ServeHTTP(recGet, reqGet)
	assert.Equal(t, int64(5), atomic.LoadInt64(&counter))

	reqGet2 := httptest.NewRequest(http.MethodGet, "/api/v1/entities", nil)
	reqGet2.Header.Set("X-Idempotency-Key", "get-key-1")
	recGet2 := httptest.NewRecorder()
	handler.ServeHTTP(recGet2, reqGet2)
	assert.Equal(t, int64(6), atomic.LoadInt64(&counter))
}
