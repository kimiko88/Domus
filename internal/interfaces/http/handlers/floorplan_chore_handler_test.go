package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appchore "github.com/domus-hub/domus/internal/application/chore"
	domainchore "github.com/domus-hub/domus/internal/domain/chore"
	"github.com/domus-hub/domus/internal/domain/floorplan"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockFloorplanRepo struct {
	floors  map[string]*floorplan.Floor
	markers map[string]*floorplan.Marker
}

func newMockFloorplanRepo() *mockFloorplanRepo {
	return &mockFloorplanRepo{
		floors:  make(map[string]*floorplan.Floor),
		markers: make(map[string]*floorplan.Marker),
	}
}

func (m *mockFloorplanRepo) CreateFloor(ctx context.Context, f *floorplan.Floor) error {
	m.floors[f.ID] = f
	return nil
}
func (m *mockFloorplanRepo) GetFloor(ctx context.Context, id string) (*floorplan.Floor, error) {
	if f, ok := m.floors[id]; ok {
		return f, nil
	}
	return nil, floorplan.ErrFloorNotFound
}
func (m *mockFloorplanRepo) ListFloors(ctx context.Context) ([]*floorplan.Floor, error) {
	var list []*floorplan.Floor
	for _, f := range m.floors {
		list = append(list, f)
	}
	return list, nil
}
func (m *mockFloorplanRepo) UpdateFloor(ctx context.Context, f *floorplan.Floor) error {
	m.floors[f.ID] = f
	return nil
}
func (m *mockFloorplanRepo) DeleteFloor(ctx context.Context, id string) error {
	delete(m.floors, id)
	return nil
}
func (m *mockFloorplanRepo) ListMarkers(ctx context.Context, floorID string) ([]*floorplan.Marker, error) {
	var list []*floorplan.Marker
	for _, mark := range m.markers {
		if mark.FloorID == floorID {
			list = append(list, mark)
		}
	}
	return list, nil
}
func (m *mockFloorplanRepo) SaveMarker(ctx context.Context, mark *floorplan.Marker) error {
	m.markers[mark.ID] = mark
	return nil
}
func (m *mockFloorplanRepo) DeleteMarker(ctx context.Context, id string) error {
	delete(m.markers, id)
	return nil
}

func TestFloorplanHandler_Endpoints(t *testing.T) {
	repo := newMockFloorplanRepo()
	h := NewFloorplanHandler(repo)

	r := chi.NewRouter()
	r.Route("/floors", func(fl chi.Router) {
		fl.Get("/", h.List)
		fl.Post("/", h.Create)
		fl.Get("/{id}", h.GetByID)
		fl.Put("/{id}", h.Update)
		fl.Delete("/{id}", h.Delete)
		fl.Get("/{id}/markers", h.ListMarkers)
		fl.Put("/{id}/markers", h.SaveMarker)
		fl.Delete("/{id}/markers/{markerId}", h.DeleteMarker)
	})

	// 1. List initially empty
	req := httptest.NewRequest(http.MethodGet, "/floors", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// 2. Create Floor (bad JSON)
	req = httptest.NewRequest(http.MethodPost, "/floors", bytes.NewReader([]byte("{invalid-json")))
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)

	// 3. Create Floor valid
	floorBody, _ := json.Marshal(map[string]any{"name": "Piano Terra", "level": 0})
	req = httptest.NewRequest(http.MethodPost, "/floors", bytes.NewReader(floorBody))
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusCreated, rr.Code)

	var createdFloor floorplan.Floor
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &createdFloor))
	assert.Equal(t, "Piano Terra", createdFloor.Name)
	assert.NotEmpty(t, createdFloor.ID)

	// 4. Get Floor By ID
	req = httptest.NewRequest(http.MethodGet, "/floors/"+createdFloor.ID, nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// 5. Update Floor
	updateBody, _ := json.Marshal(map[string]any{"name": "Piano Terra Rinnovato", "level": 0})
	req = httptest.NewRequest(http.MethodPut, "/floors/"+createdFloor.ID, bytes.NewReader(updateBody))
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// 6. Save Marker
	markerBody, _ := json.Marshal(map[string]any{
		"entity_key": "light.living_room",
		"x_percent":  25.5,
		"y_percent":  40.0,
		"show_badge": true,
	})
	req = httptest.NewRequest(http.MethodPut, "/floors/"+createdFloor.ID+"/markers", bytes.NewReader(markerBody))
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	var savedMarker floorplan.Marker
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &savedMarker))
	assert.Equal(t, "light.living_room", savedMarker.EntityKey)
	assert.NotEmpty(t, savedMarker.ID)

	// 7. List Markers
	req = httptest.NewRequest(http.MethodGet, "/floors/"+createdFloor.ID+"/markers", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	var markers []*floorplan.Marker
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &markers))
	assert.Len(t, markers, 1)

	// 8. Delete Marker
	req = httptest.NewRequest(http.MethodDelete, "/floors/"+createdFloor.ID+"/markers/"+savedMarker.ID, nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusNoContent, rr.Code)

	// 9. Delete Floor
	req = httptest.NewRequest(http.MethodDelete, "/floors/"+createdFloor.ID, nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusNoContent, rr.Code)
}

