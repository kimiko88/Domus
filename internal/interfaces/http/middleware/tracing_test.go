package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/interfaces/http/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTraceContextMiddleware_GeneratesNewTrace(t *testing.T) {
	var ctxTrace *events.TraceContext

	handler := middleware.TraceContext()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxTrace = events.TraceFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	require.NotNil(t, ctxTrace)
	assert.Len(t, ctxTrace.TraceID, 32)
	assert.Len(t, ctxTrace.SpanID, 16)

	// Check response headers
	respTraceParent := rec.Header().Get(middleware.TraceparentHeader)
	assert.NotEmpty(t, respTraceParent)
	assert.True(t, strings.HasPrefix(respTraceParent, "00-"+ctxTrace.TraceID))

	respXTraceID := rec.Header().Get(middleware.XTraceIDHeader)
	assert.Equal(t, ctxTrace.TraceID, respXTraceID)
}

func TestTraceContextMiddleware_PropagatesIncomingTrace(t *testing.T) {
	incomingTraceParent := "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
	var ctxTrace *events.TraceContext

	handler := middleware.TraceContext()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxTrace = events.TraceFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(middleware.TraceparentHeader, incomingTraceParent)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	require.NotNil(t, ctxTrace)
	// Must preserve incoming TraceID
	assert.Equal(t, "4bf92f3577b34da6a3ce929d0e0e4736", ctxTrace.TraceID)
	// Must create a new child SpanID
	assert.NotEqual(t, "00f067aa0ba902b7", ctxTrace.SpanID)
	assert.Equal(t, "00f067aa0ba902b7", ctxTrace.ParentSpanID)

	// Response header must carry child traceparent
	assert.Equal(t, ctxTrace.String(), rec.Header().Get(middleware.TraceparentHeader))
	assert.Equal(t, "4bf92f3577b34da6a3ce929d0e0e4736", rec.Header().Get(middleware.XTraceIDHeader))
}
