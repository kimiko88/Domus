package network_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/infrastructure/network"
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

type mockARPReader struct {
	hosts []network.DiscoveredHost
	err   error
}

func (m *mockARPReader) ReadEntries(ctx context.Context) ([]network.DiscoveredHost, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.hosts, nil
}

func TestScanner_ScanARPAndOUIVendorFingerprinting(t *testing.T) {
	bus := &mockBus{}

	mockReader := &mockARPReader{
		hosts: []network.DiscoveredHost{
			{IP: "192.168.1.50", MAC: "08:3A:F2:11:22:33"}, // Espressif
			{IP: "192.168.1.60", MAC: "34:94:54:AA:BB:CC"}, // Shelly
			{IP: "192.168.1.70", MAC: "00:17:88:DD:EE:FF"}, // Philips Hue
			{IP: "192.168.1.80", MAC: "12:34:56:00:11:22"}, // Unknown
		},
	}

	scanner := network.NewScanner(bus, network.Config{
		ARPReader: mockReader,
	})

	ctx := context.Background()
	hosts, err := scanner.ScanARP(ctx)
	if err != nil {
		t.Fatalf("ScanARP failed: %v", err)
	}

	if len(hosts) != 4 {
		t.Fatalf("expected 4 hosts, got %d", len(hosts))
	}

	// 1. Verify Espressif
	esphost, ok := scanner.GetHost("08:3a:f2:11:22:33")
	if !ok {
		t.Fatalf("failed to find Espressif host by MAC")
	}
	if esphost.Vendor != "Espressif Systems" {
		t.Errorf("expected Espressif Systems vendor, got %q", esphost.Vendor)
	}
	if esphost.SuggestedPlugin != "esphome_mqtt" {
		t.Errorf("expected esphome_mqtt plugin, got %q", esphost.SuggestedPlugin)
	}
	if !esphost.IsIoT {
		t.Errorf("expected IsIoT=true for Espressif")
	}

	// 2. Verify Shelly
	shellyHost, ok := scanner.GetHost("192.168.1.60")
	if !ok {
		t.Fatalf("failed to find Shelly host by IP")
	}
	if shellyHost.Vendor != "Shelly (Allterco)" {
		t.Errorf("expected Shelly (Allterco), got %q", shellyHost.Vendor)
	}
	if shellyHost.SuggestedPlugin != "mqtt_generic" {
		t.Errorf("expected mqtt_generic, got %q", shellyHost.SuggestedPlugin)
	}

	// 3. Verify Hue
	hueHost, _ := scanner.GetHost("192.168.1.70")
	if hueHost.Vendor != "Philips Lighting" || hueHost.SuggestedPlugin != "zigbee2mqtt" {
		t.Errorf("unexpected Hue fingerprint: %v", hueHost)
	}

	// Give async event publisher a moment
	time.Sleep(50 * time.Millisecond)
	bus.mu.Lock()
	eventsCount := len(bus.events)
	bus.mu.Unlock()

	if eventsCount != 4 {
		t.Errorf("expected 4 DeviceDiscovered events, got %d", eventsCount)
	}
}

func TestScanner_ProbeHostPortsRefinement(t *testing.T) {
	bus := &mockBus{}
	mockReader := &mockARPReader{
		hosts: []network.DiscoveredHost{
			{IP: "192.168.1.99", MAC: "C4:4F:33:55:66:77"}, // Espressif
		},
	}

	scanner := network.NewScanner(bus, network.Config{
		ARPReader: mockReader,
	})

	_, _ = scanner.ScanARP(context.Background())

	h, _ := scanner.GetHost("192.168.1.99")
	if h.Confidence < 0.8 {
		t.Errorf("expected high confidence from OUI, got %.2f", h.Confidence)
	}
}

func TestScanner_ParseWindowsAndLinuxARP(t *testing.T) {
	// Sample Windows "arp -a" output
	windowsOutput := `
Interface: 192.168.1.25 --- 0x10
  Internet Address      Physical Address      Type
  192.168.1.1           e8-db-84-11-22-33     dynamic
  192.168.1.150         08-3a-f2-44-55-66     dynamic
  192.168.1.255         ff-ff-ff-ff-ff-ff     static
  224.0.0.22            01-00-5e-00-00-16     static
`
	// Sample Linux "ip neigh" output
	linuxOutput := `
192.168.1.1 dev eth0 lladdr e8:db:84:11:22:33 REACHABLE
192.168.1.150 dev eth0 lladdr 08:3a:f2:44:55:66 STALE
192.168.1.200 dev eth0 FAILED
`
	wHosts := network.ParseWindowsARPForTest([]byte(windowsOutput))
	if len(wHosts) != 2 {
		t.Errorf("expected 2 valid hosts parsed from Windows ARP, got %d", len(wHosts))
	}

	lHosts := network.ParseLinuxIPNeighForTest([]byte(linuxOutput))
	if len(lHosts) != 2 {
		t.Errorf("expected 2 valid hosts parsed from Linux ip neigh, got %d", len(lHosts))
	}
}

