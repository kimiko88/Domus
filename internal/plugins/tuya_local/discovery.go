package tuya_local

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"sync"
	"time"
)

// DiscoveredDevice represents a Tuya device broadcasted on the local network.
type DiscoveredDevice struct {
	IP         string `json:"ip"`
	DeviceID   string `json:"gwId"`
	Active     int    `json:"active"`
	Version    string `json:"version"`
	ProductKey string `json:"productKey"`
}

// DiscoveryListener listens on UDP ports 6666 and 6667 for Tuya discovery packets.
type DiscoveryListener struct {
	ports    []int
	onDevice func(DiscoveredDevice)
	stopCh   chan struct{}
	wg       sync.WaitGroup
	mu       sync.RWMutex
	devices  map[string]DiscoveredDevice
}

// NewDiscoveryListener creates a listener for Tuya UDP broadcasts.
func NewDiscoveryListener(onDevice func(DiscoveredDevice)) *DiscoveryListener {
	return &DiscoveryListener{
		ports:    []int{6666, 6667},
		onDevice: onDevice,
		stopCh:   make(chan struct{}),
		devices:  make(map[string]DiscoveredDevice),
	}
}

// Start opens UDP listeners on the configured ports.
func (l *DiscoveryListener) Start(ctx context.Context) error {
	for _, port := range l.ports {
		addr := net.UDPAddr{
			Port: port,
			IP:   net.IPv4zero,
		}

		conn, err := net.ListenUDP("udp4", &addr)
		if err != nil {
			slog.Warn("tuya_local: could not bind udp port (may be in use)", "port", port, "error", err)
			continue
		}

		l.wg.Add(1)
		go l.listenLoop(conn)
	}

	return nil
}

// Stop closes the UDP listeners.
func (l *DiscoveryListener) Stop() {
	close(l.stopCh)
	l.wg.Wait()
}

func (l *DiscoveryListener) listenLoop(conn *net.UDPConn) {
	defer l.wg.Done()
	defer conn.Close()

	buf := make([]byte, 2048)

	for {
		select {
		case <-l.stopCh:
			return
		default:
		}

		_ = conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, _, err := conn.ReadFrom(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			return
		}

		if n <= 0 {
			continue
		}

		data := buf[:n]
		var dev DiscoveredDevice

		// First try raw JSON decode (v3.1)
		if err := json.Unmarshal(data, &dev); err == nil && dev.DeviceID != "" {
			l.handleDiscovered(dev)
			continue
		}

		// Otherwise attempt Tuya packet decode with default discovery MD5 key
		// Tuya UDP discovery key for v3.3: MD5("yGAdlopoPVldABfn")
		md5Key := ComputeMD5Key([]byte("yGAdlopoPVldABfn"))
		if pkt, err := DecodePacket(data, md5Key); err == nil && len(pkt.Payload) > 0 {
			if err := json.Unmarshal(pkt.Payload, &dev); err == nil && dev.DeviceID != "" {
				l.handleDiscovered(dev)
			}
		}
	}
}

func (l *DiscoveryListener) handleDiscovered(dev DiscoveredDevice) {
	l.mu.Lock()
	l.devices[dev.DeviceID] = dev
	l.mu.Unlock()

	if l.onDevice != nil {
		l.onDevice(dev)
	}
}

// GetDiscovered returns all currently discovered devices.
func (l *DiscoveryListener) GetDiscovered() []DiscoveredDevice {
	l.mu.RLock()
	defer l.mu.RUnlock()

	result := make([]DiscoveredDevice, 0, len(l.devices))
	for _, d := range l.devices {
		result = append(result, d)
	}
	return result
}
