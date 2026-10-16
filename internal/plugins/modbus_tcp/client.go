package modbus_tcp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// Client handles Modbus TCP socket connections and transaction sequencing.
type Client interface {
	RoundTrip(ctx context.Context, req []byte) ([]byte, error)
	Connect(ctx context.Context) error
	Close() error
	IsConnected() bool
	NextTxID() uint16
}

// TCPClient is a thread-safe TCP socket client for Modbus.
type TCPClient struct {
	addr        string
	timeout     time.Duration
	mu          sync.Mutex
	conn        net.Conn
	connected   bool
	txCounter   uint32
}

// NewTCPClient creates a new Modbus TCPClient.
func NewTCPClient(addr string, timeout time.Duration) *TCPClient {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &TCPClient{
		addr:    addr,
		timeout: timeout,
	}
}

// NextTxID returns an incrementing transaction ID.
func (c *TCPClient) NextTxID() uint16 {
	return uint16(atomic.AddUint32(&c.txCounter, 1))
}

// Connect opens a TCP connection to the Modbus server.
func (c *TCPClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected && c.conn != nil {
		return nil
	}

	d := net.Dialer{Timeout: c.timeout}
	conn, err := d.DialContext(ctx, "tcp", c.addr)
	if err != nil {
		c.connected = false
		return fmt.Errorf("modbus_tcp: connect failed to %s: %w", c.addr, err)
	}

	c.conn = conn
	c.connected = true
	return nil
}

// Close closes the TCP connection.
func (c *TCPClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.connected = false
	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}

// IsConnected returns whether the client is currently connected.
func (c *TCPClient) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected && c.conn != nil
}

// RoundTrip sends a Modbus TCP frame and waits for the corresponding response.
func (c *TCPClient) RoundTrip(ctx context.Context, req []byte) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected || c.conn == nil {
		return nil, errors.New("modbus_tcp: client not connected")
	}

	reqHeader, err := DecodeMBAP(req)
	if err != nil {
		return nil, fmt.Errorf("modbus_tcp: invalid request frame: %w", err)
	}

	deadline := time.Now().Add(c.timeout)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	_ = c.conn.SetDeadline(deadline)

	// Send request
	if _, err := c.conn.Write(req); err != nil {
		c.connected = false
		_ = c.conn.Close()
		c.conn = nil
		return nil, fmt.Errorf("modbus_tcp: write error: %w", err)
	}

	// Read MBAP Header (7 bytes)
	hdrBuf := make([]byte, 7)
	if _, err := io.ReadFull(c.conn, hdrBuf); err != nil {
		c.connected = false
		_ = c.conn.Close()
		c.conn = nil
		return nil, fmt.Errorf("modbus_tcp: read header error: %w", err)
	}

	respHeader, err := DecodeMBAP(hdrBuf)
	if err != nil {
		return nil, fmt.Errorf("modbus_tcp: malformed response header: %w", err)
	}

	if respHeader.TransactionID != reqHeader.TransactionID {
		return nil, fmt.Errorf("modbus_tcp: txID mismatch: got %d, expected %d", respHeader.TransactionID, reqHeader.TransactionID)
	}

	if respHeader.Length < 1 {
		return nil, errors.New("modbus_tcp: invalid response length")
	}

	// Remaining bytes after UnitID in Length field is the PDU: (Length - 1) bytes
	pduLen := int(respHeader.Length - 1)
	pduBuf := make([]byte, pduLen)
	if _, err := io.ReadFull(c.conn, pduBuf); err != nil {
		c.connected = false
		_ = c.conn.Close()
		c.conn = nil
		return nil, fmt.Errorf("modbus_tcp: read pdu error: %w", err)
	}

	return pduBuf, nil
}
