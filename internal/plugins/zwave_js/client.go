// Package zwave_js implements a Z-Wave JS UI bridge via WebSocket.
package zwave_js

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// ZWaveValueID identifies a specific property on a Z-Wave node.
type ZWaveValueID struct {
	NodeID       int    `json:"nodeId"`
	CommandClass int    `json:"commandClass"`
	Endpoint     int    `json:"endpoint,omitempty"`
	Property     string `json:"property"`
	PropertyName string `json:"propertyName,omitempty"`
}

// ZWaveNode represents a discovered Z-Wave device.
type ZWaveNode struct {
	NodeID       int                    `json:"nodeId"`
	Name         string                 `json:"name"`
	Location     string                 `json:"location"`
	Ready        bool                   `json:"ready"`
	Status       string                 `json:"status"` // alive, asleep, dead
	DeviceClass  string                 `json:"deviceClass,omitempty"`
	Values       map[string]any         `json:"values"`
}

// WSMessage represents a JSON-RPC message exchanged with the Z-Wave JS UI server.
type WSMessage struct {
	MessageID string         `json:"messageId,omitempty"`
	Command   string         `json:"command,omitempty"`
	Event     string         `json:"event,omitempty"`
	Args      map[string]any `json:"args,omitempty"`
	Success   bool           `json:"success,omitempty"`
	Result    any            `json:"result,omitempty"`
}

// ValueUpdatedHandler is called whenever a Z-Wave node property changes.
type ValueUpdatedHandler func(nodeID int, commandClass int, property string, value any)

// Client manages the WebSocket connection to Z-Wave JS UI.
type Client struct {
	wsURL   string
	handler ValueUpdatedHandler

	mu        sync.Mutex
	conn      *websocket.Conn
	active    bool
	stopCh    chan struct{}
}

// NewClient constructs a new Z-Wave JS WebSocket client.
func NewClient(wsURL string, handler ValueUpdatedHandler) *Client {
	if wsURL == "" {
		wsURL = "ws://127.0.0.1:3000"
	}
	return &Client{
		wsURL:   wsURL,
		handler: handler,
		stopCh:  make(chan struct{}),
	}
}

// Connect dials the Z-Wave JS UI server and starts listening.
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	dialer := websocket.Dialer{HandshakeTimeout: 5 * time.Second}
	conn, _, err := dialer.DialContext(ctx, c.wsURL, nil)
	if err != nil {
		return fmt.Errorf("zwave_js: failed to dial %s: %w", c.wsURL, err)
	}

	c.conn = conn
	c.active = true

	// Send start_listening command to receive driver events and node values
	startMsg := WSMessage{
		MessageID: "start_listen_1",
		Command:   "start_listening",
	}
	if err := conn.WriteJSON(startMsg); err != nil {
		conn.Close()
		return fmt.Errorf("zwave_js: failed to send start_listening: %w", err)
	}

	go c.readLoop()
	return nil
}

// Close disconnects the client.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.active {
		return nil
	}
	c.active = false
	close(c.stopCh)
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// SetValue issues a setValue command to a Z-Wave node.
func (c *Client) SetValue(ctx context.Context, nodeID int, commandClass int, property string, val any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil || !c.active {
		return fmt.Errorf("zwave_js: client disconnected")
	}

	cmd := WSMessage{
		MessageID: fmt.Sprintf("cmd_%d", time.Now().UnixNano()),
		Command:   "node.set_value",
		Args: map[string]any{
			"nodeId": nodeID,
			"valueId": map[string]any{
				"commandClass": commandClass,
				"property":     property,
			},
			"value": val,
		},
	}

	return c.conn.WriteJSON(cmd)
}

func (c *Client) readLoop() {
	for {
		select {
		case <-c.stopCh:
			return
		default:
		}

		var msg map[string]any
		err := c.conn.ReadJSON(&msg)
		if err != nil {
			slog.Debug("zwave_js: read connection ended", "error", err)
			return
		}

		event, _ := msg["event"].(string)
		if event == "value updated" {
			args, _ := msg["args"].(map[string]any)
			nodeID := 0
			if n, ok := args["nodeId"].(float64); ok {
				nodeID = int(n)
			}
			valArgs, _ := args["args"].(map[string]any)
			cc := 0
			if cNum, ok := valArgs["commandClass"].(float64); ok {
				cc = int(cNum)
			}
			prop, _ := valArgs["property"].(string)
			val := valArgs["newValue"]

			if c.handler != nil && nodeID > 0 {
				c.handler(nodeID, cc, prop, val)
			}
		}
	}
}
