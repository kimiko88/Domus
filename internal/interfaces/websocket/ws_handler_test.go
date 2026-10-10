package websocket

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/application/auth"
	"github.com/domus-hub/domus/internal/domain/user"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/crypto"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	wsinfra "github.com/domus-hub/domus/internal/infrastructure/websocket"
	domusmw "github.com/domus-hub/domus/internal/interfaces/http/middleware"
	gorilla "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDBAndAuth(t *testing.T) (*auth.Service, *sqlite.Repositories) {
	t.Helper()
	db, err := sqlite.Open(config.DatabaseConfig{
		Path:         ":memory:",
		WALMode:      false,
		MaxOpenConns: 1,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	require.NoError(t, sqlite.Migrate(context.Background(), db))
	repos := sqlite.NewRepositories(db)
	hasher := crypto.NewArgon2Hasher(crypto.Argon2Params{
		Memory:      32 * 1024,
		Iterations:  1,
		Parallelism: 1,
		SaltLength:  16,
		KeyLength:   32,
	})
	authSvc := auth.NewService(repos.Users, repos.Sessions, repos.Audit, hasher, 1*time.Hour)
	return authSvc, repos
}

func TestWebSocketHandler_NoAuthService(t *testing.T) {
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 64, Workers: 1})
	defer bus.Close()

	hub := wsinfra.NewHub(bus)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	handler := NewHandler(hub, nil)
	srv := httptest.NewServer(handler)
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	conn, resp, err := gorilla.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()
	assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)

	var initMsg wsinfra.Message
	err = conn.ReadJSON(&initMsg)
	require.NoError(t, err)
	assert.Equal(t, "system.connected", initMsg.Type)

	err = conn.WriteJSON(map[string]any{"type": "ping"})
	require.NoError(t, err)

	var pongMsg wsinfra.Message
	err = conn.ReadJSON(&pongMsg)
	require.NoError(t, err)
	assert.Equal(t, "pong", pongMsg.Type)
}

func TestWebSocketHandler_WithAuth(t *testing.T) {
	authSvc, repos := setupTestDBAndAuth(t)
	ctx := context.Background()

	// Create user & session
	u := &user.User{
		ID:           "u-ws-test",
		Username:     "wsadmin",
		PasswordHash: "mockhash",
		Role:         user.RoleAdmin,
		Enabled:      true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	require.NoError(t, repos.Users.Create(ctx, u))

	sess := &user.Session{
		ID:        "sess-ws-token-1234",
		UserID:    u.ID,
		ExpiresAt: time.Now().Add(1 * time.Hour),
		CreatedAt: time.Now(),
	}
	require.NoError(t, repos.Sessions.Create(ctx, sess))

	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 64, Workers: 1})
	defer bus.Close()
	hub := wsinfra.NewHub(bus)
	hubCtx, hubCancel := context.WithCancel(context.Background())
	defer hubCancel()
	go hub.Run(hubCtx)

	handler := NewHandler(hub, authSvc)
	srv := httptest.NewServer(handler)
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	t.Run("missing auth returns 401", func(t *testing.T) {
		_, resp, err := gorilla.DefaultDialer.Dial(wsURL, nil)
		assert.Error(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("invalid session returns 401", func(t *testing.T) {
		_, resp, err := gorilla.DefaultDialer.Dial(wsURL+"?session=invalid-token", nil)
		assert.Error(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("valid session via query param connects", func(t *testing.T) {
		conn, resp, err := gorilla.DefaultDialer.Dial(wsURL+"?session="+sess.ID, nil)
		require.NoError(t, err)
		defer conn.Close()
		assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)

		var initMsg wsinfra.Message
		require.NoError(t, conn.ReadJSON(&initMsg))
		assert.Equal(t, "system.connected", initMsg.Type)
	})

	t.Run("valid session via cookie connects and receives broadcast", func(t *testing.T) {
		header := http.Header{}
		header.Add("Cookie", domusmw.SessionCookieName+"="+sess.ID)

		conn, resp, err := gorilla.DefaultDialer.Dial(wsURL, header)
		require.NoError(t, err)
		defer conn.Close()
		assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)

		var initMsg wsinfra.Message
		require.NoError(t, conn.ReadJSON(&initMsg))

		broadcastMsg := wsinfra.Message{
			Type:    "entity.state_changed",
			Payload: map[string]any{"entity_key": "light.kitchen", "state": "ON"},
		}
		hub.Broadcast(broadcastMsg)

		var rcvMsg wsinfra.Message
		require.NoError(t, conn.ReadJSON(&rcvMsg))
		assert.Equal(t, "entity.state_changed", rcvMsg.Type)
	})
}
