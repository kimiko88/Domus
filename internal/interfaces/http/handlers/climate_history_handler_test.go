package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appclimatehistory "github.com/domus-hub/domus/internal/application/climate_history"
	domclimatehistory "github.com/domus-hub/domus/internal/domain/climate_history"
	"github.com/domus-hub/domus/internal/interfaces/http/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func TestClimateHistoryHandlerSettingsHistoryAndExport(t *testing.T) {
	repos, _ := setupTestEnvironment(t)
	ctx := context.Background()
	service := appclimatehistory.NewService(repos.ClimateHistory, repos.Entities, nil)
	handler := handlers.NewClimateHistoryHandler(service)

	router := chi.NewRouter()
	router.Get("/settings", handler.GetSettings)
	router.Put("/settings", handler.UpdateSettings)
	router.Get("/history", handler.GetHistory)
	router.Get("/summary", handler.GetSummary)
	router.Get("/export.csv", handler.ExportCSV)

	t.Run("settings defaults and update", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/settings", nil)
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		require.Equal(t, http.StatusOK, res.Code)
		var settings domclimatehistory.Settings
		require.NoError(t, json.Unmarshal(res.Body.Bytes(), &settings))
		require.Equal(t, 10, settings.IntervalMinutes)

		settings.IntervalMinutes = 5
		body, err := json.Marshal(settings)
		require.NoError(t, err)
		req = httptest.NewRequest(http.MethodPut, "/settings", bytes.NewReader(body))
		res = httptest.NewRecorder()
		router.ServeHTTP(res, req)
		require.Equal(t, http.StatusOK, res.Code)
	})

	now := time.Now().UTC().Truncate(time.Second)
	temp, humidity, target := 21.2, 52.0, 22.0
	require.NoError(t, repos.ClimateHistory.SaveSamples(ctx, []domclimatehistory.Sample{{
		EntityID:"sensor-living", EntityKey:"sensor.living_temperature", EntityName:"Living temperature",
		AreaID:"living", RecordedAt:now, Temperature:&temp, Humidity:&humidity, TargetTemperature:&target,
	}}))

	t.Run("history and summary", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/history?area_id=living", nil)
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		require.Equal(t, http.StatusOK, res.Code)
		require.Contains(t, res.Body.String(), "temperature_average")

		req = httptest.NewRequest(http.MethodGet, "/summary?from="+now.Add(-time.Hour).Format(time.RFC3339)+"&to="+now.Add(time.Hour).Format(time.RFC3339), nil)
		res = httptest.NewRecorder()
		router.ServeHTTP(res, req)
		require.Equal(t, http.StatusOK, res.Code)
		require.Contains(t, res.Body.String(), "living")
	})

	t.Run("CSV export and invalid range", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/export.csv?area_id=living", nil)
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		require.Equal(t, http.StatusOK, res.Code)
		require.Contains(t, res.Header().Get("Content-Type"), "text/csv")
		require.Contains(t, res.Body.String(), "temperature_avg_c")

		req = httptest.NewRequest(http.MethodGet, "/history?from=not-a-date", nil)
		res = httptest.NewRecorder()
		router.ServeHTTP(res, req)
		require.Equal(t, http.StatusBadRequest, res.Code)
	})
}
