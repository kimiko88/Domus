package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	appCal "github.com/domus-hub/domus/internal/application/calendar"
	appCamera "github.com/domus-hub/domus/internal/application/camera"
	appMedia "github.com/domus-hub/domus/internal/application/media"
	appMesh "github.com/domus-hub/domus/internal/application/mesh"
	appNotif "github.com/domus-hub/domus/internal/application/notification"
	appOTA "github.com/domus-hub/domus/internal/application/ota"
	appShop "github.com/domus-hub/domus/internal/application/shopping"
	appStore "github.com/domus-hub/domus/internal/application/store"
	appTag "github.com/domus-hub/domus/internal/application/tag"
	appWeather "github.com/domus-hub/domus/internal/application/weather"
	domainCamera "github.com/domus-hub/domus/internal/domain/camera"
	domainShop "github.com/domus-hub/domus/internal/domain/shopping"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	"github.com/domus-hub/domus/internal/interfaces/http/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupRealSQLiteE2E(t *testing.T) (*chi.Mux, *sqlite.Repositories) {
	db, err := sqlite.Open(config.DatabaseConfig{
		Path:          ":memory:",
		WALMode:       false,
		BusyTimeoutMs: 1000,
	})
	require.NoError(t, err)

	ctx := context.Background()
	err = sqlite.Migrate(ctx, db)
	require.NoError(t, err)

	repos := sqlite.NewRepositories(db)
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 64, Workers: 1})

	camSvc := appCamera.NewService(repos.Cameras, bus)
	tagSvc := appTag.NewService(repos.Tags, bus)
	notifMgr := appNotif.NewManager(bus)
	calSvc := appCal.NewService(repos.Calendar, bus)
	mediaSvc := appMedia.NewService(repos.Media, bus)
	weatherCli := appWeather.NewClient()
	meshSvc := appMesh.NewService(repos.Devices)
	otaSvc := appOTA.NewService(repos.OTA, bus)
	storeSvc := appStore.NewService(repos.Store)
	shopSvc := appShop.NewService(repos.Shopping, bus)

	ecoH := handlers.NewEcosystemHandler(camSvc, tagSvc, notifMgr, calSvc, mediaSvc, weatherCli, meshSvc, otaSvc, storeSvc)
	shopH := handlers.NewShoppingHandler(shopSvc)

	r := chi.NewRouter()

	// Shopping
	r.Get("/api/v1/shopping", shopH.List)
	r.Post("/api/v1/shopping", shopH.Create)
	r.Post("/api/v1/shopping/{id}/toggle", shopH.Toggle)
	r.Delete("/api/v1/shopping/{id}", shopH.Delete)
	r.Post("/api/v1/shopping/clear-completed", shopH.ClearCompleted)

	// Ecosystem
	r.Get("/api/v1/cameras", ecoH.ListCameras)
	r.Post("/api/v1/cameras", ecoH.CreateCamera)
	r.Get("/api/v1/tags", ecoH.ListTags)
	r.Post("/api/v1/tags", ecoH.CreateTag)
	r.Post("/api/v1/tags/{id}/scan", ecoH.ScanTag)
	r.Get("/api/v1/schedules", ecoH.ListSchedules)
	r.Get("/api/v1/media/players", ecoH.ListMediaPlayers)
	r.Get("/api/v1/mesh/topology", ecoH.GetMeshTopology)
	r.Get("/api/v1/ota/devices", ecoH.ListOTADevices)
	r.Post("/api/v1/ota/devices/{id}/upgrade", ecoH.TriggerOTAUpgrade)
	r.Get("/api/v1/store/packages", ecoH.ListStorePackages)
	r.Post("/api/v1/store/packages/{id}/install", ecoH.InstallStorePackage)
	r.Post("/api/v1/store/packages/{id}/uninstall", ecoH.UninstallStorePackage)

	return r, repos
}

