package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	domauth "github.com/domus-hub/domus/internal/domain/auth"
	"github.com/domus-hub/domus/internal/interfaces/http/middleware"
	"github.com/stretchr/testify/require"
)

type unavailableBannedIPRepository struct{}

func (unavailableBannedIPRepository) Get(context.Context, string) (*domauth.BannedIP, error) {
	return nil, errors.New("database unavailable")
}
func (unavailableBannedIPRepository) RecordFailedAttempt(context.Context, string, int, time.Duration) (bool, error) {
	return false, errors.New("database unavailable")
}
func (unavailableBannedIPRepository) ResetAttempts(context.Context, string) error { return errors.New("database unavailable") }
func (unavailableBannedIPRepository) ListBanned(context.Context) ([]*domauth.BannedIP, error) {
	return nil, errors.New("database unavailable")
}
func (unavailableBannedIPRepository) Unban(context.Context, string) error { return errors.New("database unavailable") }

func TestIPBanMiddlewareFailsClosedWhenRepositoryUnavailable(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})
	handler := middleware.IPBanMiddleware(unavailableBannedIPRepository{})(next)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/entities", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	require.False(t, called, "request must not proceed when ban state cannot be checked")
}
