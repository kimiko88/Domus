package midea

import (
	"context"
	"errors"
	"testing"
)

type fakeDriver struct {
	protocol     Protocol
	devices      []Device
	state        map[string]any
	discoverErr  error
	executeErr   error
	readErr      error
	ignoreWrites bool
	ignoreActions bool
	executeCalls int
	readCalls    int
}

func (f *fakeDriver) Protocol() Protocol { return f.protocol }

func (f *fakeDriver) Discover(context.Context) ([]Device, error) {
	return append([]Device(nil), f.devices...), f.discoverErr
}

func (f *fakeDriver) Read(context.Context, Device) (map[string]any, error) {
	f.readCalls++
	if f.readErr != nil {
		return nil, f.readErr
	}
	return cloneState(f.state), nil
}

func (f *fakeDriver) Execute(_ context.Context, _ Device, command Command) error {
	f.executeCalls++
	if f.executeErr != nil {
		return f.executeErr
	}
	if !f.ignoreWrites {
		if f.state == nil {
			f.state = make(map[string]any)
		}
		for key, value := range command.Values {
			f.state[key] = value
		}
		if command.Action == "reset_filter" && !f.ignoreActions {
			f.state["filter_cleaning_reminder"] = false
		}
	}
	return nil
}

func dehumidifierFixture() Device {
	profile, _ := NewRegistry().Resolve("A1")
	return Device{
		ID: "dehumidifier-1", Name: "Basement dehumidifier",
		Brand: "Comfee", Model: "MDDF", TypeCode: "A1",
		Kind: profile.Kind, Protocol: ProtocolMSmartV2,
		Host: "192.168.1.40", Port: 6444,
		Capabilities: profile.Capabilities, Writable: profile.Writable,
		MinHumidity: 30, MaxHumidity: 80,
		Modes: []string{"set", "continuous", "smart"},
		FanSpeeds: []string{"low", "medium", "high"},
	}
}

func TestDefaultRegistryIncludesDehumidifier(t *testing.T) {
	profile, ok := NewRegistry().Resolve("a1")
	if !ok {
		t.Fatal("A1 dehumidifier profile not found")
	}
	if profile.Kind != KindDehumidifier {
		t.Fatalf("kind = %q, want %q", profile.Kind, KindDehumidifier)
	}
	for _, capability := range []Capability{CapTargetHumidity, CapCurrentHumidity, CapWaterTankFull, CapFanSpeed} {
		if !containsCapability(profile.Capabilities, capability) {
			t.Errorf("dehumidifier profile is missing %q", capability)
		}
	}
	for _, capability := range []Capability{CapPower, CapMode, CapTargetHumidity, CapFanSpeed} {
		if !containsCapability(profile.Writable, capability) {
			t.Errorf("dehumidifier profile cannot control %q", capability)
		}
	}
}

func TestRegistryApplyReportedNarrowsDehumidifierCapabilities(t *testing.T) {
	registry := NewRegistry()
	device := dehumidifierFixture()
	if !registry.ApplyReported(&device, []Capability{CapPower, CapCurrentHumidity, CapTargetHumidity}) {
		t.Fatal("expected A1 profile to be applied")
	}
	if !containsCapability(device.Capabilities, CapTargetHumidity) {
		t.Fatal("reported target humidity capability was lost")
	}
	if containsCapability(device.Capabilities, CapWaterTankFull) {
		t.Fatal("unreported water tank capability must not be exposed")
	}
	if !containsCapability(device.Writable, CapTargetHumidity) {
		t.Fatal("reported writable target humidity capability was lost")
	}
	if containsCapability(device.Writable, CapFanSpeed) {
		t.Fatal("unreported fan speed must not remain writable")
	}
}

func TestRegistryReturnsDefensiveCopies(t *testing.T) {
	registry := NewRegistry()
	profile, ok := registry.Resolve("A1")
	if !ok {
		t.Fatal("expected A1 profile")
	}
	profile.Capabilities[0] = "corrupted"
	again, _ := registry.Resolve("A1")
	if again.Capabilities[0] == "corrupted" {
		t.Fatal("caller mutation leaked into the registry")
	}
}

func TestDeviceValidateRejectsPublicLANTarget(t *testing.T) {
	device := dehumidifierFixture()
	device.Host = "203.0.113.25"
	if err := device.Validate(); !errors.Is(err, ErrInvalidDevice) {
		t.Fatalf("expected ErrInvalidDevice for public host, got %v", err)
	}
}

func TestDeviceValidateRejectsCloudLANEndpoint(t *testing.T) {
	device := dehumidifierFixture()
	device.Protocol = ProtocolCloud
	if err := device.Validate(); !errors.Is(err, ErrInvalidDevice) {
		t.Fatalf("expected cloud endpoint validation error, got %v", err)
	}
}

