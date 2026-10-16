// Package middleware provides security hardening HTTP middlewares.
package middleware

import (
	"net/http"
	"strings"
)

// SecurityHeaders adds recommended security response headers to every response.
// These include CSP, X-Frame-Options, X-Content-Type-Options, Referrer-Policy,
// and Permissions-Policy.
func SecurityHeaders() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()

			// Content-Security-Policy: restrictive default, allow same-origin scripts/styles
			h.Set("Content-Security-Policy",
				"default-src 'self'; "+
					"script-src 'self' 'unsafe-inline'; "+ // Vue needs inline scripts for hydration
					"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; "+
					"font-src 'self' https://fonts.gstatic.com; "+
					"img-src 'self' data:; "+
					"connect-src 'self' ws: wss:; "+
					"frame-ancestors 'none';",
			)

			// Clickjacking protection
			h.Set("X-Frame-Options", "DENY")

			// MIME-sniffing protection
			h.Set("X-Content-Type-Options", "nosniff")

			// Referrer information leakage control
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")

			// Restrict access to browser APIs
			h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), usb=()")

			// HSTS: only set if the server is expected to use HTTPS
			// (not forced here since dev may run on plain HTTP)

			next.ServeHTTP(w, r)
		})
	}
}

// CORS configures cross-origin resource sharing for the specified allowed origins.
// It handles preflight OPTIONS requests and annotates actual responses.
// Pass an empty slice to reject all cross-origin requests.
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	originSet := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		originSet[o] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			if origin != "" {
				if _, ok := originSet[origin]; ok {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Credentials", "true")
					w.Header().Set("Vary", "Origin")
				}
			}

			// Preflight
			if r.Method == http.MethodOptions {
				if _, ok := originSet[origin]; ok {
					w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
					w.Header().Set("Access-Control-Allow-Headers",
						strings.Join([]string{
							"Content-Type",
							"Authorization",
							"X-CSRF-Token",
							"X-Requested-With",
						}, ", "))
					w.Header().Set("Access-Control-Max-Age", "86400")
					w.WriteHeader(http.StatusNoContent)
					return
				}
				w.WriteHeader(http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// MaxBodySize restricts the request body to limitBytes to mitigate DoS attacks via payload inflation.
func MaxBodySize(limitBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, limitBytes)
			}
			next.ServeHTTP(w, r)
		})
	}
}

