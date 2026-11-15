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
			deviceCommand := strings.Contains(path, "/entities/") && strings.HasSuffix(path, "/command")
			serviceCall := strings.Contains(path, "/services/")
			sceneExecution := strings.Contains(path, "/scenes/") && strings.HasSuffix(path, "/apply")
			mediaControl := strings.Contains(path, "/media/players/") &&
				(strings.HasSuffix(path, "/volume") || strings.HasSuffix(path, "/state")) ||
				strings.HasSuffix(path, "/media/broadcast")
			cameraSession := strings.Contains(path, "/cameras/") && strings.HasSuffix(path, "/webrtc") ||
				strings.Contains(path, "/intercom/") && strings.HasSuffix(path, "/audio")
			automationMutation := strings.Contains(path, "/automations/")
				|| strings.HasSuffix(path, "/automations")
			if deviceCommand || serviceCall || sceneExecution || mediaControl {
				permissions := u.EffectivePermissions()
				if permissions != nil && permissions.CanControlDevices {
					next.ServeHTTP(w, r)
					return
				}
				writeJSONError(w, http.StatusForbidden, "forbidden", "device control permission required")
				return
			}
			if cameraSession {
				permissions := u.EffectivePermissions()
				if permissions != nil && permissions.CanViewCameras {
					next.ServeHTTP(w, r)
					return
				}
				writeJSONError(w, http.StatusForbidden, "forbidden", "camera viewing permission required")
				return
			}
			if automationMutation {
				permissions := u.EffectivePermissions()
				if permissions != nil && permissions.CanManageAutomations {
					next.ServeHTTP(w, r)
					return
				}
				writeJSONError(w, http.StatusForbidden, "forbidden", "automation management permission required")
				return
			}
			presenceSelfService := hasSuffix("/presence/gps") ||
				hasSuffix("/presence/room")
			choreCompletion := strings.Contains(path, "/chores/") && strings.HasSuffix(path, "/complete")
			shoppingMutation := hasSuffix("/shopping") || strings.Contains(path, "/shopping/")
			if presenceSelfService && u.Role != user.RoleGuest {
				next.ServeHTTP(w, r)
				return
			}
			if (choreCompletion || shoppingMutation) && u.Role == user.RoleKid {
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

// RequireViewAccess maps API resources to their corresponding UI section and applies the member allow-list.
func RequireViewAccess() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u := GetUser(r.Context())
			if u == nil || u.Role == user.RoleAdmin {
				next.ServeHTTP(w, r)
				return
			}
			permissions := u.EffectivePermissions()
			if permissions == nil {
				writeJSONError(w, http.StatusForbidden, "forbidden", "view access denied")
				return
			}
			path := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1"), "/")
			required := ""
			if path == "/presence/gps" || path == "/presence/room" {
				next.ServeHTTP(w, r)
				return
			}
			switch {
			case strings.HasPrefix(path, "/family"):
				required = "/family"
			case strings.HasPrefix(path, "/dashboards"):
				required = "/custom-dashboards"
			case strings.HasPrefix(path, "/automations"):
				required = "/automations"
			case strings.HasPrefix(path, "/scenes"):
				required = "/scenes"
			case strings.HasPrefix(path, "/areas"):
				required = "/areas"
			case strings.HasPrefix(path, "/energy"):
				required = "/energy"
			case strings.HasPrefix(path, "/climate"):
				required = "/climate/predictive"
			case strings.HasPrefix(path, "/logbook"):
				required = "/logbook"
			case strings.HasPrefix(path, "/guide"):
				required = "/guide"
			case strings.HasPrefix(path, "/floors"):
				required = "/floorplan"
			case strings.HasPrefix(path, "/persons") || strings.HasPrefix(path, "/presence"):
				required = "/family"
			case strings.HasPrefix(path, "/template-entities"):
				required = "/templates"
			case strings.HasPrefix(path, "/hardware") || strings.HasPrefix(path, "/ota") || strings.HasPrefix(path, "/tags"):
				required = "/devices"
			case strings.HasPrefix(path, "/helpers"):
				required = "/helpers"
			case strings.HasPrefix(path, "/intercom") || strings.HasPrefix(path, "/nvr"):
				required = "/cameras"
			case strings.HasPrefix(path, "/notifications"):
				required = "/notifications"
			case strings.HasPrefix(path, "/schedules") || strings.HasPrefix(path, "/calendar"):
				required = "/schedules"
			case strings.HasPrefix(path, "/scripts"):
				required = "/automations"
			case strings.HasPrefix(path, "/weather"):
				required = "/"
			case strings.HasPrefix(path, "/voice"):
				required = "/media"
			case strings.HasPrefix(path, "/safety") || strings.HasPrefix(path, "/anomalies"):
				required = "/settings"
			case strings.HasPrefix(path, "/webhook-configs") || strings.HasPrefix(path, "/webhooks"):
				required = "/integrations"
			case strings.HasPrefix(path, "/devices"):
				required = "/devices"
			case strings.HasPrefix(path, "/integrations"):
				required = "/integrations"
			case strings.HasPrefix(path, "/history"):
				required = "/history"
			case strings.HasPrefix(path, "/audit"):
				required = "/audit"
			case strings.HasPrefix(path, "/cameras"):
				required = "/cameras"
			case strings.HasPrefix(path, "/chores"):
				required = "/chores"
			case strings.HasPrefix(path, "/shopping"):
				required = "/shopping"
			case strings.HasPrefix(path, "/pantry"):
				required = "/pantry"
			case strings.HasPrefix(path, "/medicines"):
				required = "/medicines"
			case strings.HasPrefix(path, "/finance"):
				required = "/finance"
			case strings.HasPrefix(path, "/media"):
				required = "/media"
			case strings.HasPrefix(path, "/zones"):
				required = "/zones"
			case strings.HasPrefix(path, "/mesh"):
				required = "/mesh"
			case strings.HasPrefix(path, "/store"):
				required = "/store"
			case strings.HasPrefix(path, "/floorplan"):
				required = "/floorplan"
			case strings.HasPrefix(path, "/system"):
				required = "/settings"
			case strings.HasPrefix(path, "/entities"):
				for _, candidate := range []string{"/", "/areas", "/devices", "/energy", "/custom-dashboards", "/floorplan", "/history", "/climate/predictive", "/zones"} {
					if permissions.CanAccessView(candidate) {
						next.ServeHTTP(w, r)
						return
					}
				}
				writeJSONError(w, http.StatusForbidden, "forbidden", "entity view access denied")
				return
			}
			if required != "" && !permissions.CanAccessView(required) {
				writeJSONError(w, http.StatusForbidden, "forbidden", "view access denied")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireDeviceControl allows authenticated members with the explicit device-control grant.
func RequireDeviceControl() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u := GetUser(r.Context())
			if u == nil {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
				return
			}
			permissions := u.EffectivePermissions()
			if permissions == nil || !permissions.CanControlDevices {
				writeJSONError(w, http.StatusForbidden, "forbidden", "device control permission required")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAutomationManagement checks the explicit automation-management grant.
func RequireAutomationManagement() func(http.Handler) http.Handler {
	return requirePermission(func(p *user.UserPermissions) bool { return p.CanManageAutomations }, "automation management permission required")
}

// RequireCameraView checks the explicit camera viewing grant.
func RequireCameraView() func(http.Handler) http.Handler {
	return requirePermission(func(p *user.UserPermissions) bool { return p.CanViewCameras }, "camera viewing permission required")
}

func requirePermission(check func(*user.UserPermissions) bool, message string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u := GetUser(r.Context())
			if u == nil {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
				return
			}
			permissions := u.EffectivePermissions()
			if permissions == nil || !check(permissions) {
				writeJSONError(w, http.StatusForbidden, "forbidden", message)
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
