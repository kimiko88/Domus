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
