// Package middleware provides HTTP middlewares for Domus.
package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// RequestLogger returns a Chi-compatible middleware that logs HTTP requests using slog.
func RequestLogger() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			defer func() {
				latency := time.Since(start)
				status := ww.Status()
				if status == 0 {
					status = http.StatusOK
				}

				attrs := []any{
					"method", r.Method,
					"path", r.URL.Path,
					"status", status,
					"duration_ms", latency.Milliseconds(),
					"remote_ip", r.RemoteAddr,
					"bytes_written", ww.BytesWritten(),
				}

				if reqID := middleware.GetReqID(r.Context()); reqID != "" {
					attrs = append(attrs, "request_id", reqID)
				}

				if status >= 500 {
					slog.Error("http request error", attrs...)
				} else if status >= 400 {
					slog.Warn("http request warning", attrs...)
				} else {
					slog.Info("http request", attrs...)
				}
			}()

			next.ServeHTTP(ww, r)
		})
	}
}
