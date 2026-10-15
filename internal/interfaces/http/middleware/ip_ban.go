// Package middleware provides HTTP middlewares for security and IP rate limiting.
package middleware

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"

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

// ExtractClientIP extracts the real client IP address considering proxy headers.
func ExtractClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		ip := strings.TrimSpace(parts[0])
		if ip != "" {
			return ip
		}
	}
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return strings.TrimSpace(xrip)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
