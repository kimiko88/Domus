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
	"github.com/domus-hub/domus/internal/domain/area"
	"github.com/domus-hub/domus/internal/domain/entity"
	"github.com/domus-hub/domus/internal/domain/user"
	domusmw "github.com/domus-hub/domus/internal/interfaces/http/middleware"
	"github.com/domus-hub/domus/internal/interfaces/http/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func TestClimateHistoryReadEndpointsRespectAreaPermissions(t *testing.T) {
	repos, _ := setupTestEnvironment(t)
	ctx := context.Background()
	require.NoError(t, repos.Areas.Create(ctx, &area.Area{ID: "bedroom", Name: "Bedroom"}))
	require.NoError(t, repos.Areas.Create(ctx, &area.Area{ID: "living", Name: "Living room"}))
	allowed := &entity.Entity{ID: "climate-bedroom", EntityKey: "sensor.bedroom_temperature", Domain: "sensor", Name: "Bedroom temperature", State: "20", Attributes: "{}", Available: true, AreaID: "bedroom"}
	denied := &entity.Entity{ID: "climate-living", EntityKey: "sensor.living_temperature", Domain: "sensor", Name: "Living temperature", State: "unavailable", Attributes: "{\"device_class\":\"temperature\"}", Available: false, AreaID: "living"}
	require.NoError(t, repos.Entities.Create(ctx, allowed))
	require.NoError(t, repos.Entities.Create(ctx, denied))
	moved := &entity.Entity{ID: "climate-moved", EntityKey: "sensor.moved_temperature", Domain: "sensor", Name: "Moved temperature", State: "20", Attributes: "{}", Available: true, AreaID: "bedroom"}
	require.NoError(t, repos.Entities.Create(ctx, moved))
	now := time.Now().UTC().Truncate(time.Second)
	bedroomTemp, livingTemp := 20.0, 23.0
	require.NoError(t, repos.ClimateHistory.SaveSamples(ctx, []domclimatehistory.Sample{
		{EntityID: allowed.ID, EntityKey: allowed.EntityKey, EntityName: allowed.Name, AreaID: allowed.AreaID, RecordedAt: now, Temperature: &bedroomTemp},
		{EntityID: denied.ID, EntityKey: denied.EntityKey, EntityName: denied.Name, AreaID: denied.AreaID, RecordedAt: now, Temperature: &livingTemp},
		{EntityID: moved.ID, EntityKey: moved.EntityKey, EntityName: moved.Name, AreaID: "living", RecordedAt: now.Add(-time.Minute), Temperature: &livingTemp},
		{EntityID: moved.ID, EntityKey: moved.EntityKey, EntityName: moved.Name, AreaID: "bedroom", RecordedAt: now, Temperature: &bedroomTemp},
	}))

	service := appclimatehistory.NewService(repos.ClimateHistory, repos.Entities, nil)
	handler := handlers.NewClimateHistoryHandler(service, repos.Entities)
	router := chi.NewRouter()
	router.Get("/history", handler.GetHistory)
	router.Get("/summary", handler.GetSummary)
	router.Get("/export.csv", handler.ExportCSV)

	kid := &user.User{
		ID: "kid-user", Role: user.RoleKid,
		Permissions: &user.UserPermissions{
			Role: user.RoleKid, AllowedAreaIDs: []string{"bedroom"},
			AllowedEntityIDs: nil,
		},
	}
	request := func(path string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req = req.WithContext(domusmw.WithUser(req.Context(), kid))
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		return res
	}

	history := request("/history")
	require.Equal(t, http.StatusOK, history.Code)
	require.Contains(t, history.Body.String(), "Bedroom temperature")
	require.Contains(t, history.Body.String(), "Moved temperature")
	require.Equal(t, 1, strings.Count(history.Body.String(), "Moved temperature"))
	require.NotContains(t, history.Body.String(), "Living temperature")
	require.NotContains(t, history.Body.String(), "AreaID": "living")

	summary := request("/summary?from=" + now.Add(-time.Hour).Format(time.RFC3339) + "&to=" + now.Add(time.Hour).Format(time.RFC3339))
	require.Equal(t, http.StatusOK, summary.Code)
	require.Contains(t, summary.Body.String(), "bedroom")
	require.NotContains(t, summary.Body.String(), "living")

	export := request("/export.csv")
	require.Equal(t, http.StatusOK, export.Code)
	require.Contains(t, export.Body.String(), "Bedroom temperature")
	require.NotContains(t, export.Body.String(), "Living temperature")
}

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

	formulaValue := "=HYPERLINK(\"https://example.invalid\",\"sensor\")"
	require.NoError(t, repos.ClimateHistory.SaveSamples(ctx, []domclimatehistory.Sample{{
		EntityID: "sensor-untrusted", EntityKey: "sensor.untrusted", EntityName: formulaValue,
		AreaID: "living", RecordedAt: now.Add(time.Second), Temperature: &temp,
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
		require.Contains(t, res.Body.String(), "'=HYPERLINK")

		req = httptest.NewRequest(http.MethodGet, "/history?from=not-a-date", nil)
		res = httptest.NewRecorder()
		router.ServeHTTP(res, req)
		require.Equal(t, http.StatusBadRequest, res.Code)
	})
}
