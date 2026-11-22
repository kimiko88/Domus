// Package middleware_test tests authentication, RBAC, rate limiting, and CSRF middlewares (TDD red-to-green phase).
package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/application/auth"
	"github.com/domus-hub/domus/internal/domain/user"
	domusmw "github.com/domus-hub/domus/internal/interfaces/http/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockAuthService struct {
	mock.Mock
}

func (m *mockAuthService) ValidateSession(ctx context.Context, sessionID string) (*user.User, *user.Session, error) {
	args := m.Called(ctx, sessionID)
	var u *user.User
	if args.Get(0) != nil {
		u = args.Get(0).(*user.User)
	}
	var s *user.Session
	if args.Get(1) != nil {
		s = args.Get(1).(*user.Session)
	}
	return u, s, args.Error(2)
}

func TestRequireAuth_UnauthorizedWhenNoUser(t *testing.T) {
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	handler := domusmw.RequireAuth()(next)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.False(t, nextCalled)
}

func TestRequireRole_ForbiddenWhenInsufficientPermissions(t *testing.T) {
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	// Requires Admin, but user is Viewer
	handler := domusmw.RequireRole(user.RoleAdmin)(next)

	viewer := &user.User{
		ID:       "usr-1",
		Username: "viewer1",
		Role:     user.RoleViewer,
	}

	req := httptest.NewRequest(http.MethodGet, "/admin-only", nil)
	ctx := domusmw.WithUser(req.Context(), viewer)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
	assert.False(t, nextCalled)

	// Now with Admin user
	admin := &user.User{
		ID:       "usr-admin",
		Username: "admin",
		Role:     user.RoleAdmin,
	}
	reqAdmin := httptest.NewRequest(http.MethodGet, "/admin-only", nil)
	reqAdmin = reqAdmin.WithContext(domusmw.WithUser(reqAdmin.Context(), admin))
	rrAdmin := httptest.NewRecorder()

	handler.ServeHTTP(rrAdmin, reqAdmin)

	assert.Equal(t, http.StatusOK, rrAdmin.Code)
	assert.True(t, nextCalled)
}

func TestRateLimitMiddleware_BlocksAfterLimit(t *testing.T) {
	limiter := auth.NewRateLimiter(2, time.Minute)
	middleware := domusmw.RateLimit(limiter)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware(next)

	req := httptest.NewRequest(http.MethodPost, "/login", nil)
	req.RemoteAddr = "10.0.0.1:12345"

	// 1st request
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req)
	assert.Equal(t, http.StatusOK, rr1.Code)

	// 2nd request
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req)
	assert.Equal(t, http.StatusOK, rr2.Code)

	// 3rd request -> 429
	rr3 := httptest.NewRecorder()
	handler.ServeHTTP(rr3, req)
	assert.Equal(t, http.StatusTooManyRequests, rr3.Code)
}


func TestRequireOperatorForWrites(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	handler := domusmw.RequireOperatorForWrites()(next)

	request := func(method, path string, u *user.User) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, nil)
		if u != nil {
			req = req.WithContext(domusmw.WithUser(req.Context(), u))
		}
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		return rr
	}

	viewer := &user.User{ID: "viewer-1", Role: user.RoleViewer}
	operator := &user.User{ID: "operator-1", Role: user.RoleOperator}
	kid := &user.User{ID: "kid-1", Role: user.RoleKid}
	guest := &user.User{ID: "guest-1", Role: user.RoleGuest}

	assert.Equal(t, http.StatusUnauthorized, request(http.MethodPost, "/api/v1/scenes", nil).Code)
	assert.Equal(t, http.StatusForbidden, request(http.MethodPost, "/api/v1/scenes", viewer).Code)
	assert.Equal(t, http.StatusNoContent, request(http.MethodPost, "/api/v1/scenes", operator).Code)
	assert.Equal(t, http.StatusNoContent, request(http.MethodPost, "/api/v1/presence/gps", viewer).Code)
	assert.Equal(t, http.StatusForbidden, request(http.MethodPost, "/api/v1/presence/commute", viewer).Code)
	assert.Equal(t, http.StatusForbidden, request(http.MethodPost, "/api/v1/shopping", viewer).Code)
	assert.Equal(t, http.StatusNoContent, request(http.MethodPost, "/api/v1/shopping", kid).Code)
	assert.Equal(t, http.StatusForbidden, request(http.MethodPost, "/api/v1/chores/123/complete", viewer).Code)
	assert.Equal(t, http.StatusNoContent, request(http.MethodPost, "/api/v1/chores/123/complete", kid).Code)
	assert.Equal(t, http.StatusForbidden, request(http.MethodPost, "/api/v1/presence/gps", guest).Code)
	assert.Equal(t, http.StatusNoContent, request(http.MethodGet, "/api/v1/scenes", viewer).Code)
}

func TestRequireViewAccessHonorsMemberAllowList(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	handler := domusmw.RequireViewAccess()(next)

	kid := &user.User{ID: "kid-1", Role: user.RoleKid, Permissions: user.DefaultKidPermissions("kid-1")}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/automations", nil)
	request = request.WithContext(domusmw.WithUser(request.Context(), kid))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	assert.Equal(t, http.StatusForbidden, recorder.Code)

	allowedDashboard := httptest.NewRequest(http.MethodGet, "/api/v1/dashboards", nil)
	allowedDashboard = allowedDashboard.WithContext(domusmw.WithUser(allowedDashboard.Context(), kid))
	dashboardRecorder := httptest.NewRecorder()
	handler.ServeHTTP(dashboardRecorder, allowedDashboard)
	assert.Equal(t, http.StatusNoContent, dashboardRecorder.Code)
}

