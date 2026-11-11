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

	assert.Equal(t, http.StatusUnauthorized, request(http.MethodPost, "/api/v1/scenes", nil).Code)
	assert.Equal(t, http.StatusForbidden, request(http.MethodPost, "/api/v1/scenes", viewer).Code)
	assert.Equal(t, http.StatusNoContent, request(http.MethodPost, "/api/v1/scenes", operator).Code)
	assert.Equal(t, http.StatusNoContent, request(http.MethodPost, "/api/v1/presence/gps", viewer).Code)
	assert.Equal(t, http.StatusNoContent, request(http.MethodGet, "/api/v1/scenes", viewer).Code)
}
