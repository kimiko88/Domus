package handlers

import (
	"context"
	"errors"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/domain/device"
	"github.com/domus-hub/domus/internal/domain/entity"
	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/domain/integration"
	apppluginruntime "github.com/domus-hub/domus/internal/application/plugin_runtime"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/crypto"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
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

func TestIsLocalTuyaSetpointFunction(t *testing.T) {
	tests := []struct {
		name string
		code string
		want bool
	}{
		{name: "known target setpoint", code: "temp_set", want: true},
		{name: "case-insensitive target setpoint", code: "TEMP_SET", want: true},
		{name: "upper temperature is not DP2", code: "upper_temp", want: false},
		{name: "lower temperature is not DP2", code: "lower_temp", want: false},
		{name: "current temperature is read-only", code: "temp_current", want: false},
		{name: "unknown temperature function", code: "temperature", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isLocalTuyaSetpointFunction(tt.code); got != tt.want {
				t.Fatalf("isLocalTuyaSetpointFunction(%q) = %v, want %v", tt.code, got, tt.want)
			}
		})
	}
}

func TestTuyaCloudHandler_UpperTemperatureRetainsCloudDatapoint(t *testing.T) {
	var commandPayload string
	transport := tuyaOAuthTestTransport(func(r *http.Request) (*http.Response, error) {
		body := `{"success":true,"result":{"access_token":"fake-token","expire_time":3600}}`
		switch r.URL.Path {
		case "/v1.0/iot-03/devices/dev-upper-temp/functions":
			body = `{"success":true,"result":{"category":"wk","functions":[{"code":"upper_temp","name":"Upper temperature","type":"Integer","values":"{\"scale\":1}"}]}}`
		case "/v1.0/iot-03/devices/dev-upper-temp/commands":
			raw, err := io.ReadAll(r.Body)
			if err != nil {
				return nil, err
			}
			commandPayload = string(raw)
			body = `{"success":true,"result":{}}`
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    r,
		}, nil
	})

	client, err := tuya_cloud.New(tuya_cloud.Config{
		ClientID:   "cid",
		Secret:     "secret",
		BaseURL:    "https://openapi.tuyaeu.com",
		HTTPClient: &http.Client{Transport: transport},
	})
	if err != nil {
		t.Fatalf("create Tuya Cloud client: %v", err)
	}

	localAdapter := tuya_local.NewAdapter("tuya_local", &mockBusForTest{})
	localAdapter.RegisterDevice(tuya_local.DeviceConfig{
		DeviceID: "dev-upper-temp",
		LocalKey: "key1234567890123",
		Category: tuya_local.CategoryClimate,
	})
	handler := NewTuyaCloudHandler(client).WithLocalAdapter(localAdapter)
	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(
		http.MethodPost,
		"/devices/dev-upper-temp/commands",
		strings.NewReader(`{"commands":[{"code":"upper_temp","value":280}]}`),
	)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("SendCommands returned %d: %s", recorder.Code, recorder.Body.String())
	}
	if commandPayload != `{"commands":[{"code":"upper_temp","value":280}]}` {
		t.Fatalf("upper_temp must be sent unchanged through Tuya Cloud, got %q", commandPayload)
	}
}


func TestInferTuyaDeviceCategoryAndTelemetryRecognizesThermostatCategoryCodes(t *testing.T) {
	device := tuya_cloud.Device{
		ID: "thermostat-1", Name: "Smart Thermostat", Category: "wk",
		Status: []tuya_cloud.Status{
			{Code: "temp_current", Value: 215},
			{Code: "temp_set", Value: 220},
			{Code: "work_mode", Value: "auto"},
		},
	}

	category, entityKey, state, rawAttributes := inferTuyaDeviceCategoryAndTelemetry(device)
	if category != tuya_local.CategoryClimate {
		t.Fatalf("category = %q, want climate", category)
	}
	if entityKey != "climate.tuya_thermostat-1" {
		t.Fatalf("entity key = %q", entityKey)
	}
	if state != "auto" {
		t.Fatalf("state = %q, want auto", state)
	}
	var attributes map[string]any
	if err := json.Unmarshal([]byte(rawAttributes), &attributes); err != nil {
		t.Fatalf("decode initial thermostat attributes: %v", err)
	}
	if got := attributes["current_temperature"]; got != 21.5 {
		t.Fatalf("current_temperature = %#v, want 21.5", got)
	}
	if got := attributes["target_temperature"]; got != 22.0 {
		t.Fatalf("target_temperature = %#v, want 22", got)
	}
}

