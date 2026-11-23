package handlers

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/domain/device"
	"github.com/domus-hub/domus/internal/domain/entity"
	"github.com/domus-hub/domus/internal/domain/integration"
	"github.com/domus-hub/domus/internal/infrastructure/crypto"
	"github.com/domus-hub/domus/internal/plugins/midea"
	"github.com/go-chi/chi/v5"
)

type fakeMideaIntegrationRepo struct {
	records map[string]*integration.Integration
}

func newFakeMideaIntegrationRepo() *fakeMideaIntegrationRepo {
	return &fakeMideaIntegrationRepo{records: make(map[string]*integration.Integration)}
}
func (r *fakeMideaIntegrationRepo) Create(_ context.Context, item *integration.Integration) error {
	copy := *item
	r.records[item.ID] = &copy
	return nil
}
func (r *fakeMideaIntegrationRepo) GetByID(_ context.Context, id string) (*integration.Integration, error) {
	item := r.records[id]
	if item == nil {
		return nil, integration.ErrIntegrationNotFound
	}
	copy := *item
	return &copy, nil
}
func (r *fakeMideaIntegrationRepo) List(context.Context) ([]*integration.Integration, error) {
	result := make([]*integration.Integration, 0, len(r.records))
	for _, item := range r.records {
		copy := *item
		result = append(result, &copy)
	}
	return result, nil
}
func (r *fakeMideaIntegrationRepo) Update(_ context.Context, item *integration.Integration) error {
	if r.records[item.ID] == nil {
		return integration.ErrIntegrationNotFound
	}
	copy := *item
	r.records[item.ID] = &copy
	return nil
}
func (r *fakeMideaIntegrationRepo) SetStatus(_ context.Context, id, status, lastError string) error {
	item := r.records[id]
	if item == nil {
		return integration.ErrIntegrationNotFound
	}
	item.Status, item.LastError = status, lastError
	return nil
}
func (r *fakeMideaIntegrationRepo) Delete(_ context.Context, id string) error {
	if r.records[id] == nil {
		return integration.ErrIntegrationNotFound
	}
	delete(r.records, id)
	return nil
}

type fakeMideaDeviceRepo struct {
	items map[string]*device.Device
}
func newFakeMideaDeviceRepo() *fakeMideaDeviceRepo {
	return &fakeMideaDeviceRepo{items: make(map[string]*device.Device)}
}
func mideaDeviceExternalKey(integrationID, externalID string) string {
	return integrationID + "|" + externalID
}
func (r *fakeMideaDeviceRepo) Create(_ context.Context, item *device.Device) error {
	copy := *item
	r.items[mideaDeviceExternalKey(item.IntegrationID, item.ExternalID)] = &copy
	return nil
}
func (r *fakeMideaDeviceRepo) GetByID(_ context.Context, id string) (*device.Device, error) {
	for _, item := range r.items {
		if item.ID == id {
			copy := *item
			return &copy, nil
		}
	}
	return nil, device.ErrDeviceNotFound
}
func (r *fakeMideaDeviceRepo) GetByExternalID(_ context.Context, integrationID, externalID string) (*device.Device, error) {
	item := r.items[mideaDeviceExternalKey(integrationID, externalID)]
	if item == nil {
		return nil, device.ErrDeviceNotFound
	}
	copy := *item
	return &copy, nil
}
func (r *fakeMideaDeviceRepo) List(context.Context, device.ListFilter) ([]*device.Device, error) {
	result := make([]*device.Device, 0, len(r.items))
	for _, item := range r.items {
		copy := *item
		result = append(result, &copy)
	}
	return result, nil
}
func (r *fakeMideaDeviceRepo) Update(_ context.Context, item *device.Device) error {
	key := mideaDeviceExternalKey(item.IntegrationID, item.ExternalID)
	if r.items[key] == nil {
		return device.ErrDeviceNotFound
	}
	copy := *item
	r.items[key] = &copy
	return nil
}
func (r *fakeMideaDeviceRepo) SetOnline(_ context.Context, id string, online bool) error {
	for _, item := range r.items {
		if item.ID == id {
			item.Online = online
			return nil
		}
	}
	return device.ErrDeviceNotFound
}
func (r *fakeMideaDeviceRepo) Delete(_ context.Context, id string) error {
	for key, item := range r.items {
		if item.ID == id {
			delete(r.items, key)
			return nil
		}
	}
	return device.ErrDeviceNotFound
}

