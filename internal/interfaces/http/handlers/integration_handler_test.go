package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/application/brand"
	"github.com/domus-hub/domus/internal/domain/integration"
	"github.com/domus-hub/domus/internal/interfaces/http/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegrationHandler_ListEmptyAndWithData(t *testing.T) {
	repos, bus := setupTestEnvironment(t)
	handler := handlers.NewIntegrationHandler(repos.Integrations, bus)

	// 1. Initially empty list should return 200 with empty array (not null, not 404)
	req := httptest.NewRequest(http.MethodGet, "/integrations", nil)
	rr := httptest.NewRecorder()
	handler.List(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var list []*integration.Integration
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &list))
	assert.Empty(t, list)

	// 2. Add an integration
	ctx := context.Background()
	in := &integration.Integration{
		ID:        "mqtt_default",
		Type:      "mqtt_generic",
		Name:      "MQTT Broker",
		Enabled:   true,
		Status:    "connected",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, repos.Integrations.Create(ctx, in))

	// 3. List again
	req = httptest.NewRequest(http.MethodGet, "/integrations", nil)
	rr = httptest.NewRecorder()
	handler.List(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &list))
	assert.Len(t, list, 1)
	assert.Equal(t, "mqtt_default", list[0].ID)
}

func TestIntegrationHandler_GetByIDAndReconnect(t *testing.T) {
	repos, bus := setupTestEnvironment(t)
	handler := handlers.NewIntegrationHandler(repos.Integrations, bus)

	ctx := context.Background()
	in := &integration.Integration{
		ID:        "z2m",
		Type:      "zigbee2mqtt",
		Name:      "Zigbee2MQTT",
		Enabled:   true,
		Status:    "disconnected",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, repos.Integrations.Create(ctx, in))

	// GetByID
	r := chi.NewRouter()
	r.Get("/integrations/{id}", handler.GetByID)
	r.Post("/integrations/{id}/reconnect", handler.Reconnect)

	req := httptest.NewRequest(http.MethodGet, "/integrations/z2m", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	var fetched integration.Integration
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &fetched))
	assert.Equal(t, "z2m", fetched.ID)

	// Reconnect
	req = httptest.NewRequest(http.MethodPost, "/integrations/z2m/reconnect", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "reconnecting", resp["status"])

	// 404 for unknown ID
	req = httptest.NewRequest(http.MethodGet, "/integrations/nonexistent", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestIntegrationHandler_BrandCatalogAndDiscovery(t *testing.T) {
	repos, bus := setupTestEnvironment(t)
	brandSvc := brand.NewService(repos.Devices, repos.Entities, repos.Integrations, bus, nil)
	handler := handlers.NewIntegrationHandler(repos.Integrations, bus).WithBrandService(brandSvc)

	// 1. Get Catalog
	req := httptest.NewRequest(http.MethodGet, "/integrations/catalog", nil)
	rr := httptest.NewRecorder()
	handler.GetCatalog(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var catalog []brand.BrandCatalogEntry
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &catalog))
	assert.NotEmpty(t, catalog)

	// 2. List Discovered
	req = httptest.NewRequest(http.MethodGet, "/integrations/discovery/devices", nil)
	rr = httptest.NewRecorder()
	handler.ListDiscovered(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var discovered []*brand.DiscoveredDevice
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &discovered))
	assert.Empty(t, discovered)
}
