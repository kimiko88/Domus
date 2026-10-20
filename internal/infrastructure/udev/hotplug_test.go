package udev_test

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/infrastructure/udev"
)

type mockBus struct {
	mu     sync.Mutex
	events []events.Event
}

func (b *mockBus) Publish(ctx context.Context, e events.Event) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = append(b.events, e)
	return nil
}

func (b *mockBus) Subscribe(eventType events.EventType, handler events.Handler) func() {
	return func() {}
}

func (b *mockBus) SubscribeAll(handler events.Handler) func() {
	return func() {}
}

func TestMonitor_DetectsPluggedAndUnpluggedDongles(t *testing.T) {
	tempDir := t.TempDir()
	bus := &mockBus{}

	var pluggedCallbackCalled bool
	var callbackMutex sync.Mutex

	monitor := udev.NewMonitor(bus, udev.Config{
		DirPath:    tempDir,
		PollPeriod: 20 * time.Millisecond,
		OnPlugged: func(d udev.DiscoveredUSBSerial) {
			callbackMutex.Lock()
			pluggedCallbackCalled = true
			callbackMutex.Unlock()
		},
	})

	// 1. Initial scan with empty directory
	plugged, unplugged, err := monitor.ScanOnce()
	if err != nil {
		t.Fatalf("ScanOnce failed: %v", err)
	}
	if len(plugged) != 0 || len(unplugged) != 0 {
		t.Errorf("expected 0 plugged and 0 unplugged")
	}

	// 2. Simulate inserting a Sonoff Zigbee CC2652 dongle
	sonoffName := "usb-ITead_Sonoff_Zigbee_3.0_USB_Dongle_Plus_0001-if00-port0"
	sonoffPath := filepath.Join(tempDir, sonoffName)
	if err := os.WriteFile(sonoffPath, []byte("fake-serial-device"), 0644); err != nil {
		t.Fatalf("failed to create fake device: %v", err)
	}

	// Also simulate a Zooz 800 Z-Wave stick
	zoozName := "usb-Zooz_800_Z-Wave_Stick_789456-if00"
	zoozPath := filepath.Join(tempDir, zoozName)
	if err := os.WriteFile(zoozPath, []byte("fake-serial-device"), 0644); err != nil {
		t.Fatalf("failed to create fake device: %v", err)
	}

	plugged, _, err = monitor.ScanOnce()
	if err != nil {
		t.Fatalf("ScanOnce error: %v", err)
	}
	if len(plugged) != 2 {
		t.Fatalf("expected 2 plugged devices, got %d", len(plugged))
	}

	// Verify Sonoff classification
	var sonoffDev, zoozDev udev.DiscoveredUSBSerial
	for _, d := range plugged {
		if d.ID == sonoffName {
			sonoffDev = d
		}
		if d.ID == zoozName {
			zoozDev = d
		}
	}

	if sonoffDev.Type != udev.DeviceZigbee {
		t.Errorf("expected Sonoff to be DeviceZigbee, got %q", sonoffDev.Type)
	}
	if sonoffDev.Vendor != "ITEAD Sonoff" {
		t.Errorf("expected vendor ITEAD Sonoff, got %q", sonoffDev.Vendor)
	}

	// Verify Zooz classification
	if zoozDev.Type != udev.DeviceZWave {
		t.Errorf("expected Zooz to be DeviceZWave, got %q", zoozDev.Type)
	}

	callbackMutex.Lock()
	if !pluggedCallbackCalled {
		t.Errorf("expected onPlugged callback to be called")
	}
	callbackMutex.Unlock()

	// Check ListDevices
	devices := monitor.ListDevices()
	if len(devices) != 2 {
		t.Errorf("expected 2 devices in ListDevices, got %d", len(devices))
	}

	// 3. Simulate removing the Zooz stick
	if err := os.Remove(zoozPath); err != nil {
		t.Fatalf("failed to remove zooz device: %v", err)
	}

	_, unplugged, err = monitor.ScanOnce()
	if err != nil {
		t.Fatalf("ScanOnce error: %v", err)
	}
	if len(unplugged) != 1 || unplugged[0].ID != zoozName {
		t.Fatalf("expected zooz unplugged, got %v", unplugged)
	}

	if len(monitor.ListDevices()) != 1 {
		t.Errorf("expected 1 remaining device, got %d", len(monitor.ListDevices()))
	}

	// Allow event bus async publishing
	time.Sleep(50 * time.Millisecond)
	bus.mu.Lock()
	eventsCount := len(bus.events)
	bus.mu.Unlock()

	// Expect 2 plugged events + 1 unplugged event = 3 total events
	if eventsCount != 3 {
		t.Errorf("expected 3 events on bus, got %d", eventsCount)
	}
}

func TestMonitor_StartStopScheduler(t *testing.T) {
	tempDir := t.TempDir()
	bus := &mockBus{}

	monitor := udev.NewMonitor(bus, udev.Config{
		DirPath:    tempDir,
		PollPeriod: 10 * time.Millisecond,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	monitor.Start(ctx)
	time.Sleep(30 * time.Millisecond)
	monitor.Stop()
	// Should cleanly terminate without panic
}
