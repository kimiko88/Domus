// Package e2e provides end-to-end integration tests for the full Domus application stack.
package e2e_test

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	serverpkg "github.com/domus-hub/domus/internal/interfaces/http"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDomusApp_EndToEnd(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "domus_e2e.db")

	// 1. Config
	cfg := config.Config{
		Server: config.ServerConfig{
			Host: "127.0.0.1",
			Port: 0, // OS assigns an ephemeral free port
		},
		Database: config.DatabaseConfig{
			Path:          dbPath,
			WALMode:       true,
			BusyTimeoutMs: 5000,
			MaxOpenConns:  1,
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "text",
		},
	}

	// 2. Open SQLite & Run Migrations
	db, err := sqlite.Open(cfg.Database)
	require.NoError(t, err)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = sqlite.Migrate(ctx, db)
	require.NoError(t, err)

	// 3. Start Event Bus
	bus := eventbus.NewMemoryBus(eventbus.Config{
		BufferSize: 256,
		Workers:    2,
	})
	defer bus.Close()

	var startedEventReceived bool
	var evtWg sync.WaitGroup
	evtWg.Add(1)

	unsub := bus.Subscribe(events.SystemStarted, func(c context.Context, e events.Event) error {
		startedEventReceived = true
		evtWg.Done()
		return nil
	})
	defer unsub()

	// 4. Start HTTP Server
	const version = "1.0.0-e2e"
	startedAt := time.Now()
	srv := serverpkg.NewServer(cfg.Server, version, startedAt, db)

	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- srv.Start()
	}()

	// Wait briefly for server startup
	time.Sleep(100 * time.Millisecond)

	// Publish SystemStarted event
	err = bus.Publish(ctx, events.Event{
		ID:         uuid.NewString(),
		Type:       events.SystemStarted,
		OccurredAt: time.Now(),
		Source:     "e2e-test",
	})
	require.NoError(t, err)

	evtWg.Wait()
	assert.True(t, startedEventReceived, "SystemStarted event must be delivered through eventbus")

	// 5. Test real HTTP /health request using Chi router
	req, err := http.NewRequest(http.MethodGet, "/health", nil)
	require.NoError(t, err)

	rr := srv.Router()
	recorder := &responseRecorder{headers: make(http.Header)}
	rr.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.code)

	var healthResp map[string]any
	err = json.Unmarshal(recorder.body, &healthResp)
	require.NoError(t, err)
	assert.Equal(t, "ok", healthResp["status"])
	assert.Equal(t, version, healthResp["version"])
	assert.Equal(t, "healthy", healthResp["database"])

	// 6. Graceful Server Shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutdownCancel()

	err = srv.Shutdown(shutdownCtx)
	assert.NoError(t, err)
}

type responseRecorder struct {
	code    int
	headers http.Header
	body    []byte
}

func (r *responseRecorder) Header() http.Header {
	return r.headers
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	r.body = append(r.body, b...)
	return len(b), nil
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.code = statusCode
}