type mockChoreHandlerRepo struct {
	items map[string]*domainchore.Chore
}

func newMockChoreHandlerRepo() *mockChoreHandlerRepo {
	return &mockChoreHandlerRepo{items: make(map[string]*domainchore.Chore)}
}
func (m *mockChoreHandlerRepo) Create(ctx context.Context, c *domainchore.Chore) error {
	m.items[c.ID] = c
	return nil
}
func (m *mockChoreHandlerRepo) GetByID(ctx context.Context, id string) (*domainchore.Chore, error) {
	if c, ok := m.items[id]; ok {
		return c, nil
	}
	return nil, domainchore.ErrChoreNotFound
}
func (m *mockChoreHandlerRepo) List(ctx context.Context, f domainchore.Filter) ([]*domainchore.Chore, error) {
	var list []*domainchore.Chore
	for _, c := range m.items {
		if f.Status != "" && c.Status != f.Status {
			continue
		}
		list = append(list, c)
	}
	return list, nil
}
func (m *mockChoreHandlerRepo) Update(ctx context.Context, c *domainchore.Chore) error {
	m.items[c.ID] = c
	return nil
}
func (m *mockChoreHandlerRepo) Delete(ctx context.Context, id string) error {
	delete(m.items, id)
	return nil
}
func (m *mockChoreHandlerRepo) GetPendingDueBefore(ctx context.Context, cutoff time.Time) ([]*domainchore.Chore, error) {
	return nil, nil
}

func TestChoreHandler_Endpoints(t *testing.T) {
	repo := newMockChoreHandlerRepo()
	svc := appchore.NewService(repo, nil)
	h := NewChoreHandler(repo, svc)

	r := chi.NewRouter()
	r.Route("/chores", func(ch chi.Router) {
		ch.Get("/", h.List)
		ch.Post("/", h.Create)
		ch.Get("/{id}", h.GetByID)
		ch.Put("/{id}", h.Update)
		ch.Delete("/{id}", h.Delete)
		ch.Post("/{id}/complete", h.Complete)
	})

	// 1. Create Chore
	choreBody, _ := json.Marshal(map[string]any{
		"title":           "Pulire i pavimenti",
		"priority":        "high",
		"recurrence_type": "weekly",
	})
	req := httptest.NewRequest(http.MethodPost, "/chores", bytes.NewReader(choreBody))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusCreated, rr.Code)

	var created domainchore.Chore
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &created))
	assert.Equal(t, "Pulire i pavimenti", created.Title)
	assert.NotEmpty(t, created.ID)

	// 2. List Chores
	req = httptest.NewRequest(http.MethodGet, "/chores?status=pending", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// 3. Get By ID
	req = httptest.NewRequest(http.MethodGet, "/chores/"+created.ID, nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// 4. Update Chore
	updBody, _ := json.Marshal(map[string]any{"title": "Pulire i pavimenti a fondo", "priority": "urgent"})
	req = httptest.NewRequest(http.MethodPut, "/chores/"+created.ID, bytes.NewReader(updBody))
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// 5. Complete Chore
	req = httptest.NewRequest(http.MethodPost, "/chores/"+created.ID+"/complete", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// 6. Delete Chore
	req = httptest.NewRequest(http.MethodDelete, "/chores/"+created.ID, nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusNoContent, rr.Code)
}
