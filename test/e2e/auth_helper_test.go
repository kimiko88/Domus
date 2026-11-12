package e2e_test

import (
	"net/http"
	"net/http/httptest"
	"net/http/cookiejar"
	"net/url"
	"testing"
	"time"

	appauth "github.com/domus-hub/domus/internal/application/auth"
	"github.com/domus-hub/domus/internal/infrastructure/crypto"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	serverpkg "github.com/domus-hub/domus/internal/interfaces/http"
	domusmw "github.com/domus-hub/domus/internal/interfaces/http/middleware"
	"github.com/stretchr/testify/require"
)

// configureE2EAuth provisions a real administrator session for tests exercising
// protected domain endpoints. E2E requests should use the same auth boundary as
// production rather than bypassing it.
func configureE2EAuth(t *testing.T, repos *sqlite.Repositories) (serverpkg.ServerOption, string) {
	t.Helper()
	ctx := t.Context()
	hasher := crypto.NewArgon2Hasher(crypto.Argon2Params{
		Memory: 32 * 1024, Iterations: 1, Parallelism: 1, SaltLength: 16, KeyLength: 32,
	})
	authSvc := appauth.NewService(repos.Users, repos.Sessions, repos.Audit, hasher, 24*time.Hour)
	_, err := authSvc.SetupInitialAdmin(ctx, "e2e_admin", "E2ESecret123!")
	require.NoError(t, err)
	session, _, err := authSvc.Login(ctx, "e2e_admin", "E2ESecret123!", "127.0.0.1", "e2e-test")
	require.NoError(t, err)
	return serverpkg.WithAuth(authSvc, appauth.NewRateLimiter(100, time.Minute)), session.ID
}

// newAuthenticatedE2EClient attaches the test session cookie to every request.
func newAuthenticatedE2EClient(ts *httptest.Server, sessionID string) *http.Client {
	jar, _ := cookiejar.New(nil)
	baseURL, _ := url.Parse(ts.URL)
	jar.SetCookies(baseURL, []*http.Cookie{{Name: domusmw.SessionCookieName, Value: sessionID}})
	client := ts.Client()
	client.Jar = jar
	return client
}
