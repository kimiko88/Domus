// Package e2e_test provides end-to-end integration tests for Automations and Scenes in Domus.
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
	appauto "github.com/domus-hub/domus/internal/application/automation"
	appscene "github.com/domus-hub/domus/internal/application/scene"
	"github.com/domus-hub/domus/internal/domain/automation"
	"github.com/domus-hub/domus/internal/domain/device"
	"github.com/domus-hub/domus/internal/domain/entity"
	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/domain/integration"
	"github.com/domus-hub/domus/internal/domain/scene"
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
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAutomation_And_Scene_EndToEnd(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "ms5_e2e.db")

	cfg := config.Config{
		Server: config.ServerConfig{
			Host: "127.0.0.1",
			Port: 8903, // Dedicated port for MS-05 E2E test
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

	// 1. Database & Migrations
	db, err := sqlite.Open(cfg.Database)
	require.NoError(t, err)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	require.NoError(t, sqlite.Migrate(ctx, db))

	// 2. Repositories and Services
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

	// 3. Event Bus, Engine, and WebSocket Hub
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 256, Workers: 2})
	defer bus.Close()

	engine := appauto.NewEngine(repos.Automations, repos.Entities, bus)
	require.NoError(t, engine.Start(ctx))
	defer engine.Stop(ctx)

	sceneService := appscene.NewService(repos.Scenes, bus)

	wsHub := wsinfra.NewHub(bus)
	ctxHub, cancelHub := context.WithCancel(context.Background())
	defer cancelHub()
	go wsHub.Run(ctxHub)

	// 4. HTTP Server
	srv := serverpkg.NewServer(
		cfg.Server,
		"1.0.0-e2e",
		time.Now(),
		db,
		serverpkg.WithAuth(authService, loginLimiter),
		serverpkg.WithRepositories(repos, bus),
		serverpkg.WithWebSocket(wsHub),
		serverpkg.WithAutomation(engine, sceneService),
	)

	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- srv.Start()
	}()
	time.Sleep(100 * time.Millisecond)

	// 5. Setup MQTT Adapter with MockClient
	mockMQTT := mqtt.NewMockClient()
	now := time.Now().UTC()
	in := &integration.Integration{
		ID:        "mqtt_e2e",
		Type:      "mqtt_generic",
		Name:      "MQTT E2E",
		Enabled:   true,
		Status:    "connected",
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, repos.Integrations.Create(ctx, in))

	mqttAdapter := mqtt_generic.NewAdapter(mqtt_generic.Config{
		IntegrationID:        in.ID,
		StateTopicPattern:    "domus/+/state",
		CommandTopicTemplate: "domus/%s/set",
	}, mockMQTT, repos.Entities, repos.Devices, bus)
	require.NoError(t, mqttAdapter.Start(ctx))
	defer mqttAdapter.Stop(ctx)

	// 6. Setup Admin & Session
	adminUser, err := authService.SetupInitialAdmin(ctx, "admin", "AdminPassword123!")
	require.NoError(t, err)

	sess, _, err := authService.Login(ctx, adminUser.Username, "AdminPassword123!", "127.0.0.1", "test-agent")
	require.NoError(t, err)

	// Connect WebSocket Client
	wsURL := fmt.Sprintf("ws://127.0.0.1:8903/ws?session=%s", url.QueryEscape(sess.ID))
	wsConn, _, err := gorilla.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer wsConn.Close()

	// Initial connected frame
	var initMsg wsinfra.Message
	err = wsConn.ReadJSON(&initMsg)
	require.NoError(t, err)
	assert.Equal(t, "system.connected", initMsg.Type)

	// 7. Seed Devices and Entities
	dev := &device.Device{
		ID:            "dev-hallway",
		IntegrationID: in.ID,
		ExternalID:    "hw_01",
		Name:          "Hallway Controller",
		Online:        true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	require.NoError(t, repos.Devices.Create(ctx, dev))

	// Motion PIR sensor
	motionEnt := &entity.Entity{
		ID:            uuid.NewString(),
		DeviceID:      dev.ID,
		EntityKey:     "binary_sensor.hallway_pir",
		Domain:        "binary_sensor",
		Name:          "Hallway PIR Motion",
		State:         `{"state":false}`,
		Available:     true,
		LastChangedAt: now,
		LastUpdatedAt: now,
	}
	require.NoError(t, repos.Entities.Create(ctx, motionEnt))

	// Light sensor (ambient lux)
	luxEnt := &entity.Entity{
		ID:                uuid.NewString(),
		DeviceID:          dev.ID,
		EntityKey:         "sensor.hallway_lux",
		Domain:            "sensor",
		Name:              "Hallway Ambient Light",
		UnitOfMeasurement: "lx",
		State:             `{"value":20.0}`, // Dark (under 50 lux threshold)
		Available:         true,
		LastChangedAt:     now,
		LastUpdatedAt:     now,
	}
	require.NoError(t, repos.Entities.Create(ctx, luxEnt))

	// Hallway light
	lightEnt := &entity.Entity{
		ID:            uuid.NewString(),
		DeviceID:      dev.ID,
		EntityKey:     "light.hallway_ceiling",
		Domain:        "light",
		Name:          "Hallway Ceiling Light",
		Writable:      true,
		State:         `{"state":"OFF"}`,
		Available:     true,
		LastChangedAt: now,
		LastUpdatedAt: now,
	}
	require.NoError(t, repos.Entities.Create(ctx, lightEnt))

	// 8. Create Automation:
	// Trigger: binary_sensor.hallway_pir == true
	// Condition: sensor.hallway_lux < 50
	// Action: light.hallway_ceiling -> turn_on brightness 180
	luxBelow := 50.0
	autoRule := &automation.Automation{
		ID:          "auto-motion-light",
		Name:        "Hallway Night Motion Light",
		Description: "Turn on hallway light if motion detected and room is dark",
		Enabled:     true,
		Triggers: []automation.Trigger{
			{
				Platform:  "state",
				EntityKey: "binary_sensor.hallway_pir",
				To:        "true",
			},
		},
		Conditions: []automation.Condition{
			{
				Condition: "numeric_state",
				EntityKey: "sensor.hallway_lux",
				Below:     &luxBelow,
			},
		},
		Actions: []automation.Action{
			{
				Type:            "service",
				Service:         "turn_on",
				TargetEntityKey: "light.hallway_ceiling",
				Params:          map[string]any{"brightness": 180},
			},
		},
		ExecutionMode: "single",
	}
	require.NoError(t, repos.Automations.Create(ctx, autoRule))

	// -------------------------------------------------------------
	// TEST FLOW 1: Motion event triggers automation -> turns on light
	// -------------------------------------------------------------
	t.Run("Automation execution on sensor state trigger", func(t *testing.T) {
		// Ingest motion detected via MQTT
		mockMQTT.SimulateIncoming("domus/binary_sensor.hallway_pir/state", []byte(`{"state":true}`))

		// WebSocket receives automation.triggered
		wsTrig := readWSEvent(t, wsConn, string(events.AutomationTriggered), 3*time.Second)
		trigPayload := wsTrig.Payload.(map[string]any)
		assert.Equal(t, autoRule.ID, trigPayload["automation_id"])

		// WebSocket receives automation.run_completed
		wsCompleted := readWSEvent(t, wsConn, string(events.AutomationRunCompleted), 3*time.Second)
		compPayload := wsCompleted.Payload.(map[string]any)
		assert.Equal(t, autoRule.ID, compPayload["automation_id"])
		assert.Equal(t, "completed", compPayload["status"])

		time.Sleep(100 * time.Millisecond)

		// Verify MQTT outbound command was published to light
		published := mockMQTT.GetPublished()
		var foundLightCmd bool
		for _, p := range published {
			if p.Topic == "domus/light.hallway_ceiling/set" {
				foundLightCmd = true
				assert.Contains(t, string(p.Payload), `"turn_on"`)
				break
			}
		}
		assert.True(t, foundLightCmd, "Automation action must dispatch MQTT command to light.hallway_ceiling")

		// Verify run record stored in SQLite
		runs, err := repos.Automations.ListRuns(ctx, autoRule.ID, 5)
		require.NoError(t, err)
		require.Len(t, runs, 1)
		assert.Equal(t, automation.RunStatusCompleted, runs[0].Status)
	})

	// -------------------------------------------------------------
	// TEST FLOW 2: Scene Creation & Activation
	// -------------------------------------------------------------
	t.Run("Scene Creation and Activation via REST API", func(t *testing.T) {
		httpClient := &http.Client{Timeout: 3 * time.Second}

		// 1. Create Scene via REST
		sceneURL := "http://127.0.0.1:8903/api/v1/scenes"
		sceneBody, _ := json.Marshal(map[string]any{
			"name": "Sleep Mode",
			"icon": "bed",
			"actions": []map[string]any{
				{
					"entity_key": "light.hallway_ceiling",
					"service":    "turn_off",
				},
			},
		})
		req, _ := http.NewRequest(http.MethodPost, sceneURL, bytes.NewReader(sceneBody))
		req.AddCookie(&http.Cookie{Name: domusmw.SessionCookieName, Value: sess.ID})
		req.Header.Set("Content-Type", "application/json")

		resp, err := httpClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		var createdScene scene.Scene
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&createdScene))
		assert.Equal(t, "Sleep Mode", createdScene.Name)

		// 2. Apply Scene via REST
		applyURL := fmt.Sprintf("http://127.0.0.1:8903/api/v1/scenes/%s/apply", createdScene.ID)
		applyReq, _ := http.NewRequest(http.MethodPost, applyURL, nil)
		applyReq.AddCookie(&http.Cookie{Name: domusmw.SessionCookieName, Value: sess.ID})

		applyResp, err := httpClient.Do(applyReq)
		require.NoError(t, err)
		defer applyResp.Body.Close()
		assert.Equal(t, http.StatusOK, applyResp.StatusCode)

		// 3. WebSocket receives scene.activated
		wsScene := readWSEvent(t, wsConn, string(events.SceneActivated), 3*time.Second)
		scenePayload := wsScene.Payload.(map[string]any)
		assert.Equal(t, createdScene.ID, scenePayload["scene_id"])
		assert.Equal(t, "Sleep Mode", scenePayload["name"])

		// 4. Verify MQTT command was dispatched for scene entity
		assert.Eventually(t, func() bool {
			published := mockMQTT.GetPublished()
			for _, p := range published {
				if p.Topic == "domus/light.hallway_ceiling/set" && bytes.Contains(p.Payload, []byte("turn_off")) {
					return true
				}
			}
			return false
		}, 1*time.Second, 20*time.Millisecond, "Scene activation must publish turn_off command to light")
	})

	// 9. Shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutdownCancel()
	err = srv.Shutdown(shutdownCtx)
	assert.NoError(t, err)
}
