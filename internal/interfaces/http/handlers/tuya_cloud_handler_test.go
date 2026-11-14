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
			body = `{"success":true,"result":{"devices":[{"id":"dev-sync-1","name":"Balcony Light","local_key":"key1234567890abc","ip":"192.168.1.188","category":"dj","online":true}]}}`
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
	if len(listResp.Devices) != 1 || !listResp.Devices[0].LocalConfigured || listResp.Devices[0].LocalIP != "192.168.1.188" {
		t.Fatalf("device listing did not enrich local status properly: %+v", listResp)
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
