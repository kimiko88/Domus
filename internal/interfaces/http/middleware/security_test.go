// Package middleware — TDD RED: security headers middleware tests.
package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	domusmw "github.com/domus-hub/domus/internal/interfaces/http/middleware"
	"github.com/stretchr/testify/assert"
)

func nextOK(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func TestSecurityHeaders_SetsExpectedHeaders(t *testing.T) {
	handler := domusmw.SecurityHeaders()(http.HandlerFunc(nextOK))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Content-Security-Policy
	csp := rr.Header().Get("Content-Security-Policy")
	assert.NotEmpty(t, csp, "CSP header must be set")
	assert.Contains(t, csp, "default-src")

	// X-Frame-Options
	assert.Equal(t, "DENY", rr.Header().Get("X-Frame-Options"))

	// X-Content-Type-Options
	assert.Equal(t, "nosniff", rr.Header().Get("X-Content-Type-Options"))

	// Referrer-Policy
	assert.NotEmpty(t, rr.Header().Get("Referrer-Policy"))

	// Permissions-Policy
	assert.NotEmpty(t, rr.Header().Get("Permissions-Policy"))
}

func TestCORS_AllowsConfiguredOrigin(t *testing.T) {
	origins := []string{"http://localhost:5173"}
	handler := domusmw.CORS(origins)(http.HandlerFunc(nextOK))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, "http://localhost:5173", rr.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", rr.Header().Get("Access-Control-Allow-Credentials"))
}

func TestCORS_RejectsUnknownOrigin(t *testing.T) {
	origins := []string{"http://localhost:5173"}
	handler := domusmw.CORS(origins)(http.HandlerFunc(nextOK))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://evil.example.com")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Empty(t, rr.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_PreflightOptions(t *testing.T) {
	origins := []string{"http://localhost:5173"}
	handler := domusmw.CORS(origins)(http.HandlerFunc(nextOK))

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/entities", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type, X-CSRF-Token")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.Code)
	assert.NotEmpty(t, rr.Header().Get("Access-Control-Allow-Methods"))
	assert.Contains(t, rr.Header().Get("Access-Control-Allow-Headers"), "X-CSRF-Token")
}
