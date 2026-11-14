// Package websocket provides HTTP WebSocket handler for live real-time bidirectional communication.
package websocket

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/domus-hub/domus/internal/application/auth"
	"github.com/domus-hub/domus/internal/domain/entity"
	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/domain/user"
	"github.com/domus-hub/domus/internal/infrastructure/websocket"
	domusmw "github.com/domus-hub/domus/internal/interfaces/http/middleware"
	"github.com/google/uuid"
	gorilla "github.com/gorilla/websocket"
)

func websocketOriginAllowed(r *http.Request, allowedOrigins []string) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	// Native clients generally omit Origin. Browser clients always send it,
	// so validate it whenever present to prevent cross-site WebSocket hijacking.
	if origin == "" {
		return true
	}
	for _, allowed := range allowedOrigins {
		if origin == allowed {
			return true
		}
	}
	parsed, err := url.Parse(origin)
	if err != nil || parsed.User != nil || parsed.Host == "" ||
		(parsed.Scheme != "http" && parsed.Scheme != "https") ||
		parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return false
	}
	return strings.EqualFold(parsed.Host, r.Host)
}

// Handler handles WebSocket upgrade and client loops.
type Handler struct {
	hub         *websocket.Hub
	authService   *auth.Service
	allowedOrigins []string
	bus           events.Bus
	repo        entity.Repository
}

// NewHandler constructs a WebSocket Handler.
func NewHandler(hub *websocket.Hub, authService *auth.Service, allowedOrigins ...string) *Handler {
	return &Handler{
		hub:            hub,
		authService:    authService,
		allowedOrigins: append([]string(nil), allowedOrigins...),
	}
}

// WithBusAndRepo sets the event bus and entity repository for incoming WebSocket command dispatching.
func (h *Handler) WithBusAndRepo(bus events.Bus, repo entity.Repository) *Handler {
	h.bus = bus
	h.repo = repo
	return h
}

// ServeHTTP upgrades HTTP to WebSocket and runs reader/writer loops.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Authenticate from cookie or ?session= query parameter
	var sessionID string
	if cookie, err := r.Cookie(domusmw.SessionCookieName); err == nil && cookie.Value != "" {
		sessionID = cookie.Value
	} else if q := r.URL.Query().Get("session"); q != "" {
		sessionID = q
	} else if authHeader := strings.TrimSpace(r.Header.Get("Authorization")); strings.HasPrefix(authHeader, "Bearer ") {
		sessionID = strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	}

	var authenticatedUser *user.User
	if h.authService != nil {
		if sessionID == "" {
			http.Error(w, `{"code":"unauthorized","message":"authentication required"}`, http.StatusUnauthorized)
			return
		}
		var err error
		authenticatedUser, _, err = h.authService.ValidateSession(r.Context(), sessionID)
		if err != nil {
			http.Error(w, `{"code":"unauthorized","message":"invalid or expired session"}`, http.StatusUnauthorized)
			return
		}
	}

	upgrader := gorilla.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(req *http.Request) bool {
			return websocketOriginAllowed(req, h.allowedOrigins)
		},
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

	// All writes must go through one goroutine: gorilla/websocket permits
	// one concurrent reader and one concurrent writer, not multiple writers.
	writerStop := make(chan struct{})
	done := make(chan struct{})
	queueMessage := func(msg websocket.Message) {
		select {
		case sendCh <- msg:
		case <-writerStop:
		case <-done:
		}
	}
	go func() {
		defer close(done)
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-writerStop:
				return
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
				if h.authService != nil {
					if _, _, authErr := h.authService.ValidateSession(r.Context(), sessionID); authErr != nil {
						_ = conn.WriteMessage(gorilla.CloseMessage, gorilla.FormatCloseMessage(gorilla.ClosePolicyViolation, "session expired"))
						return
					}
				}
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
		if h.authService != nil {
			if _, _, authErr := h.authService.ValidateSession(r.Context(), sessionID); authErr != nil {
				break
			}
		}
		var clientMsg map[string]any
		if err := json.Unmarshal(message, &clientMsg); err == nil {
			msgType, _ := clientMsg["type"].(string)
			switch msgType {
			case "ping":
				queueMessage(websocket.Message{
					Type:      "pong",
					Timestamp: time.Now().UTC().Format(time.RFC3339),
				})
			case "call_service":
				domain, _ := clientMsg["domain"].(string)
				service, _ := clientMsg["service"].(string)
				entityID, _ := clientMsg["entity_id"].(string)
				data, _ := clientMsg["data"].(map[string]any)
				result := map[string]any{"success": false, "entity_id": entityID, "service": service}
				// Refresh identity on every command so role changes and session
				// revocations take effect without waiting for a reconnect.
				currentUser := authenticatedUser
				if h.authService != nil {
					var authErr error
					currentUser, _, authErr = h.authService.ValidateSession(r.Context(), sessionID)
					if authErr != nil {
						currentUser = nil
					}
				}

				switch {
				case h.authService == nil || currentUser == nil:
					result["code"] = "unauthorized"
					result["message"] = "authentication required"
				case currentUser.Role != user.RoleAdmin && currentUser.Role != user.RoleOperator:
					result["code"] = "forbidden"
					result["message"] = "operator or admin role required"
				case entityID == "" || service == "":
					result["code"] = "bad_request"
					result["message"] = "entity_id and service are required"
				case h.repo == nil:
					result["code"] = "unavailable"
					result["message"] = "entity repository unavailable"
				case h.bus == nil:
					result["code"] = "unavailable"
					result["message"] = "command dispatcher unavailable"
				default:
					ent, lookupErr := h.repo.GetByID(r.Context(), entityID)
					if lookupErr != nil || ent == nil {
						result["code"] = "not_found"
						result["message"] = "entity not found"
					} else if !ent.Writable {
						result["code"] = "not_writable"
						result["message"] = "entity is read-only"
					} else if publishErr := h.bus.Publish(r.Context(), events.Event{
						ID: uuid.NewString(), Type: events.CommandRequested,
						OccurredAt: time.Now().UTC(), Source: "user:" + currentUser.Username,
						Payload: map[string]any{
							"entity_id": ent.ID, "entity_key": ent.EntityKey,
							"service": service, "domain": domain, "params": data,
						},
					}); publishErr != nil {
						result["code"] = "dispatch_failed"
						result["message"] = "failed to dispatch command"
					} else {
						result["success"] = true
					}
				}
				queueMessage(websocket.Message{
					Type: "result", Payload: result,
					Timestamp: time.Now().UTC().Format(time.RFC3339),
				})
			}
		}
	}

	close(writerStop)
	_ = conn.Close()
	<-done
}
