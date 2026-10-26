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

	otaRepo := appOTA.NewMemoryRepository()
	_ = otaRepo.SaveDevice(context.Background(), &appOTA.DeviceOTA{
		ID:             "ota_1",
		Name:           "ESP32 Relay",
		CurrentVersion: "1.0.0",
		LatestVersion:  "1.1.0",
		HasUpdate:      true,
		UpgradeState:   appOTA.StateIdle,
	})
	otaSvc := appOTA.NewService(otaRepo, bus)

	storeRepo := appStore.NewMemoryRepository()
	_ = storeRepo.SavePackage(context.Background(), &appStore.Package{
		ID:        "pkg_bp_circadian",
		PackageID: "blueprints.circadian",
		Name:      "Circadian Lighting",
		Type:      appStore.TypeBlueprint,
		Version:   "1.0.0",
	})
	storeSvc := appStore.NewService(storeRepo)

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
	r.Get("/mesh/thread/topology", h.GetThreadTopology)
	r.Get("/system/remote-access/wireguard/server", h.GetWireGuardServer)
	r.Get("/system/remote-access/wireguard/peers", h.ListWireGuardPeers)
	r.Post("/system/remote-access/wireguard/peers", h.EnrollWireGuardPeer)
	r.Delete("/system/remote-access/wireguard/peers/{id}", h.RemoveWireGuardPeer)
	r.Get("/system/remote-access/tailscale", h.GetTailscaleStatus)
	r.Post("/system/remote-access/tailscale/connect", h.ConnectTailscale)
	r.Get("/system/backup/remote/config", h.GetRemoteBackupConfig)
	r.Post("/system/backup/remote/sync", h.TriggerRemoteBackupSync)
	r.Post("/voice/process", h.ProcessVoiceSpeech)
	r.Get("/ota/devices", h.ListOTADevices)
	r.Post("/ota/devices/{id}/upgrade", h.TriggerOTAUpgrade)
	r.Get("/store/packages", h.ListStorePackages)
	r.Post("/store/packages/{id}/install", h.InstallStorePackage)
	r.Post("/store/community/submit", h.SubmitCommunityPackage)
	r.Post("/store/community/{id}/approve", h.ApproveCommunityPackage)
	r.Post("/store/community/{id}/reject", h.RejectCommunityPackage)

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

	// Community submit package
	commBody := map[string]any{
		"package_id":     "domus.community.mesh_test",
		"name":           "Mesh Test Plugin",
		"type":           "wasm_plugin",
		"version":        "1.0.0",
		"author":         "Tester",
		"description":    "Test description",
		"repository_url": "https://github.com/domus-community/mesh_test",
		"license":        "MIT",
	}
	commRaw, _ := json.Marshal(commBody)
	req = httptest.NewRequest(http.MethodPost, "/store/community/submit", bytes.NewReader(commRaw))
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusCreated, rr.Code)

	var subResp map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &subResp))
	commID := subResp["id"].(string)

	// Approve community package
	req = httptest.NewRequest(http.MethodPost, "/store/community/"+commID+"/approve", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// Reject community package
	rejectBody, _ := json.Marshal(map[string]string{"reason": "Testing reject"})
	req = httptest.NewRequest(http.MethodPost, "/store/community/"+commID+"/reject", bytes.NewReader(rejectBody))
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestEcosystemHandler_ThreadAndRemoteAccessAndVoice(t *testing.T) {
	r, _ := setupEcosystemRouter()

	// 1. Thread Topology
	req := httptest.NewRequest(http.MethodGet, "/mesh/thread/topology", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	var threadTopo map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &threadTopo))
	assert.NotEmpty(t, threadTopo["network_name"])
	assert.Equal(t, "leader", threadTopo["local_role"])

	// 2. WireGuard Server Config
	req = httptest.NewRequest(http.MethodGet, "/system/remote-access/wireguard/server", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// 3. WireGuard Peer Enrollment
	peerBody, _ := json.Marshal(map[string]string{"name": "iPad Pro M4"})
	req = httptest.NewRequest(http.MethodPost, "/system/remote-access/wireguard/peers", bytes.NewReader(peerBody))
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusCreated, rr.Code)

	var peerResp map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &peerResp))
	peerID := peerResp["peer_id"].(string)
	assert.NotEmpty(t, peerResp["config_text"])
	assert.NotEmpty(t, peerResp["qr_code_data_uri"])

	// 4. WireGuard Peer List & Remove
	req = httptest.NewRequest(http.MethodGet, "/system/remote-access/wireguard/peers", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	req = httptest.NewRequest(http.MethodDelete, "/system/remote-access/wireguard/peers/"+peerID, nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// 5. Tailscale Status & Connect
	req = httptest.NewRequest(http.MethodGet, "/system/remote-access/tailscale", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	tsBody, _ := json.Marshal(map[string]string{"auth_key": "tskey-test-abc"})
	req = httptest.NewRequest(http.MethodPost, "/system/remote-access/tailscale/connect", bytes.NewReader(tsBody))
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// 6. Remote Backup Config & Sync
	req = httptest.NewRequest(http.MethodGet, "/system/backup/remote/config", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	req = httptest.NewRequest(http.MethodPost, "/system/backup/remote/sync", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// 7. Voice Pipeline Process
	voiceBody, _ := json.Marshal(map[string]string{
		"satellite_id": "sat_kitchen",
		"text_prompt":  "spegni tutte le luci",
	})
	req = httptest.NewRequest(http.MethodPost, "/voice/process", bytes.NewReader(voiceBody))
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	var voiceResp map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &voiceResp))
	assert.Equal(t, "spegni tutte le luci", voiceResp["transcribed_text"])
	assert.NotEmpty(t, voiceResp["response_text"])
}
