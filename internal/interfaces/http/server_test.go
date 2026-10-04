// Package http_test provides integration tests for the Domus HTTP server (TDD red-to-green phase).
package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/application/auth"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/crypto"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	serverpkg "github.com/domus-hub/domus/internal/interfaces/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type dummyPinger struct{}

func (d *dummyPinger) Ping() error {
	return nil
}

func TestServer_HealthEndpoint(t *testing.T) {
	cfg := config.ServerConfig{
		Host: "127.0.0.1",
		Port: 8080,
	}

	srv := serverpkg.NewServer(cfg, "1.0.0", time.Now(), &dummyPinger{})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	srv.Router().ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]any
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp["status"])
	assert.Equal(t, "1.0.0", resp["version"])
}

func TestServer_APIV1_HealthEndpoint(t *testing.T) {
	cfg := config.ServerConfig{
		Host: "127.0.0.1",
		Port: 8080,
	}

	srv := serverpkg.NewServer(cfg, "1.0.0", time.Now(), &dummyPinger{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/health", nil)
	rr := httptest.NewRecorder()

	srv.Router().ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestServer_WithAuthRoutes(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "server_auth_test.db")

	db, err := sqlite.Open(config.DatabaseConfig{
		Path:          dbPath,
		WALMode:       true,
		BusyTimeoutMs: 5000,
		MaxOpenConns:  1,
	})
	require.NoError(t, err)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, sqlite.Migrate(ctx, db))

	repos := sqlite.NewRepositories(db)
	hasher := crypto.NewArgon2Hasher(crypto.Argon2Params{
		Memory:      32 * 1024,
		Iterations:  1,
		Parallelism: 1,
		SaltLength:  16,
		KeyLength:   32,
	})
	authSvc := auth.NewService(repos.Users, repos.Sessions, repos.Audit, hasher, 24*time.Hour)
	rateLimiter := auth.NewRateLimiter(10, time.Minute)

	cfg := config.ServerConfig{
		Host: "127.0.0.1",
		Port: 8080,
	}

	srv := serverpkg.NewServer(cfg, "1.0.0", time.Now(), db, serverpkg.WithAuth(authSvc, rateLimiter))

	// Verify /api/v1/auth/setup-status endpoint exists and returns true
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/setup-status", nil)
	rr := httptest.NewRecorder()
	srv.Router().ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]any
	err = json.Unmarshal(rr.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, true, resp["setup_required"])
}

func TestServer_StartAndShutdown(t *testing.T) {
	cfg := config.ServerConfig{
		Host: "127.0.0.1",
		Port: 0, // OS assigns an ephemeral free port
	}

	srv := serverpkg.NewServer(cfg, "1.0.0", time.Now(), &dummyPinger{})

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start()
	}()

	// Wait briefly for server startup
	time.Sleep(50 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := srv.Shutdown(ctx)
	assert.NoError(t, err)

	startErr := <-errCh
	assert.True(t, startErr == nil || startErr == http.ErrServerClosed)
}

func TestServer_SwaggerAndDocsEndpoints(t *testing.T) {
	cfg := config.ServerConfig{
		Host: "127.0.0.1",
		Port: 8080,
	}

	srv := serverpkg.NewServer(cfg, "1.0.0", time.Now(), &dummyPinger{})

	// 1. /docs endpoint
	reqDocs := httptest.NewRequest(http.MethodGet, "/docs", nil)
	rrDocs := httptest.NewRecorder()
	srv.Router().ServeHTTP(rrDocs, reqDocs)
	assert.Equal(t, http.StatusOK, rrDocs.Code)
	assert.Contains(t, rrDocs.Header().Get("Content-Type"), "text/html")
	assert.Contains(t, rrDocs.Body.String(), "SwaggerUIBundle")

	// 2. /swagger endpoint
	reqSwagger := httptest.NewRequest(http.MethodGet, "/swagger", nil)
	rrSwagger := httptest.NewRecorder()
	srv.Router().ServeHTTP(rrSwagger, reqSwagger)
	assert.Equal(t, http.StatusOK, rrSwagger.Code)
	assert.Contains(t, rrSwagger.Header().Get("Content-Type"), "text/html")

	// 3. /api/openapi.yaml endpoint
	reqSpec := httptest.NewRequest(http.MethodGet, "/api/openapi.yaml", nil)
	rrSpec := httptest.NewRecorder()
	srv.Router().ServeHTTP(rrSpec, reqSpec)
	assert.Equal(t, http.StatusOK, rrSpec.Code)
	assert.Contains(t, rrSpec.Header().Get("Content-Type"), "yaml")
	assert.Contains(t, rrSpec.Body.String(), "openapi: 3.1.0")
}

