// Package middleware provides authentication, RBAC, and security middlewares.
package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// CSRFProtection verifies the double-submit token for authenticated browser
// sessions. Bearer-token clients are not subject to cookie CSRF because they
// must explicitly attach their Authorization header.
func CSRFProtection() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet || r.Method == http.MethodHead ||
				r.Method == http.MethodOptions || r.Method == http.MethodTrace {
				next.ServeHTTP(w, r)
				return
			}
			_, hasSessionCookie := r.Cookie(SessionCookieName)
			if !hasSessionCookie || GetUser(r.Context()) == nil {
				next.ServeHTTP(w, r)
				return
			}
			cookie, err := r.Cookie(CSRFCookieName)
			header := strings.TrimSpace(r.Header.Get("X-CSRF-Token"))
			if err != nil || len(cookie.Value) != 32 || len(header) != len(cookie.Value) ||
				subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(header)) != 1 {
				writeJSONError(w, http.StatusForbidden, "csrf_validation_failed", "CSRF token missing or invalid")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