func TestInferTuyaDeviceCategoryAndTelemetryRecognizesTemperatureHumiditySensors(t *testing.T) {
	device := tuya_cloud.Device{
		ID: "sensor-1", Name: "Temperature and Humidity Sensor", Category: "wsdcg",
		Status: []tuya_cloud.Status{
			{Code: "va_temperature", Value: 217},
			{Code: "va_humidity", Value: 48},
		},
	}

	category, entityKey, state, rawAttributes := inferTuyaDeviceCategoryAndTelemetry(device)
	if category != tuya_local.CategorySensor {
		t.Fatalf("category = %q, want sensor", category)
	}
	if entityKey != "sensor.tuya_sensor-1" {
		t.Fatalf("entity key = %q", entityKey)
	}
	if state != "active" {
		t.Fatalf("state = %q, want active", state)
	}
	var attributes map[string]any
	if err := json.Unmarshal([]byte(rawAttributes), &attributes); err != nil {
		t.Fatalf("decode initial sensor attributes: %v", err)
	}
	if got := attributes["current_temperature"]; got != 21.7 {
		t.Fatalf("current_temperature = %#v, want 21.7", got)
	}
	if got := attributes["current_humidity"]; got != float64(48) {
		t.Fatalf("current_humidity = %#v, want 48", got)
	}
}

func TestInferTuyaDeviceCategoryAndTelemetryDoesNotClassifyPlainSwitchAsClimate(t *testing.T) {
	device := tuya_cloud.Device{
		ID: "switch-1", Name: "Wall Switch", Category: "kg",
		Status: []tuya_cloud.Status{{Code: "switch_1", Value: true}},
	}
	category, entityKey, _, _ := inferTuyaDeviceCategoryAndTelemetry(device)
	if category != tuya_local.CategorySwitch {
		t.Fatalf("category = %q, want switch", category)
	}
	if entityKey != "switch.tuya_switch-1" {
		t.Fatalf("entity key = %q", entityKey)
	}
}


type tuyaCloudTestDeviceRepo struct {
	items map[string]*device.Device
}

func (r *tuyaCloudTestDeviceRepo) Create(_ context.Context, item *device.Device) error {
	if r.items == nil { r.items = make(map[string]*device.Device) }
	copy := *item
	r.items[item.ID] = &copy
	return nil
}
func (r *tuyaCloudTestDeviceRepo) GetByID(_ context.Context, id string) (*device.Device, error) {
	if item := r.items[id]; item != nil { copy := *item; return &copy, nil }
	return nil, device.ErrDeviceNotFound
}
func (r *tuyaCloudTestDeviceRepo) GetByExternalID(_ context.Context, integrationID, externalID string) (*device.Device, error) {
	for _, item := range r.items {
		if item.IntegrationID == integrationID && item.ExternalID == externalID { copy := *item; return &copy, nil }
	}
	return nil, device.ErrDeviceNotFound
}
func (r *tuyaCloudTestDeviceRepo) List(context.Context, device.ListFilter) ([]*device.Device, error) {
	items := make([]*device.Device, 0, len(r.items))
	for _, item := range r.items { copy := *item; items = append(items, &copy) }
	return items, nil
}
func (r *tuyaCloudTestDeviceRepo) Update(_ context.Context, item *device.Device) error {
	if r.items[item.ID] == nil { return device.ErrDeviceNotFound }
	copy := *item
	r.items[item.ID] = &copy
	return nil
}
func (r *tuyaCloudTestDeviceRepo) SetOnline(_ context.Context, id string, online bool) error {
	item := r.items[id]
	if item == nil { return device.ErrDeviceNotFound }
	item.Online = online
	return nil
}
func (r *tuyaCloudTestDeviceRepo) Delete(_ context.Context, id string) error {
	if r.items[id] == nil { return device.ErrDeviceNotFound }
	delete(r.items, id)
	return nil
}

type tuyaCloudTestEntityRepo struct {
	items map[string]*entity.Entity
}

func (r *tuyaCloudTestEntityRepo) Create(_ context.Context, item *entity.Entity) error {
	if r.items == nil { r.items = make(map[string]*entity.Entity) }
	copy := *item
	r.items[item.ID] = &copy
	return nil
}
func (r *tuyaCloudTestEntityRepo) GetByID(_ context.Context, id string) (*entity.Entity, error) {
	if item := r.items[id]; item != nil { copy := *item; return &copy, nil }
	return nil, entity.ErrEntityNotFound
}
func (r *tuyaCloudTestEntityRepo) GetByKey(_ context.Context, key string) (*entity.Entity, error) {
	for _, item := range r.items {
		if item.EntityKey == key { copy := *item; return &copy, nil }
	}
	return nil, entity.ErrEntityNotFound
}
func (r *tuyaCloudTestEntityRepo) List(context.Context, entity.ListFilter) ([]*entity.Entity, error) {
	items := make([]*entity.Entity, 0, len(r.items))
	for _, item := range r.items { copy := *item; items = append(items, &copy) }
	return items, nil
}
func (r *tuyaCloudTestEntityRepo) Update(_ context.Context, item *entity.Entity) error {
	if r.items[item.ID] == nil { return entity.ErrEntityNotFound }
	copy := *item
	r.items[item.ID] = &copy
	return nil
}
func (r *tuyaCloudTestEntityRepo) UpdateState(_ context.Context, id, stateJSON, attributesJSON string, available bool, at time.Time) error {
	item := r.items[id]
	if item == nil { return entity.ErrEntityNotFound }
	item.State, item.Attributes, item.Available, item.LastUpdatedAt = stateJSON, attributesJSON, available, at
	return nil
}
func (r *tuyaCloudTestEntityRepo) Delete(_ context.Context, id string) error {
	if r.items[id] == nil { return entity.ErrEntityNotFound }
	delete(r.items, id)
	return nil
}