func TestValidateCommandHonorsDehumidifierCapabilitiesAndLimits(t *testing.T) {
	device := dehumidifierFixture()
	if err := ValidateCommand(device, Command{Values: map[string]any{
		"power": true, "target_humidity": 45, "fan_speed": "medium",
	}}); err != nil {
		t.Fatalf("valid dehumidifier command rejected: %v", err)
	}
	if err := ValidateCommand(device, Command{Values: map[string]any{"target_humidity": 90}}); !errors.Is(err, ErrInvalidCommand) {
		t.Fatalf("expected humidity range rejection, got %v", err)
	}
	if err := ValidateCommand(device, Command{Values: map[string]any{"target_temperature": 22.0}}); !errors.Is(err, ErrInvalidCommand) {
		t.Fatalf("expected unsupported field rejection, got %v", err)
	}
	if err := ValidateCommand(device, Command{Values: map[string]any{"mode": "heat"}}); !errors.Is(err, ErrInvalidCommand) {
		t.Fatalf("expected unsupported mode rejection, got %v", err)
	}
}

func TestAdapterDiscoversAndRegistersDehumidifier(t *testing.T) {
	device := dehumidifierFixture()
	driver := &fakeDriver{protocol: ProtocolMSmartV2, devices: []Device{device}}
	adapter := NewAdapter(driver)
	found, err := adapter.Discover(context.Background())
	if err != nil {
		t.Fatalf("discovery failed: %v", err)
	}
	if len(found) != 1 || found[0].Kind != KindDehumidifier {
		t.Fatalf("unexpected discovery result: %+v", found)
	}
	if got := len(adapter.ListDevices()); got != 1 {
		t.Fatalf("registered devices = %d, want 1", got)
	}
}

func TestAdapterCommandWritesOnceAndVerifiesReadBack(t *testing.T) {
	device := dehumidifierFixture()
	driver := &fakeDriver{
		protocol: ProtocolMSmartV2,
		state: map[string]any{"power": false, "target_humidity": 60, "fan_speed": "low"},
	}
	adapter := NewAdapter(driver)
	if err := adapter.Register(device); err != nil {
		t.Fatalf("register device: %v", err)
	}
	state, err := adapter.Command(context.Background(), device.ID, Command{Values: map[string]any{
		"power": true, "target_humidity": 45,
	}})
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if driver.executeCalls != 1 || driver.readCalls != 1 {
		t.Fatalf("write/read counts = %d/%d, want 1/1", driver.executeCalls, driver.readCalls)
	}
	if state["power"] != true || state["target_humidity"] != 45 {
		t.Fatalf("unexpected verified state: %#v", state)
	}
}

func TestAdapterDoesNotRetryUncertainPhysicalWrite(t *testing.T) {
	device := dehumidifierFixture()
	driver := &fakeDriver{protocol: ProtocolMSmartV2, executeErr: errors.New("connection reset")}
	adapter := NewAdapter(driver)
	if err := adapter.Register(device); err != nil {
		t.Fatalf("register device: %v", err)
	}
	_, err := adapter.Command(context.Background(), device.ID, Command{Values: map[string]any{"power": true}})
	if err == nil {
		t.Fatal("expected uncertain delivery error")
	}
	if driver.executeCalls != 1 || driver.readCalls != 0 {
		t.Fatalf("write/read counts = %d/%d, want 1/0", driver.executeCalls, driver.readCalls)
	}
}

func TestAdapterDetectsReadBackMismatch(t *testing.T) {
	device := dehumidifierFixture()
	driver := &fakeDriver{
		protocol: ProtocolMSmartV2,
		state: map[string]any{"power": false},
		ignoreWrites: true,
	}
	adapter := NewAdapter(driver)
	if err := adapter.Register(device); err != nil {
		t.Fatalf("register device: %v", err)
	}
	_, err := adapter.Command(context.Background(), device.ID, Command{Values: map[string]any{"power": true}})
	if !errors.Is(err, ErrReadBackMismatch) {
		t.Fatalf("expected read-back mismatch, got %v", err)
	}
	if driver.executeCalls != 1 || driver.readCalls != 1 {
		t.Fatalf("write/read counts = %d/%d, want 1/1", driver.executeCalls, driver.readCalls)
	}
}

func TestAdapterDistinguishesReadBackUnavailable(t *testing.T) {
	device := dehumidifierFixture()
	driver := &fakeDriver{
		protocol: ProtocolMSmartV2,
		readErr: errors.New("timeout"),
	}
	adapter := NewAdapter(driver)
	if err := adapter.Register(device); err != nil {
		t.Fatalf("register device: %v", err)
	}
	_, err := adapter.Command(context.Background(), device.ID, Command{Values: map[string]any{"power": true}})
	if !errors.Is(err, ErrReadBackUnavailable) {
		t.Fatalf("expected read-back unavailable, got %v", err)
	}
	if driver.executeCalls != 1 {
		t.Fatalf("physical write calls = %d, want 1", driver.executeCalls)
	}
}

