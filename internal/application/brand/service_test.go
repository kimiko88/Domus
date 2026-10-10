package brand

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/domain/device"
	"github.com/domus-hub/domus/internal/domain/entity"
	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/domain/integration"
)

// In-memory mock repositories for testing
type mockDeviceRepo struct {
	devices map[string]*device.Device
}

func (m *mockDeviceRepo) Create(ctx context.Context, d *device.Device) error {
	m.devices[d.ID] = d
	return nil
}
func (m *mockDeviceRepo) GetByID(ctx context.Context, id string) (*device.Device, error) {
	d, ok := m.devices[id]
	if !ok {
		return nil, device.ErrDeviceNotFound
	}
	return d, nil
}
func (m *mockDeviceRepo) GetByExternalID(ctx context.Context, intID, extID string) (*device.Device, error) {
	for _, d := range m.devices {
		if d.IntegrationID == intID && d.ExternalID == extID {
			return d, nil
		}
	}
	return nil, device.ErrDeviceNotFound
}
func (m *mockDeviceRepo) List(ctx context.Context, filter device.ListFilter) ([]*device.Device, error) {
	var list []*device.Device
	for _, d := range m.devices {
		list = append(list, d)
	}
	return list, nil
}
func (m *mockDeviceRepo) Update(ctx context.Context, d *device.Device) error {
	m.devices[d.ID] = d
	return nil
}
func (m *mockDeviceRepo) SetOnline(ctx context.Context, id string, online bool) error {
	if d, ok := m.devices[id]; ok {
		d.Online = online
	}
	return nil
}
func (m *mockDeviceRepo) Delete(ctx context.Context, id string) error {
	delete(m.devices, id)
	return nil
}

type mockEntityRepo struct {
	entities map[string]*entity.Entity
}

func (m *mockEntityRepo) Create(ctx context.Context, e *entity.Entity) error {
	m.entities[e.ID] = e
	return nil
}
func (m *mockEntityRepo) GetByID(ctx context.Context, id string) (*entity.Entity, error) {
	e, ok := m.entities[id]
	if !ok {
		return nil, entity.ErrEntityNotFound
	}
	return e, nil
}
func (m *mockEntityRepo) GetByKey(ctx context.Context, key string) (*entity.Entity, error) {
	for _, e := range m.entities {
		if e.EntityKey == key {
			return e, nil
		}
	}
	return nil, entity.ErrEntityNotFound
}
func (m *mockEntityRepo) List(ctx context.Context, filter entity.ListFilter) ([]*entity.Entity, error) {
	var list []*entity.Entity
	for _, e := range m.entities {
		list = append(list, e)
	}
	return list, nil
}
func (m *mockEntityRepo) Update(ctx context.Context, e *entity.Entity) error {
	m.entities[e.ID] = e
	return nil
}
func (m *mockEntityRepo) UpdateState(ctx context.Context, id string, stateJSON, attributesJSON string, available bool, at time.Time) error {
	if e, ok := m.entities[id]; ok {
		e.State = stateJSON
		e.Attributes = attributesJSON
		e.Available = available
		e.LastChangedAt = at
		e.LastUpdatedAt = at
	}
	return nil
}
func (m *mockEntityRepo) Delete(ctx context.Context, id string) error {
	delete(m.entities, id)
	return nil
}

type mockIntegrationRepo struct {
	integrations map[string]*integration.Integration
}

func (m *mockIntegrationRepo) Create(ctx context.Context, in *integration.Integration) error {
	m.integrations[in.ID] = in
	return nil
}
func (m *mockIntegrationRepo) GetByID(ctx context.Context, id string) (*integration.Integration, error) {
	in, ok := m.integrations[id]
	if !ok {
		return nil, integration.ErrIntegrationNotFound
	}
	return in, nil
}
func (m *mockIntegrationRepo) List(ctx context.Context) ([]*integration.Integration, error) {
	var list []*integration.Integration
	for _, in := range m.integrations {
		list = append(list, in)
	}
	return list, nil
}
func (m *mockIntegrationRepo) Update(ctx context.Context, in *integration.Integration) error {
	m.integrations[in.ID] = in
	return nil
}
func (m *mockIntegrationRepo) SetStatus(ctx context.Context, id string, status, lastError string) error {
	if in, ok := m.integrations[id]; ok {
		in.Status = status
		in.LastError = lastError
	}
	return nil
}
func (m *mockIntegrationRepo) Delete(ctx context.Context, id string) error {
	delete(m.integrations, id)
	return nil
}

type mockBus struct{}

