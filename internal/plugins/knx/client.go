package knx

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// InboundHandler is called when a cEMI telegram is received from the KNX bus.
type InboundHandler func(frame *CEMIFrame)

// Client is the interface for interacting with a KNXnet/IP router or interface.
type Client interface {
	Connect(ctx context.Context) error
	Close() error
	IsConnected() bool
	SendTelegram(dstGroupAddr uint16, apci uint16, data []byte) error
	SetInboundHandler(handler InboundHandler)
}

// UDPClient implements KNXnet/IP tunneling over UDP.
type UDPClient struct {
	addr       string
	channelID  byte
	seqCounter uint32
	conn       *net.UDPConn
	connected  bool
	mu         sync.Mutex
	logger     *slog.Logger
	inboundFn  InboundHandler
	stopChan   chan struct{}
}

// NewUDPClient creates a new KNXnet/IP UDP client.
func NewUDPClient(addr string, logger *slog.Logger) *UDPClient {
	if logger == nil {
		logger = slog.Default()
	}
	return &UDPClient{
		addr:     addr,
		logger:   logger.With("component", "knx_client"),
		stopChan: make(chan struct{}),
	}
}

// SetInboundHandler registers the callback for incoming KNX telegrams.
func (c *UDPClient) SetInboundHandler(handler InboundHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.inboundFn = handler
}

// Connect resolves the address, opens the UDP socket, and initiates the tunnel.
func (c *UDPClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected && c.conn != nil {
		return nil
	}

	raddr, err := net.ResolveUDPAddr("udp", c.addr)
	if err != nil {
		return fmt.Errorf("knx: resolve udp addr %s failed: %w", c.addr, err)
	}

	conn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		return fmt.Errorf("knx: dial udp failed: %w", err)
	}

	c.conn = conn
	c.connected = true
	c.channelID = 1 // Assigned channel id (in standard mock or gateway handshake)

	c.stopChan = make(chan struct{})
	go c.listenLoop()

	c.logger.Info("connected to knx router", "addr", c.addr)
	return nil
}

// Close disconnects the UDP socket.
func (c *UDPClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return nil
	}

	c.connected = false
	close(c.stopChan)

	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}

// IsConnected returns connection status.
func (c *UDPClient) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected && c.conn != nil
}

// SendTelegram sends an outbound cEMI frame to the KNX bus via TUNNELLING_REQ.
func (c *UDPClient) SendTelegram(dstGroupAddr uint16, apci uint16, data []byte) error {
	c.mu.Lock()
	conn := c.conn
	connected := c.connected
	chID := c.channelID
	c.mu.Unlock()

	if !connected || conn == nil {
		return errors.New("knx: client not connected")
	}

	seq := byte(atomic.AddUint32(&c.seqCounter, 1) % 256)
	cemi := BuildCEMIFrame(0x0000, dstGroupAddr, apci, data)
	req := BuildTunnellingRequest(chID, seq, cemi)

	_, err := conn.Write(req)
	if err != nil {
		return fmt.Errorf("knx: write tunnelling req error: %w", err)
	}

	return nil
}

func (c *UDPClient) listenLoop() {
	buf := make([]byte, 1024)

	for {
		select {
		case <-c.stopChan:
			return
		default:
		}

		c.mu.Lock()
		conn := c.conn
		c.mu.Unlock()

		if conn == nil {
			return
		}

		_ = conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, err := conn.Read(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			return
		}

		if n < 6 {
			continue
		}

		hdr, err := DecodeKNXHeader(buf[:6])
		if err != nil {
			continue
		}

		switch hdr.ServiceType {
		case ServiceTunnellingReq:
			if n < 10 {
				continue
			}
			// Connection Header: [0x04, ChannelID, SequenceCounter, Reserved]
			channelID := buf[6]
			seqCounter := buf[7]

			// Send ACK immediately
			ack := BuildTunnellingAck(channelID, seqCounter, 0x00)
			_, _ = conn.Write(ack)

			// Parse cEMI frame starting at offset 10
			cemiBytes := buf[10:n]
			frame, err := ParseCEMIFrame(cemiBytes)
			if err != nil {
				c.logger.Debug("failed to parse inbound cemi frame", "err", err)
				continue
			}

			c.mu.Lock()
			fn := c.inboundFn
			c.mu.Unlock()

			if fn != nil {
				fn(frame)
			}
		}
	}
}
