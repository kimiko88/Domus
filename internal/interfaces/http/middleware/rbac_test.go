package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/domus-hub/domus/internal/domain/user"
	"github.com/domus-hub/domus/internal/interfaces/http/middleware"
	"github.com/stretchr/testify/assert"
)

func TestRBAC_RequireSettingsPermission(t *testing.T) {
	handler := middleware.RequireSettingsPermission()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Case 1: Unauthenticated -> 401
	req1 := httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil)
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)
	assert.Equal(t, http.StatusUnauthorized, rr1.Code)

	// Case 2: Operator (non-admin) -> 403
	opUser := &user.User{ID: "u1", Role: user.RoleOperator}
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil)
	req2 = req2.WithContext(middleware.WithUser(req2.Context(), opUser))
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)
	assert.Equal(t, http.StatusForbidden, rr2.Code)

	// Case 3: Admin -> 200
	adminUser := &user.User{ID: "u0", Role: user.RoleAdmin}
	req3 := httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil)
	req3 = req3.WithContext(middleware.WithUser(req3.Context(), adminUser))
	rr3 := httptest.NewRecorder()
	handler.ServeHTTP(rr3, req3)
	assert.Equal(t, http.StatusOK, rr3.Code)
}
