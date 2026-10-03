// Package e2e provides end-to-end integration tests for MQTT ingestion, EventBus, and real-time WebSocket delivery.
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
	"github.com/domus-hub/domus/internal/domain/device"
	"github.com/domus-hub/domus/internal/domain/entity"
	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/domain/integration"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/crypto"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/domus-hub/domus/internal/infrastructure/mqtt"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	wsinfra "github.com/domus-hub/domus/internal/infrastructure/websocket"
	serverpkg "github.com/domus-hub/domus/internal/interfaces/http"
	domusmw "github.com/domus-hub/domus/internal/interfaces/http/middleware"
	"github.com/domus-hub/domus/internal/plugins/mqtt_generic"
	gorilla "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMQTT_To_EventBus_To_WebSocket_EndToEnd(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "mqtt_ws_e2e.db")

	cfg := config.Config{
		Server: config.ServerConfig{
			Host: "127.0.0.1",
			Port: 8899, // Specific test port
		},
		Database: config.DatabaseConfig{
			Path:          dbPath,
			WALMode:       true,
			BusyTimeoutMs: 5000,
			MaxOpenConns:  1,
		},
		Auth: config.AuthConfig{
			SessionTTL:           config.Duration{Duration: 24 * time.Hour},
			RateLimitLoginPerMin: 20,
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

	// 3. Start Event Bus & WebSocket Hub
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 128, Workers: 2})
	defer bus.Close()

	wsHub := wsinfra.NewHub(bus)
	ctxHub, cancelHub := context.WithCancel(context.Background())
	defer cancelHub()
	go wsHub.Run(ctxHub)

	// 4. Start HTTP Server
	srv := serverpkg.NewServer(
		cfg.Server,
		"1.0.0-e2e",
		time.Now(),
		db,
		serverpkg.WithAuth(authService, loginLimiter),
		serverpkg.WithRepositories(repos, bus),
		serverpkg.WithWebSocket(wsHub),
	)

	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- srv.Start()
	}()
	time.Sleep(100 * time.Millisecond)

	// 5. Setup MQTT Adapter with MockClient
	mockMQTT := mqtt.NewMockClient()
	adapter := mqtt_generic.NewAdapter(mqtt_generic.Config{
		IntegrationID:        "mqtt_test",
		StateTopicPattern:    "domus/+/state",
		CommandTopicTemplate: "domus/%s/set",
	}, mockMQTT, repos.Entities, repos.Devices, bus)

	require.NoError(t, adapter.Start(ctx))
	defer adapter.Stop(ctx)

	// 6. Setup Admin and Session
	adminUser, err := authService.SetupInitialAdmin(ctx, "admin", "AdminPassword123!")
	require.NoError(t, err)

	sess, _, err := authService.Login(ctx, adminUser.Username, "AdminPassword123!", "127.0.0.1", "test-agent")
	require.NoError(t, err)

	// 7. Seed Integration, Device, and Entity
	now := time.Now().UTC()
	in := &integration.Integration{
		ID:        "mqtt_test",
		Type:      "mqtt_generic",
		Name:      "MQTT Generic Test",
		Enabled:   true,
		Status:    "connected",
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, repos.Integrations.Create(ctx, in))

	dev := &device.Device{
		ID:            "dev-ceiling-light",
		IntegrationID: in.ID,
		ExternalID:    "light_01",
		Name:          "Salon Ceiling Light",
		Online:        true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	require.NoError(t, repos.Devices.Create(ctx, dev))

	ent := &entity.Entity{
		ID:            "ent-ceiling-light",
		DeviceID:      dev.ID,
		EntityKey:     "light.salon_ceiling",
		Domain:        "light",
		Name:          "Salon Ceiling Light",
		Writable:      true,
		State:         `{"state":"OFF"}`,
		Available:     true,
		LastChangedAt: now,
		LastUpdatedAt: now,
	}
	require.NoError(t, repos.Entities.Create(ctx, ent))

	// 8. Connect real WebSocket client
	wsURL := fmt.Sprintf("ws://127.0.0.1:8899/ws?session=%s", url.QueryEscape(sess.ID))
	wsConn, _, err := gorilla.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer wsConn.Close()

	// Read initial connected frame
	var initMsg wsinfra.Message
	err = wsConn.ReadJSON(&initMsg)
	require.NoError(t, err)
	assert.Equal(t, "system.connected", initMsg.Type)

	// 9. Simulate incoming MQTT message
	mockMQTT.SimulateIncoming("domus/light.salon_ceiling/state", []byte(`{"state":"ON","brightness":255}`))

	// 10. Read forwarded message from WebSocket
	var wsFrame wsinfra.Message
	err = wsConn.ReadJSON(&wsFrame)
	require.NoError(t, err)

	assert.Equal(t, string(events.EntityStateChanged), wsFrame.Type)
	payloadMap, ok := wsFrame.Payload.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "light.salon_ceiling", payloadMap["entity_key"])
	assert.Equal(t, `{"state":"ON","brightness":255}`, payloadMap["state"])

	// 11. Verify state was persisted in SQLite
	updatedEnt, err := repos.Entities.GetByID(ctx, ent.ID)
	require.NoError(t, err)
	assert.Equal(t, `{"state":"ON","brightness":255}`, updatedEnt.State)

	// 12. Send command via REST API to control entity
	cmdURL := fmt.Sprintf("http://127.0.0.1:8899/api/v1/entities/%s/command", ent.ID)
	cmdBody, _ := json.Marshal(map[string]any{
		"service": "turn_off",
		"params":  map[string]any{},
	})
	httpReq, err := http.NewRequest(http.MethodPost, cmdURL, bytes.NewReader(cmdBody))
	require.NoError(t, err)
	httpReq.AddCookie(&http.Cookie{
		Name:  domusmw.SessionCookieName,
		Value: sess.ID,
	})
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusAccepted, resp.StatusCode)

	// Allow event bus and adapter to process
	time.Sleep(100 * time.Millisecond)

	// Verify MQTT published message
	published := mockMQTT.GetPublished()
	require.NotEmpty(t, published)
	lastPub := published[len(published)-1]
	assert.Equal(t, "domus/light.salon_ceiling/set", lastPub.Topic)
	assert.Contains(t, string(lastPub.Payload), "turn_off")

	// 13. Clean shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutdownCancel()
	err = srv.Shutdown(shutdownCtx)
	assert.NoError(t, err)
}
