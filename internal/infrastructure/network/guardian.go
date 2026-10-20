// Package network provides IoT network isolation monitoring, VLAN boundary checks,
// and WAN leak alerts to keep local-first appliances strictly offline.
package network

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/google/uuid"
)

// FlowRecord represents an observed network packet or connection flow.
type FlowRecord struct {
	SourceIP        string    `json:"source_ip"`
	SourceMAC       string    `json:"source_mac,omitempty"`
	DestinationIP   string    `json:"destination_ip"`
	DestinationPort int       `json:"destination_port"`
	Protocol        string    `json:"protocol"` // "tcp", "udp", "icmp"
	Timestamp       time.Time `json:"timestamp"`
	Bytes           int64     `json:"bytes,omitempty"`
}

// LeakAlert details an unauthorized attempt by an IoT device to reach external WAN.
type LeakAlert struct {
	ID            string     `json:"id"`
	SourceIP      string     `json:"source_ip"`
	SourceMAC     string     `json:"source_mac,omitempty"`
	DeviceName    string     `json:"device_name,omitempty"`
	DestinationIP string     `json:"destination_ip"`
	Port          int        `json:"port"`
	Protocol      string     `json:"protocol"`
	DetectedAt    time.Time  `json:"detected_at"`
	Reason        string     `json:"reason"`
	Flow          FlowRecord `json:"flow"`
}

// GuardianConfig configures network boundaries and whitelist rules.
type GuardianConfig struct {
	IoTSlotsSubnets      []string          `json:"iot_subnets"`       // e.g. ["192.168.30.0/24"]
	IsolatedMACs         []string          `json:"isolated_macs"`     // Specific isolated MACs
	AllowedExternalIPs   []string          `json:"allowed_external"`  // e.g. Public NTP ["129.6.15.28"]
	DeviceAliases        map[string]string `json:"device_aliases"`    // IP or MAC -> Name
	AlertCooldownSeconds int               `json:"cooldown_seconds"`  // Prevents notification flooding
}

// Guardian monitors local IoT traffic for unauthorized external WAN communications.
type Guardian struct {
	mu           sync.RWMutex
	cfg          GuardianConfig
	subnets      []*net.IPNet
	allowedIPs   map[string]bool
	isolatedMACs map[string]bool
	lastAlerts   map[string]time.Time // key: source_ip -> last alert time
	history      []LeakAlert
	bus          events.Bus
}

// NewGuardian instantiates an IoT Network Guardian.
func NewGuardian(bus events.Bus, cfg GuardianConfig) (*Guardian, error) {
	g := &Guardian{
		cfg:          cfg,
		allowedIPs:   make(map[string]bool),
		isolatedMACs: make(map[string]bool),
		lastAlerts:   make(map[string]time.Time),
		history:      make([]LeakAlert, 0, 50),
		bus:          bus,
	}

	for _, s := range cfg.IoTSlotsSubnets {
		_, ipNet, err := net.ParseCIDR(s)
		if err != nil {
			return nil, fmt.Errorf("guardian: invalid IoT subnet CIDR %q: %w", s, err)
		}
		g.subnets = append(g.subnets, ipNet)
	}

	for _, ip := range cfg.AllowedExternalIPs {
		g.allowedIPs[ip] = true
	}

	for _, mac := range cfg.IsolatedMACs {
		norm := normalizeMAC(mac)
		if norm != "" {
			g.isolatedMACs[norm] = true
		}
	}

	if g.cfg.AlertCooldownSeconds <= 0 {
		g.cfg.AlertCooldownSeconds = 60
	}

	return g, nil
}

