// Package handlers_test provides unit tests for Area, Device, and Entity HTTP handlers (TDD red-to-green phase).
package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	appauto "github.com/domus-hub/domus/internal/application/automation"
	"github.com/domus-hub/domus/internal/domain/device"
	"github.com/domus-hub/domus/internal/domain/entity"
	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/domain/integration"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	"github.com/domus-hub/domus/internal/interfaces/http/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestEnvironment(t *testing.T) (*sqlite.Repositories, *eventbus.MemoryBus) {
	t.Helper()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "handlers_test.db")

	db, err := sqlite.Open(config.DatabaseConfig{
		Path:          dbPath,
		WALMode:       true,
		BusyTimeoutMs: 5000,
		MaxOpenConns:  1,
	})
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, sqlite.Migrate(ctx, db))

	t.Cleanup(func() { _ = db.Close() })

	repos := sqlite.NewRepositories(db)
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 64, Workers: 2})
	t.Cleanup(func() { _ = bus.Close() })

	return repos, bus
}

func TestAreaHandler_ListAndCreate(t *testing.T) {
	repos, _ := setupTestEnvironment(t)
	handler := handlers.NewAreaHandler(repos.Areas)

	// 1. Create Area
	payload, _ := json.Marshal(map[string]any{
		"id":   "kitchen",
		"name": "Kitchen",
		"icon": "mdi:chef-hat",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/areas", bytes.NewReader(payload))
	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)

	// 2. List Areas
	reqList := httptest.NewRequest(http.MethodGet, "/api/v1/areas", nil)
	rrList := httptest.NewRecorder()
	handler.List(rrList, reqList)

	assert.Equal(t, http.StatusOK, rrList.Code)
	var list []map[string]any
	require.NoError(t, json.Unmarshal(rrList.Body.Bytes(), &list))
	assert.Len(t, list, 1)
	assert.Equal(t, "Kitchen", list[0]["name"])
}

func TestAreaHandler_UpdateAndDelete(t *testing.T) {
	repos, _ := setupTestEnvironment(t)
	handler := handlers.NewAreaHandler(repos.Areas)

	// Create
	payload, _ := json.Marshal(map[string]any{
		"id":   "living",
		"name": "Living Room",
		"icon": "sofa",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/areas", bytes.NewReader(payload))
	rr := httptest.NewRecorder()
	handler.Create(rr, req)
	assert.Equal(t, http.StatusCreated, rr.Code)

	r := chi.NewRouter()
	r.Put("/api/v1/areas/{id}", handler.Update)
	r.Delete("/api/v1/areas/{id}", handler.Delete)

	// Update
	updatePayload, _ := json.Marshal(map[string]any{
		"name":       "Salon",
		"icon":       "tv",
		"sort_order": 5,
	})
	reqUpdate := httptest.NewRequest(http.MethodPut, "/api/v1/areas/living", bytes.NewReader(updatePayload))
	rrUpdate := httptest.NewRecorder()
	r.ServeHTTP(rrUpdate, reqUpdate)
	assert.Equal(t, http.StatusOK, rrUpdate.Code)

	// Verify update
	var updated map[string]any
	require.NoError(t, json.Unmarshal(rrUpdate.Body.Bytes(), &updated))
	assert.Equal(t, "Salon", updated["name"])
	assert.Equal(t, "tv", updated["icon"])

	// Delete
	reqDelete := httptest.NewRequest(http.MethodDelete, "/api/v1/areas/living", nil)
	rrDelete := httptest.NewRecorder()
	r.ServeHTTP(rrDelete, reqDelete)
	assert.Equal(t, http.StatusNoContent, rrDelete.Code)

	// Delete again should return 404
	reqDeleteNotFound := httptest.NewRequest(http.MethodDelete, "/api/v1/areas/living", nil)
	rrDeleteNotFound := httptest.NewRecorder()
	r.ServeHTTP(rrDeleteNotFound, reqDeleteNotFound)
	assert.Equal(t, http.StatusNotFound, rrDeleteNotFound.Code)
}

func TestDeviceHandler_ListAndGet(t *testing.T) {
	repos, _ := setupTestEnvironment(t)
	ctx := context.Background()

	// Seed integration and device
	in := &integration.Integration{
		ID:        "in-1",
		Type:      "mqtt",
		Name:      "MQTT",
		Status:    "connected",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, repos.Integrations.Create(ctx, in))

	dev := &device.Device{
		ID:            uuid.NewString(),
		IntegrationID: in.ID,
		ExternalID:    "sensor-1",
		Name:          "Temp Sensor",
		Online:        true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	require.NoError(t, repos.Devices.Create(ctx, dev))

	handler := handlers.NewDeviceHandler(repos.Devices)

	// List
	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil)
	rr := httptest.NewRecorder()
	handler.List(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var list []map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &list))
	assert.Len(t, list, 1)
	assert.Equal(t, "Temp Sensor", list[0]["name"])

	// Get by ID using Chi URLParam context
	r := chi.NewRouter()
	r.Get("/api/v1/devices/{id}", handler.GetByID)

	reqGet := httptest.NewRequest(http.MethodGet, "/api/v1/devices/"+dev.ID, nil)
	rrGet := httptest.NewRecorder()
	r.ServeHTTP(rrGet, reqGet)

	assert.Equal(t, http.StatusOK, rrGet.Code)
	var fetched map[string]any
	require.NoError(t, json.Unmarshal(rrGet.Body.Bytes(), &fetched))
	assert.Equal(t, dev.ID, fetched["id"])
}

func TestEntityHandler_CommandEmitsEvent(t *testing.T) {
	repos, bus := setupTestEnvironment(t)
	ctx := context.Background()

	ent := &entity.Entity{
		ID:            uuid.NewString(),
		EntityKey:     "light.ceiling",
		Domain:        "light",
		Name:          "Ceiling Light",
		Writable:      true,
		State:         `{"state":"OFF"}`,
		Available:     true,
		LastChangedAt: time.Now(),
		LastUpdatedAt: time.Now(),
	}
	require.NoError(t, repos.Entities.Create(ctx, ent))

	handler := handlers.NewEntityHandler(repos.Entities, bus)

	var emittedCmd events.Event
	cmdReceived := make(chan struct{}, 1)
	unsub := bus.Subscribe(events.CommandRequested, func(c context.Context, e events.Event) error {
		emittedCmd = e
		cmdReceived <- struct{}{}
		return nil
	})
	defer unsub()

	// Dispatch command via router
	r := chi.NewRouter()
	r.Post("/api/v1/entities/{id}/command", handler.Command)

	cmdPayload, _ := json.Marshal(map[string]any{
		"service": "turn_on",
		"params":  map[string]any{"brightness": 128},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/entities/"+ent.ID+"/command", bytes.NewReader(cmdPayload))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusAccepted, rr.Code)

	select {
	case <-cmdReceived:
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for CommandRequested event")
	}

	assert.Equal(t, events.CommandRequested, emittedCmd.Type)
	payloadMap := emittedCmd.Payload.(map[string]any)
	assert.Equal(t, "light.ceiling", payloadMap["entity_key"])
	assert.Equal(t, "turn_on", payloadMap["service"])
}


func TestEntityHandler_ApplyClimateProfile(t *testing.T) {
	repos, bus := setupTestEnvironment(t)
	ctx := context.Background()
	in := &integration.Integration{
		ID: "climate-integration", Type: "test", Name: "Climate", Status: "connected",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, repos.Integrations.Create(ctx, in))
	dev := &device.Device{
		ID: "climate-device", IntegrationID: in.ID, ExternalID: "climate",
		Name: "Climate", Online: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, repos.Devices.Create(ctx, dev))
	thermostat := &entity.Entity{
		ID: "climate-entity-id", DeviceID: dev.ID, EntityKey: "climate.living_room",
		Domain: "climate", Name: "Living room", Writable: true, Available: true,
		State: `{"state":"heat"}`, Attributes: `{"target_temperature":20,"min_temp":5,"max_temp":30}`,
	}
	require.NoError(t, repos.Entities.Create(ctx, thermostat))

	engine := appauto.NewEngine(repos.Automations, repos.Entities, bus)
	require.NoError(t, engine.Start(ctx))
	defer func() { _ = engine.Stop(context.Background()) }()
	handler := handlers.NewEntityHandler(repos.Entities, bus).WithClimateEngine(engine)
	router := chi.NewRouter()
	router.Post("/api/v1/entities/{id}/climate-profile", handler.ApplyClimateProfile)

	var received float64
	commandReceived := make(chan struct{}, 1)
	unsub := bus.Subscribe(string(events.CommandRequested), func(_ context.Context, event events.Event) error {
		if event.Payload["entity_key"] == "climate.living_room" {
			params, _ := event.Payload["params"].(map[string]any)
			if temperature, ok := params["temperature"].(float64); ok {
				received = temperature
				commandReceived <- struct{}{}
			}
		}
		return nil
	})
	defer unsub()

	body := bytes.NewBufferString(`{"profile":"custom","temperature":19.5,"duration_minutes":0}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/entities/climate-entity-id/climate-profile", body)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusAccepted, rr.Code)
	select {
	case <-commandReceived:
		assert.Equal(t, 19.5, received)
	case <-time.After(2 * time.Second):
		t.Fatal("thermostat profile command was not published")
	}
}
