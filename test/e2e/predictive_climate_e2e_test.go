// Package e2e_test provides end-to-end tests for the Domus Smart Home platform.
package e2e_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	appclimate "github.com/domus-hub/domus/internal/application/climate"
	appauth "github.com/domus-hub/domus/internal/application/auth"
	"github.com/domus-hub/domus/internal/infrastructure/crypto"
	domclimate "github.com/domus-hub/domus/internal/domain/climate"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	serverpkg "github.com/domus-hub/domus/internal/interfaces/http"
	domusmw "github.com/domus-hub/domus/internal/interfaces/http/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPredictiveClimate_EndToEnd(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "predictive_climate_e2e.db")

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
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "text",
		},
	}

	db, err := sqlite.Open(cfg.Database)
	require.NoError(t, err)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err = sqlite.Migrate(ctx, db)
	require.NoError(t, err)

	bus := eventbus.NewMemoryBus(eventbus.Config{
		BufferSize: 256,
		Workers:    2,
	})
	defer bus.Close()

	repos := sqlite.NewRepositories(db)
	hasher := crypto.NewArgon2Hasher(crypto.Argon2Params{
		Memory: 32 * 1024, Iterations: 1, Parallelism: 1, SaltLength: 16, KeyLength: 32,
	})
	authSvc := appauth.NewService(repos.Users, repos.Sessions, repos.Audit, hasher, 24*time.Hour)
	_, err = authSvc.SetupInitialAdmin(ctx, "e2e_admin", "E2ESecret123!")
	require.NoError(t, err)
	session, _, err := authSvc.Login(ctx, "e2e_admin", "E2ESecret123!", "127.0.0.1", "predictive-e2e")
	require.NoError(t, err)

	srv := serverpkg.NewServer(cfg.Server, "1.0.0-test", time.Now(), db,
		serverpkg.WithRepositories(repos, bus),
		serverpkg.WithAuth(authSvc, appauth.NewRateLimiter(100, time.Minute)),
	)

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	client := ts.Client()
	doRequest := func(req *http.Request) (*http.Response, error) {
		req.AddCookie(&http.Cookie{Name: domusmw.SessionCookieName, Value: session.ID})
		return client.Do(req)
	}
	post := func(url string, body []byte) (*http.Response, error) {
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
		if err != nil { return nil, err }
		req.Header.Set("Content-Type", "application/json")
		return doRequest(req)
	}
	get := func(url string) (*http.Response, error) {
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil { return nil, err }
		return doRequest(req)
	}

	// STEP 1: Create a multi-zone predictive room (South-facing Living Room with Radiant Floor)
	zonePayload := domclimate.PredictiveZoneConfig{
		Name:                 "Living Room Floor Heating",
		TRVEntityKey:         "climate.living_radiant_floor",
		TempSensorKey:        "sensor.living_room_temperature",
		PressureSensorKey:    "sensor.living_room_barometer",
		HumiditySensorKey:    "sensor.living_room_humidity",
		WindowSensorKey:      "binary_sensor.living_room_balcony",
		TargetTemperature:    21.0,
		Orientation:          domclimate.OrientationSouth,
		WindowAreaSqm:        7.5,
		EmitterType:          domclimate.EmitterFloorRadiant,
		ThermalInertiaHours:  4.0,
		PreheatEnabled:       true,
		SolarCutoffEnabled:   true,
		PressureBoostEnabled: true,
		Enabled:              true,
	}
	bodyBytes, _ := json.Marshal(zonePayload)
	resp, err := post(ts.URL+"/api/v1/climate/predictive/zones", bodyBytes)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var createdZone domclimate.PredictiveZoneConfig
	err = json.NewDecoder(resp.Body).Decode(&createdZone)
	resp.Body.Close()
	require.NoError(t, err)
	assert.NotEmpty(t, createdZone.ID)
	assert.Equal(t, "Living Room Floor Heating", createdZone.Name)
	assert.Equal(t, domclimate.EmitterFloorRadiant, createdZone.EmitterType)

	// STEP 2: Trigger manual evaluation on the created zone
	resp, err = post(ts.URL+"/api/v1/climate/predictive/zones/"+createdZone.ID+"/evaluate", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var eval domclimate.PredictiveEvaluation
	err = json.NewDecoder(resp.Body).Decode(&eval)
	resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, createdZone.ID, eval.ZoneID)
	assert.Greater(t, eval.CalculatedSetpoint, 15.0)
	assert.NotEmpty(t, eval.Reason)
	assert.NotEmpty(t, eval.Mode)

	// STEP 3: Fetch real-time system dashboard status
	resp, err = get(ts.URL + "/api/v1/climate/predictive/status")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var status appclimate.SystemPredictiveStatus
	err = json.NewDecoder(resp.Body).Decode(&status)
	resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, 1, status.TotalZones)
	assert.Equal(t, 1, status.EnabledZones)
	assert.Len(t, status.Zones, 1)
	assert.Equal(t, createdZone.ID, status.Zones[0].Config.ID)
	assert.NotNil(t, status.Zones[0].LatestEvaluation)

	// STEP 4: Fetch historical evaluations for this zone
	resp, err = get(ts.URL + "/api/v1/climate/predictive/zones/" + createdZone.ID + "/evaluations")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var evalsRes map[string]any
	err = json.NewDecoder(resp.Body).Decode(&evalsRes)
	resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, float64(1), evalsRes["count"])

	// STEP 5: Thermal Profile Auto-learning & Optimal Start/Stop
	// 5.1 Fetch initial default thermal profile for Radiant Floor
	resp, err = get(ts.URL + "/api/v1/climate/predictive/zones/" + createdZone.ID + "/profile")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var initialProfile domclimate.ThermalProfile
	err = json.NewDecoder(resp.Body).Decode(&initialProfile)
	resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, createdZone.ID, initialProfile.ZoneID)
	assert.Equal(t, 0.6, initialProfile.HeatRatePerHr)
	assert.Equal(t, 0, initialProfile.SamplesCount)

	// 5.2 Train model with active heating cycle observation
	trainCycle := domclimate.ThermalCycleObservation{
		CycleType:       "heating",
		StartTemp:       19.0,
		EndTemp:         20.8,
		DurationMinutes: 60,
	}
	trainBytes, _ := json.Marshal(trainCycle)
	resp, err = post(ts.URL+"/api/v1/climate/predictive/zones/"+createdZone.ID+"/train", trainBytes)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var trainedProfile domclimate.ThermalProfile
	err = json.NewDecoder(resp.Body).Decode(&trainedProfile)
	resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, 1, trainedProfile.SamplesCount)
	assert.Greater(t, trainedProfile.HeatRatePerHr, 0.6)

	// 5.3 Optimal Start Recommendation
	targetTime := time.Now().UTC().Add(2 * time.Hour).Format(time.RFC3339)
	resp, err = get(ts.URL + "/api/v1/climate/predictive/zones/" + createdZone.ID + "/optimal-start?target_temp=21.0&target_time=" + targetTime)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var preheatRec domclimate.PreheatRecommendation
	err = json.NewDecoder(resp.Body).Decode(&preheatRec)
	resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, createdZone.ID, preheatRec.ZoneID)
	assert.Greater(t, preheatRec.PreheatMinutes, 0)
	assert.NotEmpty(t, preheatRec.Reason)

	// 5.4 Optimal Stop (Coast-Down) Recommendation
	resp, err = get(ts.URL + "/api/v1/climate/predictive/zones/" + createdZone.ID + "/optimal-stop")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var coastRec domclimate.CoastDownRecommendation
	err = json.NewDecoder(resp.Body).Decode(&coastRec)
	resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, createdZone.ID, coastRec.ZoneID)
	assert.GreaterOrEqual(t, coastRec.CoastDownMinutes, 15)
	assert.Greater(t, coastRec.EnergySavedKwh, 0.0)
	assert.NotEmpty(t, coastRec.Reason)

	// STEP 6: Delete zone
	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/v1/climate/predictive/zones/"+createdZone.ID, nil)
	resp, err = doRequest(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}
