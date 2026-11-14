// Package middleware provides authentication, RBAC, and security middlewares.
package middleware

import (
	"crypto/subtle"
	"net/http"
	"net/url"
	"strings"
)

// CSRFProtection verifies the double-submit token for authenticated browser
// sessions. Bearer-token clients are not subject to cookie CSRF because they
// must explicitly attach their Authorization header.
func CSRFProtection(allowedOrigins ...string) func(http.Handler) http.Handler {
	originSet := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins { originSet[origin] = struct{}{} }
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet || r.Method == http.MethodHead ||
				r.Method == http.MethodOptions || r.Method == http.MethodTrace {
				next.ServeHTTP(w, r)
				return
			}
			// Origin validation also protects unauthenticated state-changing
			// endpoints (notably login) against cross-site request forgery.
			origin := r.Header.Get("Origin")
			if origin == "" {
				if referer := r.Header.Get("Referer"); referer != "" {
					if parsed, err := url.Parse(referer); err != nil || !csrfOriginAllowed(parsed.Scheme+"://"+parsed.Host, r.Host, originSet) {
						writeJSONError(w, http.StatusForbidden, "csrf_origin_rejected", "cross-origin request rejected")
						return
					}
				}
			} else if !csrfOriginAllowed(origin, r.Host, originSet) {
				writeJSONError(w, http.StatusForbidden, "csrf_origin_rejected", "cross-origin request rejected")
				return
			}
			_, cookieErr := r.Cookie(SessionCookieName)
			if cookieErr != nil || GetUser(r.Context()) == nil {
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

func csrfOriginAllowed(origin, host string, allowed map[string]struct{}) bool {
	if _, ok := allowed[origin]; ok { return true }
	parsed, err := url.Parse(origin)
	if err != nil || parsed.User != nil || parsed.Host == "" ||
		(parsed.Scheme != "http" && parsed.Scheme != "https") ||
		parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return false
	}
	return strings.EqualFold(parsed.Host, host)
}
