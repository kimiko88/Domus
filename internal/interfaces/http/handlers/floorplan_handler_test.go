package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/domus-hub/domus/internal/domain/floorplan"
	"github.com/domus-hub/domus/internal/interfaces/http/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFloorplanHandler_Lifecycle(t *testing.T) {
	repos, _ := setupTestEnvironment(t)
	handler := handlers.NewFloorplanHandler(repos.Floors)

	r := chi.NewRouter()
	r.Get("/floors", handler.List)
	r.Post("/floors", handler.Create)
	r.Get("/floors/{id}", handler.GetByID)
	r.Put("/floors/{id}", handler.Update)
	r.Delete("/floors/{id}", handler.Delete)
	r.Get("/floors/{id}/markers", handler.ListMarkers)
	r.Put("/floors/{id}/markers", handler.SaveMarker)
	r.Delete("/floors/{id}/markers/{markerId}", handler.DeleteMarker)

	// 1. Create a Floor (Piano Terra, Level 0)
	createBody, _ := json.Marshal(map[string]any{
		"name":       "Piano Terra",
		"level":      0,
		"plan_svg":   "<svg><rect width='100' height='100'/></svg>",
		"sort_order": 0,
	})
	req := httptest.NewRequest(http.MethodPost, "/floors", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	var created floorplan.Floor
	require.NoError(t, json.NewDecoder(w.Body).Decode(&created))
	assert.NotEmpty(t, created.ID)
	assert.Equal(t, "Piano Terra", created.Name)
	assert.Equal(t, 0, created.Level)

	// 2. Get Floor by ID
	req = httptest.NewRequest(http.MethodGet, "/floors/"+created.ID, nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var fetched floorplan.Floor
	require.NoError(t, json.NewDecoder(w.Body).Decode(&fetched))
	assert.Equal(t, created.ID, fetched.ID)

	// 3. Update Floor
	updateBody, _ := json.Marshal(map[string]any{
		"name":       "Piano Terra Ristrutturato",
		"level":      0,
		"sort_order": 1,
	})
	req = httptest.NewRequest(http.MethodPut, "/floors/"+created.ID, bytes.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var updated floorplan.Floor
	require.NoError(t, json.NewDecoder(w.Body).Decode(&updated))
	assert.Equal(t, "Piano Terra Ristrutturato", updated.Name)

	// 4. Save a Marker on the Floor
	markerBody, _ := json.Marshal(map[string]any{
		"entity_key": "light.living_room_ceiling",
		"x_percent":  45.5,
		"y_percent":  70.2,
		"show_badge": true,
	})
	req = httptest.NewRequest(http.MethodPut, "/floors/"+created.ID+"/markers", bytes.NewReader(markerBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var marker floorplan.Marker
	require.NoError(t, json.NewDecoder(w.Body).Decode(&marker))
	assert.NotEmpty(t, marker.ID)
	assert.Equal(t, 45.5, marker.XPercent)
	assert.Equal(t, 70.2, marker.YPercent)

	// 5. List Markers on Floor
	req = httptest.NewRequest(http.MethodGet, "/floors/"+created.ID+"/markers", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var markers []*floorplan.Marker
	require.NoError(t, json.NewDecoder(w.Body).Decode(&markers))
	assert.Len(t, markers, 1)
	assert.Equal(t, "light.living_room_ceiling", markers[0].EntityKey)

	// 6. Delete Marker
	req = httptest.NewRequest(http.MethodDelete, "/floors/"+created.ID+"/markers/"+marker.ID, nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)

	// 7. List Markers again -> should be empty
	req = httptest.NewRequest(http.MethodGet, "/floors/"+created.ID+"/markers", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var emptyMarkers []*floorplan.Marker
	require.NoError(t, json.NewDecoder(w.Body).Decode(&emptyMarkers))
	assert.Empty(t, emptyMarkers)

	// 8. Delete Floor
	req = httptest.NewRequest(http.MethodDelete, "/floors/"+created.ID, nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}
