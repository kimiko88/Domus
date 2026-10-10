package handlers_test

import (
	"bytes"
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
	appStore "github.com/domus-hub/domus/internal/application/store"
	appTag "github.com/domus-hub/domus/internal/application/tag"
	appWeather "github.com/domus-hub/domus/internal/application/weather"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/domus-hub/domus/internal/interfaces/http/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupEcosystemRouter() (*chi.Mux, *handlers.EcosystemHandler) {
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 64, Workers: 1})
	camSvc := appCamera.NewService(appCamera.NewMemoryRepository(), bus)
	tagSvc := appTag.NewService(appTag.NewMemoryRepository(), bus)
	notifMgr := appNotif.NewManager(bus)
	calSvc := appCal.NewService(appCal.NewMemoryRepository(), bus)
	mediaSvc := appMedia.NewService(appMedia.NewMemoryRepository(), bus)
	weatherCli := appWeather.NewClient()
	meshSvc := appMesh.NewService()
	otaSvc := appOTA.NewService(bus)
	storeSvc := appStore.NewService()

	h := handlers.NewEcosystemHandler(camSvc, tagSvc, notifMgr, calSvc, mediaSvc, weatherCli, meshSvc, otaSvc, storeSvc)

	r := chi.NewRouter()
	r.Get("/cameras", h.ListCameras)
	r.Post("/cameras", h.CreateCamera)
	r.Post("/cameras/{id}/webrtc", h.NegotiateWebRTC)
	r.Get("/tags", h.ListTags)
	r.Post("/tags", h.CreateTag)
	r.Post("/tags/{id}/scan", h.ScanTag)
	r.Get("/schedules", h.ListSchedules)
	r.Get("/media/players", h.ListMediaPlayers)
	r.Get("/weather/forecast", h.GetWeatherForecast)
	r.Get("/weather/solar-forecast", h.GetSolarForecast)
	r.Get("/mesh/topology", h.GetMeshTopology)
	r.Get("/ota/devices", h.ListOTADevices)
	r.Post("/ota/devices/{id}/upgrade", h.TriggerOTAUpgrade)
	r.Get("/store/packages", h.ListStorePackages)
	r.Post("/store/packages/{id}/install", h.InstallStorePackage)

	return r, h
}

func TestEcosystemHandler_CamerasAndWebRTC(t *testing.T) {
	r, _ := setupEcosystemRouter()

	body := map[string]any{
		"name":       "Telecamera Balcone",
		"stream_url": "rtsp://192.168.1.50/live",
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/cameras", bytes.NewReader(raw))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusCreated, rr.Code)

	var camResp map[string]any
	err := json.Unmarshal(rr.Body.Bytes(), &camResp)
	require.NoError(t, err)
	camID := camResp["id"].(string)

	// WebRTC negotiation
	webrtcBody := map[string]any{"offer_sdp": "v=0\r\no=test\r\n"}
	wraw, _ := json.Marshal(webrtcBody)
	req = httptest.NewRequest(http.MethodPost, "/cameras/"+camID+"/webrtc", bytes.NewReader(wraw))
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestEcosystemHandler_TagsScan(t *testing.T) {
	r, _ := setupEcosystemRouter()

	body := map[string]any{
		"name":             "NFC Salotto",
		"tag_id":           "04:FA:21:88",
		"action_type":      "scene",
		"action_target_id": "scene_cinema",
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/tags", bytes.NewReader(raw))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusCreated, rr.Code)

	// Scan
	scanReq := httptest.NewRequest(http.MethodPost, "/tags/04:FA:21:88/scan", bytes.NewReader([]byte(`{"scanned_by":"iphone_15"}`)))
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, scanReq)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestEcosystemHandler_WeatherAndMesh(t *testing.T) {
	r, _ := setupEcosystemRouter()

	// Weather
	req := httptest.NewRequest(http.MethodGet, "/weather/forecast?lat=45.4&lon=9.1", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// Solar
	req = httptest.NewRequest(http.MethodGet, "/weather/solar-forecast?kwp=6", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// Mesh
	req = httptest.NewRequest(http.MethodGet, "/mesh/topology?protocol=zigbee", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestEcosystemHandler_OTAAndStore(t *testing.T) {
	r, _ := setupEcosystemRouter()

	// OTA devices
	req := httptest.NewRequest(http.MethodGet, "/ota/devices", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// Upgrade
	req = httptest.NewRequest(http.MethodPost, "/ota/devices/ota_1/upgrade", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// Store packages
	req = httptest.NewRequest(http.MethodGet, "/store/packages", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// Install package
	req = httptest.NewRequest(http.MethodPost, "/store/packages/pkg_bp_circadian/install", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}