type fakeMideaEntityRepo struct {
	items map[string]*entity.Entity
}
func newFakeMideaEntityRepo() *fakeMideaEntityRepo {
	return &fakeMideaEntityRepo{items: make(map[string]*entity.Entity)}
}
func (r *fakeMideaEntityRepo) Create(_ context.Context, item *entity.Entity) error {
	copy := *item
	r.items[item.EntityKey] = &copy
	return nil
}
func (r *fakeMideaEntityRepo) GetByID(_ context.Context, id string) (*entity.Entity, error) {
	for _, item := range r.items {
		if item.ID == id {
			copy := *item
			return &copy, nil
		}
	}
	return nil, entity.ErrEntityNotFound
}
func (r *fakeMideaEntityRepo) GetByKey(_ context.Context, key string) (*entity.Entity, error) {
	item := r.items[key]
	if item == nil {
		return nil, entity.ErrEntityNotFound
	}
	copy := *item
	return &copy, nil
}
func (r *fakeMideaEntityRepo) List(context.Context, entity.ListFilter) ([]*entity.Entity, error) {
	result := make([]*entity.Entity, 0, len(r.items))
	for _, item := range r.items {
		copy := *item
		result = append(result, &copy)
	}
	return result, nil
}
func (r *fakeMideaEntityRepo) Update(_ context.Context, item *entity.Entity) error {
	if r.items[item.EntityKey] == nil {
		return entity.ErrEntityNotFound
	}
	copy := *item
	r.items[item.EntityKey] = &copy
	return nil
}
func (r *fakeMideaEntityRepo) UpdateState(_ context.Context, id, stateJSON, attributesJSON string, available bool, at time.Time) error {
	for _, item := range r.items {
		if item.ID == id {
			item.State, item.Attributes, item.Available, item.LastUpdatedAt = stateJSON, attributesJSON, available, at
			return nil
		}
	}
	return entity.ErrEntityNotFound
}
func (r *fakeMideaEntityRepo) Delete(_ context.Context, id string) error {
	for key, item := range r.items {
		if item.ID == id {
			delete(r.items, key)
			return nil
		}
	}
	return entity.ErrEntityNotFound
}

func newMideaHandlerFixture(t *testing.T) (*MideaHandler, *fakeMideaIntegrationRepo, *fakeMideaDeviceRepo, *fakeMideaEntityRepo) {
	t.Helper()
	cipher, err := crypto.NewEnvelopeCipher(bytes.Repeat([]byte{0x45}, 32))
	if err != nil {
		t.Fatal(err)
	}
	integrations := newFakeMideaIntegrationRepo()
	devices := newFakeMideaDeviceRepo()
	entities := newFakeMideaEntityRepo()
	handler, err := NewMideaHandler(integrations, cipher, devices, entities)
	if err != nil {
		t.Fatal(err)
	}
	handler.discover = func(context.Context) ([]midea.DiscoveredAppliance, error) {
		return []midea.DiscoveredAppliance{{
			ID: "123456789", Name: "Midea dehumidifier", Host: "192.168.1.50",
			Port: 6444, Model: "MDDF16WF", TypeCode: "A1",
			Protocol: midea.ProtocolMSmartV2, Supported: true,
		}}, nil
	}
	return handler, integrations, devices, entities
}

func TestMideaHandlerRequiresDependencies(t *testing.T) {
	if _, err := NewMideaHandler(nil, nil, nil, nil); err == nil {
		t.Fatal("Midea handler must require encrypted storage and Domus repositories")
	}
}

