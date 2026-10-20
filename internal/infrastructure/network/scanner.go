// Package network implements active and passive local network discovery,
// ARP snooping, mDNS resolution, and IoT hardware vendor fingerprinting.
package network

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/google/uuid"
)

// DiscoveredHost represents a physical device detected on the local network.
type DiscoveredHost struct {
	IP                 string            `json:"ip"`
	MAC                string            `json:"mac"`
	Hostname           string            `json:"hostname,omitempty"`
	Vendor             string            `json:"vendor,omitempty"`
	SuggestedPlugin    string            `json:"suggested_plugin,omitempty"`
	Confidence         float64           `json:"confidence"` // 0.0 - 1.0
	OpenPorts          []int             `json:"open_ports,omitempty"`
	FirstSeenAt        time.Time         `json:"first_seen_at"`
	LastSeenAt         time.Time         `json:"last_seen_at"`
	IsIoT              bool              `json:"is_iot"`
	Metadata           map[string]string `json:"metadata,omitempty"`
}

// Common IoT Vendor OUI prefixes (normalized without colons, 6 hex chars).
var knownOUIVendors = map[string]struct {
	Vendor string
	Plugin string
}{
	// Espressif / WLED / ESPHome
	"083AF2": {Vendor: "Espressif Systems", Plugin: "esphome_mqtt"},
	"244CAB": {Vendor: "Espressif Systems", Plugin: "esphome_mqtt"},
	"30AEA4": {Vendor: "Espressif Systems", Plugin: "esphome_mqtt"},
	"485519": {Vendor: "Espressif Systems", Plugin: "esphome_mqtt"},
	"84F3EB": {Vendor: "Espressif Systems", Plugin: "esphome_mqtt"},
	"A4CF12": {Vendor: "Espressif Systems", Plugin: "esphome_mqtt"},
	"C44F33": {Vendor: "Espressif Systems", Plugin: "esphome_mqtt"},
	"DC4F22": {Vendor: "Espressif Systems", Plugin: "esphome_mqtt"},

	// Shelly / Allterco
	"349454": {Vendor: "Shelly (Allterco)", Plugin: "mqtt_generic"},
	"E8DB84": {Vendor: "Shelly (Allterco)", Plugin: "mqtt_generic"},
	"C45BBE": {Vendor: "Shelly (Allterco)", Plugin: "mqtt_generic"},

	// Philips Hue / Signify
	"001788": {Vendor: "Philips Lighting", Plugin: "zigbee2mqtt"},
	"ECB5FA": {Vendor: "Philips Lighting", Plugin: "zigbee2mqtt"},

	// Sonoff / ITEAD
	"001F54": {Vendor: "Sonoff (ITEAD)", Plugin: "esphome_mqtt"},
	"600194": {Vendor: "Sonoff (ITEAD)", Plugin: "esphome_mqtt"},

	// Tuya Smart
	"10521C": {Vendor: "Tuya Smart", Plugin: "tuya_local"},
	"D88039": {Vendor: "Tuya Smart", Plugin: "tuya_local"},

	// Raspberry Pi Foundation
	"B4E62D": {Vendor: "Raspberry Pi Foundation", Plugin: "domus_satellite"},
	"286DCD": {Vendor: "Raspberry Pi Foundation", Plugin: "domus_satellite"},
	"D83ADD": {Vendor: "Raspberry Pi Foundation", Plugin: "domus_satellite"},

	// Ubiquiti Networks
	"0026BB": {Vendor: "Ubiquiti Networks", Plugin: "unifi_network"},
	"702C1F": {Vendor: "Ubiquiti Networks", Plugin: "unifi_network"},
}

// ARPCacheReader defines an interface for obtaining ARP entries (allows mock injection).
type ARPCacheReader interface {
	ReadEntries(ctx context.Context) ([]DiscoveredHost, error)
}

// SystemARPCacheReader inspects the OS ARP table via native CLI utilities.
type SystemARPCacheReader struct{}

func (s *SystemARPCacheReader) ReadEntries(ctx context.Context) ([]DiscoveredHost, error) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "arp", "-a")
	} else {
		cmd = exec.CommandContext(ctx, "ip", "neigh")
	}

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute system arp command: %w", err)
	}

	if runtime.GOOS == "windows" {
		return parseWindowsARP(out), nil
	}
	return parseLinuxIPNeigh(out), nil
}