func TestAdapterResetFilterVerifiesReminderCleared(t *testing.T) {
	device := dehumidifierFixture()
	driver := &fakeDriver{
		protocol: ProtocolMSmartV2,
		state: map[string]any{"power": true, "filter_cleaning_reminder": true},
	}
	adapter := NewAdapter(driver)
	if err := adapter.Register(device); err != nil {
		t.Fatal(err)
	}
	state, err := adapter.Command(context.Background(), device.ID, Command{Action: "reset_filter"})
	if err != nil {
		t.Fatalf("reset filter should succeed after read-back clears reminder: %v", err)
	}
	if state["filter_cleaning_reminder"] != false {
		t.Fatalf("filter reminder state = %#v, want false", state["filter_cleaning_reminder"])
	}
	if driver.executeCalls != 1 || driver.readCalls != 1 {
		t.Fatalf("write/read calls = %d/%d, want 1/1", driver.executeCalls, driver.readCalls)
	}
}

func TestAdapterResetFilterFailsWhenReminderRemainsSet(t *testing.T) {
	device := dehumidifierFixture()
	driver := &fakeDriver{
		protocol: ProtocolMSmartV2,
		state: map[string]any{"filter_cleaning_reminder": true},
		ignoreActions: true,
	}
	adapter := NewAdapter(driver)
	if err := adapter.Register(device); err != nil {
		t.Fatal(err)
	}
	_, err := adapter.Command(context.Background(), device.ID, Command{Action: "reset_filter"})
	if !errors.Is(err, ErrReadBackMismatch) {
		t.Fatalf("expected filter reset read-back mismatch, got %v", err)
	}
	if driver.executeCalls != 1 || driver.readCalls != 1 {
		t.Fatalf("write/read calls = %d/%d, want 1/1", driver.executeCalls, driver.readCalls)
	}
}

func TestA1ProfileDoesNotAdvertiseUnimplementedTelemetry(t *testing.T) {
	profile, ok := NewRegistry().Resolve("A1")
	if !ok {
		t.Fatal("A1 profile not found")
	}
	for _, capability := range []Capability{
		CapAnion, CapPump, CapWaterLevelSet, CapFilterReminder,
		CapPower, CapMode, CapTargetHumidity, CapFanSpeed,
	} {
		if !containsCapability(profile.Capabilities, capability) {
			t.Errorf("A1 profile missing verified capability %q", capability)
		}
	}
	for _, capability := range []Capability{CapEnergy, CapErrorCode, CapFilterLife} {
		if containsCapability(profile.Capabilities, capability) {
			t.Errorf("A1 profile advertises unsupported telemetry %q", capability)
		}
	}
	for _, capability := range []Capability{CapPower, CapMode, CapTargetHumidity, CapFanSpeed, CapAnion, CapPump, CapWaterLevelSet} {
		if !containsCapability(profile.Writable, capability) {
			t.Errorf("A1 profile missing verified write capability %q", capability)
		}
	}
	for _, capability := range []Capability{CapFilterReminder, CapCurrentHumidity, CapCurrentTemperature, CapWaterTankFull} {
		if containsCapability(profile.Writable, capability) {
			t.Errorf("read-only A1 capability %q marked writable", capability)
		}
	}
}

func TestAdapterRejectsDeviceWithoutDriver(t *testing.T) {
	adapter := NewAdapter()
	err := adapter.Register(dehumidifierFixture())
	if !errors.Is(err, ErrNoDriver) {
		t.Fatalf("expected ErrNoDriver, got %v", err)
	}
}

func TestAdapterRejectsConflictingProtocolDiscovery(t *testing.T) {
	first := dehumidifierFixture()
	second := dehumidifierFixture()
	second.Protocol = ProtocolMSmartV3
	second.Host = "192.168.1.41"
	adapter := NewAdapter(
		&fakeDriver{protocol: ProtocolMSmartV2, devices: []Device{first}},
		&fakeDriver{protocol: ProtocolMSmartV3, devices: []Device{second}},
	)
	if _, err := adapter.Discover(context.Background()); err == nil {
		t.Fatal("expected conflicting protocol discovery for the same device id to fail")
	}
}

func TestAdapterRejectsDriverProtocolMismatch(t *testing.T) {
	device := dehumidifierFixture()
	device.Protocol = ProtocolMSmartV1
	driver := &fakeDriver{protocol: ProtocolMSmartV2, devices: []Device{device}}
	_, err := NewAdapter(driver).Discover(context.Background())
	if err == nil {
		t.Fatal("expected protocol mismatch to fail discovery")
	}
}
