// Package websocket provides HTTP WebSocket handler for live real-time bidirectional communication.
package websocket

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/domus-hub/domus/internal/application/auth"
	"github.com/domus-hub/domus/internal/infrastructure/websocket"
	domusmw "github.com/domus-hub/domus/internal/interfaces/http/middleware"
	gorilla "github.com/gorilla/websocket"
)

var upgrader = gorilla.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Local-first: allow local network origins
		return true
	},
}

// Handler handles WebSocket upgrade and client loops.
type Handler struct {
	hub         *websocket.Hub
	authService *auth.Service
}

// NewHandler constructs a WebSocket Handler.
func NewHandler(hub *websocket.Hub, authService *auth.Service) *Handler {
	return &Handler{
		hub:         hub,
		authService: authService,
	}
}

// ServeHTTP upgrades HTTP to WebSocket and runs reader/writer loops.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Authenticate from cookie or ?session= query parameter
	var sessionID string
	if cookie, err := r.Cookie(domusmw.SessionCookieName); err == nil && cookie.Value != "" {
		sessionID = cookie.Value
	} else if q := r.URL.Query().Get("session"); q != "" {
		sessionID = q
	}

	if h.authService != nil {
		if sessionID == "" {
			http.Error(w, `{"code":"unauthorized","message":"authentication required"}`, http.StatusUnauthorized)
			return
		}
		_, _, err := h.authService.ValidateSession(r.Context(), sessionID)
		if err != nil {
			http.Error(w, `{"code":"unauthorized","message":"invalid or expired session"}`, http.StatusUnauthorized)
			return
		}
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Warn("websocket upgrade failed", "error", err)
		return
	}
	defer conn.Close()

	sendCh := make(chan websocket.Message, 64)
	client := h.hub.Register(sessionID, sendCh)
	defer h.hub.Unregister(client)

	// Send initial connected acknowledgement frame
	_ = conn.WriteJSON(websocket.Message{
		Type:      "system.connected",
		Payload:   map[string]any{"authenticated": true},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})

	// Writer loop (goroutine)
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case msg, ok := <-sendCh:
				if !ok {
					_ = conn.WriteMessage(gorilla.CloseMessage, []byte{})
					return
				}
				_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
				if err := conn.WriteJSON(msg); err != nil {
					return
				}

			case <-ticker.C:
				_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
				if err := conn.WriteMessage(gorilla.PingMessage, nil); err != nil {
					return
				}
			}
		}
	}()

	// Reader loop (blocks until disconnect)
	conn.SetReadLimit(4096)
	_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}
		var clientMsg map[string]any
		if err := json.Unmarshal(message, &clientMsg); err == nil {
			if clientMsg["type"] == "ping" {
				_ = conn.WriteJSON(websocket.Message{
					Type:      "pong",
					Timestamp: time.Now().UTC().Format(time.RFC3339),
				})
			}
		}
	}

	<-done
}