// Scanner coordinates network scans, OUI identification, and port checking.
type Scanner struct {
	mu           sync.RWMutex
	hosts        map[string]DiscoveredHost // Key: MAC or IP
	bus          events.Bus
	arpReader    ARPCacheReader
	checkPorts   []int
	scanTimeout  time.Duration
}

// Config defines Scanner options.
type Config struct {
	ARPReader   ARPCacheReader
	CheckPorts  []int
	ScanTimeout time.Duration
}

// NewScanner instantiates a network scanner.
func NewScanner(bus events.Bus, cfg Config) *Scanner {
	reader := cfg.ARPReader
	if reader == nil {
		reader = &SystemARPCacheReader{}
	}
	ports := cfg.CheckPorts
	if len(ports) == 0 {
		ports = []int{80, 554, 502, 3671, 6053, 8080}
	}
	timeout := cfg.ScanTimeout
	if timeout <= 0 {
		timeout = 2 * time.Second
	}

	return &Scanner{
		hosts:       make(map[string]DiscoveredHost),
		bus:         bus,
		arpReader:   reader,
		checkPorts:  ports,
		scanTimeout: timeout,
	}
}

// ScanARP performs an immediate ARP table examination and fingerprints detected nodes.
func (s *Scanner) ScanARP(ctx context.Context) ([]DiscoveredHost, error) {
	rawHosts, err := s.arpReader.ReadEntries(ctx)
	if err != nil {
		return nil, err
	}

	var newlyDiscovered []DiscoveredHost
	now := time.Now()

	s.mu.Lock()

	for _, h := range rawHosts {
		h.FirstSeenAt = now
		h.LastSeenAt = now
		if norm := normalizeMAC(h.MAC); norm != "" {
			h.MAC = norm
		}

		// Enrich with OUI Vendor and Suggested Plugin
		fingerprintHost(&h)

		key := h.MAC
		if key == "" {
			key = h.IP
		}

		existing, exists := s.hosts[key]
		if !exists {
			s.hosts[key] = h
			newlyDiscovered = append(newlyDiscovered, h)
		} else {
			existing.LastSeenAt = now
			existing.IP = h.IP
			if h.Hostname != "" {
				existing.Hostname = h.Hostname
			}
			s.hosts[key] = existing
		}
	}

	slog.Info("network arp scan complete",
		"total_cached", len(s.hosts),
		"new_hosts", len(newlyDiscovered),
	)

	result := make([]DiscoveredHost, 0, len(s.hosts))
	for _, h := range s.hosts {
		result = append(result, h)
	}
	s.mu.Unlock()

	// Emit domain events for newly discovered hosts
	if s.bus != nil && len(newlyDiscovered) > 0 {
		go func(hosts []DiscoveredHost) {
			for _, h := range hosts {
				pubCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				_ = s.bus.Publish(pubCtx, events.Event{
					ID:         uuid.NewString(),
					Type:       events.DeviceDiscovered,
					OccurredAt: now,
					Source:     "network.scanner",
					Payload: map[string]any{
						"ip":               h.IP,
						"mac":              h.MAC,
						"vendor":           h.Vendor,
						"suggested_plugin": h.SuggestedPlugin,
						"confidence":       h.Confidence,
					},
				})
				cancel()
			}
		}(newlyDiscovered)
	}

	return result, nil
}

// ProbeHostPorts tests which common IoT ports are open on a discovered IP.
func (s *Scanner) ProbeHostPorts(ctx context.Context, ip string) ([]int, error) {
	var open []int
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, port := range s.checkPorts {
		wg.Add(1)
		go func(p int) {
			defer wg.Done()
			addr := fmt.Sprintf("%s:%d", ip, p)
			d := net.Dialer{Timeout: 300 * time.Millisecond}
			conn, err := d.DialContext(ctx, "tcp", addr)
			if err == nil {
				_ = conn.Close()
				mu.Lock()
				open = append(open, p)
				mu.Unlock()
			}
		}(port)
	}

	wg.Wait()

	s.mu.Lock()
	for k, h := range s.hosts {
		if h.IP == ip {
			h.OpenPorts = open
			refineFingerprintWithPorts(&h)
			s.hosts[k] = h
			break
		}
	}
	s.mu.Unlock()

	return open, nil
}

