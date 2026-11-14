// Package handlers_test tests the AuthHandler HTTP endpoints (TDD red-to-green phase).
package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/application/auth"
	domauth "github.com/domus-hub/domus/internal/domain/auth"
	"github.com/domus-hub/domus/internal/domain/user"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/crypto"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	"github.com/domus-hub/domus/internal/interfaces/http/handlers"
	domusmw "github.com/domus-hub/domus/internal/interfaces/http/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupAuthHandler(t *testing.T) (*handlers.AuthHandler, *auth.Service, *sqlite.Repositories) {
	t.Helper()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "auth_handler_test.db")

	db, err := sqlite.Open(config.DatabaseConfig{
		Path:          dbPath,
		WALMode:       true,
		BusyTimeoutMs: 5000,
		MaxOpenConns:  1,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	require.NoError(t, sqlite.Migrate(ctx, db))

	t.Cleanup(func() {
		_ = db.Close()
	})

	repos := sqlite.NewRepositories(db)
	hasher := crypto.NewArgon2Hasher(crypto.Argon2Params{
		Memory:      32 * 1024,
		Iterations:  1,
		Parallelism: 1,
		SaltLength:  16,
		KeyLength:   32,
	})

	svc := auth.NewService(repos.Users, repos.Sessions, repos.Audit, hasher, 24*time.Hour)
	handler := handlers.NewAuthHandler(svc, false)
	return handler, svc, repos
}

func TestAuthHandler_Setup(t *testing.T) {
	handler, _, _ := setupAuthHandler(t)

	// 1. Initial setup succeeds
	body := map[string]string{
		"username": "admin",
		"password": "Password123!",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/setup", bytes.NewReader(jsonBody))
	rr := httptest.NewRecorder()

	handler.Setup(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)

	var resp map[string]any
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "admin", resp["username"])
	assert.Equal(t, "admin", resp["role"])

	// 2. Second setup returns 409 Conflict
	rr2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/setup", bytes.NewReader(jsonBody))
	handler.Setup(rr2, req2)

	assert.Equal(t, http.StatusConflict, rr2.Code)
}

func TestAuthHandler_LoginAndLogout(t *testing.T) {
	handler, svc, _ := setupAuthHandler(t)
	ctx := context.Background()

	// Provision admin
	_, err := svc.SetupInitialAdmin(ctx, "admin", "Password123!")
	require.NoError(t, err)

	// 1. Login with invalid password -> 401
	badLoginBody, _ := json.Marshal(map[string]string{
		"username": "admin",
		"password": "WrongPassword!",
	})
	reqBad := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(badLoginBody))
	rrBad := httptest.NewRecorder()
	handler.Login(rrBad, reqBad)

	assert.Equal(t, http.StatusUnauthorized, rrBad.Code)

	// 2. Login with correct password -> 200 + Set-Cookie
	goodLoginBody, _ := json.Marshal(map[string]string{
		"username": "admin",
		"password": "Password123!",
	})
	reqGood := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(goodLoginBody))
	rrGood := httptest.NewRecorder()
	handler.Login(rrGood, reqGood)

	assert.Equal(t, http.StatusOK, rrGood.Code)

	// Check cookie
	cookies := rrGood.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == domusmw.SessionCookieName {
			sessionCookie = c
			break
		}
	}
	require.NotNil(t, sessionCookie, "domus_session cookie must be set")
	assert.True(t, sessionCookie.HttpOnly)
	assert.NotEmpty(t, sessionCookie.Value)

	// 3. Logout clears cookie -> 204
	reqLogout := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	reqLogout.AddCookie(sessionCookie)
	reqLogout = reqLogout.WithContext(domusmw.WithSession(reqLogout.Context(), &user.Session{ID: sessionCookie.Value}))
	rrLogout := httptest.NewRecorder()

	handler.Logout(rrLogout, reqLogout)
	assert.Equal(t, http.StatusNoContent, rrLogout.Code)

	logoutCookies := rrLogout.Result().Cookies()
	var clearedCookie *http.Cookie
	for _, c := range logoutCookies {
		if c.Name == domusmw.SessionCookieName {
			clearedCookie = c
			break
		}
	}
	require.NotNil(t, clearedCookie)
	assert.Equal(t, -1, clearedCookie.MaxAge)
}

func TestDisable2FARequiresPasswordAndCurrentSecondFactor(t *testing.T) {
	handler, svc, repos := setupAuthHandler(t)
	ctx := context.Background()
	admin, err := svc.SetupInitialAdmin(ctx, "secure_admin", "Password123!")
	require.NoError(t, err)

	secret, err := auth.GenerateTOTPSecret()
	require.NoError(t, err)
	require.NoError(t, repos.TwoFactor.Save(ctx, &domauth.TwoFactorConfig{
		UserID: admin.ID, Secret: secret, IsEnabled: true,
		BackupCodes: []string{"ABCD-EFGH"},
	}))
	handler.WithSecurity(repos.TwoFactor, nil)

	call := func(password, code string) *httptest.ResponseRecorder {
		body, marshalErr := json.Marshal(map[string]string{"password": password, "two_factor_code": code})
		require.NoError(t, marshalErr)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/2fa/disable", bytes.NewReader(body))
		req = req.WithContext(domusmw.WithUser(req.Context(), admin))
		rr := httptest.NewRecorder()
		handler.Disable2FA(rr, req)
		return rr
	}

	require.Equal(t, http.StatusUnauthorized, call("WrongPassword!", auth.GenerateTOTPPasscode(secret, time.Now().UTC())).Code)
	require.Equal(t, http.StatusUnauthorized, call("Password123!", "").Code)
	cfg, err := repos.TwoFactor.Get(ctx, admin.ID)
	require.NoError(t, err)
	require.True(t, cfg.IsEnabled, "failed re-authentication must not disable 2FA")

	code := auth.GenerateTOTPPasscode(secret, time.Now().UTC())
	require.Equal(t, http.StatusOK, call("Password123!", code).Code)
	_, err = repos.TwoFactor.Get(ctx, admin.ID)
	require.ErrorIs(t, err, domauth.ErrTwoFactorNotFound)
}


func TestSetup2FADoesNotReplaceEnabledConfiguration(t *testing.T) {
	handler, svc, repos := setupAuthHandler(t)
	ctx := context.Background()
	admin, err := svc.SetupInitialAdmin(ctx, "two_factor_admin", "Password123!")
	require.NoError(t, err)
	original := &domauth.TwoFactorConfig{
		UserID: admin.ID, Secret: "JBSWY3DPEHPK3PXP", IsEnabled: true,
		BackupCodes: []string{"ABCD-EFGH"},
	}
	require.NoError(t, repos.TwoFactor.Save(ctx, original))
	handler.WithSecurity(repos.TwoFactor, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/2fa/setup", nil)
	req = req.WithContext(domusmw.WithUser(req.Context(), admin))
	rr := httptest.NewRecorder()
	handler.Setup2FA(rr, req)

	require.Equal(t, http.StatusConflict, rr.Code)
	persisted, err := repos.TwoFactor.Get(ctx, admin.ID)
	require.NoError(t, err)
	require.True(t, persisted.IsEnabled)
	require.Equal(t, original.Secret, persisted.Secret)
	require.Equal(t, original.BackupCodes, persisted.BackupCodes)
}
