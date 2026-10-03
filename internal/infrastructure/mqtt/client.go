// Package mqtt provides the MQTT client interface, Paho wrapper, and testing mock.
package mqtt

import (
	"context"
	"crypto/tls"
	"errors"
	"sync"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
)

var (
	ErrNotConnected = errors.New("mqtt: client is not connected")
)

// MessageHandler is the signature for functions receiving incoming MQTT payloads.
type MessageHandler func(topic string, payload []byte)

// Client is the interface used by adapters to interact with an MQTT broker.
type Client interface {
	Connect(ctx context.Context) error
	Disconnect(quiesceMs uint)
	IsConnected() bool
	Publish(ctx context.Context, topic string, qos byte, retained bool, payload []byte) error
	Subscribe(ctx context.Context, topic string, qos byte, handler MessageHandler) error
	Unsubscribe(ctx context.Context, topics ...string) error
}

// Config defines connection parameters for an MQTT broker.
type Config struct {
	BrokerURL    string
	ClientID     string
	Username     string
	Password     string
	CleanSession bool
	KeepAlive    time.Duration
	TLSConfig    *tls.Config
}

// PahoClient implements Client using eclipse/paho.mqtt.golang.
type PahoClient struct {
	client paho.Client
}

// NewPahoClient creates a new PahoClient configured with the given options.
func NewPahoClient(cfg Config) *PahoClient {
	opts := paho.NewClientOptions()
	opts.AddBroker(cfg.BrokerURL)
	opts.SetClientID(cfg.ClientID)
	opts.SetCleanSession(cfg.CleanSession)
	if cfg.Username != "" {
		opts.SetUsername(cfg.Username)
		opts.SetPassword(cfg.Password)
	}
	if cfg.KeepAlive > 0 {
		opts.SetKeepAlive(cfg.KeepAlive)
	}
	if cfg.TLSConfig != nil {
		opts.SetTLSConfig(cfg.TLSConfig)
	}
	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(30 * time.Second)

	return &PahoClient{
		client: paho.NewClient(opts),
	}
}

// Connect initiates connection to the MQTT broker.
func (c *PahoClient) Connect(ctx context.Context) error {
	token := c.client.Connect()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-token.Done():
		return token.Error()
	}
}

// Disconnect gracefully terminates the broker connection.
func (c *PahoClient) Disconnect(quiesceMs uint) {
	c.client.Disconnect(quiesceMs)
}

// IsConnected returns whether the client is currently connected.
func (c *PahoClient) IsConnected() bool {
	return c.client.IsConnected()
}

// Publish sends a payload to the given topic.
func (c *PahoClient) Publish(ctx context.Context, topic string, qos byte, retained bool, payload []byte) error {
	if !c.IsConnected() {
		return ErrNotConnected
	}
	token := c.client.Publish(topic, qos, retained, payload)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-token.Done():
		return token.Error()
	}
}

// Subscribe listens for messages on topic.
func (c *PahoClient) Subscribe(ctx context.Context, topic string, qos byte, handler MessageHandler) error {
	pahoHandler := func(_ paho.Client, msg paho.Message) {
		handler(msg.Topic(), msg.Payload())
	}
	token := c.client.Subscribe(topic, qos, pahoHandler)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-token.Done():
		return token.Error()
	}
}

// Unsubscribe stops listening for messages on the specified topics.
func (c *PahoClient) Unsubscribe(ctx context.Context, topics ...string) error {
	token := c.client.Unsubscribe(topics...)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-token.Done():
		return token.Error()
	}
}

// MockClient is an in-memory Client implementation for unit and integration testing.
type MockClient struct {
	mu          sync.RWMutex
	connected   bool
	subscribers map[string][]MessageHandler
	published   []PublishedMessage
}

// PublishedMessage records an outbound publish call on the mock.
type PublishedMessage struct {
	Topic    string
	QoS      byte
	Retained bool
	Payload  []byte
}

// NewMockClient constructs a MockClient.
func NewMockClient() *MockClient {
	return &MockClient{
		connected:   true,
		subscribers: make(map[string][]MessageHandler),
	}
}

func (m *MockClient) Connect(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = true
	return nil
}

func (m *MockClient) Disconnect(quiesceMs uint) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = false
}

func (m *MockClient) IsConnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connected
}

func (m *MockClient) Publish(ctx context.Context, topic string, qos byte, retained bool, payload []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.connected {
		return ErrNotConnected
	}
	m.published = append(m.published, PublishedMessage{
		Topic:    topic,
		QoS:      qos,
		Retained: retained,
		Payload:  payload,
	})

	// Dispatch to subscribers
	for pattern, handlers := range m.subscribers {
		if matchTopic(pattern, topic) {
			for _, h := range handlers {
				go h(topic, payload)
			}
		}
	}
	return nil
}

func (m *MockClient) Subscribe(ctx context.Context, topic string, qos byte, handler MessageHandler) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.subscribers[topic] = append(m.subscribers[topic], handler)
	return nil
}

func (m *MockClient) Unsubscribe(ctx context.Context, topics ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, t := range topics {
		delete(m.subscribers, t)
	}
	return nil
}

// GetPublished returns a copy of messages published to the mock.
func (m *MockClient) GetPublished() []PublishedMessage {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]PublishedMessage, len(m.published))
	copy(out, m.published)
	return out
}

// SimulateIncoming dispatches an incoming MQTT message to matching handlers.
func (m *MockClient) SimulateIncoming(topic string, payload []byte) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for pattern, handlers := range m.subscribers {
		if matchTopic(pattern, topic) {
			for _, h := range handlers {
				h(topic, payload)
			}
		}
	}
}

func matchTopic(pattern, topic string) bool {
	if pattern == "#" || pattern == topic {
		return true
	}
	pParts := splitTopic(pattern)
	tParts := splitTopic(topic)

	for i := 0; i < len(pParts); i++ {
		if pParts[i] == "#" {
			return true
		}
		if i >= len(tParts) {
			return false
		}
		if pParts[i] != "+" && pParts[i] != tParts[i] {
			return false
		}
	}
	return len(pParts) == len(tParts)
}

func splitTopic(t string) []string {
	var parts []string
	curr := ""
	for i := 0; i < len(t); i++ {
		if t[i] == '/' {
			parts = append(parts, curr)
			curr = ""
		} else {
			curr += string(t[i])
		}
	}
	parts = append(parts, curr)
	return parts
}
