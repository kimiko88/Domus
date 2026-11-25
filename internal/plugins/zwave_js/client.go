// Package zwave_js implements a Z-Wave JS UI bridge via WebSocket.
package zwave_js

import (
	"context"
	"fmt"
	"net/http"
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
	NodeID      int            `json:"nodeId"`
	Name        string         `json:"name"`
	Location    string         `json:"location"`
	Ready       bool           `json:"ready"`
	Status      string         `json:"status"`
	DeviceClass string         `json:"deviceClass,omitempty"`
	Values      map[string]any `json:"values"`
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

// Client manages one reconnectable WebSocket session to Z-Wave JS UI.
type Client struct {
	wsURL   string
	handler ValueUpdatedHandler

	connectMu sync.Mutex
	mu        sync.Mutex
	conn      *websocket.Conn
	active    bool
	authToken string
	done      chan struct{}
}

// NewClient constructs a new Z-Wave JS WebSocket client.
func NewClient(wsURL string, handler ValueUpdatedHandler) *Client {
	if wsURL == "" {
		wsURL = "ws://127.0.0.1:3000"
	}
	return &Client{wsURL: wsURL, handler: handler}
}

// SetAuthToken configures the optional Z-Wave JS UI Bearer token before connecting.
func (c *Client) SetAuthToken(token string) {
	c.mu.Lock()
	c.authToken = token
	c.mu.Unlock()
}

// Connected reports whether the current WebSocket session is active.
func (c *Client) Connected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.active && c.conn != nil
}

// Connect establishes one WebSocket session and starts its reader.
// Repeated calls while connected are idempotent; after a disconnect the client
// can connect again without reusing a closed stop channel.
func (c *Client) Connect(ctx context.Context) error {
	c.connectMu.Lock()
	defer c.connectMu.Unlock()

	c.mu.Lock()
	if c.active && c.conn != nil {
		c.mu.Unlock()
		return nil
	}
	previousDone := c.done
	token := c.authToken
	c.mu.Unlock()

	if previousDone != nil {
		select {
		case <-previousDone:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	headers := make(http.Header)
	if token != "" {
		headers.Set("Authorization", "Bearer "+token)
	}
	dialer := websocket.Dialer{HandshakeTimeout: 5 * time.Second}
	conn, response, err := dialer.DialContext(ctx, c.wsURL, headers)
	if err != nil {
		if response != nil && response.Body != nil {
			_ = response.Body.Close()
		}
		return fmt.Errorf("zwave_js: failed to dial %s: %w", c.wsURL, err)
	}

	startMsg := WSMessage{MessageID: "start_listen_1", Command: "start_listening"}
	if err := conn.WriteJSON(startMsg); err != nil {
		_ = conn.Close()
		return fmt.Errorf("zwave_js: failed to send start_listening: %w", err)
	}

	done := make(chan struct{})
	c.mu.Lock()
	c.conn = conn
	c.active = true
	c.done = done
	c.mu.Unlock()

	go c.readLoop(conn, done)
	return nil
}

// Close disconnects the current session and waits for its reader to exit.
func (c *Client) Close() error {
	c.connectMu.Lock()
	defer c.connectMu.Unlock()

	c.mu.Lock()
	conn, done := c.conn, c.done
	c.conn = nil
	c.active = false
	c.done = nil
	c.mu.Unlock()

	if conn != nil {
		if err := conn.Close(); err != nil {
			return err
		}
	}
	if done != nil {
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			return fmt.Errorf("zwave_js: timed out waiting for WebSocket reader to stop")
		}
	}
	return nil
}

// SetValue issues a setValue command to a Z-Wave node.
func (c *Client) SetValue(ctx context.Context, nodeID int, commandClass int, property string, val any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil || !c.active {
		return fmt.Errorf("zwave_js: client disconnected")
	}
	cmd := WSMessage{
		MessageID: fmt.Sprintf("cmd_%d", time.Now().UnixNano()),
		Command: "node.set_value",
		Args: map[string]any{
			"nodeId": nodeID,
			"valueId": map[string]any{"commandClass": commandClass, "property": property},
			"value": val,
		},
	}
	if err := c.conn.WriteJSON(cmd); err != nil {
		return fmt.Errorf("zwave_js: write node.set_value: %w", err)
	}
	return nil
}

func (c *Client) readLoop(conn *websocket.Conn, done chan struct{}) {
	defer close(done)
	defer func() {
		_ = conn.Close()
		c.mu.Lock()
		if c.conn == conn {
			c.conn = nil
			c.active = false
		}
		c.mu.Unlock()
	}()

	for {
		var msg map[string]any
		if err := conn.ReadJSON(&msg); err != nil {
			slog.Debug("zwave_js: read connection ended", "error", err)
			return
		}
		event, _ := msg["event"].(string)
		if event != "value updated" {
			continue
		}
		args, _ := msg["args"].(map[string]any)
		nodeID := 0
		if n, ok := args["nodeId"].(float64); ok {
			nodeID = int(n)
		}
		valueArgs, _ := args["args"].(map[string]any)
		cc := 0
		if n, ok := valueArgs["commandClass"].(float64); ok {
			cc = int(n)
		}
		property, _ := valueArgs["property"].(string)
		value := valueArgs["newValue"]
		if c.handler != nil && nodeID > 0 && cc > 0 {
			c.handler(nodeID, cc, property, value)
		}
	}
}
