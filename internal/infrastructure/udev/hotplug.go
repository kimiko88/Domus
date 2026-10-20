// Package udev provides automated USB serial device hotplug detection,
// persistent by-id symlink mapping, and dynamic plugin notification.
package udev

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/google/uuid"
)

// DeviceType identifies the smart home hardware protocol served by the USB dongle.
type DeviceType string

const (
	DeviceZigbee  DeviceType = "zigbee"
	DeviceZWave   DeviceType = "zwave"
	DeviceRF      DeviceType = "rf_433"
	DeviceUnknown DeviceType = "unknown"
)

// DiscoveredUSBSerial represents an enumerated serial adapter.
type DiscoveredUSBSerial struct {
	ID         string     `json:"id"`
	Path       string     `json:"path"`
	RealDevice string     `json:"real_device"`
	Type       DeviceType `json:"type"`
	Vendor     string     `json:"vendor"`
	Product    string     `json:"product"`
	PluggedAt  time.Time  `json:"plugged_at"`
}

// knownSignatures maps substrings in /dev/serial/by-id/ filenames to device types and vendors.
var knownSignatures = []struct {
	Substr  string
	Type    DeviceType
	Vendor  string
	Product string
}{
	// Zigbee
	{"ITead_Sonoff_Zigbee", DeviceZigbee, "ITEAD Sonoff", "Dongle Plus (CC2652 / CP2102N)"},
	{"Sonoff", DeviceZigbee, "ITEAD Sonoff", "Zigbee Dongle"},
	{"CC2652", DeviceZigbee, "Texas Instruments", "CC2652 Coordinator"},
	{"ConBee", DeviceZigbee, "dresden elektronik", "ConBee II/III Zigbee Gateway"},
	{"SkyConnect", DeviceZigbee, "Nabu Casa", "SkyConnect USB Coordinator"},
	{"Silicon_Labs", DeviceZigbee, "Silicon Labs", "CP210x UART Bridge (Zigbee/Thread)"},

	// Z-Wave
	{"Zooz_800", DeviceZWave, "Zooz", "Z-Wave Plus 800 Series Stick"},
	{"Zooz_700", DeviceZWave, "Zooz", "Z-Wave Plus 700 Series Stick"},
	{"Aeotec", DeviceZWave, "Aeotec", "Z-Stick Gen5/7"},
	{"Z-Wave", DeviceZWave, "Generic Z-Wave", "Serial USB Controller"},

	// RF 433 / SDR
	{"RTL2838", DeviceRF, "Realtek", "RTL-SDR / RTL2838 433MHz Receiver"},
	{"RFXtrx433", DeviceRF, "RFXCOM", "RFXtrx433 Transceiver"},
}

// Monitor tracks additions and removals of USB serial devices.
type Monitor struct {
	mu         sync.RWMutex
	dirPath    string
	devices    map[string]DiscoveredUSBSerial // key: by-id filename
	bus        events.Bus
	pollPeriod time.Duration
	stopCh     chan struct{}
	stopOnce   sync.Once
	onPlugged  func(d DiscoveredUSBSerial)
}

// Config configures the Monitor.
type Config struct {
	DirPath    string
	PollPeriod time.Duration
	OnPlugged  func(d DiscoveredUSBSerial)
}

// NewMonitor creates a USB hotplug monitor.
func NewMonitor(bus events.Bus, cfg Config) *Monitor {
	dir := cfg.DirPath
	if dir == "" {
		dir = "/dev/serial/by-id"
	}
	period := cfg.PollPeriod
	if period <= 0 {
		period = 2 * time.Second
	}

	return &Monitor{
		dirPath:    dir,
		devices:    make(map[string]DiscoveredUSBSerial),
		bus:        bus,
		pollPeriod: period,
		stopCh:     make(chan struct{}),
		onPlugged:  cfg.OnPlugged,
	}
}

// ScanOnce performs a single examination of the serial by-id directory.
func (m *Monitor) ScanOnce() ([]DiscoveredUSBSerial, []DiscoveredUSBSerial, error) {
	entries, err := os.ReadDir(m.dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Directory does not exist (no serial devices or non-Linux OS)
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("udev: failed to read %s: %w", m.dirPath, err)
	}

	current := make(map[string]DiscoveredUSBSerial)
	now := time.Now()

	for _, entry := range entries {
		name := entry.Name()
		fullPath := filepath.Join(m.dirPath, name)

		realDev, _ := filepath.EvalSymlinks(fullPath)
		if realDev == "" {
			realDev = fullPath
		}

		devType, vendor, product := classifyDevice(name)

		current[name] = DiscoveredUSBSerial{
			ID:         name,
			Path:       fullPath,
			RealDevice: realDev,
			Type:       devType,
			Vendor:     vendor,
			Product:    product,
			PluggedAt:  now,
		}
	}

	m.mu.Lock()
	var plugged []DiscoveredUSBSerial
	var unplugged []DiscoveredUSBSerial

	// Detect new devices
	for id, dev := range current {
		if _, exists := m.devices[id]; !exists {
			plugged = append(plugged, dev)
			m.devices[id] = dev
		}
	}

	// Detect removed devices
	for id, dev := range m.devices {
		if _, exists := current[id]; !exists {
			unplugged = append(unplugged, dev)
			delete(m.devices, id)
		}
	}
	m.mu.Unlock()

	// Emit events and invoke callbacks
	for _, dev := range plugged {
		slog.Info("USB smart home adapter attached",
			"id", dev.ID,
			"type", dev.Type,
			"vendor", dev.Vendor,
			"path", dev.Path,
		)
		if m.onPlugged != nil {
			m.onPlugged(dev)
		}
		if m.bus != nil {
			go m.publishEvent(events.USBDevicePlugged, dev)
		}
	}

	for _, dev := range unplugged {
		slog.Warn("USB smart home adapter detached",
			"id", dev.ID,
			"type", dev.Type,
			"path", dev.Path,
		)
		if m.bus != nil {
			go m.publishEvent(events.USBDeviceUnplugged, dev)
		}
	}

	return plugged, unplugged, nil
}

// ListDevices returns all currently connected serial devices.
func (m *Monitor) ListDevices() []DiscoveredUSBSerial {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]DiscoveredUSBSerial, 0, len(m.devices))
	for _, d := range m.devices {
		result = append(result, d)
	}
	return result
}

// Start launches the background polling loop.
func (m *Monitor) Start(ctx context.Context) {
	// Immediate scan
	_, _, _ = m.ScanOnce()

	go func() {
		ticker := time.NewTicker(m.pollPeriod)
		defer ticker.Stop()

		for {
			select {
			case <-m.stopCh:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				_, _, _ = m.ScanOnce()
			}
		}
	}()
}

// Stop cleanly terminates the monitor.
func (m *Monitor) Stop() {
	m.stopOnce.Do(func() {
		close(m.stopCh)
	})
}

func (m *Monitor) publishEvent(eventType events.EventType, dev DiscoveredUSBSerial) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = m.bus.Publish(ctx, events.Event{
		ID:         uuid.NewString(),
		Type:       eventType,
		OccurredAt: time.Now(),
		Source:     "udev.monitor",
		Payload:    dev,
	})
}

func classifyDevice(name string) (DeviceType, string, string) {
	for _, sig := range knownSignatures {
		if strings.Contains(name, sig.Substr) {
			return sig.Type, sig.Vendor, sig.Product
		}
	}
	return DeviceUnknown, "Generic Serial Device", "USB Serial Adapter"
}
