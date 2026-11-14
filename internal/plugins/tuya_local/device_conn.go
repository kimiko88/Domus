package tuya_local

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// DeviceClient handles a persistent TCP connection to a single Tuya device on port 6668.
type DeviceClient struct {
	DeviceID string
	IP       string
	LocalKey []byte
	Version  string
	Category DeviceCategory

	conn     net.Conn
	mu       sync.Mutex
	seq      atomic.Uint32
	closed   atomic.Bool
	stopCh   chan struct{}
	onStatus func(dps map[string]any)
}

// NewDeviceClient creates a new client for a Tuya device.
func NewDeviceClient(deviceID, ip string, localKey []byte, version string, cat DeviceCategory, onStatus func(map[string]any)) *DeviceClient {
	if version == "" {
		version = "3.3"
	}
	return &DeviceClient{
		DeviceID: deviceID,
		IP:       ip,
		LocalKey: localKey,
		Version:  version,
		Category: cat,
		stopCh:   make(chan struct{}),
		onStatus: onStatus,
	}
}

// Connect establishes the TCP connection and starts heartbeat + read loops.
func (c *DeviceClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.IP == "" {
		return errors.New("tuya: cannot connect without device IP")
	}

	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}

	addr := net.JoinHostPort(c.IP, "6668")
	dialer := net.Dialer{Timeout: 5 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("tuya: failed to connect to %s: %w", addr, err)
	}

	c.conn = conn
	c.closed.Store(false)

	select {
	case <-c.stopCh:
		c.stopCh = make(chan struct{})
	default:
	}

	go c.readLoop()
	go c.heartbeatLoop()

	return nil
}

// Close disconnects the client.
func (c *DeviceClient) Close() error {
	if c.closed.Swap(true) {
		return nil
	}
	close(c.stopCh)

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// SendCommand encodes and transmits a DPS update to the device.
// Automatically reconnects if the connection was dropped or idle.
func (c *DeviceClient) SendCommand(ctx context.Context, dps map[string]any) error {
	payload, err := BuildControlPayload(c.DeviceID, dps)
	if err != nil {
		return err
	}

	seq := c.seq.Add(1)
	frame, err := EncodePacket(seq, CmdControl, payload, c.LocalKey, c.Version)
	if err != nil {
		return err
	}

	c.mu.Lock()
	if c.conn == nil || c.closed.Load() {
		c.mu.Unlock()
		if err := c.Connect(ctx); err != nil {
			return fmt.Errorf("tuya: failed to reconnect before command: %w", err)
		}
		c.mu.Lock()
	}

	if c.conn == nil {
		c.mu.Unlock()
		return errors.New("tuya: connection is nil")
	}

	_ = c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	_, err = c.conn.Write(frame)
	if err != nil {
		_ = c.conn.Close()
		c.conn = nil
		c.mu.Unlock()

		if recErr := c.Connect(ctx); recErr == nil {
			c.mu.Lock()
			if c.conn != nil {
				_ = c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
				_, err = c.conn.Write(frame)
			}
			c.mu.Unlock()
		}
		return err
	}
	c.mu.Unlock()
	return nil
}

func (c *DeviceClient) heartbeatLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			if c.closed.Load() {
				return
			}
			seq := c.seq.Add(1)
			frame, err := EncodePacket(seq, CmdHeartbeat, nil, c.LocalKey, c.Version)
			if err != nil {
				continue
			}

			c.mu.Lock()
			if c.conn != nil {
				_ = c.conn.SetWriteDeadline(time.Now().Add(3 * time.Second))
				_, _ = c.conn.Write(frame)
			}
			c.mu.Unlock()
		}
	}
}

func (c *DeviceClient) readLoop() {
	buf := make([]byte, 4096)

	for {
		if c.closed.Load() {
			return
		}

		c.mu.Lock()
		conn := c.conn
		c.mu.Unlock()

		if conn == nil {
			return
		}

		_ = conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		n, err := conn.Read(buf)
		if err != nil {
			if errors.Is(err, io.EOF) || c.closed.Load() {
				return
			}
			continue
		}

		if n < 16 {
			continue
		}

		packet, err := DecodePacket(buf[:n], c.LocalKey)
		if err != nil {
			continue
		}

		if len(packet.Payload) > 0 {
			var resp struct {
				DPS map[string]any `json:"dps"`
			}
			if err := json.Unmarshal(packet.Payload, &resp); err == nil && len(resp.DPS) > 0 {
				if c.onStatus != nil {
					c.onStatus(resp.DPS)
				}
			}
		}
	}
}