func TestMideaConfigureRequiresRecentLANDiscovery(t *testing.T) {
	handler, _, _, _ := newMideaHandlerFixture(t)
	body := "{\"id\":\"123456789\",\"name\":\"Dryer\",\"type_code\":\"A1\",\"host\":\"192.168.1.50\",\"protocol\":\"midea_msmart_v2\"}"
	request := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/midea/devices", strings.NewReader(body))
	response := httptest.NewRecorder()
	handler.ConfigureDevice(response, request)
	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, want conflict; body=%s", response.Code, response.Body.String())
	}
	if len(handler.adapter.ListDevices()) != 0 {
		t.Fatal("undiscovered LAN target must never be registered")
	}
}

func TestMideaConfigureProvisionsDeviceEntitiesAndEncryptedConfig(t *testing.T) {
	handler, integrations, devices, entities := newMideaHandlerFixture(t)
	discovered, err := handler.discover(context.Background())
	if err != nil { t.Fatal(err) }
	handler.discovered = map[string]midea.DiscoveredAppliance{discovered[0].ID: discovered[0]}

	body := "{\"id\":\"123456789\",\"name\":\"Basement dehumidifier\",\"brand\":\"Comfee\",\"model\":\"MDDF16WF\",\"type_code\":\"A1\",\"host\":\"192.168.1.50\",\"port\":6444,\"protocol\":\"midea_msmart_v2\"}"
	request := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/midea/devices", strings.NewReader(body))
	response := httptest.NewRecorder()
	handler.ConfigureDevice(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want created; body=%s", response.Code, response.Body.String())
	}
	if got := len(handler.adapter.ListDevices()); got != 1 {
		t.Fatalf("runtime devices = %d, want 1", got)
	}
	if got := len(devices.items); got != 1 {
		t.Fatalf("Domus device records = %d, want 1", got)
	}
	if got := len(entities.items); got != 4 {
		t.Fatalf("Domus entities = %d, want 4 (humidifier, humidity, temperature, tank)", got)
	}
	record, err := integrations.GetByID(context.Background(), mideaIntegrationID)
	if err != nil { t.Fatal(err) }
	if !strings.HasPrefix(string(record.ConfigEncrypted), "ENC:v1:") {
		t.Fatal("integration credentials/configuration were not encrypted at rest")
	}
	if strings.Contains(string(record.ConfigEncrypted), "192.168.1.50") {
		t.Fatal("encrypted integration record unexpectedly contains plaintext address")
	}
	if strings.Contains(response.Body.String(), "token") || strings.Contains(response.Body.String(), "key_hex") {
		t.Fatal("API response leaked secret field names")
	}
}

func TestMideaConfigureRejectsDiscoveryProtocolMismatch(t *testing.T) {
	handler, _, _, _ := newMideaHandlerFixture(t)
	discovered, err := handler.discover(context.Background())
	if err != nil { t.Fatal(err) }
	handler.discovered = map[string]midea.DiscoveredAppliance{discovered[0].ID: discovered[0]}
	body := "{\"id\":\"123456789\",\"name\":\"Basement dehumidifier\",\"type_code\":\"A1\",\"host\":\"192.168.1.50\",\"port\":6444,\"protocol\":\"midea_msmart_v3\"}"
	request := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/midea/devices", strings.NewReader(body))
	response := httptest.NewRecorder()
	handler.ConfigureDevice(response, request)
	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, want discovery conflict; body=%s", response.Code, response.Body.String())
	}
}

func TestMideaConfigureRequiresValidV3Credentials(t *testing.T) {
	handler, _, _, _ := newMideaHandlerFixture(t)
	discovered, err := handler.discover(context.Background())
	if err != nil { t.Fatal(err) }
	item := discovered[0]
	item.Protocol = midea.ProtocolMSmartV3
	handler.discovered = map[string]midea.DiscoveredAppliance{item.ID: item}
	body := "{\"id\":\"123456789\",\"name\":\"Basement dehumidifier\",\"type_code\":\"A1\",\"host\":\"192.168.1.50\",\"port\":6444,\"protocol\":\"midea_msmart_v3\",\"token\":\"short\",\"key\":\"short\"}"
	request := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/midea/devices", strings.NewReader(body))
	response := httptest.NewRecorder()
	handler.ConfigureDevice(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want bad request; body=%s", response.Code, response.Body.String())
	}
	if len(handler.adapter.ListDevices()) != 0 {
		t.Fatal("invalid V3 credentials must not register a device")
	}
}

