package middleware

import (
	"net/http"

	"github.com/domus-hub/domus/internal/domain/events"
)

// TraceparentHeader is the canonical W3C traceparent HTTP header name.
const TraceparentHeader = "traceparent"

// XTraceIDHeader is the correlated trace ID header for clients and logs.
const XTraceIDHeader = "X-Trace-ID"

// TraceContext returns a middleware that extracts or generates a W3C traceparent
// and propagates the trace context through HTTP requests, responses, and the request context.
func TraceContext() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rawHeader := r.Header.Get(TraceparentHeader)

			var currentTrace *events.TraceContext
			if rawHeader != "" {
				parentTrace, err := events.ParseTraceParent(rawHeader)
				if err == nil {
					// Derive child span for current HTTP handler execution
					currentTrace = events.NewChildSpan(parentTrace)
				}
			}

			if currentTrace == nil {
				// No valid incoming traceparent; start a new root trace
				currentTrace = events.NewTraceContext()
			}

			// Propagate trace to context
			ctx := events.ContextWithTrace(r.Context(), currentTrace)

			// Set response headers for client-side correlation
			w.Header().Set(TraceparentHeader, currentTrace.String())
			w.Header().Set(XTraceIDHeader, currentTrace.TraceID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
