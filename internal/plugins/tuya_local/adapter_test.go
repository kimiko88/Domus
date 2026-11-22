package tuya_local

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/domain/device"
	"github.com/domus-hub/domus/internal/domain/entity"
	"github.com/domus-hub/domus/internal/domain/events"
	_ "modernc.org/sqlite"
)

type mockBus struct {
	published []events.Event
}

func (m *mockBus) Publish(ctx context.Context, evt events.Event) error {
	m.published = append(m.published, evt)
	return nil
}

func (m *mockBus) Subscribe(eventType events.EventType, handler events.Handler) func() {
	return func() {}
}

func (m *mockBus) SubscribeAll(handler events.Handler) func() {
	return func() {}
}

func TestAdapter_Lifecycle(t *testing.T) {
	bus := &mockBus{}
	adapter := NewAdapter("tuya_test", bus)

	if adapter.ID() != "tuya_test" {
		t.Errorf("expected ID tuya_test, got %s", adapter.ID())
	}
	if adapter.Type() != "tuya_local" {
		t.Errorf("expected Type tuya_local, got %s", adapter.Type())
	}

	ctx := context.Background()
	if err := adapter.Start(ctx); err != nil {
		t.Fatalf("start failed: %v", err)
	}

	if err := adapter.HealthCheck(ctx); err != nil {
		t.Errorf("health check failed: %v", err)
	}

	adapter.RegisterDevice(DeviceConfig{
		DeviceID:  "dev_001",
		IP:        "127.0.0.1",
		LocalKey:  "0123456789abcdef",
		Category:  CategorySwitch,
		EntityKey: "switch.living_room_socket",
	})

	time.Sleep(50 * time.Millisecond)

	if err := adapter.Stop(ctx); err != nil {
		t.Fatalf("stop failed: %v", err)
	}

	if err := adapter.HealthCheck(ctx); err == nil {
		t.Errorf("expected health check failure after stop")
	}
}

func TestParseCategory(t *testing.T) {
	tests := []struct {
		input    string
		expected DeviceCategory
	}{
		{"kg", CategorySwitch},
		{"switch", CategorySwitch},
		{"cz", CategoryPlug},
		{"plug", CategoryPlug},
		{"socket", CategoryPlug},
		{"dj", CategoryLight},
		{"light", CategoryLight},
		{"dd", CategoryLight},
		{"wk", CategoryClimate},
		{"climate", CategoryClimate},
		{"unknown_xyz", CategorySwitch},
	}

	for _, tc := range tests {
		if got := ParseCategory(tc.input); got != tc.expected {
			t.Errorf("ParseCategory(%q) = %v, want %v", tc.input, got, tc.expected)
		}
	}
}

func TestAdapter_RegisterAndPersistDevice(t *testing.T) {
	bus := &mockBus{}
	adapter := NewAdapter("tuya_test", bus)

	// Discovery records the sender address, which is the actual LAN address.
	adapter.discovery.handleDiscovered(DiscoveredDevice{
		DeviceID: "dev_test_99",
		IP:       "192.168.1.77",
		Version:  "3.4",
	})

	ctx := context.Background()
	cfg := DeviceConfig{
		DeviceID: "dev_test_99",
		LocalKey: "localkey12345678",
		Category: CategoryLight,
	}

	err := adapter.RegisterAndPersistDevice(ctx, cfg, "Smart Ceiling Lamp")
	if err != nil {
		t.Fatalf("RegisterAndPersistDevice failed: %v", err)
	}

	if !adapter.IsDeviceConfigured("dev_test_99") {
		t.Errorf("device should be configured")
	}

	devices := adapter.GetConfiguredDevices()
	if len(devices) != 1 {
		t.Fatalf("expected 1 configured device, got %d", len(devices))
	}
	if devices[0].Name != "Smart Ceiling Lamp" || devices[0].Category != CategoryLight {
		t.Errorf("unexpected device in configs: %+v", devices[0])
	}
	if devices[0].IP != "192.168.1.77" || devices[0].Version != "3.4" {
		t.Fatalf("expected LAN discovery IP/version, got IP=%q version=%q", devices[0].IP, devices[0].Version)
	}
}

func TestAdapter_UnregisterDevice(t *testing.T) {
	adapter := NewAdapter("tuya_test", &mockBus{})
	adapter.RegisterDevice(DeviceConfig{
		DeviceID: "dev_remove_1",
		LocalKey: "0123456789abcdef",
		Category: CategoryClimate,
		EntityKey: "climate.tuya_dev_remove_1",
	})
	if !adapter.IsDeviceConfigured("dev_remove_1") {
		t.Fatal("device should be configured before removal")
	}
	if err := adapter.UnregisterDevice(context.Background(), "dev_remove_1"); err != nil {
		t.Fatalf("UnregisterDevice failed: %v", err)
	}
	if adapter.IsDeviceConfigured("dev_remove_1") {
		t.Fatal("device should no longer be configured")
	}
}

