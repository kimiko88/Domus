package tuya_local

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"
)

// DiscoveredDevice represents a Tuya / Smart Life device broadcast on the local network.
type DiscoveredDevice struct {
	IP         string `json:"ip"`
	DeviceID   string `json:"gwId"`
	Active     int    `json:"active"`
	Version    string `json:"version"`
	ProductKey string `json:"productKey"`
}

// DiscoveryListener listens on UDP ports 6666 and 6667 for Tuya discovery packets.
type DiscoveryListener struct {
	ports       []int
	onDevice    func(DiscoveredDevice)
	stopCh      chan struct{}
	wg          sync.WaitGroup
	lifecycleMu sync.Mutex
	mu          sync.RWMutex
	devices     map[string]DiscoveredDevice
	started     bool
}

// NewDiscoveryListener creates a listener for Tuya UDP broadcasts.
func NewDiscoveryListener(onDevice func(DiscoveredDevice)) *DiscoveryListener {
	return &DiscoveryListener{
		ports:    []int{6666, 6667},
		onDevice: onDevice,
		devices:  make(map[string]DiscoveredDevice),
	}
}

// Start opens UDP listeners on the configured ports. It is safe to call repeatedly.
func (l *DiscoveryListener) Start(ctx context.Context) error {
	l.lifecycleMu.Lock()
	defer l.lifecycleMu.Unlock()

	l.mu.Lock()
	if l.started {
		l.mu.Unlock()
		return nil
	}
	l.stopCh = make(chan struct{})
	stopCh := l.stopCh
	l.started = true
	l.mu.Unlock()

	bound := 0
	for _, port := range l.ports {
		addr := net.UDPAddr{Port: port, IP: net.IPv4zero}
		conn, err := net.ListenUDP("udp4", &addr)
		if err != nil {
			slog.Warn("tuya_local: could not bind udp port (may be in use)", "port", port, "error", err)
			continue
		}
		bound++
		l.wg.Add(1)
		go l.listenLoop(conn, stopCh)
	}
	if bound == 0 {
		l.mu.Lock()
		l.started = false
		close(stopCh)
		l.mu.Unlock()
		return errors.New("tuya_local: no discovery UDP port could be bound")
	}

	go func() {
		select {
		case <-ctx.Done():
			l.Stop()
		case <-stopCh:
		}
	}()
	return nil
}

// Stop closes the UDP listeners. It is safe to call repeatedly and after a failed Start.
func (l *DiscoveryListener) Stop() {
	l.lifecycleMu.Lock()
	defer l.lifecycleMu.Unlock()

	l.mu.Lock()
	if !l.started {
		l.mu.Unlock()
		return
	}
	l.started = false
	close(l.stopCh)
	l.mu.Unlock()
	l.wg.Wait()
}

func validDeviceID(id string) bool {
	id = strings.TrimSpace(id)
	if id == "" || len(id) > 128 {
		return false
	}
	for _, r := range id {
		if !(r >= 'a' && r <= 'z') && !(r >= 'A' && r <= 'Z') &&
			!(r >= '0' && r <= '9') && r != '-' && r != '_' {
			return false
		}
	}
	return true
}

func (l *DiscoveryListener) listenLoop(conn *net.UDPConn, stopCh <-chan struct{}) {
	defer l.wg.Done()
	defer conn.Close()

	buf := make([]byte, 2048)
	for {
		select {
		case <-stopCh:
			return
		default:
		}

		_ = conn.SetReadDeadline(time.Now().Add(time.Second))
		n, source, err := conn.ReadFromUDP(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			return
		}
		if n <= 0 || source == nil || source.IP == nil {
			continue
		}
		if !(source.IP.IsPrivate() || source.IP.IsLoopback() || source.IP.IsLinkLocalUnicast()) {
			continue
		}

		data := buf[:n]
		var dev DiscoveredDevice
		if err := json.Unmarshal(data, &dev); err != nil {
			md5Key := ComputeMD5Key([]byte("yGAdlopoPVldABfn"))
			pkt, decodeErr := DecodePacket(data, md5Key)
			if decodeErr != nil || len(pkt.Payload) == 0 || json.Unmarshal(pkt.Payload, &dev) != nil {
				continue
			}
		}
		if !validDeviceID(dev.DeviceID) {
			continue
		}

		// Never trust the address embedded in an unauthenticated broadcast packet.
		dev.IP = source.IP.String()
		l.handleDiscovered(dev)
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

// GetDevice returns a discovered device by device ID if found on the local network.
func (l *DiscoveryListener) GetDevice(deviceID string) (DiscoveredDevice, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	d, ok := l.devices[deviceID]
	return d, ok
}
