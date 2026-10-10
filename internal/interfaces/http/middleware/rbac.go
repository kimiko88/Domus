package middleware

import (
	"net/http"

	"github.com/domus-hub/domus/internal/domain/user"
)

// RequireSettingsPermission ensures the user is an admin or has settings management rights.
func RequireSettingsPermission() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u := GetUser(r.Context())
			if u == nil {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
				return
			}
			if u.Role != user.RoleAdmin {
				writeJSONError(w, http.StatusForbidden, "forbidden", "admin permissions required to modify system settings")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