func TestTuyaCloudHandler_SyncLocalProvisionsClimateSensorWithoutLocalKey(t *testing.T) {
	var detailRequests int
	transport := tuyaOAuthTestTransport(func(r *http.Request) (*http.Response, error) {
		body := `{"success":true,"result":{"access_token":"fake-token","expire_time":3600}}`
		switch {
		case r.URL.Path == "/v1.0/devices":
			body = `{"success":true,"result":{"devices":[{"id":"thermo-1","name":"Sensore ambiente","product_name":"Temperature and Humidity Sensor","category":"wsdcg","online":true,"status":[{"code":"va_temperature","value":218},{"code":"va_humidity","value":47}] }]}}`
		case r.URL.Path == "/v1.0/devices/thermo-1":
			detailRequests++
			body = `{"success":true,"result":{"id":"thermo-1","name":"Sensore ambiente","product_name":"Temperature and Humidity Sensor","category":"wsdcg","online":true,"status":[{"code":"va_temperature","value":218},{"code":"va_humidity","value":47}]}}`
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header: make(http.Header),
			Body: io.NopCloser(strings.NewReader(body)),
			Request: r,
		}, nil
	})
	client, err := tuya_cloud.New(tuya_cloud.Config{
		ClientID: "cid", Secret: "secret", BaseURL: "https://openapi.tuyaeu.com",
		HTTPClient: &http.Client{Transport: transport},
	})
	if err != nil { t.Fatal(err) }

	devices := &tuyaCloudTestDeviceRepo{items: make(map[string]*device.Device)}
	entities := &tuyaCloudTestEntityRepo{items: make(map[string]*entity.Entity)}
	adapter := tuya_local.NewAdapter("tuya_local", &mockBusForTest{}).WithRepositories(devices, entities)
	handler := NewTuyaCloudHandler(client).WithLocalAdapter(adapter)
	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/sync-local", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("SyncLocal status=%d body=%s", recorder.Code, recorder.Body.String())
	}

	var response struct {
		Success bool `json:"success"`
		SyncedCount int `json:"synced_count"`
		LocalControlCount int `json:"local_control_count"`
		ClimateSourcesCount int `json:"climate_sources_count"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil { t.Fatal(err) }
	if !response.Success || response.SyncedCount != 1 || response.LocalControlCount != 0 || response.ClimateSourcesCount != 1 {
		t.Fatalf("unexpected sync result: %+v body=%s", response, recorder.Body.String())
	}
	if detailRequests != 1 {
		t.Fatalf("detail requests=%d, want 1 when the device list omitted a Local Key", detailRequests)
	}

	deviceRecord, err := devices.GetByID(context.Background(), "tuya_thermo-1")
	if err != nil { t.Fatalf("cloud-only Tuya device was not persisted: %v", err) }
	if deviceRecord.IntegrationID != "tuya_cloud" || !deviceRecord.Online {
		t.Fatalf("unexpected device record: %+v", deviceRecord)
	}
	entityRecord, err := entities.GetByKey(context.Background(), "sensor.tuya_thermo-1")
	if err != nil { t.Fatalf("climate sensor entity was not provisioned: %v", err) }
	if entityRecord.Domain != "sensor" || entityRecord.DeviceID != deviceRecord.ID || entityRecord.Writable {
		t.Fatalf("cloud sensor should be associated and read-only: %+v", entityRecord)
	}
	var attributes map[string]any
	if err := json.Unmarshal([]byte(entityRecord.Attributes), &attributes); err != nil { t.Fatal(err) }
	if attributes["current_temperature"] != 21.8 || attributes["current_humidity"] != float64(47) ||
		attributes["tuya_climate_source"] != true {
		t.Fatalf("climate readings/marker missing from entity: %#v", attributes)
	}
}


func TestInferTuyaDeviceCategoryAndTelemetryMarksThermoHygrometerWithoutInitialStatus(t *testing.T) {
	device := tuya_cloud.Device{
		ID: "thermo-generic-1",
		Name: "TH01",
		ProductName: "Smart home environment monitor",
		Category: "thermo_hygrometer",
		Status: nil,
	}

	category, entityKey, state, rawAttributes := inferTuyaDeviceCategoryAndTelemetry(device)
	if category != tuya_local.CategorySensor {
		t.Fatalf("category = %q, want sensor", category)
	}
	if entityKey != "sensor.tuya_thermo-generic-1" {
		t.Fatalf("entity key = %q", entityKey)
	}
	if state != "active" {
		t.Fatalf("state = %q, want active", state)
	}
	if !isTuyaClimateSource(device, category) {
		t.Fatal("known thermo_hygrometer product category must be a climate history source")
	}

	var attributes map[string]any
	if err := json.Unmarshal([]byte(rawAttributes), &attributes); err != nil {
		t.Fatalf("decode initial sensor attributes: %v", err)
	}
	if attributes["tuya_climate_source"] != true ||
		attributes["tuya_sensor_kind"] != "temperature_humidity" ||
		attributes["device_class"] != "temperature" ||
		attributes["unit_of_measurement"] != "°C" {
		t.Fatalf("thermo-hygrometer climate metadata missing before first reading: %#v", attributes)
	}
}


type tuyaFailingListRepository struct {
	integration.Repository
}

func (r tuyaFailingListRepository) List(context.Context) ([]*integration.Integration, error) {
	return nil, errors.New("synthetic integration repository failure")
}

func TestTuyaCloudProviderFallsBackWhenPluginRegistryCannotInspectInstances(t *testing.T) {
	db, err := sqlite.Open(config.DatabaseConfig{
		Path: ":memory:", WALMode: false, BusyTimeoutMs: 1000, MaxOpenConns: 1,
	})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := sqlite.Migrate(context.Background(), db); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	cipher, err := crypto.NewEnvelopeCipher([]byte("tuya-provider-fallback-test-key"))
	if err != nil {
		t.Fatalf("create test cipher: %v", err)
	}
	repos := sqlite.NewRepositories(db)
	repos.Cipher = cipher
	repos.Integrations = tuyaFailingListRepository{Repository: newPluginHandlerIntegrationRepo()}
	manager, err := apppluginruntime.NewManager(repos, nil)
	if err != nil {
		t.Fatalf("create plugin manager: %v", err)
	}

	handler := NewTuyaCloudHandler(nil, manager)
	provider, err := handler.providerForConfig(context.Background(), tuya_cloud.Config{
		ClientID: "test-client", Secret: "test-secret",
		BaseURL: "https://openapi.tuyaeu.com", UID: "eu-test-user",
	})
	if err != nil {
		t.Fatalf("providerForConfig should preserve Tuya Cloud availability: %v", err)
	}
	if _, ok := provider.(*tuya_cloud.Client); !ok {
		t.Fatalf("expected direct compatibility client fallback, got %T", provider)
	}
}

func TestTuyaCloudProviderReusesPreparedCustomInstanceID(t *testing.T) {
	repo := newPluginHandlerIntegrationRepo()
	cipher, err := crypto.NewEnvelopeCipher([]byte("tuya-provider-custom-id-test-key"))
	if err != nil {
		t.Fatalf("create test cipher: %v", err)
	}
	manager := newPluginRuntimeHandlerManager(t, repo, cipher)
	adapter, err := manager.Prepare(context.Background(), tuya_cloud.PluginType, "tuya_cloud_custom", []byte(`{"client_id":"old-client","secret":"old-secret","base_url":"https://openapi.tuyaeu.com","uid":"eu-old-user"}`))
	if err != nil {
		t.Fatalf("prepare custom Tuya instance: %v", err)
	}

	handler := NewTuyaCloudHandler(nil, manager)
	provider, err := handler.providerForConfig(context.Background(), tuya_cloud.Config{
		ClientID: "new-client", Secret: "new-secret",
		BaseURL: "https://openapi.tuyaeu.com", UID: "eu-new-user",
	})
	if err != nil {
		t.Fatalf("providerForConfig: %v", err)
	}
	expectedProvider, ok := adapter.(tuya_cloud.Provider)
	if !ok {
		t.Fatalf("prepared adapter does not implement tuya_cloud.Provider: %T", adapter)
	}
	if provider != expectedProvider {
		t.Fatalf("expected existing custom-ID adapter to be reused, got %T", provider)
	}
	if got := provider.Config(); got.ClientID != "new-client" || got.UID != "eu-new-user" {
		t.Fatalf("provider credentials were not refreshed: client_id=%q uid=%q", got.ClientID, got.UID)
	}
}