func TestRequireDeviceControlUsesExplicitGrant(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	handler := domusmw.RequireDeviceControl()(next)

	viewer := &user.User{ID: "viewer-1", Role: user.RoleViewer}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/entities/entity-1/command", nil)
	request = request.WithContext(domusmw.WithUser(request.Context(), viewer))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	assert.Equal(t, http.StatusForbidden, recorder.Code)

	operator := &user.User{ID: "operator-1", Role: user.RoleOperator}
	operatorRequest := httptest.NewRequest(http.MethodPost, "/api/v1/entities/entity-1/command", nil)
	operatorRequest = operatorRequest.WithContext(domusmw.WithUser(operatorRequest.Context(), operator))
	operatorRecorder := httptest.NewRecorder()
	handler.ServeHTTP(operatorRecorder, operatorRequest)
	assert.Equal(t, http.StatusNoContent, operatorRecorder.Code)
}

func TestRequireUnrestrictedDeviceControlBlocksScopedBulkActions(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	handler := domusmw.RequireUnrestrictedDeviceControl()(next)

	operator := &user.User{ID: "operator-1", Role: user.RoleOperator}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/scenes/scene-1/apply", nil)
	request = request.WithContext(domusmw.WithUser(request.Context(), operator))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	assert.Equal(t, http.StatusNoContent, recorder.Code)

	scoped := &user.User{
		ID: "scoped-1",
		Role: user.RoleOperator,
		Permissions: &user.UserPermissions{
			AllowedAreaIDs: []string{"kitchen"},
			CanControlDevices: true,
		},
	}
	scopedRequest := httptest.NewRequest(http.MethodPost, "/api/v1/scenes/scene-1/apply", nil)
	scopedRequest = scopedRequest.WithContext(domusmw.WithUser(scopedRequest.Context(), scoped))
	scopedRecorder := httptest.NewRecorder()
	handler.ServeHTTP(scopedRecorder, scopedRequest)
	assert.Equal(t, http.StatusForbidden, scopedRecorder.Code)
}


func TestRequireViewAccessMapsAlarmAndAssistantRoutes(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	handler := domusmw.RequireViewAccess()(next)
	request := func(method, path string, member *user.User) int {
		req := httptest.NewRequest(method, path, nil)
		req = req.WithContext(domusmw.WithUser(req.Context(), member))
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
		return recorder.Code
	}

	viewer := &user.User{ID: "viewer-1", Role: user.RoleViewer, Permissions: user.DefaultPermissions("viewer-1", user.RoleViewer)}
	assert.Equal(t, http.StatusNoContent, request(http.MethodGet, "/api/v1/alarm", viewer))
	assert.Equal(t, http.StatusForbidden, request(http.MethodGet, "/api/v1/presence/vacation", viewer))
	assert.Equal(t, http.StatusNoContent, request(http.MethodPost, "/api/v1/assistant/chat", viewer))
	assert.Equal(t, http.StatusForbidden, request(http.MethodGet, "/api/v1/assistant/config", viewer))
	assert.Equal(t, http.StatusForbidden, request(http.MethodGet, "/api/v1/assistant/satellites", viewer))

	kid := &user.User{ID: "kid-1", Role: user.RoleKid, Permissions: user.DefaultPermissions("kid-1", user.RoleKid)}
	assert.Equal(t, http.StatusNoContent, request(http.MethodGet, "/api/v1/assistant/satellites", kid))
}

func TestRequireViewAccessSeparatesClimateHistoryFromPredictiveClimate(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	handler := domusmw.RequireViewAccess()(next)
	request := func(path string, member *user.User) int {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req = req.WithContext(domusmw.WithUser(req.Context(), member))
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
		return recorder.Code
	}

	viewer := &user.User{ID: "viewer-climate", Role: user.RoleViewer, Permissions: user.DefaultPermissions("viewer-climate", user.RoleViewer)}
	assert.Equal(t, http.StatusNoContent, request("/api/v1/climate/history/history", viewer))
	assert.Equal(t, http.StatusForbidden, request("/api/v1/climate/predictive", viewer))

	kid := &user.User{ID: "kid-climate", Role: user.RoleKid, Permissions: user.DefaultPermissions("kid-climate", user.RoleKid)}
	assert.Equal(t, http.StatusForbidden, request("/api/v1/climate/history/history", kid))
}

func TestRequireSettingsPermissionUsesExplicitGrant(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	handler := domusmw.RequireSettingsPermission()
	request := func(member *user.User) int {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/assistant/config", nil)
		req = req.WithContext(domusmw.WithUser(req.Context(), member))
		recorder := httptest.NewRecorder()
		handler(next).ServeHTTP(recorder, req)
		return recorder.Code
	}

	operator := &user.User{ID: "operator-1", Role: user.RoleOperator}
	assert.Equal(t, http.StatusForbidden, request(operator))

	granted := &user.User{
		ID: "operator-granted",
		Role: user.RoleOperator,
		Permissions: &user.UserPermissions{CanManageSettings: true},
	}
	assert.Equal(t, http.StatusNoContent, request(granted))
	assert.Equal(t, http.StatusNoContent, request(&user.User{ID: "admin-1", Role: user.RoleAdmin}))
}