func (b *mockBus) Publish(ctx context.Context, event events.Event) error {
	return nil
}
func (b *mockBus) Subscribe(eventType events.EventType, handler events.Handler) func() {
	return func() {}
}
func (b *mockBus) SubscribeAll(handler events.Handler) func() {
	return func() {}
}
func (b *mockBus) Close() error { return nil }

func TestBrandCatalog(t *testing.T) {
	cat := DefaultCatalog()
	if len(cat) < 6 {
		t.Fatalf("expected at least 6 brand catalog entries, got %d", len(cat))
	}

	foundShelly := false
	for _, entry := range cat {
		if entry.ID == BrandShelly {
			foundShelly = true
			if !entry.ZeroCloud {
				t.Errorf("expected Shelly to be ZeroCloud = true")
			}
			if len(entry.SupportedModels) == 0 {
				t.Errorf("expected Shelly to list supported models")
			}
		}
	}
	if !foundShelly {
		t.Errorf("Shelly brand entry not found in catalog")
	}
}

func TestProbeShellyGen2(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rpc/Shelly.GetDeviceInfo" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":    "shellyplus1pm-112233",
				"model": "SNSW-001P16EU",
				"mac":   "AABBCCDDEEFF",
				"gen":   2,
				"ver":   "1.4.2",
				"app":   "Plus1PM",
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	scanner := NewScanner(ts.Client())

	dev, err := scanner.ProbeIP(context.Background(), u.Host, BrandShelly)
	if err != nil {
		t.Fatalf("expected successful Shelly probe, got: %v", err)
	}

	if dev.Brand != BrandShelly {
		t.Errorf("expected brand shelly, got %s", dev.Brand)
	}
	if dev.Model != "SNSW-001P16EU" {
		t.Errorf("expected model SNSW-001P16EU, got %s", dev.Model)
	}
	if len(dev.SuggestedEntities) < 3 {
		t.Errorf("expected at least 3 suggested entities for Shelly 1PM, got %d", len(dev.SuggestedEntities))
	}
}

func TestProbeWLED(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/json/info" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ver":  "0.14.0",
				"name": "WLED-Salotto",
				"mac":  "112233445566",
				"leds": map[string]any{
					"count": 144,
					"fps":   42,
				},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	scanner := NewScanner(ts.Client())

	dev, err := scanner.ProbeIP(context.Background(), u.Host, BrandWLED)
	if err != nil {
		t.Fatalf("expected successful WLED probe, got: %v", err)
	}

	if dev.Brand != BrandWLED {
		t.Errorf("expected brand wled, got %s", dev.Brand)
	}
	if dev.Name != "WLED-Salotto ("+u.Host+")" {
		t.Errorf("unexpected name: %s", dev.Name)
	}
}

func TestPairDeviceAndSendCommand(t *testing.T) {
	switchSetCalled := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rpc/Shelly.GetDeviceInfo":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":    "shellyplus1pm-998877",
				"model": "SNSW-001P16EU",
				"mac":   "998877665544",
				"gen":   2,
				"ver":   "1.4.2",
				"app":   "Plus1PM",
			})
		case "/rpc/Switch.Set":
			switchSetCalled = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"was_on": false}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	port, _ := strconv.Atoi(u.Port())

	devRepo := &mockDeviceRepo{devices: make(map[string]*device.Device)}
	entRepo := &mockEntityRepo{entities: make(map[string]*entity.Entity)}
	intRepo := &mockIntegrationRepo{integrations: make(map[string]*integration.Integration)}
	bus := &mockBus{}

	svc := NewService(devRepo, entRepo, intRepo, bus, ts.Client())

	// 1. Pair the device
	d, entities, err := svc.PairDevice(context.Background(), PairRequest{
		Brand: BrandShelly,
		IP:    u.Host,
		Port:  port,
		Name:  "Shelly Cucina",
	})
	if err != nil {
		t.Fatalf("PairDevice failed: %v", err)
	}

	if d.Name != "Shelly Cucina" {
		t.Errorf("expected device name Shelly Cucina, got %s", d.Name)
	}
	if len(entities) == 0 {
		t.Fatalf("expected created entities, got 0")
	}

	// 2. Locate the switch entity key
	var switchKey string
	for _, ent := range entities {
		if ent.Domain == "switch" {
			switchKey = ent.EntityKey
			break
		}
	}
	if switchKey == "" {
		t.Fatalf("switch entity key not found in created entities")
	}

	// 3. Send command to switch
	err = svc.SendCommand(context.Background(), switchKey, "turn_on", nil)
	if err != nil {
		t.Fatalf("SendCommand failed: %v", err)
	}

	if !switchSetCalled {
		t.Errorf("expected /rpc/Switch.Set to be called on Shelly device")
	}
}
