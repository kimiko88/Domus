// Package e2e_test tests multi-client family scenarios with granular RBAC permissions.
package e2e_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/application/auth"
	"github.com/domus-hub/domus/internal/domain/entity"
	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/crypto"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	wsinfra "github.com/domus-hub/domus/internal/infrastructure/websocket"
	serverpkg "github.com/domus-hub/domus/internal/interfaces/http"
	domusmw "github.com/domus-hub/domus/internal/interfaces/http/middleware"
	gorilla "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFamily_MultiClient_RBAC_EndToEnd(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "family_multiclient_e2e.db")

	const testPort = 8905
	cfg := config.Config{
		Server: config.ServerConfig{
			Host: "127.0.0.1",
			Port: testPort,
		},
		Database: config.DatabaseConfig{
			Path:          dbPath,
			WALMode:       true,
			BusyTimeoutMs: 5000,
			MaxOpenConns:  1,
		},
		Auth: config.AuthConfig{
			SessionTTL:           config.Duration{Duration: 24 * time.Hour},
			RateLimitLoginPerMin: 100,
			Argon2Memory:         32 * 1024,
			Argon2Time:           1,
			Argon2Threads:        1,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := sqlite.Open(cfg.Database)
	require.NoError(t, err)
	defer db.Close()

	require.NoError(t, sqlite.Migrate(ctx, db))

	repos := sqlite.NewRepositories(db)
	hasher := crypto.NewArgon2Hasher(crypto.Argon2Params{
		Memory:      cfg.Auth.Argon2Memory,
		Iterations:  cfg.Auth.Argon2Time,
		Parallelism: cfg.Auth.Argon2Threads,
	})
	authService := auth.NewService(
		repos.Users,
		repos.Sessions,
		repos.Audit,
		hasher,
		cfg.Auth.SessionTTL.Duration,
	).WithInvites(repos.Invites)

	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 256, Workers: 2})
	defer bus.Close()

	wsHub := wsinfra.NewHub(bus)
	ctxHub, cancelHub := context.WithCancel(context.Background())
	defer cancelHub()
	go wsHub.Run(ctxHub)

	srv := serverpkg.NewServer(
		cfg.Server,
		"1.0.0-e2e",
		time.Now(),
		db,
		serverpkg.WithAuth(authService, nil),
		serverpkg.WithRepositories(repos, bus),
		serverpkg.WithWebSocket(wsHub),
	)

	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- srv.Start()
	}()
	time.Sleep(100 * time.Millisecond)
	defer func() {
		_ = srv.Shutdown(context.Background())
	}()

	// 1. Initial Setup: Family Creator (Admin)
	creatorAdmin, err := authService.SetupInitialAdmin(ctx, "creator_admin", "AdminPass123!")
	require.NoError(t, err)

	adminSess, _, err := authService.Login(ctx, "creator_admin", "AdminPass123!", "127.0.0.1", "CreatorBrowser")
	require.NoError(t, err)

	// Create a controllable entity
	now := time.Now().UTC()
	testEnt := &entity.Entity{
		ID:            "ent-family-lamp",
		EntityKey:     "light.family_lamp",
		Domain:        "light",
		Name:          "Family Living Room Lamp",
		Writable:      true,
		State:         `{"state":"OFF"}`,
		Available:     true,
		LastChangedAt: now,
		LastUpdatedAt: now,
	}
	require.NoError(t, repos.Entities.Create(ctx, testEnt))

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", testPort)
	client := &http.Client{Timeout: 3 * time.Second}

	// Test CSRF token: 16 random bytes encoded as 32 hexadecimal characters.
	const csrfToken = "0123456789abcdef0123456789abcdef"

	// Helper for HTTP requests
	postJSON := func(endpoint string, body any, sessionCookie string) (*http.Response, map[string]any) {
		b, _ := json.Marshal(body)
		req, _ := http.NewRequest(http.MethodPost, baseURL+endpoint, bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		if sessionCookie != "" {
			req.AddCookie(&http.Cookie{Name: domusmw.SessionCookieName, Value: sessionCookie})
			req.AddCookie(&http.Cookie{Name: domusmw.CSRFCookieName, Value: csrfToken})
			req.Header.Set("X-CSRF-Token", csrfToken)
		}
		resp, err := client.Do(req)
		require.NoError(t, err)
		var res map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&res)
		_ = resp.Body.Close()
		return resp, res
	}

	// 2. Creator invites family member 1 as 'operator'
	resp, inviteOpData := postJSON("/api/v1/family/invites", map[string]any{
		"role":             "operator",
		"expires_in_hours": 48,
	}, adminSess.ID)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	opToken, ok := inviteOpData["token"].(string)
	require.True(t, ok, "invite response must contain a token")
	require.NotEmpty(t, opToken)

	// 3. Creator invites family member 2 as 'viewer'
	resp, inviteViewData := postJSON("/api/v1/family/invites", map[string]any{
		"role":             "viewer",
		"expires_in_hours": 48,
	}, adminSess.ID)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	viewToken, ok := inviteViewData["token"].(string)
	require.True(t, ok, "invite response must contain a token")
	require.NotEmpty(t, viewToken)

	// 4. Family Member 1 registers via invite link
	bOp, _ := json.Marshal(map[string]any{
		"token":    opToken,
		"username": "mom_operator",
		"password": "MomPassword123!",
		"email":    "mom@family.local",
	})
	reqOp, _ := http.NewRequest(http.MethodPost, baseURL+"/api/v1/auth/register", bytes.NewReader(bOp))
	reqOp.Header.Set("Content-Type", "application/json")
	respOp, err := client.Do(reqOp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, respOp.StatusCode)
	var momCookie string
	for _, c := range respOp.Cookies() {
		if c.Name == domusmw.SessionCookieName {
			momCookie = c.Value
		}
	}
	respOp.Body.Close()
	require.NotEmpty(t, momCookie)

	// 5. Family Member 2 registers via invite link
	bView, _ := json.Marshal(map[string]any{
		"token":    viewToken,
		"username": "kid_viewer",
		"password": "KidPassword123!",
		"email":    "kid@family.local",
	})
	reqView, _ := http.NewRequest(http.MethodPost, baseURL+"/api/v1/auth/register", bytes.NewReader(bView))
	reqView.Header.Set("Content-Type", "application/json")
	respView, err := client.Do(reqView)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, respView.StatusCode)
	var kidCookie string
	for _, c := range respView.Cookies() {
		if c.Name == domusmw.SessionCookieName {
			kidCookie = c.Value
		}
	}
	respView.Body.Close()
	require.NotEmpty(t, kidCookie)

	// 6. Connect 3 simultaneous WebSocket clients (Admin, Mom, Kid)
	dialWS := func(sess string) *gorilla.Conn {
		wsURL := fmt.Sprintf("ws://127.0.0.1:%d/ws?session=%s", testPort, url.QueryEscape(sess))
		conn, _, err := gorilla.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		var connAck wsinfra.Message
		_ = conn.ReadJSON(&connAck)
		assert.Equal(t, "system.connected", connAck.Type)
		return conn
	}

	wsAdmin := dialWS(adminSess.ID)
	defer wsAdmin.Close()

	wsMom := dialWS(momCookie)
	defer wsMom.Close()

	wsKid := dialWS(kidCookie)
	defer wsKid.Close()

	// Hub should report 3 active clients
	assert.Equal(t, 3, wsHub.ClientCount())

	// 7. Mom (Operator) turns on the living room lamp
	resp, _ = postJSON(fmt.Sprintf("/api/v1/entities/%s/command", testEnt.ID), map[string]any{
		"service": "turn_on",
	}, momCookie)
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)

	// 8. Kid (Viewer) tries to turn off the lamp -> blocked (403 Forbidden)
	resp, _ = postJSON(fmt.Sprintf("/api/v1/entities/%s/command", testEnt.ID), map[string]any{
		"service": "turn_off",
	}, kidCookie)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	// 9. Mom (Operator) tries to create another invite -> blocked (403 Forbidden)
	resp, _ = postJSON("/api/v1/family/invites", map[string]any{
		"role": "admin",
	}, momCookie)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	// 10. Simulate state update published on EventBus; all 3 clients must receive it
	require.NoError(t, bus.Publish(ctx, events.Event{
		ID:         "evt-state-1",
		Type:       events.EntityStateChanged,
		OccurredAt: time.Now(),
		Source:     "core",
		Payload: map[string]any{
			"entity_key": testEnt.EntityKey,
			"state":      `{"state":"ON"}`,
		},
	}))

	readMsg := func(conn *gorilla.Conn) wsinfra.Message {
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		var msg wsinfra.Message
		err := conn.ReadJSON(&msg)
		require.NoError(t, err)
		return msg
	}

	msgAdmin := readMsg(wsAdmin)
	msgMom := readMsg(wsMom)
	msgKid := readMsg(wsKid)

	assert.Equal(t, string(events.EntityStateChanged), msgAdmin.Type)
	assert.Equal(t, string(events.EntityStateChanged), msgMom.Type)
	assert.Equal(t, string(events.EntityStateChanged), msgKid.Type)

	// 11. Admin promotes Kid to 'operator'
	kidUser, err := repos.Users.GetByUsername(ctx, "kid_viewer")
	require.NoError(t, err)

	bRole, _ := json.Marshal(map[string]any{"role": "operator"})
	reqRole, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/api/v1/family/members/%s/role", baseURL, kidUser.ID), bytes.NewReader(bRole))
	reqRole.Header.Set("Content-Type", "application/json")
	reqRole.AddCookie(&http.Cookie{Name: domusmw.SessionCookieName, Value: adminSess.ID})
	reqRole.AddCookie(&http.Cookie{Name: domusmw.CSRFCookieName, Value: csrfToken})
	reqRole.Header.Set("X-CSRF-Token", csrfToken)
	respRole, err := client.Do(reqRole)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, respRole.StatusCode)
	respRole.Body.Close()

	// 12. Kid (now operator) can successfully turn off the lamp!
	resp, _ = postJSON(fmt.Sprintf("/api/v1/entities/%s/command", testEnt.ID), map[string]any{
		"service": "turn_off",
	}, kidCookie)
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)

	_ = creatorAdmin
}
