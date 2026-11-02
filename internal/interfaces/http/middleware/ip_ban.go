// Package middleware provides HTTP middlewares for security and IP rate limiting.
package middleware

import (
	"encoding/json"
	"net"
	"net/http"

	domauth "github.com/domus-hub/domus/internal/domain/auth"
)

// IPBanMiddleware blocks requests originating from banned IP addresses.
func IPBanMiddleware(bannedRepo domauth.BannedIPRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if bannedRepo == nil {
				next.ServeHTTP(w, r)
				return
			}

			clientIP := ExtractClientIP(r)
			bannedRecord, err := bannedRepo.Get(r.Context(), clientIP)
			if err == nil && bannedRecord != nil && bannedRecord.IsBanned() {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"error":        "IP_BANNED",
					"message":      "Il tuo indirizzo IP è stato temporaneamente bloccato per tentativi multipli falliti",
					"banned_until": bannedRecord.BannedUntil,
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ExtractClientIP extracts the verified client IP address from the request.
// It relies exclusively on r.RemoteAddr, which represents the authenticated
// TCP peer set by the Go HTTP server and (when applicable) rewritten by a
// trusted proxy middleware. Raw X-Forwarded-For / X-Real-IP headers are
// intentionally ignored here to prevent IP-spoofing via header injection
// (CVE-2026-114437 / go-chi/chi ClientIPFromXFF bypass).
func ExtractClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
