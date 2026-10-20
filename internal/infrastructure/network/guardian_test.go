package network_test

import (
	"context"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/infrastructure/network"
)

func TestGuardian_DetectsWANLeakFromIsolatedIoT(t *testing.T) {
	bus := &mockBus{}

	cfg := network.GuardianConfig{
		IoTSlotsSubnets: []string{"192.168.30.0/24"},
		IsolatedMACs:    []string{"34:94:54:11:22:33"},
		AllowedExternalIPs: []string{
			"129.6.15.28", // NIST NTP Server
		},
		DeviceAliases: map[string]string{
			"192.168.30.50":     "Kitchen Smart Plug",
			"34:94:54:11:22:33": "Living Room Light Strip",
		},
		AlertCooldownSeconds: 2,
	}

	guardian, err := network.NewGuardian(bus, cfg)
	if err != nil {
		t.Fatalf("NewGuardian failed: %v", err)
	}

	// 1. Internal LAN communication should NOT trigger leak
	isLeak, alert := guardian.InspectFlow(network.FlowRecord{
		SourceIP:        "192.168.30.50",
		DestinationIP:   "192.168.1.10",
		DestinationPort: 1883,
		Protocol:        "tcp",
	})
	if isLeak || alert != nil {
		t.Errorf("internal LAN flow should not trigger leak alert")
	}

	// 2. Whitelisted external IP should NOT trigger leak
	isLeak, alert = guardian.InspectFlow(network.FlowRecord{
		SourceIP:        "192.168.30.50",
		DestinationIP:   "129.6.15.28", // NIST NTP
		DestinationPort: 123,
		Protocol:        "udp",
	})
	if isLeak || alert != nil {
		t.Errorf("whitelisted NTP IP should not trigger leak alert")
	}

	// 3. Unauthorized external WAN connection to arbitrary cloud server
	isLeak, alert = guardian.InspectFlow(network.FlowRecord{
		SourceIP:        "192.168.30.50",
		DestinationIP:   "45.33.32.156", // External WAN host
		DestinationPort: 8883,
		Protocol:        "tcp",
	})
	if !isLeak || alert == nil {
		t.Fatalf("expected WAN leak to be detected!")
	}
	if alert.DeviceName != "Kitchen Smart Plug" {
		t.Errorf("expected device alias 'Kitchen Smart Plug', got %q", alert.DeviceName)
	}
	if alert.Port != 8883 {
		t.Errorf("expected port 8883, got %d", alert.Port)
	}

	// 4. Repeated connection within cooldown should return isLeak=true but no duplicate alert
	isLeak2, alert2 := guardian.InspectFlow(network.FlowRecord{
		SourceIP:        "192.168.30.50",
		DestinationIP:   "45.33.32.156",
		DestinationPort: 8883,
		Protocol:        "tcp",
	})
	if !isLeak2 {
		t.Errorf("expected leak to still be identified")
	}
	if alert2 != nil {
		t.Errorf("expected duplicate alert to be suppressed by cooldown")
	}

	// Check history
	history := guardian.RecentAlerts()
	if len(history) != 1 {
		t.Errorf("expected 1 alert in history, got %d", len(history))
	}

	// Check domain event
	time.Sleep(50 * time.Millisecond)
	bus.mu.Lock()
	eventsCount := len(bus.events)
	bus.mu.Unlock()
	if eventsCount != 1 {
		t.Errorf("expected 1 domain event emitted, got %d", eventsCount)
	}
}

func TestGuardian_IsolatedMACCheck(t *testing.T) {
	bus := &mockBus{}
	guardian, err := network.NewGuardian(bus, network.GuardianConfig{
		IsolatedMACs: []string{"aa:bb:cc:dd:ee:ff"},
	})
	if err != nil {
		t.Fatalf("failed to init guardian: %v", err)
	}

	// Host on main subnet, but MAC is specifically flagged as untrusted IoT
	isLeak, alert := guardian.InspectFlow(network.FlowRecord{
		SourceIP:        "192.168.1.99",
		SourceMAC:       "AA:BB:CC:DD:EE:FF",
		DestinationIP:   "93.184.216.34", // example.com
		DestinationPort: 80,
		Protocol:        "tcp",
	})

	if !isLeak || alert == nil {
		t.Errorf("expected leak alert for isolated MAC")
	}
}

type mockFirewallClient struct {
	isolatedMAC string
	isolatedIP  string
	reason      string
}

func (m *mockFirewallClient) IsolateClient(ctx context.Context, mac, ip, reason string) error {
	m.isolatedMAC = mac
	m.isolatedIP = ip
	m.reason = reason
	return nil
}

func (m *mockFirewallClient) UnblockClient(ctx context.Context, mac, ip string) error {
	return nil
}

func TestGuardian_QuarantineWithFirewallClient(t *testing.T) {
	bus := &mockBus{}
	guardian, err := network.NewGuardian(bus, network.GuardianConfig{})
	if err != nil {
		t.Fatalf("failed to init guardian: %v", err)
	}

	fw := &mockFirewallClient{}
	guardian.SetFirewall(fw)

	ctx := context.Background()
	err = guardian.QuarantineDevice(ctx, "00:11:22:33:44:55", "192.168.1.200", "Critical WAN leak detected")
	if err != nil {
		t.Fatalf("QuarantineDevice failed: %v", err)
	}

	if fw.isolatedMAC != "00:11:22:33:44:55" || fw.isolatedIP != "192.168.1.200" {
		t.Errorf("expected firewall to isolate 00:11:22:33:44:55 / 192.168.1.200, got %s / %s", fw.isolatedMAC, fw.isolatedIP)
	}

	bus.mu.Lock()
	eventsCount := len(bus.events)
	bus.mu.Unlock()
	if eventsCount != 1 {
		t.Errorf("expected 1 event emitted on bus, got %d", eventsCount)
	}
}

