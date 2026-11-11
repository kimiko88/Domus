// Package middleware provides authentication, RBAC, and security middlewares.
package middleware

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strings"

	"github.com/domus-hub/domus/internal/application/auth"
	"github.com/domus-hub/domus/internal/domain/user"
)

// SessionCookieName is the name of the HTTP-only session cookie.
const SessionCookieName = "domus_session"

// CSRFCookieName is the name of the CSRF token cookie.
const CSRFCookieName = "domus_csrf"

type contextKey string

const (
	userContextKey    contextKey = "domus_user"
	sessionContextKey contextKey = "domus_session"
)

// WithUser returns a child context containing the authenticated user.
func WithUser(ctx context.Context, u *user.User) context.Context {
	return context.WithValue(ctx, userContextKey, u)
}

// GetUser retrieves the authenticated user from the context, or nil if unauthenticated.
func GetUser(ctx context.Context) *user.User {
	if val := ctx.Value(userContextKey); val != nil {
		if u, ok := val.(*user.User); ok {
			return u
		}
	}
	return nil
}

// WithSession returns a child context containing the active session.
func WithSession(ctx context.Context, s *user.Session) context.Context {
	return context.WithValue(ctx, sessionContextKey, s)
}

// GetSession retrieves the active session from the context, or nil if not present.
func GetSession(ctx context.Context) *user.Session {
	if val := ctx.Value(sessionContextKey); val != nil {
		if s, ok := val.(*user.Session); ok {
			return s
		}
	}
	return nil
}

// Authenticate extracts and validates the session cookie or Bearer token, attaching User & Session to context.
func Authenticate(authService *auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := ""
			if authHeader := r.Header.Get("Authorization"); strings.HasPrefix(authHeader, "Bearer ") {
				token = strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
			} else if cookie, err := r.Cookie(SessionCookieName); err == nil && cookie.Value != "" {
				token = cookie.Value
			}

			if token != "" {
				u, sess, err := authService.ValidateSession(r.Context(), token)
				if err == nil && u != nil && sess != nil {
					ctx := WithUser(r.Context(), u)
					ctx = WithSession(ctx, sess)
					r = r.WithContext(ctx)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAuth enforces that an authenticated user is present in the context.
func RequireAuth() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if GetUser(r.Context()) == nil {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireOperatorForWrites prevents read-only and restricted roles from invoking
// mutating domain endpoints. A small set of personal/family actions remains
// available to authenticated members with lower roles.
func RequireOperatorForWrites() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions:
				next.ServeHTTP(w, r)
				return
			}

			u := GetUser(r.Context())
			if u == nil {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
				return
			}

			path := strings.TrimSuffix(r.URL.Path, "/")
			hasSuffix := func(suffix string) bool { return path == suffix || strings.HasSuffix(path, suffix) }
			presenceSelfService := hasSuffix("/presence/gps") ||
				hasSuffix("/presence/room")
			choreCompletion := strings.Contains(path, "/chores/") && strings.HasSuffix(path, "/complete")
			shoppingMutation := hasSuffix("/shopping") || strings.Contains(path, "/shopping/")
			if presenceSelfService || choreCompletion {
				next.ServeHTTP(w, r)
				return
			}
			if shoppingMutation && u.Role != user.RoleGuest {
				next.ServeHTTP(w, r)
				return
			}
			if u.Role != user.RoleOperator && u.Role != user.RoleAdmin {
				writeJSONError(w, http.StatusForbidden, "forbidden", "operator or admin role required")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole enforces that the authenticated user possesses one of the allowed roles.
func RequireRole(allowedRoles ...user.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u := GetUser(r.Context())
			if u == nil {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
				return
			}

			for _, role := range allowedRoles {
				if u.Role == role {
					next.ServeHTTP(w, r)
					return
				}
			}

			writeJSONError(w, http.StatusForbidden, "forbidden", "insufficient permissions")
		})
	}
}

// RateLimit limits requests per client IP.
func RateLimit(limiter *auth.RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := extractIP(r)
			if !limiter.Allow(ip) {
				writeJSONError(w, http.StatusTooManyRequests, "rate_limited", "too many requests, please try again later")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// extractIP returns the verified client IP from r.RemoteAddr.
// X-Forwarded-For / X-Real-IP headers are intentionally ignored to prevent
// IP-spoofing (CVE-2026-114437). The chi RealIP middleware (when present)
// is the single authorised place to rewrite RemoteAddr from a trusted proxy.
func extractIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func writeJSONError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"code":    code,
		"message": message,
	})
}
