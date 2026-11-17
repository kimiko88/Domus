package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appclimate "github.com/domus-hub/domus/internal/application/climate"
	appweather "github.com/domus-hub/domus/internal/application/weather"
	domclimate "github.com/domus-hub/domus/internal/domain/climate"
	"github.com/domus-hub/domus/internal/domain/entity"
	"github.com/domus-hub/domus/internal/domain/user"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	"github.com/domus-hub/domus/internal/interfaces/http/handlers"
	domusmw "github.com/domus-hub/domus/internal/interfaces/http/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupPredictiveHandlerTest(t *testing.T) (*chi.Mux, *sqlite.EntityRepository) {
	t.Helper()
	db, err := sqlite.Open(config.DatabaseConfig{
		Path:    ":memory:",
		WALMode: false,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	ctx := context.Background()
	err = sqlite.Migrate(ctx, db)
	require.NoError(t, err)

	entities := sqlite.NewEntityRepository(db)
	now := time.Now().UTC()
	for _, seeded := range []*entity.Entity{
		{
			ID: "predictive-handler-room-temperature", EntityKey: "sensor.cucina_temp",
			Domain: "sensor", Name: "Temperatura cucina", UnitOfMeasurement: "°C",
			State: "19.0", Attributes: "{}", Available: true,
			LastChangedAt: now, LastUpdatedAt: now,
		},
		{
			ID: "predictive-handler-room-pressure", EntityKey: "sensor.cucina_pressione",
			Domain: "sensor", Name: "Pressione cucina", UnitOfMeasurement: "hPa",
			State: "1013.25", Attributes: "{}", Available: true,
			LastChangedAt: now, LastUpdatedAt: now,
		},
	} {
		require.NoError(t, entities.Create(ctx, seeded))
	}

	repo := sqlite.NewPredictiveClimateRepository(db)
	weatherCli := appweather.NewClient()
	svc := appclimate.NewPredictiveService(repo, entities, weatherCli, nil, nil)
	handler := handlers.NewPredictiveClimateHandler(svc)

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		admin := &user.User{ID: "test-admin", Role: user.RoleAdmin}
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			next.ServeHTTP(w, req.WithContext(domusmw.WithUser(req.Context(), admin)))
		})
	})
	r.Route("/api/v1/climate/predictive", func(sub chi.Router) {
		handler.RegisterRoutes(sub)
	})
	return r, entities
}

func requirePredictiveHTTPStatus(t *testing.T, rec *httptest.ResponseRecorder, expected int) {
	t.Helper()
	require.Equalf(t, expected, rec.Code, "HTTP response body: %s", rec.Body.String())
}

func TestPredictiveClimateHandler_Flow(t *testing.T) {
	router, entities := setupPredictiveHandlerTest(t)

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

	requirePredictiveHTTPStatus(t, rec, http.StatusCreated)
	var created domclimate.PredictiveZoneConfig
	err := json.NewDecoder(rec.Body).Decode(&created)
	require.NoError(t, err)
	assert.NotEmpty(t, created.ID)
	assert.Equal(t, "Cucina Sud", created.Name)

	// 2. List Zones
	req = httptest.NewRequest(http.MethodGet, "/api/v1/climate/predictive/zones", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	requirePredictiveHTTPStatus(t, rec, http.StatusOK)
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

	// 5. Get Thermal Profile
	req = httptest.NewRequest(http.MethodGet, "/api/v1/climate/predictive/zones/"+created.ID+"/profile", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	var profile domclimate.ThermalProfile
	err = json.NewDecoder(rec.Body).Decode(&profile)
	require.NoError(t, err)
	assert.Equal(t, created.ID, profile.ZoneID)
	assert.Equal(t, 1.2, profile.HeatRatePerHr)

	// 6. Train Zone from real observation
	trainPayload := domclimate.ThermalCycleObservation{
		CycleType:       "heating",
		StartTemp:       19.0,
		EndTemp:         20.5,
		DurationMinutes: 60,
	}
	trainBody, _ := json.Marshal(trainPayload)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/climate/predictive/zones/"+created.ID+"/train", bytes.NewReader(trainBody))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	var trainedProfile domclimate.ThermalProfile
	err = json.NewDecoder(rec.Body).Decode(&trainedProfile)
	require.NoError(t, err)
	assert.Equal(t, 1, trainedProfile.SamplesCount)
	assert.Greater(t, trainedProfile.HeatRatePerHr, 1.2)

	// 7. Get Optimal Start
	req = httptest.NewRequest(http.MethodGet, "/api/v1/climate/predictive/zones/"+created.ID+"/optimal-start?target_temp=21.0", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	var preheat domclimate.PreheatRecommendation
	err = json.NewDecoder(rec.Body).Decode(&preheat)
	require.NoError(t, err)
	assert.Equal(t, created.ID, preheat.ZoneID)
	assert.Greater(t, preheat.PreheatMinutes, 0)
	assert.NotEmpty(t, preheat.Reason)

	// Coast-down is only useful once the room has reached its comfort target.
	require.NoError(t, entities.UpdateState(
		context.Background(),
		"predictive-handler-room-temperature",
		"20.5",
		"{}",
		true,
		time.Now().UTC(),
	))

	// 8. Get Optimal Stop
	req = httptest.NewRequest(http.MethodGet, "/api/v1/climate/predictive/zones/"+created.ID+"/optimal-stop", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	var coast domclimate.CoastDownRecommendation
	err = json.NewDecoder(rec.Body).Decode(&coast)
	require.NoError(t, err)
	assert.Equal(t, created.ID, coast.ZoneID)
	assert.GreaterOrEqual(t, coast.CoastDownMinutes, 15)
	assert.NotEmpty(t, coast.Reason)

	// 9. Delete Zone
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/climate/predictive/zones/"+created.ID, nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}