func TestTrustedLANIP(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"192.168.1.42", "192.168.1.42"},
		{"10.0.0.8", "10.0.0.8"},
		{"172.16.0.9", "172.16.0.9"},
		{"87.123.45.67", ""},
		{"8.8.8.8", ""},
		{"", ""},
		{"not-an-ip", ""},
	}
	for _, tt := range tests {
		if got := trustedLANIP(tt.input); got != tt.want {
			t.Errorf("trustedLANIP(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestAdapter_RejectsPublicIPOnRegistrationAndDiscovery(t *testing.T) {
	adapter := NewAdapter("tuya_test", &mockBus{})
	adapter.RegisterDevice(DeviceConfig{
		DeviceID: "dev_public_ip",
		IP: "87.123.45.67",
		LocalKey: "0123456789abcdef",
		Category: CategorySwitch,
	})
	if got := adapter.GetDeviceIP("dev_public_ip"); got != "" {
		t.Fatalf("public IP must not be registered as LAN address, got %q", got)
	}

	adapter.handleDiscoveredDevice(DiscoveredDevice{
		DeviceID: "dev_public_ip",
		IP: "87.123.45.67",
		Version: "3.5",
	})
	if got := adapter.GetDeviceIP("dev_public_ip"); got != "" {
		t.Fatalf("public IP must not be accepted from discovery, got %q", got)
	}
}

func TestAdapter_LoadStoredDevicesClearsUntrustedIPWithSingleConnection(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)

	_, err = db.Exec(`
		CREATE TABLE tuya_local_devices (
			device_id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			ip TEXT NOT NULL,
			local_key_enc TEXT NOT NULL,
			version TEXT NOT NULL,
			category TEXT NOT NULL,
			entity_key TEXT NOT NULL,
			updated_at INTEGER NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("create Tuya table: %v", err)
	}
	_, err = db.Exec(`
		INSERT INTO tuya_local_devices
			(device_id, name, ip, local_key_enc, version, category, entity_key, updated_at)
		VALUES
			('dev_legacy_wan', 'Legacy Tuya device', '87.123.45.67',
			 '0123456789abcdef', '3.3', 'switch', 'switch.tuya_legacy', 1)
	`)
	if err != nil {
		t.Fatalf("seed Tuya device: %v", err)
	}

	adapter := NewAdapter("tuya_test", &mockBus{}).WithPersistence(db, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := adapter.loadStoredDevices(ctx); err != nil {
		t.Fatalf("load stored devices: %v", err)
	}

	var storedIP string
	if err := db.QueryRowContext(context.Background(),
		"SELECT ip FROM tuya_local_devices WHERE device_id = ?", "dev_legacy_wan",
	).Scan(&storedIP); err != nil {
		t.Fatalf("read sanitized device IP: %v", err)
	}
	if storedIP != "" {
		t.Fatalf("expected persisted public IP to be cleared, got %q", storedIP)
	}

	if !adapter.IsDeviceConfigured("dev_legacy_wan") {
		t.Fatal("device configuration should be retained after IP cleanup")
	}
	deviceIP := adapter.GetDeviceIP("dev_legacy_wan")
	if deviceIP != "" {
		t.Fatalf("expected configured IP to be empty until LAN discovery, got %q", deviceIP)
	}

	devices := adapter.GetConfiguredDevices()
	if len(devices) != 1 || devices[0].LocalKey != "0123456789abcdef" {
		t.Fatalf("device credentials should be retained after IP cleanup, got %+v", devices)
	}
}

func TestAdapter_WaitForDiscoveredLANIPUsesFreshAnnouncement(t *testing.T) {
	adapter := NewAdapter("tuya_test", &mockBus{})
	adapter.RegisterDevice(DeviceConfig{
		DeviceID: "dev_lan_refresh",
		LocalKey: "0123456789abcdef",
		Version:  "3.3",
		Category: CategoryClimate,
	})

	// Simulate the UDP listener receiving a packet after LocalTuya sync.
	adapter.handleDiscoveredDevice(DiscoveredDevice{
		DeviceID: "dev_lan_refresh",
		IP:       "192.168.1.88",
		Version:  "3.5",
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	cfg, found := adapter.waitForDiscoveredLANIP(ctx, "dev_lan_refresh")
	if !found {
		t.Fatal("expected a fresh private LAN discovery address")
	}
	if cfg.IP != "192.168.1.88" {
		t.Fatalf("resolved IP = %q, want 192.168.1.88", cfg.IP)
	}
	if cfg.Version != "3.5" {
		t.Fatalf("resolved version = %q, want 3.5", cfg.Version)
	}
}

type memoryTuyaEntityRepo struct {
	entity *entity.Entity
}

func (r *memoryTuyaEntityRepo) Create(_ context.Context, item *entity.Entity) error {
	r.entity = item
	return nil
}
func (r *memoryTuyaEntityRepo) GetByID(_ context.Context, id string) (*entity.Entity, error) {
	if r.entity != nil && r.entity.ID == id { return r.entity, nil }
	return nil, entity.ErrEntityNotFound
}
func (r *memoryTuyaEntityRepo) GetByKey(_ context.Context, key string) (*entity.Entity, error) {
	if r.entity != nil && r.entity.EntityKey == key { return r.entity, nil }
	return nil, entity.ErrEntityNotFound
}
func (r *memoryTuyaEntityRepo) List(context.Context, entity.ListFilter) ([]*entity.Entity, error) {
	if r.entity == nil { return nil, nil }
	return []*entity.Entity{r.entity}, nil
}
func (r *memoryTuyaEntityRepo) Update(context.Context, *entity.Entity) error { return nil }
func (r *memoryTuyaEntityRepo) UpdateState(_ context.Context, id, stateJSON, attributesJSON string, available bool, at time.Time) error {
	if r.entity == nil || r.entity.ID != id { return entity.ErrEntityNotFound }
	r.entity.State = stateJSON
	r.entity.Attributes = attributesJSON
	r.entity.Available = available
	r.entity.LastUpdatedAt = at
	return nil
}
func (r *memoryTuyaEntityRepo) Delete(context.Context, string) error { r.entity = nil; return nil }

func TestAdapter_HandleDeviceStatusPersistsClimateTelemetryAndPreservesCloudFields(t *testing.T) {
	repo := &memoryTuyaEntityRepo{entity: &entity.Entity{
		ID: "climate-1", EntityKey: "climate.tuya_thermostat", Domain: "climate",
		State: `{"state":"heat"}`,
		Attributes: `{"current_humidity":46,"cloud_marker":"kept"}`,
		Available: true,
	}}
	adapter := NewAdapter("tuya_test", &mockBus{}).WithRepositories(nil, repo)
	adapter.handleDeviceStatus(DeviceConfig{
		DeviceID: "thermostat", Category: CategoryClimate, EntityKey: "climate.tuya_thermostat",
	}, map[string]any{"2": 225, "3": 217, "4": 1})

	var state map[string]any
	if err := json.Unmarshal([]byte(repo.entity.State), &state); err != nil {
		t.Fatalf("decode persisted state: %v", err)
	}
	if state["state"] != "heat" {
		t.Fatalf("state = %#v, expected previous heat state to be preserved", state["state"])
	}
	var attrs map[string]any
	if err := json.Unmarshal([]byte(repo.entity.Attributes), &attrs); err != nil {
		t.Fatalf("decode persisted attributes: %v", err)
	}
	if attrs["current_temperature"] != 21.7 || attrs["target_temperature"] != 22.5 {
		t.Fatalf("expected live climate DPS telemetry, got %#v", attrs)
	}
	if attrs["current_humidity"] != float64(46) || attrs["cloud_marker"] != "kept" {
		t.Fatalf("cloud fields should be preserved while merging DPS data, got %#v", attrs)
	}
}

func TestAdapter_HandleDeviceStatusPreservesTuyaSensorCloudTelemetry(t *testing.T) {
	repo := &memoryTuyaEntityRepo{entity: &entity.Entity{
		ID: "sensor-1", EntityKey: "sensor.tuya_sensor", Domain: "sensor",
		State: `{"state":"active"}`,
		Attributes: `{"current_temperature":21.4,"current_humidity":49}`,
		Available: true,
	}}
	adapter := NewAdapter("tuya_test", &mockBus{}).WithRepositories(nil, repo)
	adapter.handleDeviceStatus(DeviceConfig{
		DeviceID: "sensor", Category: CategorySensor, EntityKey: "sensor.tuya_sensor",
	}, map[string]any{"1": 42})

	var attrs map[string]any
	if err := json.Unmarshal([]byte(repo.entity.Attributes), &attrs); err != nil {
		t.Fatalf("decode persisted sensor attributes: %v", err)
	}
	if attrs["current_temperature"] != float64(21.4) || attrs["current_humidity"] != float64(49) {
		t.Fatalf("cloud temperature/humidity readings should survive unmapped local DPS, got %#v", attrs)
	}
	if attrs["dp_1"] != float64(42) {
		t.Fatalf("raw sensor DPS should also be retained, got %#v", attrs)
	}
}


type memoryTuyaDeviceRepo struct {
	devices map[string]*device.Device
}

func (r *memoryTuyaDeviceRepo) Create(_ context.Context, item *device.Device) error {
	if r.devices == nil { r.devices = make(map[string]*device.Device) }
	if _, exists := r.devices[item.ID]; exists { return errors.New("duplicate device") }
	copy := *item
	r.devices[item.ID] = &copy
	return nil
}
func (r *memoryTuyaDeviceRepo) GetByID(_ context.Context, id string) (*device.Device, error) {
	if item := r.devices[id]; item != nil { copy := *item; return &copy, nil }
	return nil, device.ErrDeviceNotFound
}
func (r *memoryTuyaDeviceRepo) GetByExternalID(_ context.Context, integrationID, externalID string) (*device.Device, error) {
	for _, item := range r.devices {
		if item.IntegrationID == integrationID && item.ExternalID == externalID { copy := *item; return &copy, nil }
	}
	return nil, device.ErrDeviceNotFound
}
func (r *memoryTuyaDeviceRepo) List(context.Context, device.ListFilter) ([]*device.Device, error) {
	items := make([]*device.Device, 0, len(r.devices))
	for _, item := range r.devices { copy := *item; items = append(items, &copy) }
	return items, nil
}
func (r *memoryTuyaDeviceRepo) Update(_ context.Context, item *device.Device) error {
	if r.devices[item.ID] == nil { return device.ErrDeviceNotFound }
	copy := *item
	r.devices[item.ID] = &copy
	return nil
}
func (r *memoryTuyaDeviceRepo) SetOnline(_ context.Context, id string, online bool) error {
	item := r.devices[id]
	if item == nil { return device.ErrDeviceNotFound }
	item.Online = online
	return nil
}
func (r *memoryTuyaDeviceRepo) Delete(_ context.Context, id string) error {
	if r.devices[id] == nil { return device.ErrDeviceNotFound }
	delete(r.devices, id)
	return nil
}

func TestAdapter_ProvisionCloudDeviceWithoutLocalKeyCreatesReadOnlyClimateSource(t *testing.T) {
	devices := &memoryTuyaDeviceRepo{devices: make(map[string]*device.Device)}
	entities := &memoryTuyaEntityRepo{}
	adapter := NewAdapter("tuya_local", &mockBus{}).WithRepositories(devices, entities)

	cfg := DeviceConfig{
		DeviceID: "thermo-hygrometer-1",
		Name: "Tuya thermo-hygrometer",
		Category: CategorySensor,
		EntityKey: "sensor.tuya_thermo-hygrometer-1",
		IntegrationID: "tuya_cloud",
		Online: true,
		InitialState: "active",
		InitialAttributes: `{"device_class":"temperature","unit_of_measurement":"°C","current_temperature":21.8,"current_humidity":47,"tuya_climate_source":true,"tuya_sensor_kind":"temperature_humidity"}`,
	}
	if err := adapter.ProvisionCloudDevice(context.Background(), cfg, cfg.Name); err != nil {
		t.Fatalf("ProvisionCloudDevice returned error: %v", err)
	}
	if adapter.IsDeviceConfigured(cfg.DeviceID) {
		t.Fatal("cloud-only source must not be registered as a LocalTuya controllable device")
	}

	dev, err := devices.GetByID(context.Background(), "tuya_"+cfg.DeviceID)
	if err != nil { t.Fatalf("cloud device was not persisted: %v", err) }
	if dev.IntegrationID != "tuya_cloud" || dev.ExternalID != cfg.DeviceID || !dev.Online {
		t.Fatalf("unexpected persisted device: %+v", dev)
	}

	if entities.entity == nil { t.Fatal("climate sensor entity was not created") }
	if entities.entity.Domain != "sensor" || entities.entity.DeviceID != dev.ID || entities.entity.Writable {
		t.Fatalf("cloud telemetry entity must be associated and read-only: %+v", entities.entity)
	}
	var attributes map[string]any
	if err := json.Unmarshal([]byte(entities.entity.Attributes), &attributes); err != nil {
		t.Fatalf("decode provisioned attributes: %v", err)
	}
	if attributes["current_temperature"] != 21.8 || attributes["current_humidity"] != float64(47) ||
		attributes["tuya_climate_source"] != true {
		t.Fatalf("climate telemetry not retained in provisioned entity: %#v", attributes)
	}
}