func TestMideaHandlerRestoresEncryptedConfiguration(t *testing.T) {
	handler, integrations, devices, entities := newMideaHandlerFixture(t)
	discovered, err := handler.discover(context.Background())
	if err != nil { t.Fatal(err) }
	handler.discovered = map[string]midea.DiscoveredAppliance{discovered[0].ID: discovered[0]}
	body := "{\"id\":\"123456789\",\"name\":\"Basement dehumidifier\",\"type_code\":\"A1\",\"host\":\"192.168.1.50\",\"protocol\":\"midea_msmart_v2\"}"
	request := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/midea/devices", strings.NewReader(body))
	response := httptest.NewRecorder()
	handler.ConfigureDevice(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("configure status = %d; body=%s", response.Code, response.Body.String())
	}

	cipher, err := crypto.NewEnvelopeCipher(bytes.Repeat([]byte{0x45}, 32))
	if err != nil { t.Fatal(err) }
	restored, err := NewMideaHandler(integrations, cipher, devices, entities)
	if err != nil { t.Fatalf("restore encrypted config: %v", err) }
	if got := len(restored.adapter.ListDevices()); got != 1 {
		t.Fatalf("restored runtime devices = %d, want 1", got)
	}
}

func TestMideaDeleteRemovesRuntimeAndRegistryRecords(t *testing.T) {
	handler, integrations, devices, entities := newMideaHandlerFixture(t)
	discovered, err := handler.discover(context.Background())
	if err != nil { t.Fatal(err) }
	handler.discovered = map[string]midea.DiscoveredAppliance{discovered[0].ID: discovered[0]}
	body := "{\"id\":\"123456789\",\"name\":\"Basement dehumidifier\",\"type_code\":\"A1\",\"host\":\"192.168.1.50\",\"protocol\":\"midea_msmart_v2\"}"
	request := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/midea/devices", strings.NewReader(body))
	response := httptest.NewRecorder()
	handler.ConfigureDevice(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("configure status = %d; body=%s", response.Code, response.Body.String())
	}
	router := chi.NewRouter()
	router.Delete("/devices/{id}", handler.DeleteDevice)
	deleteRequest := httptest.NewRequest(http.MethodDelete, "/devices/123456789", nil)
	deleteResponse := httptest.NewRecorder()
	router.ServeHTTP(deleteResponse, deleteRequest)
	if deleteResponse.Code != http.StatusOK {
		t.Fatalf("delete status = %d; body=%s", deleteResponse.Code, deleteResponse.Body.String())
	}
	if len(handler.adapter.ListDevices()) != 0 || len(devices.items) != 0 || len(entities.items) != 0 {
		t.Fatal("Midea runtime device or Domus registry records were not removed")
	}
	if len(handler.config.Devices) != 0 {
		t.Fatal("persisted configuration still contains deleted device")
	}
	if _, err := integrations.GetByID(context.Background(), mideaIntegrationID); err != nil {
		t.Fatalf("integration status record should remain for disconnected state: %v", err)
	}
}

func TestMideaSafeErrorRedactsCredentialHints(t *testing.T) {
	for _, raw := range []string{"bad token value", "invalid password", "key_hex mismatch", "secret rejected"} {
		got := safeMideaError(errors.New(raw))
		lower := strings.ToLower(got)
		if strings.Contains(lower, "token") || strings.Contains(lower, "password") ||
			strings.Contains(lower, "key_hex") || strings.Contains(lower, "secret") {
			t.Errorf("sensitive error detail was not redacted: %q", got)
		}
	}
}

var _ integration.Repository = (*fakeMideaIntegrationRepo)(nil)
var _ device.Repository = (*fakeMideaDeviceRepo)(nil)
var _ entity.Repository = (*fakeMideaEntityRepo)(nil)
