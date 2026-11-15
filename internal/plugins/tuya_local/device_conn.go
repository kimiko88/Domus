package tuya_local

import (
	"bytes"
	"context"
	"encoding/binary"
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
type commandWaiter struct {
	sequence uint32
	command uint32
	expected map[string]any
	ack chan error
	confirmed chan map[string]any
}

type DeviceClient struct {
	DeviceID string
	IP       string
	LocalKey []byte
	Version  string
	Category DeviceCategory

	conn     net.Conn
	mu        sync.Mutex
	commandMu sync.Mutex
	pendingMu sync.Mutex
	pending   *commandWaiter
	seq       atomic.Uint32
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

// SendCommand waits for the Tuya ACK and the reported DPS state. A successful
// TCP write alone is not proof that the device executed the requested action.
func (c *DeviceClient) SendCommand(ctx context.Context, dps map[string]any) (map[string]any, error) {
	if len(dps) == 0 { return nil, errors.New("tuya: refusing to send an empty DPS command") }
	c.commandMu.Lock()
	defer c.commandMu.Unlock()
	payload, err := BuildControlPayload(c.DeviceID, dps)
	if err != nil { return nil, err }
	seq := c.seq.Add(1)
	frame, err := EncodePacket(seq, CmdControl, payload, c.LocalKey, c.Version)
	if err != nil { return nil, err }
	waiter := &commandWaiter{sequence: seq, command: CmdControl, expected: cloneDPS(dps), ack: make(chan error, 1), confirmed: make(chan map[string]any, 1)}
	c.pendingMu.Lock()
	c.pending = waiter
	c.pendingMu.Unlock()
	defer func() {
		c.pendingMu.Lock()
		if c.pending == waiter { c.pending = nil }
		c.pendingMu.Unlock()
	}()
	c.mu.Lock()
	if c.conn == nil || c.closed.Load() {
		c.mu.Unlock()
		if err := c.Connect(ctx); err != nil { return nil, fmt.Errorf("tuya: failed to reconnect before command: %w", err) }
		c.mu.Lock()
	}
	if c.conn == nil { c.mu.Unlock(); return nil, errors.New("tuya: connection is nil") }
	_ = c.conn.SetWriteDeadline(time.Now().Add(5*time.Second))
	_, err = c.conn.Write(frame)
	if err != nil {
		_ = c.conn.Close()
		c.conn = nil
		c.mu.Unlock()
		return nil, fmt.Errorf("tuya: command write failed: %w", err)
	}
	c.mu.Unlock()
	timer := time.NewTimer(5*time.Second)
	defer timer.Stop()
	ackReceived := false
	var confirmed map[string]any
	for !ackReceived || confirmed == nil {
		select {
		case ackErr := <-waiter.ack:
			if ackErr != nil { return nil, ackErr }
			ackReceived = true
		case status := <-waiter.confirmed:
			confirmed = status
		case <-ctx.Done():
			return nil, fmt.Errorf("tuya: command outcome unconfirmed: %w", ctx.Err())
		case <-timer.C:
			if !ackReceived { return nil, errors.New("tuya: device acknowledgement timeout; command delivery is unconfirmed") }
			return nil, errors.New("tuya: device acknowledged the command but did not report the requested DPS state")
		}
	}
	return confirmed, nil
}

func cloneDPS(dps map[string]any) map[string]any {
	copy := make(map[string]any, len(dps))
	for key, value := range dps { copy[key] = value }
	return copy
}

func dpsMatch(expected, actual map[string]any) bool {
	for key, want := range expected {
		got, ok := actual[key]
		if !ok || !sameDPSValue(want, got) { return false }
	}
	return true
}

func sameDPSValue(a, b any) bool {
	if af, ok := numericDPSValue(a); ok {
		if bf, ok := numericDPSValue(b); ok { return af == bf }
	}
	return fmt.Sprint(a) == fmt.Sprint(b)
}

func numericDPSValue(value any) (float64, bool) {
	switch number := value.(type) {
	case int: return float64(number), true
	case int64: return float64(number), true
	case float64: return number, true
	case float32: return float64(number), true
	default: return 0, false
	}
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
	readBuf := make([]byte, 4096)
	pending := make([]byte, 0, 8192)

	for {
		if c.closed.Load() { return }
		c.mu.Lock()
		conn := c.conn
		c.mu.Unlock()
		if conn == nil { return }

		_ = conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		n, err := conn.Read(readBuf)
		if err != nil {
			if errors.Is(err, io.EOF) || c.closed.Load() { return }
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() { continue }
			return
		}
		pending = append(pending, readBuf[:n]...)

		// TCP is a byte stream: one Read may contain a partial frame or several
		// coalesced frames. Extract complete 55AA packets before decoding.
		for {
			prefixAt := bytes.Index(pending, PrefixHeader)
			if prefixAt < 0 {
				if len(pending) > len(PrefixHeader)-1 {
					pending = append(pending[:0], pending[len(pending)-(len(PrefixHeader)-1):]...)
				}
				break
			}
			if prefixAt > 0 { pending = pending[prefixAt:] }
			if len(pending) < 16 { break }
			payloadLen := binary.BigEndian.Uint32(pending[12:16])
			if payloadLen < 8 || payloadLen > 1<<20 {
				pending = pending[1:]
				continue
			}
			frameLen := 16 + int(payloadLen)
			if len(pending) < frameLen { break }
			frame := append([]byte(nil), pending[:frameLen]...)
			pending = pending[frameLen:]
			packet, decodeErr := DecodeResponsePacket(frame, c.LocalKey)
			if decodeErr != nil { continue }
			c.handleResponsePacket(packet)
		}
	}
}

func (c *DeviceClient) handleResponsePacket(packet *Packet) {
	var dps map[string]any
	if len(packet.Payload) > 0 {
		var resp struct {
			DPS map[string]any `json:"dps"`
			Data struct { DPS map[string]any `json:"dps"` } `json:"data"`
		}
		if err := json.Unmarshal(packet.Payload, &resp); err == nil {
			dps = resp.DPS
			if len(dps) == 0 { dps = resp.Data.DPS }
		}
	}
	if len(dps) > 0 && c.onStatus != nil { c.onStatus(dps) }

	c.pendingMu.Lock()
	waiter := c.pending
	if waiter != nil {
		if packet.Sequence == waiter.sequence && packet.Command == waiter.command {
			if packet.ReturnCode != 0 {
				select { case waiter.ack <- fmt.Errorf("tuya: device rejected command with return code %d", packet.ReturnCode): default: }
			} else {
				select { case waiter.ack <- nil: default: }
			}
		}
		if len(dps) > 0 && dpsMatch(waiter.expected, dps) {
			select { case waiter.confirmed <- cloneDPS(dps): default: }
		}
	}
	c.pendingMu.Unlock()
}