func TestEcosystem_RealSQLite_E2E_Flow(t *testing.T) {
	r, repos := setupRealSQLiteE2E(t)
	ctx := context.Background()

	// 1. Shopping List E2E
	t.Run("Shopping List E2E CRUD", func(t *testing.T) {
		// Verify initial empty state
		req := httptest.NewRequest(http.MethodGet, "/api/v1/shopping", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "[]\n", rr.Body.String())

		// Create Item
		itemPayload, _ := json.Marshal(map[string]any{
			"name":     "Pane Casereccio",
			"quantity": "1kg",
			"category": "alimentari",
			"added_by": "Marco",
		})
		req = httptest.NewRequest(http.MethodPost, "/api/v1/shopping", bytes.NewReader(itemPayload))
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusCreated, rr.Code)

		var created domainShop.Item
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &created))
		assert.NotEmpty(t, created.ID)
		assert.Equal(t, "Pane Casereccio", created.Name)

		// Verify record exists in SQLite directly
		dbItem, err := repos.Shopping.GetItem(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, "Pane Casereccio", dbItem.Name)
		assert.False(t, dbItem.Completed)

		// Toggle Item
		togglePayload, _ := json.Marshal(map[string]any{
			"completed_by": "Giulia",
		})
		req = httptest.NewRequest(http.MethodPost, "/api/v1/shopping/"+created.ID+"/toggle", bytes.NewReader(togglePayload))
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)

		var toggled domainShop.Item
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &toggled))
		assert.True(t, toggled.Completed)

		// Clear Completed
		req = httptest.NewRequest(http.MethodPost, "/api/v1/shopping/clear-completed", nil)
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)

		// Verify empty again
		req = httptest.NewRequest(http.MethodGet, "/api/v1/shopping", nil)
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "[]\n", rr.Body.String())
	})

	// 2. Cameras E2E
	t.Run("Cameras E2E", func(t *testing.T) {
		// Initial empty
		req := httptest.NewRequest(http.MethodGet, "/api/v1/cameras", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "[]\n", rr.Body.String())

		// Create Camera
		camPayload, _ := json.Marshal(map[string]any{
			"id":                "cam_garden",
			"name":              "Telecamera Giardino",
			"stream_url":        "rtsp://192.168.1.150/live",
			"stream_type":       "webrtc",
			"detection_enabled": true,
		})
		req = httptest.NewRequest(http.MethodPost, "/api/v1/cameras", bytes.NewReader(camPayload))
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusCreated, rr.Code)

		// Read back via GET
		req = httptest.NewRequest(http.MethodGet, "/api/v1/cameras", nil)
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)

		var cams []domainCamera.Camera
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &cams))
		assert.Len(t, cams, 1)
		assert.Equal(t, "Telecamera Giardino", cams[0].Name)
	})

	// 3. NFC Tags E2E
	t.Run("NFC Tags E2E", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/tags", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "[]\n", rr.Body.String())

		tagPayload, _ := json.Marshal(map[string]any{
			"tag_id":           "04:22:33:44:55",
			"name":             "Apertura Cancello",
			"action_type":      "automation",
			"action_target_id": "auto_gate_open",
		})
		req = httptest.NewRequest(http.MethodPost, "/api/v1/tags", bytes.NewReader(tagPayload))
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusCreated, rr.Code)

		// Scan tag
		req = httptest.NewRequest(http.MethodPost, "/api/v1/tags/04:22:33:44:55/scan", nil)
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	})

	// 4. OTA & Store E2E
	t.Run("OTA and Store Packages E2E", func(t *testing.T) {
		// Pre-insert an OTA device into SQLite
		err := repos.OTA.SaveDevice(ctx, &appOTA.DeviceOTA{
			ID:             "ota_thermostat_01",
			Name:           "Termostato Salotto",
			DeviceKey:      "zigbee_term_01",
			Platform:       "zigbee",
			CurrentVersion: "1.0.4",
			LatestVersion:  "1.1.0",
			HasUpdate:      true,
			UpgradeState:   appOTA.StateIdle,
		})
		require.NoError(t, err)

		// List OTA devices via HTTP
		req := httptest.NewRequest(http.MethodGet, "/api/v1/ota/devices", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)

		var otaDevs []*appOTA.DeviceOTA
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &otaDevs))
		assert.Len(t, otaDevs, 1)
		assert.Equal(t, "Termostato Salotto", otaDevs[0].Name)

		// Trigger Upgrade
		req = httptest.NewRequest(http.MethodPost, "/api/v1/ota/devices/ota_thermostat_01/upgrade", nil)
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)

		// Pre-insert a package into Store in SQLite
		err = repos.Store.SavePackage(ctx, &appStore.Package{
			ID:          "pkg_aqara_cube",
			PackageID:   "community.aqara.cube",
			Name:        "Aqara Cube Controller",
			Type:        appStore.TypeWasmPlugin,
			Version:     "1.0.0",
			Author:      "DomusCommunity",
			Description: "Sandbox WASM cube interpreter",
			Installed:   false,
		})
		require.NoError(t, err)

		// List packages via HTTP
		req = httptest.NewRequest(http.MethodGet, "/api/v1/store/packages", nil)
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)

		var pkgs []*appStore.Package
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &pkgs))
		assert.Len(t, pkgs, 1)
		assert.False(t, pkgs[0].Installed)

		// Install package via HTTP
		req = httptest.NewRequest(http.MethodPost, "/api/v1/store/packages/pkg_aqara_cube/install", nil)
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)

		// Verify in SQLite that package is marked installed
		dbPkg, err := repos.Store.GetPackage(ctx, "pkg_aqara_cube")
		require.NoError(t, err)
		assert.True(t, dbPkg.Installed)

		// Uninstall package via HTTP
		req = httptest.NewRequest(http.MethodPost, "/api/v1/store/packages/pkg_aqara_cube/uninstall", nil)
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)

		dbPkg, err = repos.Store.GetPackage(ctx, "pkg_aqara_cube")
		require.NoError(t, err)
		assert.False(t, dbPkg.Installed)
	})
}
