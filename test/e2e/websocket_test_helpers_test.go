package e2e_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	wsinfra "github.com/domus-hub/domus/internal/infrastructure/websocket"
	gorilla "github.com/gorilla/websocket"
)

// wsEventReader preserves frames that arrive before the event currently awaited.
// The event bus intentionally has multiple workers, so two related domain events
// can be broadcast in a different order from the order in which they were queued.
type wsEventReader struct {
	conn    *gorilla.Conn
	mu      sync.Mutex
	pending map[string][]wsinfra.Message
}

func newWSEventReader(conn *gorilla.Conn) *wsEventReader {
	return &wsEventReader{
		conn:    conn,
		pending: make(map[string][]wsinfra.Message),
	}
}

// Read returns the next frame of the requested type without discarding frames
// of other types. Calls on one reader are serialized because Gorilla WebSocket
// permits only one concurrent reader per connection.
func (r *wsEventReader) Read(t *testing.T, eventType string, timeout time.Duration) wsinfra.Message {
	t.Helper()

	r.mu.Lock()
	defer r.mu.Unlock()

	if msg, ok := r.takePending(eventType); ok {
		return msg
	}
	if err := r.conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		t.Fatalf("setting websocket read deadline while waiting for %s: %v", eventType, err)
		return wsinfra.Message{}
	}

	for {
		var msg wsinfra.Message
		if err := r.conn.ReadJSON(&msg); err != nil {
			t.Fatalf("failed reading ws event (waiting for %s): %v", eventType, err)
			return wsinfra.Message{}
		}
		if msg.Type == eventType {
			return msg
		}
		r.buffer(msg)
	}
}

func (r *wsEventReader) buffer(msg wsinfra.Message) {
	r.pending[msg.Type] = append(r.pending[msg.Type], msg)
}

func (r *wsEventReader) takePending(eventType string) (wsinfra.Message, bool) {
	messages := r.pending[eventType]
	if len(messages) == 0 {
		return wsinfra.Message{}, false
	}

	msg := messages[0]
	if len(messages) == 1 {
		delete(r.pending, eventType)
	} else {
		r.pending[eventType] = messages[1:]
	}
	return msg, true
}

func TestWSEventReaderPreservesOutOfOrderFrames(t *testing.T) {
	upgrader := gorilla.Upgrader{
		CheckOrigin: func(*http.Request) bool { return true },
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		conn, err := upgrader.Upgrade(w, req, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Completion is deliberately sent before the trigger. A caller waiting
		// for the trigger must retain the completion for its next Read call.
		for _, eventType := range []string{
			"automation.run_completed",
			"automation.triggered",
			"scene.activated",
		} {
			if err := conn.WriteJSON(wsinfra.Message{Type: eventType}); err != nil {
				return
			}
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := gorilla.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial test websocket: %v", err)
	}
	defer conn.Close()

	reader := newWSEventReader(conn)
	for _, eventType := range []string{
		"automation.triggered",
		"automation.run_completed",
		"scene.activated",
	} {
		msg := reader.Read(t, eventType, time.Second)
		if msg.Type != eventType {
			t.Fatalf("Read() returned %q, want %q", msg.Type, eventType)
		}
	}
}
