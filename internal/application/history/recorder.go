// Package history provides application services for state history recording and retention.
package history

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
	domhistory "github.com/domus-hub/domus/internal/domain/history"
)

// RecorderConfig tunes the batching and flush behavior of the state recorder.
type RecorderConfig struct {
	BatchSize     int
	FlushInterval time.Duration
}

// Recorder batches entity state changes and commits them asynchronously to SQLite.
type Recorder struct {
	repo        domhistory.Repository
	bus         events.Bus
	cfg         RecorderConfig
	queue       chan *domhistory.StateHistory
	stopCh      chan struct{}
	doneCh      chan struct{}
	unsubscribe func()
	mu          sync.Mutex
	running     bool
}

// NewRecorder constructs a new batched state history Recorder.
func NewRecorder(repo domhistory.Repository, bus events.Bus, cfg RecorderConfig) *Recorder {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 50
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = 2 * time.Second
	}

	return &Recorder{
		repo:   repo,
		bus:    bus,
		cfg:    cfg,
		queue:  make(chan *domhistory.StateHistory, 1000),
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}
}

// Start begins listening to domain events and flushes state entries in batches.
func (r *Recorder) Start(ctx context.Context) error {
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		return nil
	}
	r.running = true
	r.mu.Unlock()

	if r.bus != nil {
		r.unsubscribe = r.bus.Subscribe(events.EntityStateChanged, func(c context.Context, e events.Event) error {
			r.handleStateChangedEvent(e)
			return nil
		})
	}

	go r.runLoop()
	return nil
}

// RecordState enqueues an explicit state snapshot.
func (r *Recorder) RecordState(entityID, state string, at time.Time) {
	if at.IsZero() {
		at = time.Now().UTC()
	}
	entry := &domhistory.StateHistory{
		EntityID:   entityID,
		State:      state,
		RecordedAt: at,
	}

	select {
	case r.queue <- entry:
	default:
		slog.Warn("state history queue is full, dropping entry", "entity_id", entityID)
	}
}

func (r *Recorder) handleStateChangedEvent(e events.Event) {
	payload, ok := e.Payload.(map[string]any)
	if !ok {
		return
	}

	var entityID string
	if id, ok := payload["entity_id"].(string); ok && id != "" {
		entityID = id
	} else if key, ok := payload["entity_key"].(string); ok && key != "" {
		entityID = key
	}

	if entityID == "" {
		return
	}

	var stateStr string
	if s, ok := payload["state"].(string); ok {
		stateStr = s
	} else if payload["state"] != nil {
		if b, err := json.Marshal(payload["state"]); err == nil {
			stateStr = string(b)
		}
	}
	if stateStr == "" {
		stateStr = "null"
	}

	r.RecordState(entityID, stateStr, e.OccurredAt)
}

func (r *Recorder) runLoop() {
	defer close(r.doneCh)
	ticker := time.NewTicker(r.cfg.FlushInterval)
	defer ticker.Stop()

	var batch []*domhistory.StateHistory

	flush := func() {
		if len(batch) == 0 {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := r.repo.SaveBatch(ctx, batch); err != nil {
			slog.Error("failed to flush state history batch", "count", len(batch), "error", err)
		}
		batch = nil
	}

	for {
		select {
		case entry := <-r.queue:
			batch = append(batch, entry)
			if len(batch) >= r.cfg.BatchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		case <-r.stopCh:
			// Drain remaining items in queue before shutting down
			drain := true
			for drain {
				select {
				case entry := <-r.queue:
					batch = append(batch, entry)
				default:
					drain = false
				}
			}
			flush()
			return
		}
	}
}

// Stop unsubscribes from the event bus, drains the queue, and flushes any pending entries.
func (r *Recorder) Stop() {
	r.mu.Lock()
	if !r.running {
		r.mu.Unlock()
		return
	}
	r.running = false
	r.mu.Unlock()

	if r.unsubscribe != nil {
		r.unsubscribe()
		r.unsubscribe = nil
	}

	close(r.stopCh)
	<-r.doneCh
}
