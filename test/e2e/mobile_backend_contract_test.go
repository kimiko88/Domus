// Package e2e_test provides end-to-end contract validation for the native mobile apps (iOS & Android).
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
	appauto "github.com/domus-hub/domus/internal/application/automation"
	"github.com/domus-hub/domus/internal/application/energy"
	"github.com/domus-hub/domus/internal/application/presence"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/crypto"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	serverpkg "github.com/domus-hub/domus/internal/interfaces/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMobileBackendContract_EndToEnd(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "domus_mobile_e2e.db")

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
			RateLimitLoginPerMin: 50,
			Argon2Time:           1,
			Argon2Memory:         32 * 1024,
			Argon2Threads:        1,
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "text",
		},
	}

	db, err := sqlite.Open(cfg.Database)
	require.NoError(t, err)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = sqlite.Migrate(ctx, db)
	require.NoError(t, err)

	repos := sqlite.NewRepositories(db)
	bus := eventbus.NewMemoryBus(eventbus.Config{
		BufferSize: 256,
		Workers:    2,
	})
	defer bus.Close()

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

	geofence := presence.HomeGeofence{
		Latitude:  45.4642,
		Longitude: 9.1900,
		RadiusM:   150.0,
	}
	presenceSvc := presence.NewService(geofence, bus)
	energyCtrl := energy.NewController(energy.LoadSheddingConfig{}, bus)
	sunEngine := appauto.NewSunEngine(45.4642, 9.1900)

	const version = "1.0.0-mobile-e2e"
	startedAt := time.Now()
	srv := serverpkg.NewServer(
		cfg.Server,
		version,
		startedAt,
		db,
		serverpkg.WithAuth(authService, loginLimiter),
		serverpkg.WithRepositories(repos, bus),
		serverpkg.WithAdvancedFeatures(energyCtrl, presenceSvc, sunEngine),
	)
	router := srv.Router()

	// 1. Initial Setup: Register First Admin (matching mobile setup flow)
	registerPayload := map[string]string{
		"username": "mobile_admin",
		"password": "Password1234!",
	}
	regBytes, _ := json.Marshal(registerPayload)
	regReq, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/setup", bytes.NewReader(regBytes))
	regReq.Header.Set("Content-Type", "application/json")
	regRec := &responseRecorder{headers: make(http.Header)}
	router.ServeHTTP(regRec, regReq)

	assert.Equal(t, http.StatusCreated, regRec.code)

	// 2. Mobile App Login: POST /api/v1/auth/login
	loginPayload := map[string]string{
		"username": "mobile_admin",
		"password": "Password1234!",
	}
	loginBytes, _ := json.Marshal(loginPayload)
	loginReq, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBytes))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := &responseRecorder{headers: make(http.Header)}
	router.ServeHTTP(loginRec, loginReq)

	require.Equal(t, http.StatusOK, loginRec.code)

	var loginResp map[string]any
	err = json.Unmarshal(loginRec.body, &loginResp)
	require.NoError(t, err)
	token, ok := loginResp["token"].(string)
	require.True(t, ok, "Token must be returned in mobile login response")
	require.NotEmpty(t, token)

	// 3. Mobile Snapshot: GET /api/v1/entities with Bearer Token
	entitiesReq, _ := http.NewRequest(http.MethodGet, "/api/v1/entities", nil)
	entitiesReq.Header.Set("Authorization", "Bearer "+token)
	entitiesRec := &responseRecorder{headers: make(http.Header)}
	router.ServeHTTP(entitiesRec, entitiesReq)

	require.Equal(t, http.StatusOK, entitiesRec.code)
	var entitiesList []map[string]any
	err = json.Unmarshal(entitiesRec.body, &entitiesList)
	require.NoError(t, err)

	// 4. Mobile Areas: GET /api/v1/areas with Bearer Token
	areasReq, _ := http.NewRequest(http.MethodGet, "/api/v1/areas", nil)
	areasReq.Header.Set("Authorization", "Bearer "+token)
	areasRec := &responseRecorder{headers: make(http.Header)}
	router.ServeHTTP(areasRec, areasReq)

	require.Equal(t, http.StatusOK, areasRec.code)
	var areasList []map[string]any
	err = json.Unmarshal(areasRec.body, &areasList)
	require.NoError(t, err)

	// 5. Mobile Background Geofence Presence: POST /api/v1/presence/gps with Bearer Token
	gpsPayload := map[string]any{
		"latitude":  45.4642,
		"longitude": 9.1900,
		"accuracy":  15.5,
		"source":    "native_ios_android_geofence",
	}
	gpsBytes, _ := json.Marshal(gpsPayload)
	gpsReq, _ := http.NewRequest(http.MethodPost, "/api/v1/presence/gps", bytes.NewReader(gpsBytes))
	gpsReq.Header.Set("Authorization", "Bearer "+token)
	gpsReq.Header.Set("Content-Type", "application/json")
	gpsRec := &responseRecorder{headers: make(http.Header)}
	router.ServeHTTP(gpsRec, gpsReq)

	assert.True(t, gpsRec.code == http.StatusOK || gpsRec.code == http.StatusNoContent || gpsRec.code == http.StatusCreated)
}