// InspectFlow checks a network connection. If an isolated IoT node contacts external WAN,
// it triggers a security alert and publishes a domain event.
func (g *Guardian) InspectFlow(flow FlowRecord) (bool, *LeakAlert) {
	if flow.Timestamp.IsZero() {
		flow.Timestamp = time.Now()
	}

	srcIP := net.ParseIP(flow.SourceIP)
	dstIP := net.ParseIP(flow.DestinationIP)
	if srcIP == nil || dstIP == nil {
		return false, nil
	}

	// 1. Determine if source device is subject to IoT isolation
	isIoT := g.isDeviceIsolated(srcIP, flow.SourceMAC)
	if !isIoT {
		return false, nil
	}

	// 2. Check if destination is external WAN (not private RFC1918 / loopback / link-local / multicast)
	if isPrivateOrLocal(dstIP) {
		return false, nil
	}

	// 3. Check whitelist (e.g. approved NTP servers)
	g.mu.RLock()
	if g.allowedIPs[dstIP.String()] {
		g.mu.RUnlock()
		return false, nil
	}

	// 4. Rate-limit alerts per device
	cooldown := time.Duration(g.cfg.AlertCooldownSeconds) * time.Second
	lastTime, exists := g.lastAlerts[flow.SourceIP]
	if exists && time.Since(lastTime) < cooldown {
		g.mu.RUnlock()
		return true, nil // Still a leak, but muted by cooldown
	}
	g.mu.RUnlock()

	// 5. Construct Alert
	deviceName := "Unknown IoT Device"
	if alias, ok := g.cfg.DeviceAliases[flow.SourceIP]; ok {
		deviceName = alias
	} else if flow.SourceMAC != "" {
		if alias, ok := g.cfg.DeviceAliases[normalizeMAC(flow.SourceMAC)]; ok {
			deviceName = alias
		}
	}

	alert := LeakAlert{
		ID:            uuid.NewString(),
		SourceIP:      flow.SourceIP,
		SourceMAC:     flow.SourceMAC,
		DeviceName:    deviceName,
		DestinationIP: flow.DestinationIP,
		Port:          flow.DestinationPort,
		Protocol:      flow.Protocol,
		DetectedAt:    flow.Timestamp,
		Reason:        fmt.Sprintf("Isolated IoT node attempted outbound WAN connection to %s:%d via %s", flow.DestinationIP, flow.DestinationPort, flow.Protocol),
		Flow:          flow,
	}

	g.mu.Lock()
	g.lastAlerts[flow.SourceIP] = flow.Timestamp
	if len(g.history) >= 100 {
		g.history = g.history[1:] // Maintain rolling buffer of 100 alerts
	}
	g.history = append(g.history, alert)
	g.mu.Unlock()

	slog.Warn("IoT WAN LEAK DETECTED",
		"device", deviceName,
		"src_ip", alert.SourceIP,
		"dst_ip", alert.DestinationIP,
		"dst_port", alert.Port,
		"protocol", alert.Protocol,
	)

	// Publish domain event
	if g.bus != nil {
		go func(a LeakAlert) {
			pubCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = g.bus.Publish(pubCtx, events.Event{
				ID:         a.ID,
				Type:       events.NetworkLeakAlert,
				OccurredAt: a.DetectedAt,
				Source:     "network.guardian",
				Payload:    a,
			})
		}(alert)
	}

	return true, &alert
}

// RecentAlerts returns the history of detected leak attempts.
func (g *Guardian) RecentAlerts() []LeakAlert {
	g.mu.RLock()
	defer g.mu.RUnlock()
	res := make([]LeakAlert, len(g.history))
	copy(res, g.history)
	return res
}

func (g *Guardian) isDeviceIsolated(ip net.IP, mac string) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if mac != "" {
		norm := normalizeMAC(mac)
		if g.isolatedMACs[norm] {
			return true
		}
	}

	for _, subnet := range g.subnets {
		if subnet.Contains(ip) {
			return true
		}
	}
	return false
}

// isPrivateOrLocal tests whether an IP is RFC1918, loopback, multicast, or link-local.
func isPrivateOrLocal(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil {
		// 10.0.0.0/8
		if ip4[0] == 10 {
			return true
		}
		// 172.16.0.0/12
		if ip4[0] == 172 && (ip4[1] >= 16 && ip4[1] <= 31) {
			return true
		}
		// 192.168.0.0/16
		if ip4[0] == 192 && ip4[1] == 168 {
			return true
		}
		// Carrier-grade NAT 100.64.0.0/10
		if ip4[0] == 100 && (ip4[1] >= 64 && ip4[1] <= 127) {
			return true
		}
	}
	return false
}
