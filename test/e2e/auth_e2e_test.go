// Package e2e provides end-to-end integration tests for Domus authentication flows.
package e2e_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/application/auth"
	"github.com/domus-hub/domus/internal/domain/user"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/crypto"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	serverpkg "github.com/domus-hub/domus/internal/interfaces/http"
	domusmw "github.com/domus-hub/domus/internal/interfaces/http/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthFlow_EndToEnd(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "domus_auth_e2e.db")

	cfg := config.Config{
		Server: config.ServerConfig{
			Host: "127.0.0.1",
			Port: 0,
		},
		Database: config.DatabaseConfig{
			Path:          dbPath,
			WALMode:       true,
			BusyTimeoutMs: 5000,
			MaxOpenConns:  1,
		},
		Auth: config.AuthConfig{
			SessionTTL:           config.Duration{Duration: 24 * time.Hour},
			RateLimitLoginPerMin: 10,
			Argon2Time:           1,
			Argon2Memory:         32 * 1024,
			Argon2Threads:        1,
		},
	}

	// 1. Open SQLite & Run Migrations
	db, err := sqlite.Open(cfg.Database)
	require.NoError(t, err)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	require.NoError(t, sqlite.Migrate(ctx, db))

	// 2. Initialize Repositories and Services
	repos := sqlite.NewRepositories(db)
	hasher := crypto.NewArgon2Hasher(crypto.Argon2Params{
		Memory:      cfg.Auth.Argon2Memory,
		Iterations:  cfg.Auth.Argon2Time,
		Parallelism: cfg.Auth.Argon2Threads,
		SaltLength:  16,
		KeyLength:   32,
	})
	authService := auth.NewService(
		repos.Users,
		repos.Sessions,
		repos.Audit,
		hasher,
		cfg.Auth.SessionTTL.Duration,
	)
	loginLimiter := auth.NewRateLimiter(cfg.Auth.RateLimitLoginPerMin, time.Minute)

	// 3. Start Event Bus
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 64, Workers: 2})
	defer bus.Close()

	// 4. Start HTTP Server
	srv := serverpkg.NewServer(
		cfg.Server,
		"1.0.0-e2e",
		time.Now(),
		db,
		serverpkg.WithAuth(authService, loginLimiter),
	)

	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- srv.Start()
	}()
	time.Sleep(50 * time.Millisecond)

	router := srv.Router()

	// Step A: GET /api/v1/auth/setup-status -> setup_required == true
	{
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/auth/setup-status", nil)
		rr := &responseRecorder{headers: make(http.Header)}
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.code)
		var body map[string]any
		require.NoError(t, json.Unmarshal(rr.body, &body))
		assert.Equal(t, true, body["setup_required"])
	}

	// Step B: GET /api/v1/auth/me without cookie -> 401 Unauthorized
	{
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		rr := &responseRecorder{headers: make(http.Header)}
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.code)
	}

	// Step C: POST /api/v1/auth/setup -> 201 Created
	{
		setupPayload, _ := json.Marshal(map[string]string{
			"username": "admin",
			"password": "SuperSecretPassword123!",
		})
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/setup", bytes.NewReader(setupPayload))
		rr := &responseRecorder{headers: make(http.Header)}
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusCreated, rr.code)
		var body map[string]any
		require.NoError(t, json.Unmarshal(rr.body, &body))
		assert.Equal(t, "admin", body["username"])
		assert.Equal(t, string(user.RoleAdmin), body["role"])
	}

	// Step D: POST /api/v1/auth/setup again -> 409 Conflict
	{
		setupPayload, _ := json.Marshal(map[string]string{
			"username": "second_admin",
			"password": "SuperSecretPassword123!",
		})
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/setup", bytes.NewReader(setupPayload))
		rr := &responseRecorder{headers: make(http.Header)}
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusConflict, rr.code)
	}

	// Step E: POST /api/v1/auth/login with wrong password -> 401 Unauthorized
	{
		badLogin, _ := json.Marshal(map[string]string{
			"username": "admin",
			"password": "WrongPassword!",
		})
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(badLogin))
		rr := &responseRecorder{headers: make(http.Header)}
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.code)
	}

	// Step F: POST /api/v1/auth/login with correct password -> 200 OK + domus_session cookie
	var sessionCookie *http.Cookie
	{
		goodLogin, _ := json.Marshal(map[string]string{
			"username": "admin",
			"password": "SuperSecretPassword123!",
		})
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(goodLogin))
		rr := &responseRecorder{headers: make(http.Header)}
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.code)
		fakeResp := http.Response{Header: rr.headers}
		for _, c := range fakeResp.Cookies() {
			if c.Name == domusmw.SessionCookieName {
				sessionCookie = c
				break
			}
		}
		require.NotNil(t, sessionCookie, "domus_session cookie must be present")
		assert.True(t, sessionCookie.HttpOnly)
		assert.NotEmpty(t, sessionCookie.Value)
	}

	// Step G: GET /api/v1/auth/me with session cookie -> 200 OK
	{
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		req.AddCookie(sessionCookie)
		rr := &responseRecorder{headers: make(http.Header)}
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.code)
		var body map[string]any
		require.NoError(t, json.Unmarshal(rr.body, &body))
		assert.Equal(t, "admin", body["username"])
		assert.Equal(t, string(user.RoleAdmin), body["role"])
	}

	// Step H: POST /api/v1/auth/logout -> 204 No Content + clears cookie
	{
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
		req.AddCookie(sessionCookie)
		rr := &responseRecorder{headers: make(http.Header)}
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNoContent, rr.code)

		fakeResp := http.Response{Header: rr.headers}
		var clearedCookie *http.Cookie
		for _, c := range fakeResp.Cookies() {
			if c.Name == domusmw.SessionCookieName {
				clearedCookie = c
				break
			}
		}
		require.NotNil(t, clearedCookie)
		assert.Equal(t, -1, clearedCookie.MaxAge)
	}

	// Step I: GET /api/v1/auth/me after logout -> 401 Unauthorized
	{
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		req.AddCookie(sessionCookie)
		rr := &responseRecorder{headers: make(http.Header)}
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.code)
	}

	// Step J: Audit logs verification
	entries, err := repos.Audit.List(ctx, 10)
	require.NoError(t, err)
	assert.NotEmpty(t, entries)

	actions := make(map[string]bool)
	for _, e := range entries {
		actions[e.Action] = true
	}
	assert.True(t, actions["auth.setup.admin_created"], "admin setup must be recorded in audit log")
	assert.True(t, actions["auth.login.failed"], "failed login must be recorded in audit log")
	assert.True(t, actions["auth.login.success"], "successful login must be recorded in audit log")
	assert.True(t, actions["auth.logout"], "logout must be recorded in audit log")

	// Step K: Clean shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutdownCancel()
	err = srv.Shutdown(shutdownCtx)
	assert.NoError(t, err)
}
