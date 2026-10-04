package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/domain/entity"
	"github.com/domus-hub/domus/internal/domain/history"
	"github.com/domus-hub/domus/internal/interfaces/http/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockWSClientCounter struct {
	count int
}

func (m *mockWSClientCounter) ClientCount() int {
	return m.count
}

func TestHistoryHandler(t *testing.T) {
	repos, bus := setupTestEnvironment(t)
	ctx := context.Background()

	// Create test entity
	testEntity := &entity.Entity{
		ID:        "ent_hist_test",
		EntityKey: "sensor.room_temp",
		Domain:    "sensor",
		Name:      "Room Temperature",
		Writable:  false,
		Available: true,
	}
	require.NoError(t, repos.Entities.Create(ctx, testEntity))

	// Insert history entries
	now := time.Now().UTC().Truncate(time.Second)
	entries := []*history.StateHistory{
		{EntityID: testEntity.ID, State: `{"temp": 20.0}`, RecordedAt: now.Add(-2 * time.Hour)},
		{EntityID: testEntity.ID, State: `{"temp": 21.0}`, RecordedAt: now.Add(-1 * time.Hour)},
		{EntityID: testEntity.ID, State: `{"temp": 22.5}`, RecordedAt: now},
	}
	require.NoError(t, repos.History.SaveBatch(ctx, entries))

	handler := handlers.NewHistoryHandler(repos.History, repos.Entities)

	r := chi.NewRouter()
	r.Get("/entities/{id}/history", handler.GetEntityHistory)

	t.Run("returns history for entity by UUID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/entities/ent_hist_test/history", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var res []map[string]any
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &res))
		assert.Len(t, res, 3)
	})

	t.Run("returns history for entity by key", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/entities/sensor.room_temp/history", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var res []map[string]any
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &res))
		assert.Len(t, res, 3)
	})

	t.Run("filters by time window and limit", func(t *testing.T) {
		fromStr := now.Add(-90 * time.Minute).Format(time.RFC3339)
		req := httptest.NewRequest(http.MethodGet, "/entities/sensor.room_temp/history?from="+fromStr+"&limit=1", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var res []map[string]any
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &res))
		assert.Len(t, res, 1)
	})

	t.Run("returns 404 for unknown entity", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/entities/unknown_ent/history", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	_ = bus
}

func TestSystemMetricsHandler(t *testing.T) {
	repos, _ := setupTestEnvironment(t)
	ctx := context.Background()

	require.NoError(t, repos.Entities.Create(ctx, &entity.Entity{
		ID:        "ent_1",
		EntityKey: "switch.desk",
		Domain:    "switch",
		Name:      "Desk",
	}))

	startedAt := time.Now().Add(-60 * time.Second)
	wsCounter := &mockWSClientCounter{count: 3}
	metricsHandler := handlers.NewSystemMetricsHandler(startedAt, repos, wsCounter, true)

	req := httptest.NewRequest(http.MethodGet, "/system/metrics", nil)
	rr := httptest.NewRecorder()

	metricsHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))

	assert.Equal(t, "ok", body["status"])
	assert.GreaterOrEqual(t, body["uptime_s"].(float64), float64(60))
	assert.Equal(t, float64(1), body["entity_count"])
	assert.Equal(t, float64(0), body["device_count"])
	assert.Equal(t, float64(3), body["ws_clients"])
	assert.Equal(t, true, body["mqtt_connected"])
}