// ListHosts returns all currently known discovered hosts.
func (s *Scanner) ListHosts() []DiscoveredHost {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]DiscoveredHost, 0, len(s.hosts))
	for _, h := range s.hosts {
		result = append(result, h)
	}
	return result
}

// GetHost retrieves a specific host by MAC or IP.
func (s *Scanner) GetHost(id string) (DiscoveredHost, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	normID := id
	if norm := normalizeMAC(id); norm != "" {
		normID = norm
	}

	h, ok := s.hosts[normID]
	if ok {
		return h, true
	}
	for _, item := range s.hosts {
		if item.IP == id || strings.EqualFold(item.MAC, id) {
			return item, true
		}
	}
	return DiscoveredHost{}, false
}

// fingerprintHost enriches the host based on MAC OUI.
func fingerprintHost(h *DiscoveredHost) {
	cleanMAC := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(h.MAC, ":", ""), "-", ""))
	if len(cleanMAC) >= 6 {
		oui := cleanMAC[:6]
		if info, ok := knownOUIVendors[oui]; ok {
			h.Vendor = info.Vendor
			h.SuggestedPlugin = info.Plugin
			h.Confidence = 0.85
			h.IsIoT = true
			return
		}
	}

	// Fallback generic heuristic
	if h.Vendor == "" {
		h.Vendor = "Generic LAN Host"
		h.Confidence = 0.1
	}
}

// refineFingerprintWithPorts adjusts plugin suggestion based on open TCP ports.
func refineFingerprintWithPorts(h *DiscoveredHost) {
	hasPort := func(p int) bool {
		for _, open := range h.OpenPorts {
			if open == p {
				return true
			}
		}
		return false
	}

	if hasPort(554) {
		h.SuggestedPlugin = "camera_onvif"
		h.Confidence = 0.95
		h.IsIoT = true
	} else if hasPort(6053) {
		h.SuggestedPlugin = "esphome_mqtt"
		h.Confidence = 0.98
		h.IsIoT = true
	} else if hasPort(502) {
		h.SuggestedPlugin = "modbus_tcp"
		h.Confidence = 0.95
		h.IsIoT = true
	} else if hasPort(3671) {
		h.SuggestedPlugin = "knx"
		h.Confidence = 0.95
		h.IsIoT = true
	}
}

// parseWindowsARP parses lines from "arp -a".
func parseWindowsARP(output []byte) []DiscoveredHost {
	var hosts []DiscoveredHost
	scanner := bufio.NewScanner(bytes.NewReader(output))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		fields := strings.Fields(line)
		if len(fields) >= 3 {
			ip := net.ParseIP(fields[0])
			if ip != nil && !ip.IsLoopback() && !ip.IsMulticast() {
				mac := normalizeMAC(fields[1])
				// Ignore broadcast FF-FF-FF-FF-FF-FF or incomplete
				if mac != "" && !strings.EqualFold(mac, "ff:ff:ff:ff:ff:ff") {
					hosts = append(hosts, DiscoveredHost{
						IP:  ip.String(),
						MAC: mac,
					})
				}
			}
		}
	}
	return hosts
}

// parseLinuxIPNeigh parses lines from "ip neigh".
func parseLinuxIPNeigh(output []byte) []DiscoveredHost {
	var hosts []DiscoveredHost
	scanner := bufio.NewScanner(bytes.NewReader(output))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		fields := strings.Fields(line)
		if len(fields) >= 5 {
			ip := net.ParseIP(fields[0])
			if ip != nil {
				for i, f := range fields {
					if f == "lladdr" && i+1 < len(fields) {
						mac := normalizeMAC(fields[i+1])
						if mac != "" {
							hosts = append(hosts, DiscoveredHost{
								IP:  ip.String(),
								MAC: mac,
							})
						}
					}
				}
			}
		}
	}
	return hosts
}

// normalizeMAC formats MAC into standard lower-case colon-separated format.
func normalizeMAC(raw string) string {
	clean := strings.ReplaceAll(strings.ReplaceAll(raw, "-", ":"), ".", "")
	hw, err := net.ParseMAC(clean)
	if err != nil {
		return ""
	}
	return hw.String()
}
