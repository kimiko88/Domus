package e2e_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/application/auth"
	appauto "github.com/domus-hub/domus/internal/application/automation"
	"github.com/domus-hub/domus/internal/application/energy"
	"github.com/domus-hub/domus/internal/application/presence"
	domainchore "github.com/domus-hub/domus/internal/domain/chore"
	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/domain/floorplan"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/crypto"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	serverpkg "github.com/domus-hub/domus/internal/interfaces/http"
	domusmw "github.com/domus-hub/domus/internal/interfaces/http/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdvancedFeatures_EndToEnd(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "advanced_e2e.db")

	cfg := config.Config{
		Server: config.ServerConfig{
			Host: "127.0.0.1",
			Port: 8999,
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

	// 1. Initialize SQLite database & migrations
	db, err := sqlite.Open(cfg.Database)
	require.NoError(t, err)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	require.NoError(t, sqlite.Migrate(ctx, db))

	repos := sqlite.NewRepositories(db)

	// 2. Event Bus
	bus := eventbus.NewMemoryBus(eventbus.Config{
		BufferSize: 256,
		Workers:    2,
	})
	defer bus.Close()

	// 3. Auth Service
	hasher := crypto.NewArgon2Hasher(crypto.Argon2Params{
		Memory:      32 * 1024,
		Iterations:  1,
		Parallelism: 1,
		SaltLength:  16,
		KeyLength:   32,
	})
	authSvc := auth.NewService(repos.Users, repos.Sessions, repos.Audit, hasher, 24*time.Hour)
	rateLimiter := auth.NewRateLimiter(50, 1*time.Minute)

	// 4. Advanced Services Wiring
	energyCfg := energy.LoadSheddingConfig{
		Enabled:          true,
		MeterEntityKey:   "sensor.power_total",
		MaxPowerWatts:    3000.0,
		OverloadDuration: 1,
		RecoveryWatts:    2200.0,
		RecoveryDelaySec: 2,
		PriorityLoads:    []string{"switch.boiler", "switch.air_con"},
	}
	energyCtrl := energy.NewController(energyCfg, bus)

	geofence := presence.HomeGeofence{
		Latitude:  45.4642,
		Longitude: 9.1900,
		RadiusM:   150.0,
	}
	presenceSvc := presence.NewService(geofence, bus)

	sunEngine := appauto.NewSunEngine(45.4642, 9.1900)

	// 5. Build HTTP Server
	srv := serverpkg.NewServer(
		cfg.Server,
		"1.0.0-e2e",
		time.Now(),
		db,
		serverpkg.WithAuth(authSvc, rateLimiter),
		serverpkg.WithRepositories(repos, bus),
		serverpkg.WithAdvancedFeatures(energyCtrl, presenceSvc, sunEngine),
	)

	router := srv.Router()

	// Step A: Setup admin
	setupPayload, _ := json.Marshal(map[string]string{
		"username": "admin",
		"password": "SuperSecretPassword123!",
	})
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/setup", bytes.NewReader(setupPayload))
	rr := &responseRecorder{headers: make(http.Header)}
	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusCreated, rr.code)

	// Step B: Login to get session cookie
	loginPayload, _ := json.Marshal(map[string]string{
		"username": "admin",
		"password": "SuperSecretPassword123!",
	})
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginPayload))
	rr = &responseRecorder{headers: make(http.Header)}
	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.code)

	var sessionCookie *http.Cookie
	fakeResp := http.Response{Header: rr.headers}
	for _, c := range fakeResp.Cookies() {
		if c.Name == domusmw.SessionCookieName {
			sessionCookie = c
			break
		}
	}
	require.NotNil(t, sessionCookie, "Session cookie must be returned upon login")

	// Test CSRF token: 16 random bytes encoded as 32 hexadecimal characters.
	const csrfToken = "0123456789abcdef0123456789abcdef"

	// Helper to send HTTP requests with session cookie
	doReq := func(method, path string, body []byte) *responseRecorder {
		var req *http.Request
		if body != nil {
			req, _ = http.NewRequest(method, path, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
		} else {
			req, _ = http.NewRequest(method, path, nil)
		}
		req.AddCookie(sessionCookie)
		req.AddCookie(&http.Cookie{Name: domusmw.CSRFCookieName, Value: csrfToken})
		req.Header.Set("X-CSRF-Token", csrfToken)
		rec := &responseRecorder{headers: make(http.Header)}
		router.ServeHTTP(rec, req)
		return rec
	}

	// ==========================================
	// FLOW A: Floorplan & House Map Management
	// ==========================================
	t.Run("Flow A: Floorplan and Room Markers", func(t *testing.T) {
		// 1. Create a Floor
		floorPayload, _ := json.Marshal(map[string]any{
			"name":       "Piano Terra",
			"level":      0,
			"plan_svg":   "<svg><rect width='100' height='100'/></svg>",
			"sort_order": 1,
		})
		res := doReq(http.MethodPost, "/api/v1/floors", floorPayload)
		require.Equal(t, http.StatusCreated, res.code)

		var floor floorplan.Floor
		require.NoError(t, json.Unmarshal(res.body, &floor))
		assert.Equal(t, "Piano Terra", floor.Name)
		assert.Equal(t, 0, floor.Level)

		// 2. Add an Entity Marker on the floor
		markerPayload, _ := json.Marshal(map[string]any{
			"entity_key": "light.living_room",
			"x_percent":  35.5,
			"y_percent":  72.0,
			"show_badge": true,
		})
		res = doReq(http.MethodPut, "/api/v1/floors/"+floor.ID+"/markers", markerPayload)
		require.Equal(t, http.StatusOK, res.code)

		var marker floorplan.Marker
		require.NoError(t, json.Unmarshal(res.body, &marker))
		assert.Equal(t, "light.living_room", marker.EntityKey)
		assert.Equal(t, 35.5, marker.XPercent)
		assert.Equal(t, 72.0, marker.YPercent)

		// 3. Query all markers for this floor
		res = doReq(http.MethodGet, "/api/v1/floors/"+floor.ID+"/markers", nil)
		require.Equal(t, http.StatusOK, res.code)
		var markers []*floorplan.Marker
		require.NoError(t, json.Unmarshal(res.body, &markers))
		require.Len(t, markers, 1)
		assert.Equal(t, marker.ID, markers[0].ID)
	})

	// ==========================================
	// FLOW B: Chores Lifecycle & Notifications
	// ==========================================
	t.Run("Flow B: Chores Lifecycle and Automated Rescheduling", func(t *testing.T) {
		var notificationReceived bool
		var nWg sync.WaitGroup
		var once sync.Once
		nWg.Add(1)

		unsub := bus.Subscribe(events.NotificationCreated, func(ctx context.Context, evt events.Event) error {
			once.Do(func() {
				notificationReceived = true
				nWg.Done()
			})
			return nil
		})
		defer unsub()

		// 1. Create a recurring weekly chore
		dueDate := time.Now().UTC().Add(24 * time.Hour)
		chorePayload, _ := json.Marshal(map[string]any{
			"title":           "Pulizia filtri deumidificatore",
			"priority":        "high",
			"recurrence_type": "weekly",
			"due_date":        dueDate.Format(time.RFC3339),
		})
		res := doReq(http.MethodPost, "/api/v1/chores", chorePayload)
		require.Equal(t, http.StatusCreated, res.code)

		var createdChore domainchore.Chore
		require.NoError(t, json.Unmarshal(res.body, &createdChore))
		assert.Equal(t, "Pulizia filtri deumidificatore", createdChore.Title)
		assert.Equal(t, domainchore.StatusPending, createdChore.Status)

		// 2. Complete the chore
		res = doReq(http.MethodPost, "/api/v1/chores/"+createdChore.ID+"/complete", nil)
		require.Equal(t, http.StatusOK, res.code)

		nWg.Wait()
		assert.True(t, notificationReceived, "Completing chore must trigger notification event")

		// 3. List chores: must have original completed + new pending instance scheduled +7 days
		res = doReq(http.MethodGet, "/api/v1/chores", nil)
		require.Equal(t, http.StatusOK, res.code)

		var choresList []*domainchore.Chore
		require.NoError(t, json.Unmarshal(res.body, &choresList))
		require.Len(t, choresList, 2)

		var hasCompleted, hasNextPending bool
		for _, c := range choresList {
			if c.ID == createdChore.ID && c.Status == domainchore.StatusCompleted {
				hasCompleted = true
			}
			if c.ID != createdChore.ID && c.Status == domainchore.StatusPending && c.RecurrenceType == domainchore.RecurrenceWeekly {
				hasNextPending = true
			}
		}
		assert.True(t, hasCompleted, "Original chore must be marked completed")
		assert.True(t, hasNextPending, "Next recurring chore instance must be spawned")
	})

	// ==========================================
	// FLOW C: Energy Load Shedding & Overload
	// ==========================================
	t.Run("Flow C: Energy Load Shedding", func(t *testing.T) {
		// 1. Query initial status
		res := doReq(http.MethodGet, "/api/v1/energy/load-shedding", nil)
		require.Equal(t, http.StatusOK, res.code)

		var status map[string]any
		require.NoError(t, json.Unmarshal(res.body, &status))
		assert.True(t, status["enabled"].(bool))
		assert.Equal(t, 3000.0, status["max_power_watts"])

		// 2. Simulate overload reading
		now := time.Now().UTC()
		// First reading initiates overload timer
		_ = energyCtrl.ProcessPowerReading(ctx, 3500.0, now)
		// Second reading 2 seconds later exceeds duration (1s) and sheds lowest priority load
		err = energyCtrl.ProcessPowerReading(ctx, 3500.0, now.Add(2*time.Second))
		require.NoError(t, err)

		statusAfter := energyCtrl.GetStatus()
		assert.True(t, statusAfter["is_overloaded"].(bool))
		shedded := statusAfter["shedded_loads"].([]string)
		assert.Contains(t, shedded, "switch.boiler")

		// 3. Power drops below recovery threshold
		_ = energyCtrl.ProcessPowerReading(ctx, 1800.0, now.Add(3*time.Second))
		_ = energyCtrl.ProcessPowerReading(ctx, 1800.0, now.Add(6*time.Second))

		statusRecovered := energyCtrl.GetStatus()
		assert.False(t, statusRecovered["is_overloaded"].(bool))
	})

	// ==========================================
	// FLOW D: Multi-Sensor Presence Detection
	// ==========================================
	t.Run("Flow D: Presence Geofencing and Transitions", func(t *testing.T) {
		// 1. First GPS update inside geofence triggers transition to JustArrived
		gpsInside, _ := json.Marshal(map[string]any{
			"latitude":  45.4642,
			"longitude": 9.1900,
		})
		res := doReq(http.MethodPost, "/api/v1/presence/gps", gpsInside)
		require.Equal(t, http.StatusOK, res.code)

		var respInside map[string]any
		require.NoError(t, json.Unmarshal(res.body, &respInside))
		assert.NotEmpty(t, respInside["user_id"])
		assert.Equal(t, string(presence.StateJustArrived), respInside["state"])

		// 2. Second GPS update confirms user is firmly Home
		res = doReq(http.MethodPost, "/api/v1/presence/gps", gpsInside)
		require.Equal(t, http.StatusOK, res.code)
		var respHome map[string]any
		require.NoError(t, json.Unmarshal(res.body, &respHome))
		assert.Equal(t, string(presence.StateHome), respHome["state"])

		// 3. Query presence summary: anyone_home must be true
		res = doReq(http.MethodGet, "/api/v1/presence/status", nil)
		require.Equal(t, http.StatusOK, res.code)
		var summary map[string]any
		require.NoError(t, json.Unmarshal(res.body, &summary))
		assert.True(t, summary["anyone_home"].(bool))

		// 4. GPS update outside geofence triggers transition to JustLeft
		gpsOutside, _ := json.Marshal(map[string]any{
			"latitude":  41.9028,
			"longitude": 12.4964,
		})
		res = doReq(http.MethodPost, "/api/v1/presence/gps", gpsOutside)
		require.Equal(t, http.StatusOK, res.code)

		var respOutside map[string]any
		require.NoError(t, json.Unmarshal(res.body, &respOutside))
		assert.Equal(t, string(presence.StateJustLeft), respOutside["state"])

		// 5. Subsequent GPS update outside confirms Away
		res = doReq(http.MethodPost, "/api/v1/presence/gps", gpsOutside)
		require.Equal(t, http.StatusOK, res.code)
		var respAway map[string]any
		require.NoError(t, json.Unmarshal(res.body, &respAway))
		assert.Equal(t, string(presence.StateAway), respAway["state"])
	})

	// ==========================================
	// FLOW E: Astronomical Solar Position API
	// ==========================================
	t.Run("Flow E: Astronomical Solar Position", func(t *testing.T) {
		res := doReq(http.MethodGet, "/api/v1/system/sun?time=2026-06-21T12:00:00Z", nil)
		require.Equal(t, http.StatusOK, res.code)

		var pos appauto.SunPosition
		require.NoError(t, json.Unmarshal(res.body, &pos))
		assert.True(t, pos.IsDay, "Noon on summer solstice must be daylight")
		assert.Greater(t, pos.Elevation, 50.0, "Elevation at solar noon in Milan must be high")
		assert.NotZero(t, pos.Sunrise)
		assert.NotZero(t, pos.Sunset)
	})
}
