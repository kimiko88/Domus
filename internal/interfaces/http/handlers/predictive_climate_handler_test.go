package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	appclimate "github.com/domus-hub/domus/internal/application/climate"
	appweather "github.com/domus-hub/domus/internal/application/weather"
	domclimate "github.com/domus-hub/domus/internal/domain/climate"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	"github.com/domus-hub/domus/internal/interfaces/http/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupPredictiveHandlerTest(t *testing.T) (*chi.Mux, *appclimate.PredictiveService) {
	t.Helper()
	db, err := sqlite.Open(config.DatabaseConfig{
		Path:    ":memory:",
		WALMode: false,
	})
	require.NoError(t, err)

	ctx := context.Background()
	err = sqlite.Migrate(ctx, db)
	require.NoError(t, err)

	repo := sqlite.NewPredictiveClimateRepository(db)
	weatherCli := appweather.NewClient()
	svc := appclimate.NewPredictiveService(repo, nil, weatherCli, nil, nil)
	handler := handlers.NewPredictiveClimateHandler(svc)

	r := chi.NewRouter()
	r.Route("/api/v1/climate/predictive", func(sub chi.Router) {
		handler.RegisterRoutes(sub)
	})
	return r, svc
}

func TestPredictiveClimateHandler_Flow(t *testing.T) {
	router, _ := setupPredictiveHandlerTest(t)

	// 1. Create Zone
	zonePayload := domclimate.PredictiveZoneConfig{
		Name:                 "Cucina Sud",
		TRVEntityKey:         "climate.valvola_cucina",
		TempSensorKey:        "sensor.cucina_temp",
		PressureSensorKey:    "sensor.cucina_pressione",
		TargetTemperature:    20.5,
		Orientation:          domclimate.OrientationSouth,
		WindowAreaSqm:        4.0,
		EmitterType:          domclimate.EmitterRadiator,
		ThermalInertiaHours:  2.0,
		PreheatEnabled:       true,
		SolarCutoffEnabled:   true,
		PressureBoostEnabled: true,
		Enabled:              true,
	}
	body, _ := json.Marshal(zonePayload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/climate/predictive/zones", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	var created domclimate.PredictiveZoneConfig
	err := json.NewDecoder(rec.Body).Decode(&created)
	require.NoError(t, err)
	assert.NotEmpty(t, created.ID)
	assert.Equal(t, "Cucina Sud", created.Name)

	// 2. List Zones
	req = httptest.NewRequest(http.MethodGet, "/api/v1/climate/predictive/zones", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var listRes map[string]any
	err = json.NewDecoder(rec.Body).Decode(&listRes)
	require.NoError(t, err)
	assert.Equal(t, float64(1), listRes["total"])

	// 3. Evaluate Zone
	req = httptest.NewRequest(http.MethodPost, "/api/v1/climate/predictive/zones/"+created.ID+"/evaluate", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var eval domclimate.PredictiveEvaluation
	err = json.NewDecoder(rec.Body).Decode(&eval)
	require.NoError(t, err)
	assert.Equal(t, created.ID, eval.ZoneID)
	assert.Greater(t, eval.CalculatedSetpoint, 15.0)

	// 4. Status
	req = httptest.NewRequest(http.MethodGet, "/api/v1/climate/predictive/status", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var status appclimate.SystemPredictiveStatus
	err = json.NewDecoder(rec.Body).Decode(&status)
	require.NoError(t, err)
	assert.Equal(t, 1, status.TotalZones)
	assert.Equal(t, 1, status.EnabledZones)

	// 5. Delete Zone
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/climate/predictive/zones/"+created.ID, nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}
