// Package e2e_test tests complete end-to-end integration for Zigbee2MQTT, ESPHome, and Cloud resilience adapters.
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
	"github.com/domus-hub/domus/internal/plugins/cloud"
	"github.com/domus-hub/domus/internal/plugins/esphome_mqtt"
	"github.com/domus-hub/domus/internal/plugins/zigbee2mqtt"
	gorilla "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestZigbee2MQTT_ESPHome_Cloud_EndToEnd(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "ms4_e2e.db")

	cfg := config.Config{
		Server: config.ServerConfig{
			Host: "127.0.0.1",
			Port: 8901, // Dedicated test port
		},
		Database: config.DatabaseConfig{
			Path:          dbPath,
			WALMode:       true,
			BusyTimeoutMs: 5000,
			MaxOpenConns:  1,
		},
		Auth: config.AuthConfig{
			SessionTTL:           config.Duration{Duration: 24 * time.Hour},
			RateLimitLoginPerMin: 50,
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
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 256, Workers: 2})
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

	// 5. Connect Mock MQTT Broker & Initialize Adapters
	mockMQTT := mqtt.NewMockClient()

	now := time.Now().UTC()
	require.NoError(t, repos.Integrations.Create(ctx, &integration.Integration{
		ID:        "z2m_e2e",
		Type:      "zigbee2mqtt",
		Name:      "Zigbee2MQTT E2E",
		Enabled:   true,
		Status:    "connected",
		CreatedAt: now,
		UpdatedAt: now,
	}))
	require.NoError(t, repos.Integrations.Create(ctx, &integration.Integration{
		ID:        "esphome_e2e",
		Type:      "esphome_mqtt",
		Name:      "ESPHome E2E",
		Enabled:   true,
		Status:    "connected",
		CreatedAt: now,
		UpdatedAt: now,
	}))
	require.NoError(t, repos.Integrations.Create(ctx, &integration.Integration{
		ID:        "cloud_e2e",
		Type:      "cloud_tuya",
		Name:      "Cloud E2E",
		Enabled:   true,
		Status:    "connected",
		CreatedAt: now,
		UpdatedAt: now,
	}))

	z2mAdapter := zigbee2mqtt.NewAdapter(zigbee2mqtt.Config{
		IntegrationID: "z2m_e2e",
		BaseTopic:     "zigbee2mqtt",
	}, mockMQTT, repos.Devices, repos.Entities, bus)
	require.NoError(t, z2mAdapter.Start(ctx))
	defer z2mAdapter.Stop(ctx)

	esphomeAdapter := esphome_mqtt.NewAdapter(esphome_mqtt.Config{
		IntegrationID: "esphome_e2e",
		TopicPrefix:   "",
	}, mockMQTT, repos.Devices, repos.Entities, bus)
	require.NoError(t, esphomeAdapter.Start(ctx))
	defer esphomeAdapter.Stop(ctx)

	mockCloud := cloud.NewMockProvider("mock_cloud")
	cloudAdapter := cloud.NewAdapter(cloud.Config{
		IntegrationID:    "cloud_e2e",
		Provider:         mockCloud,
		FailureThreshold: 2,
		ResetTimeout:     100 * time.Millisecond,
	}, bus)
	require.NoError(t, cloudAdapter.Start(ctx))
	defer cloudAdapter.Stop(ctx)

	// 6. Setup Admin & Session for WebSocket and REST client
	adminUser, err := authService.SetupInitialAdmin(ctx, "admin", "AdminPassword123!")
	require.NoError(t, err)

	sess, _, err := authService.Login(ctx, adminUser.Username, "AdminPassword123!", "127.0.0.1", "test-agent")
	require.NoError(t, err)

	wsURL := fmt.Sprintf("ws://127.0.0.1:8901/ws?session=%s", url.QueryEscape(sess.ID))
	wsConn, _, err := gorilla.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer wsConn.Close()

	// Initial system.connected frame
	var initMsg wsinfra.Message
	err = wsConn.ReadJSON(&initMsg)
	require.NoError(t, err)
	assert.Equal(t, "system.connected", initMsg.Type)

	httpClient := &http.Client{Timeout: 3 * time.Second}

	// ---------------------------------------------------------
	// PART 1: Zigbee2MQTT Discovery, Telemetry, and Outbound Command
	// ---------------------------------------------------------
	t.Run("Zigbee2MQTT Flow", func(t *testing.T) {
		bridgeDevicesJSON := `[
			{
				"ieee_address": "0x00158d0001a1b2c3",
				"type": "Router",
				"friendly_name": "living_room_bulb",
				"definition": {
					"model": "TRADFRI-1000LM",
					"vendor": "IKEA",
					"description": "IKEA Smart Bulb",
					"exposes": [
						{
							"type": "light",
							"features": [
								{"name": "state", "property": "state", "type": "binary"},
								{"name": "brightness", "property": "brightness", "type": "numeric"}
							]
						}
					]
				}
			}
		]`

		// Publish discovery payload
		mockMQTT.SimulateIncoming("zigbee2mqtt/bridge/devices", []byte(bridgeDevicesJSON))

		// Allow processing
		time.Sleep(100 * time.Millisecond)

		// Verify device auto-created in SQLite
		dev, err := repos.Devices.GetByExternalID(ctx, "z2m_e2e", "0x00158d0001a1b2c3")
		require.NoError(t, err)
		assert.Equal(t, "living_room_bulb", dev.Name)
		assert.Equal(t, "IKEA", dev.Manufacturer)

		// Verify entity auto-created in SQLite
		ent, err := repos.Entities.GetByKey(ctx, "light.living_room_bulb")
		require.NoError(t, err)
		assert.Equal(t, "light", ent.Domain)
		assert.True(t, ent.Writable)

		// Publish telemetry
		mockMQTT.SimulateIncoming("zigbee2mqtt/living_room_bulb", []byte(`{"state":"ON","brightness":220}`))

		// WebSocket should receive state change
		wsFrame := readWSEvent(t, wsConn, string(events.EntityStateChanged), 3*time.Second)
		payloadMap := wsFrame.Payload.(map[string]any)
		assert.Equal(t, "light.living_room_bulb", payloadMap["entity_key"])
		assert.Contains(t, payloadMap["state"], `"state":"ON"`)
		assert.Contains(t, payloadMap["state"], `"brightness":220`)

		// Send REST command to turn off
		cmdURL := fmt.Sprintf("http://127.0.0.1:8901/api/v1/entities/%s/command", ent.ID)
		cmdBody, _ := json.Marshal(map[string]any{
			"service": "turn_off",
		})
		req, _ := http.NewRequest(http.MethodPost, cmdURL, bytes.NewReader(cmdBody))
		req.AddCookie(&http.Cookie{Name: domusmw.SessionCookieName, Value: sess.ID})
		req.AddCookie(&http.Cookie{Name: domusmw.CSRFCookieName, Value: csrfToken})
		req.Header.Set("X-CSRF-Token", csrfToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := httpClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusAccepted, resp.StatusCode)

		time.Sleep(100 * time.Millisecond)

		// Verify command published to zigbee2mqtt
		published := mockMQTT.GetPublished()
		var foundCmd bool
		for _, p := range published {
			if p.Topic == "zigbee2mqtt/living_room_bulb/set" {
				foundCmd = true
				assert.Contains(t, string(p.Payload), `"state":"OFF"`)
				break
			}
		}
		assert.True(t, foundCmd, "Z2M command must be published to zigbee2mqtt/living_room_bulb/set")
	})

	// ---------------------------------------------------------
	// PART 2: ESPHome MQTT Auto-Discovery, Telemetry, and Control
	// ---------------------------------------------------------
	t.Run("ESPHome Flow", func(t *testing.T) {
		// 1. ESPHome publishes sensor telemetry
		mockMQTT.SimulateIncoming("garden_node/sensor/soil_moisture/state", []byte("48.5"))

		time.Sleep(100 * time.Millisecond)

		// Verify device auto-created
		gardenDev, err := repos.Devices.GetByExternalID(ctx, "esphome_e2e", "garden_node")
		require.NoError(t, err)
		assert.Equal(t, "garden_node", gardenDev.Name)
		assert.Equal(t, "ESPHome", gardenDev.Manufacturer)

		// Verify entity auto-created
		sensorEnt, err := repos.Entities.GetByKey(ctx, "sensor.garden_node_soil_moisture")
		require.NoError(t, err)
		assert.Equal(t, "sensor", sensorEnt.Domain)
		assert.Equal(t, `{"value":48.5}`, sensorEnt.State)

		// WebSocket receives state change
		wsFrame := readWSEvent(t, wsConn, string(events.EntityStateChanged), 3*time.Second)
		payloadMap := wsFrame.Payload.(map[string]any)
		assert.Equal(t, "sensor.garden_node_soil_moisture", payloadMap["entity_key"])
		assert.Equal(t, `{"value":48.5}`, payloadMap["state"])

		// 2. ESPHome publishes switch state
		mockMQTT.SimulateIncoming("garden_node/switch/irrigation/state", []byte("OFF"))

		time.Sleep(100 * time.Millisecond)

		switchEnt, err := repos.Entities.GetByKey(ctx, "switch.garden_node_irrigation")
		require.NoError(t, err)
		assert.True(t, switchEnt.Writable)

		// Read websocket frame for switch state
		_ = readWSEvent(t, wsConn, string(events.EntityStateChanged), 3*time.Second)

		// 3. Send REST command to turn on irrigation
		cmdURL := fmt.Sprintf("http://127.0.0.1:8901/api/v1/entities/%s/command", switchEnt.ID)
		cmdBody, _ := json.Marshal(map[string]any{
			"service": "turn_on",
		})
		req, _ := http.NewRequest(http.MethodPost, cmdURL, bytes.NewReader(cmdBody))
		req.AddCookie(&http.Cookie{Name: domusmw.SessionCookieName, Value: sess.ID})
		req.AddCookie(&http.Cookie{Name: domusmw.CSRFCookieName, Value: csrfToken})
		req.Header.Set("X-CSRF-Token", csrfToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := httpClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusAccepted, resp.StatusCode)

		time.Sleep(100 * time.Millisecond)

		// Verify ESPHome command published
		published := mockMQTT.GetPublished()
		var foundESPCmd bool
		for _, p := range published {
			if p.Topic == "garden_node/switch/irrigation/command" {
				foundESPCmd = true
				assert.Equal(t, "ON", string(p.Payload))
				break
			}
		}
		assert.True(t, foundESPCmd, "ESPHome command must be published to garden_node/switch/irrigation/command")

		// 4. Send LWT offline status
		mockMQTT.SimulateIncoming("garden_node/status", []byte("offline"))

		time.Sleep(100 * time.Millisecond)

		updatedDev, err := repos.Devices.GetByID(ctx, gardenDev.ID)
		require.NoError(t, err)
		assert.False(t, updatedDev.Online, "Device must be marked offline upon receiving LWT offline")
	})

	// ---------------------------------------------------------
	// PART 3: Cloud Circuit Breaker & Graceful Offline Degradation
	// ---------------------------------------------------------
	t.Run("Cloud Circuit Breaker & Offline Degradation Flow", func(t *testing.T) {
		// Healthy execution
		require.NoError(t, cloudAdapter.HealthCheck(ctx))
		require.NoError(t, cloudAdapter.Execute(ctx, "cloud_plug", "turn_on", nil))

		// Simulate cloud outage
		mockCloud.SetOnline(false)

		// First failure
		_ = cloudAdapter.Execute(ctx, "cloud_plug", "turn_on", nil)
		// Second failure trips circuit breaker
		_ = cloudAdapter.Execute(ctx, "cloud_plug", "turn_on", nil)

		// Circuit breaker is open: fails fast with 0 blocking
		start := time.Now()
		err := cloudAdapter.Execute(ctx, "cloud_plug", "turn_on", nil)
		elapsed := time.Since(start)

		assert.ErrorIs(t, err, cloud.ErrCircuitBreakerOpen)
		assert.Less(t, elapsed, 10*time.Millisecond, "Open circuit breaker must fail fast immediately")

		// Read websocket frame for integration degraded state
		wsFrame := readWSEvent(t, wsConn, string(events.IntegrationStateChanged), 3*time.Second)
		payloadMap := wsFrame.Payload.(map[string]any)
		assert.Equal(t, "cloud_e2e", payloadMap["integration_id"])
		assert.Equal(t, "degraded", payloadMap["status"])
	})

	// 7. Clean Shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutdownCancel()
	err = srv.Shutdown(shutdownCtx)
	assert.NoError(t, err)
}

func readWSEvent(t *testing.T, wsConn *gorilla.Conn, eventType string, timeout time.Duration) wsinfra.Message {
	_ = wsConn.SetReadDeadline(time.Now().Add(timeout))
	for {
		var msg wsinfra.Message
		err := wsConn.ReadJSON(&msg)
		if err != nil {
			t.Fatalf("failed reading ws event (waiting for %s): %v", eventType, err)
		}
		if msg.Type == eventType {
			return msg
		}
	}
}
