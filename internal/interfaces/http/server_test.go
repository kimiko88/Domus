// Package http_test provides integration tests for the Domus HTTP server (TDD red-to-green phase).
package http_test

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
	"github.com/domus-hub/domus/internal/domain/entity"
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

func TestServer_RBAC_And_FamilyRoutes(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "server_rbac_test.db")

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
	authSvc := auth.NewService(repos.Users, repos.Sessions, repos.Audit, hasher, 24*time.Hour).WithInvites(repos.Invites)
	rateLimiter := auth.NewRateLimiter(100, time.Minute)

	// Create test users: Admin, Operator, Viewer
	adminUser, err := authSvc.SetupInitialAdmin(ctx, "admin_user", "AdminSecret123!")
	require.NoError(t, err)
	adminSess, _, err := authSvc.Login(ctx, "admin_user", "AdminSecret123!", "127.0.0.1", "test-agent")
	require.NoError(t, err)

	pwHash, err := hasher.HashPassword("MemberSecret123!")
	require.NoError(t, err)

	now := time.Now().UTC()
	opUser := &user.User{
		ID:           "usr-operator-1",
		Username:     "operator_user",
		PasswordHash: pwHash,
		Role:         user.RoleOperator,
		Enabled:      true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	require.NoError(t, repos.Users.Create(ctx, opUser))
	opSess := &user.Session{
		ID:        "sess-op-1",
		UserID:    opUser.ID,
		ExpiresAt: now.Add(24 * time.Hour),
		CreatedAt: now,
	}
	require.NoError(t, repos.Sessions.Create(ctx, opSess))

	viewUser := &user.User{
		ID:           "usr-viewer-1",
		Username:     "viewer_user",
		PasswordHash: pwHash,
		Role:         user.RoleViewer,
		Enabled:      true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	require.NoError(t, repos.Users.Create(ctx, viewUser))
	viewSess := &user.Session{
		ID:        "sess-view-1",
		UserID:    viewUser.ID,
		ExpiresAt: now.Add(24 * time.Hour),
		CreatedAt: now,
	}
	require.NoError(t, repos.Sessions.Create(ctx, viewSess))

	// Create test entity
	ent := &entity.Entity{
		ID:            "ent-1",
		EntityKey:     "light.living_room",
		Domain:        "light",
		Name:          "Living Room Light",
		Writable:      true,
		State:         `{"state":"OFF"}`,
		Available:     true,
		LastChangedAt: now,
		LastUpdatedAt: now,
	}
	require.NoError(t, repos.Entities.Create(ctx, ent))

	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 128, Workers: 1})
	defer bus.Close()

	cfg := config.ServerConfig{Host: "127.0.0.1", Port: 8080}
	srv := serverpkg.NewServer(
		cfg,
		"1.0.0",
		now,
		db,
		serverpkg.WithAuth(authSvc, rateLimiter),
		serverpkg.WithRepositories(repos, bus),
	)

	// Helper to send request with session cookie
	doReq := func(method, path string, sessID string, body []byte) *httptest.ResponseRecorder {
		var req *http.Request
		if body != nil {
			req = httptest.NewRequest(method, path, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
		} else {
			req = httptest.NewRequest(method, path, nil)
		}
		if sessID != "" {
			req.AddCookie(&http.Cookie{Name: domusmw.SessionCookieName, Value: sessID})
		}
		rr := httptest.NewRecorder()
		srv.Router().ServeHTTP(rr, req)
		return rr
	}

	// 1. Unauthenticated request to /api/v1/entities -> 401
	rr := doReq(http.MethodGet, "/api/v1/entities", "", nil)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)

	// 2. Viewer: can view entities
	rr = doReq(http.MethodGet, "/api/v1/entities", viewSess.ID, nil)
	assert.Equal(t, http.StatusOK, rr.Code)

	// 3. Viewer: CANNOT execute commands -> 403 Forbidden
	cmdBody, _ := json.Marshal(map[string]any{"service": "turn_on"})
	rr = doReq(http.MethodPost, "/api/v1/entities/"+ent.ID+"/command", viewSess.ID, cmdBody)
	assert.Equal(t, http.StatusForbidden, rr.Code)

	// 4. Viewer: CANNOT invite members -> 403 Forbidden
	inviteBody, _ := json.Marshal(map[string]any{"role": "viewer", "expires_in_hours": 24})
	rr = doReq(http.MethodPost, "/api/v1/family/invites", viewSess.ID, inviteBody)
	assert.Equal(t, http.StatusForbidden, rr.Code)

	// 5. Viewer: CAN view family members -> 200 OK
	rr = doReq(http.MethodGet, "/api/v1/family/members", viewSess.ID, nil)
	assert.Equal(t, http.StatusOK, rr.Code)
	var membersResp map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &membersResp)
	assert.Equal(t, float64(3), membersResp["total"])

	// 6. Operator: CAN execute commands -> 202 Accepted
	rr = doReq(http.MethodPost, "/api/v1/entities/"+ent.ID+"/command", opSess.ID, cmdBody)
	assert.Equal(t, http.StatusAccepted, rr.Code)

	// 7. Operator: CANNOT invite members -> 403 Forbidden
	rr = doReq(http.MethodPost, "/api/v1/family/invites", opSess.ID, inviteBody)
	assert.Equal(t, http.StatusForbidden, rr.Code)

	// 8. Admin: CAN execute commands -> 202 Accepted
	rr = doReq(http.MethodPost, "/api/v1/entities/"+ent.ID+"/command", adminSess.ID, cmdBody)
	assert.Equal(t, http.StatusAccepted, rr.Code)

	// 9. Admin: CAN invite family members -> 201 Created
	adminInviteBody, _ := json.Marshal(map[string]any{"role": "operator", "expires_in_hours": 48})
	rr = doReq(http.MethodPost, "/api/v1/family/invites", adminSess.ID, adminInviteBody)
	assert.Equal(t, http.StatusCreated, rr.Code)
	var inviteResp map[string]any
	err = json.Unmarshal(rr.Body.Bytes(), &inviteResp)
	require.NoError(t, err)
	token, ok := inviteResp["token"].(string)
	require.True(t, ok)
	assert.NotEmpty(t, token)
	assert.Equal(t, "/register?token="+token, inviteResp["invite_url"])

	// 10. Public: Validate invite token -> 200 OK
	rr = doReq(http.MethodGet, "/api/v1/auth/invite/"+token, "", nil)
	assert.Equal(t, http.StatusOK, rr.Code)
	var valResp map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &valResp)
	assert.Equal(t, true, valResp["valid"])
	assert.Equal(t, "operator", valResp["role"])

	// 11. Public: Complete registration with invite -> 201 Created
	regBody, _ := json.Marshal(map[string]any{
		"token":    token,
		"username": "kid_member",
		"password": "KidPassword123!",
		"email":    "kid@domus.local",
	})
	rr = doReq(http.MethodPost, "/api/v1/auth/register", "", regBody)
	assert.Equal(t, http.StatusCreated, rr.Code)

	// Session cookie should be set
	cookies := rr.Result().Cookies()
	var newSessCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == domusmw.SessionCookieName {
			newSessCookie = c
			break
		}
	}
	require.NotNil(t, newSessCookie)
	assert.NotEmpty(t, newSessCookie.Value)

	// 12. Newly registered user is an operator, can control entities
	rr = doReq(http.MethodPost, "/api/v1/entities/"+ent.ID+"/command", newSessCookie.Value, cmdBody)
	assert.Equal(t, http.StatusAccepted, rr.Code)

	// 13. Admin can update member role
	kidUser, err := repos.Users.GetByUsername(ctx, "kid_member")
	require.NoError(t, err)
	updateRoleBody, _ := json.Marshal(map[string]any{"role": "viewer"})
	rr = doReq(http.MethodPut, "/api/v1/family/members/"+kidUser.ID+"/role", adminSess.ID, updateRoleBody)
	assert.Equal(t, http.StatusOK, rr.Code)

	// Kid is now viewer, command should be forbidden
	rr = doReq(http.MethodPost, "/api/v1/entities/"+ent.ID+"/command", newSessCookie.Value, cmdBody)
	assert.Equal(t, http.StatusForbidden, rr.Code)

	// 14. Admin cannot demote last admin
	demoteAdminBody, _ := json.Marshal(map[string]any{"role": "viewer"})
	rr = doReq(http.MethodPut, "/api/v1/family/members/"+adminUser.ID+"/role", adminSess.ID, demoteAdminBody)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}
