package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/plugins/tuya_cloud"
	"github.com/domus-hub/domus/internal/plugins/tuya_local"
	"github.com/go-chi/chi/v5"
)

type mockBusForTest struct{}

func (m *mockBusForTest) Publish(ctx context.Context, evt events.Event) error { return nil }
func (m *mockBusForTest) Subscribe(eventType events.EventType, handler events.Handler) func() {
	return func() {}
}
func (m *mockBusForTest) SubscribeAll(handler events.Handler) func() { return func() {} }

func TestTuyaCloudHandler_SyncLocalAndLanDispatch(t *testing.T) {
	// 1. Mock Tuya Cloud HTTP transport
	transport := tuyaOAuthTestTransport(func(r *http.Request) (*http.Response, error) {
		body := `{"success":true,"result":{"access_token":"fake-token","expire_time":3600}}`
		if strings.HasPrefix(r.URL.Path, "/v1.0/devices") {
			body = `{"success":true,"result":{"devices":[{"id":"dev-sync-1","name":"Balcony Light","local_key":"key1234567890abc","ip":"203.0.113.188","category":"dj","online":true}]}}`
		}
		return &http.Response{
			StatusCode: 200,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    r,
		}, nil
	})

	cloudClient, err := tuya_cloud.New(tuya_cloud.Config{
		ClientID:   "cid",
		Secret:     "secret",
		BaseURL:    "https://openapi.tuyaeu.com",
		HTTPClient: &http.Client{Transport: transport},
	})
	if err != nil {
		t.Fatalf("failed to create cloud client: %v", err)
	}

	bus := &mockBusForTest{}
	localAdapter := tuya_local.NewAdapter("tuya_local", bus)
	// A previous sync may have persisted Tuya Cloud's WAN address as if it
	// were the device LAN address. A new sync must clear that stale value.
	localAdapter.RegisterDevice(tuya_local.DeviceConfig{
		DeviceID: "dev-sync-1",
		IP:       "198.51.100.44",
		LocalKey: "old-local-key-123",
		Category: tuya_local.CategoryLight,
	})

	handler := NewTuyaCloudHandler(cloudClient).WithLocalAdapter(localAdapter)

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	// 2. Execute SyncLocal
	syncReq := httptest.NewRequest(http.MethodPost, "/sync-local", nil)
	syncRec := httptest.NewRecorder()
	r.ServeHTTP(syncRec, syncReq)

	if syncRec.Code != http.StatusOK {
		t.Fatalf("SyncLocal returned status %d, body: %s", syncRec.Code, syncRec.Body.String())
	}

	var syncResp struct {
		Success     bool                      `json:"success"`
		SyncedCount int                       `json:"synced_count"`
		Devices     []tuya_local.DeviceConfig `json:"devices"`
	}
	if err := json.Unmarshal(syncRec.Body.Bytes(), &syncResp); err != nil {
		t.Fatalf("failed to parse sync response: %v", err)
	}
	if !syncResp.Success || syncResp.SyncedCount != 1 {
		t.Fatalf("unexpected sync result: %+v", syncResp)
	}

	// Verify device is now configured in Tuya Local with local_key
	if !localAdapter.IsDeviceConfigured("dev-sync-1") {
		t.Fatalf("device dev-sync-1 should be registered in tuya_local")
	}

	// 3. ListDevices should now reflect local_configured = true
	listReq := httptest.NewRequest(http.MethodGet, "/devices", nil)
	listRec := httptest.NewRecorder()
	r.ServeHTTP(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("ListDevices status %d: %s", listRec.Code, listRec.Body.String())
	}
	var listResp struct {
		Devices []struct {
			ID              string `json:"id"`
			LocalConfigured bool   `json:"local_configured"`
			LocalIP         string `json:"local_ip"`
		} `json:"devices"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("failed to parse list response: %v", err)
	}
	if len(listResp.Devices) != 1 || !listResp.Devices[0].LocalConfigured || listResp.Devices[0].LocalIP != "" {
		t.Fatalf("cloud-reported WAN IP must not be used as LocalTuya IP: %+v", listResp)
	}
	if got := localAdapter.GetDeviceIP("dev-sync-1"); got != "" {
		t.Fatalf("stale Cloud/WAN IP must be cleared until LAN discovery, got %q", got)
	}
	if len(syncResp.Devices) != 1 || syncResp.Devices[0].IP != "" {
		t.Fatalf("SyncLocal must not return Cloud IP as a LAN address: %+v", syncResp.Devices)
	}
}

func TestTuyaCloudHandler_SmartLifeUIDDevices(t *testing.T) {
	transport := tuyaOAuthTestTransport(func(r *http.Request) (*http.Response, error) {
		body := `{"success":true,"result":{"access_token":"fake-token","expire_time":3600}}`
		if r.URL.Path == "/v1.0/users/eu_user_999/devices" {
			body = `{"success":true,"result":[{"id":"dev-qr-1","name":"Smart Life Switch","local_key":"key999888777666","ip":"192.168.1.99","category":"kg","online":true}]}`
		}
		return &http.Response{
			StatusCode: 200,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    r,
		}, nil
	})

	cloudClient, err := tuya_cloud.New(tuya_cloud.Config{
		ClientID:   "cid",
		Secret:     "secret",
		BaseURL:    "https://openapi.tuyaeu.com",
		UID:        "eu_user_999",
		HTTPClient: &http.Client{Transport: transport},
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	handler := NewTuyaCloudHandler(cloudClient)
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	req := httptest.NewRequest(http.MethodGet, "/devices", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("ListDevices status %d: %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Devices []tuya_cloud.Device `json:"devices"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	if len(resp.Devices) != 1 || resp.Devices[0].ID != "dev-qr-1" || resp.Devices[0].LocalKey != "key999888777666" {
		t.Fatalf("unexpected devices: %+v", resp.Devices)
	}
}

func TestTuyaCloudHandler_SyncLocalAppliesExplicitEmptySelection(t *testing.T) {
	transport := tuyaOAuthTestTransport(func(r *http.Request) (*http.Response, error) {
		body := `{"success":true,"result":{"access_token":"fake-token","expire_time":3600}}`
		if r.URL.Path == "/v1.0/users/eu_user_999/devices" {
			body = `{"success":true,"result":[{"id":"dev-keep-out","name":"Plug","local_key":"key1234567890123","ip":"192.168.1.21","category":"cz","online":true}]}`
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})
	client, err := tuya_cloud.New(tuya_cloud.Config{ClientID:"cid", Secret:"secret", BaseURL:"https://openapi.tuyaeu.com", UID:"eu_user_999", HTTPClient:&http.Client{Transport:transport}})
	if err != nil { t.Fatal(err) }
	adapter := tuya_local.NewAdapter("tuya_local", &mockBusForTest{})
	adapter.RegisterDevice(tuya_local.DeviceConfig{DeviceID:"dev-keep-out", LocalKey:"key1234567890123", Category:tuya_local.CategoryPlug, EntityKey:"switch.tuya_dev-keep-out"})
	handler := NewTuyaCloudHandler(client).WithLocalAdapter(adapter)
	r := chi.NewRouter()
	handler.RegisterRoutes(r)
	req := httptest.NewRequest(http.MethodPost, "/sync-local", strings.NewReader(`{"device_ids":[]}`))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK { t.Fatalf("SyncLocal returned %d: %s", rec.Code, rec.Body.String()) }
	if adapter.IsDeviceConfigured("dev-keep-out") { t.Fatal("explicit empty selection must remove previously configured LocalTuya devices") }
	var body struct { SyncedCount int `json:"synced_count"` }
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil { t.Fatal(err) }
	if body.SyncedCount != 0 { t.Fatalf("expected no selected devices, got %d", body.SyncedCount) }
}

func TestTuyaCloudHandler_ListFunctionsUsesDeviceFunctionAPI(t *testing.T) {
	transport := tuyaOAuthTestTransport(func(r *http.Request) (*http.Response, error) {
		body := `{"success":true,"result":{"access_token":"fake-token","expire_time":3600}}`
		if r.URL.Path == "/v1.0/iot-03/devices/dev-functions-1/functions" {
			body = `{"success":true,"result":{"category":"wk","functions":[{"code":"temp_set","name":"Target temperature","desc":"Set target temperature","type":"Integer","values":"{}"}]}}`
		}
		return &http.Response{StatusCode:200, Header:make(http.Header), Body:io.NopCloser(strings.NewReader(body)), Request:r}, nil
	})
	client, err := tuya_cloud.New(tuya_cloud.Config{ClientID:"cid", Secret:"secret", BaseURL:"https://openapi.tuyaeu.com", HTTPClient:&http.Client{Transport:transport}})
	if err != nil { t.Fatal(err) }
	handler := NewTuyaCloudHandler(client)
	r := chi.NewRouter()
	handler.RegisterRoutes(r)
	req := httptest.NewRequest(http.MethodGet, "/devices/dev-functions-1/functions", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK { t.Fatalf("ListFunctions returned %d: %s", rec.Code, rec.Body.String()) }
	var response struct { Functions []tuya_cloud.Function `json:"functions"` }
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil { t.Fatal(err) }
	if len(response.Functions) != 1 || response.Functions[0].Code != "temp_set" || response.Functions[0].Type != "Integer" || response.Functions[0].Desc != "Set target temperature" {
		t.Fatalf("unexpected Tuya function metadata: %+v", response.Functions)
	}
}

func TestTuyaCloudHandler_SyncLocalDoesNotRemoveOtherLocalTuyaAccounts(t *testing.T) {
	transport := tuyaOAuthTestTransport(func(r *http.Request) (*http.Response, error) {
		body := `{"success":true,"result":{"access_token":"fake-token","expire_time":3600}}`
		if r.URL.Path == "/v1.0/users/eu_user_999/devices" {
			body = `{"success":true,"result":[{"id":"account-device","name":"Smart Plug","local_key":"key1234567890123","ip":"192.168.1.22","category":"cz","online":true}]}`
		}
		return &http.Response{StatusCode:200, Header:make(http.Header), Body:io.NopCloser(strings.NewReader(body)), Request:r}, nil
	})
	client, err := tuya_cloud.New(tuya_cloud.Config{ClientID:"cid", Secret:"secret", BaseURL:"https://openapi.tuyaeu.com", UID:"eu_user_999", HTTPClient:&http.Client{Transport:transport}})
	if err != nil { t.Fatal(err) }
	adapter := tuya_local.NewAdapter("tuya_local", &mockBusForTest{})
	adapter.RegisterDevice(tuya_local.DeviceConfig{DeviceID:"unrelated-local-device", LocalKey:"localkey123456789", Category:tuya_local.CategorySwitch, EntityKey:"switch.other"})
	handler := NewTuyaCloudHandler(client).WithLocalAdapter(adapter)
	r := chi.NewRouter()
	handler.RegisterRoutes(r)
	req := httptest.NewRequest(http.MethodPost, "/sync-local", strings.NewReader(`{"device_ids":[]}`))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK { t.Fatalf("SyncLocal returned %d: %s", rec.Code, rec.Body.String()) }
	if !adapter.IsDeviceConfigured("unrelated-local-device") {
		t.Fatal("selection update must not remove devices outside the configured Smart Life account")
	}
}
