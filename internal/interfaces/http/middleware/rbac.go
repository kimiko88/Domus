package middleware

import "net/http"

// RequireSettingsPermission ensures the user is an admin or has settings management rights.
func RequireSettingsPermission() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u := GetUser(r.Context())
			if u == nil {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
				return
			}
			permissions := u.EffectivePermissions()
			if permissions == nil || !permissions.CanManageSettings {
				writeJSONError(w, http.StatusForbidden, "forbidden", "settings management permission required")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
