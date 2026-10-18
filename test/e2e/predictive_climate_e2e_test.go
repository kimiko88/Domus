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
	domclimate "github.com/domus-hub/domus/internal/domain/climate"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	serverpkg "github.com/domus-hub/domus/internal/interfaces/http"
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
	srv := serverpkg.NewServer(cfg.Server, "1.0.0-test", time.Now(), db,
		serverpkg.WithRepositories(repos, bus),
	)

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	client := ts.Client()

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
	resp, err := client.Post(ts.URL+"/api/v1/climate/predictive/zones", "application/json", bytes.NewReader(bodyBytes))
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
	resp, err = client.Post(ts.URL+"/api/v1/climate/predictive/zones/"+createdZone.ID+"/evaluate", "application/json", nil)
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
	resp, err = client.Get(ts.URL + "/api/v1/climate/predictive/status")
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
	resp, err = client.Get(ts.URL + "/api/v1/climate/predictive/zones/" + createdZone.ID + "/evaluations")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var evalsRes map[string]any
	err = json.NewDecoder(resp.Body).Decode(&evalsRes)
	resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, float64(1), evalsRes["count"])

	// STEP 5: Delete zone
	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/v1/climate/predictive/zones/"+createdZone.ID, nil)
	resp, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}
