// Package websocket provides real-time broadcast and WebSocket connection handling.
package websocket

import (
	"context"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
)

// Message is the standard frame sent from server to WebSocket clients.
type Message struct {
	Type      string `json:"type"`
	Payload   any    `json:"payload"`
	Timestamp string `json:"ts,omitempty"`
}

// ClientHandle represents an active registered WebSocket listener.
type ClientHandle struct {
	ID   string
	Send chan Message
}

// Hub manages active WebSocket clients and broadcasts events.
type Hub struct {
	mu        sync.RWMutex
	clients   map[*ClientHandle]bool
	broadcast chan Message
	bus       events.Bus
}

// NewHub constructs a new WebSocket Hub.
func NewHub(bus events.Bus) *Hub {
	h := &Hub{
		clients:   make(map[*ClientHandle]bool),
		broadcast: make(chan Message, 256),
		bus:       bus,
	}

	if bus != nil {
		h.subscribeToBus()
	}

	return h
}

// Run begins the hub event loop. It runs until ctx is cancelled.
func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		case msg := <-h.broadcast:
			if msg.Timestamp == "" {
				msg.Timestamp = time.Now().UTC().Format(time.RFC3339)
			}
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.Send <- msg:
				default:
					// Drop if client buffer is full to prevent lagging client from blocking the hub
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Register registers a new client channel.
func (h *Hub) Register(id string, send chan Message) *ClientHandle {
	handle := &ClientHandle{ID: id, Send: send}
	h.mu.Lock()
	h.clients[handle] = true
	h.mu.Unlock()
	return handle
}

// Unregister removes a client handle.
func (h *Hub) Unregister(handle *ClientHandle) {
	h.mu.Lock()
	delete(h.clients, handle)
	h.mu.Unlock()
}

// Broadcast enqueues a message for broadcast to all connected clients.
func (h *Hub) Broadcast(msg Message) {
	select {
	case h.broadcast <- msg:
	default:
		// Queue full, drop non-critical broadcast
	}
}

// ClientCount returns the current number of connected clients.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

func (h *Hub) subscribeToBus() {
	forwardHandler := func(ctx context.Context, e events.Event) error {
		h.Broadcast(Message{
			Type:      string(e.Type),
			Payload:   e.Payload,
			Timestamp: e.OccurredAt.UTC().Format(time.RFC3339),
		})
		return nil
	}

	h.bus.Subscribe(events.EntityStateChanged, forwardHandler)
	h.bus.Subscribe(events.DeviceOnline, forwardHandler)
	h.bus.Subscribe(events.DeviceOffline, forwardHandler)
	h.bus.Subscribe(events.DeviceDiscovered, forwardHandler)
	h.bus.Subscribe(events.IntegrationStateChanged, forwardHandler)
	h.bus.Subscribe(events.NotificationCreated, forwardHandler)
	h.bus.Subscribe(events.SceneActivated, forwardHandler)
}
