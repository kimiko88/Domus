package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/domus-hub/domus/internal/domain/user"
	domusmw "github.com/domus-hub/domus/internal/interfaces/http/middleware"
	"github.com/stretchr/testify/require"
)

func TestCSRFProtectionRejectsMissingAndMismatchedTokens(t *testing.T) {
	handler := domusmw.CSRFProtection()(http.HandlerFunc(nextOK))
	for _, token := range []string{"", "ffffffffffffffffffffffffffffffff"} {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/example", nil)
		req.AddCookie(&http.Cookie{Name: domusmw.SessionCookieName, Value: "session"})
		req.AddCookie(&http.Cookie{Name: domusmw.CSRFCookieName, Value: "0123456789abcdef0123456789abcdef"})
		if token != "" { req.Header.Set("X-CSRF-Token", token) }
		req = req.WithContext(domusmw.WithUser(req.Context(), &user.User{ID: "user-1"}))
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		require.Equal(t, http.StatusForbidden, rr.Code)
	}
}

func TestCSRFProtectionAcceptsMatchingDoubleSubmitToken(t *testing.T) {
	const token = "0123456789abcdef0123456789abcdef"
	req := httptest.NewRequest(http.MethodPost, "/api/v1/example", nil)
	req.AddCookie(&http.Cookie{Name: domusmw.SessionCookieName, Value: "session"})
	req.AddCookie(&http.Cookie{Name: domusmw.CSRFCookieName, Value: token})
	req.Header.Set("X-CSRF-Token", token)
	req = req.WithContext(domusmw.WithUser(req.Context(), &user.User{ID: "user-1"}))
	rr := httptest.NewRecorder()
	domusmw.CSRFProtection()(http.HandlerFunc(nextOK)).ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
}

func TestCSRFProtectionDoesNotBlockBearerOnlyRequests(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/example", nil)
	req.Header.Set("Authorization", "Bearer token")
	rr := httptest.NewRecorder()
	domusmw.CSRFProtection()(http.HandlerFunc(nextOK)).ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
}
