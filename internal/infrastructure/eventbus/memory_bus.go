// Package eventbus implements an in-memory, bounded, concurrent event bus
// fulfilling the domain events.Bus interface.
package eventbus

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/domus-hub/domus/internal/domain/events"
)

// ErrBusClosed is returned when publishing to an already closed bus.
var ErrBusClosed = errors.New("eventbus: bus is closed")

// ErrQueueFull is returned when the internal event buffer has reached capacity.
var ErrQueueFull = errors.New("eventbus: queue is full")

// Config defines the operational parameters for MemoryBus.
type Config struct {
	// BufferSize is the capacity of the internal event queue.
	BufferSize int
	// Workers is the number of background worker goroutines consuming the queue.
	Workers int
}

// MemoryBus is a thread-safe, bounded, in-memory implementation of events.Bus.
type MemoryBus struct {
	cfg Config

	mu        sync.RWMutex
	specific  map[events.EventType]map[uint64]events.Handler
	wildcards map[uint64]events.Handler
	nextSubID atomic.Uint64

	queue  chan events.Event
	done   chan struct{}
	closed atomic.Bool
	wg     sync.WaitGroup
}

// NewMemoryBus constructs and starts a new MemoryBus instance.
func NewMemoryBus(cfg Config) *MemoryBus {
	if cfg.BufferSize <= 0 {
		cfg.BufferSize = 1024
	}
	if cfg.Workers < 0 {
		cfg.Workers = 4
	}

	b := &MemoryBus{
		cfg:       cfg,
		specific:  make(map[events.EventType]map[uint64]events.Handler),
		wildcards: make(map[uint64]events.Handler),
		queue:     make(chan events.Event, cfg.BufferSize),
		done:      make(chan struct{}),
	}

	for i := 0; i < cfg.Workers; i++ {
		b.wg.Add(1)
		go b.worker()
	}

	return b
}

// Publish enqueues an event for asynchronous delivery to matching subscribers.
// Returns an error if the bus is closed, context is canceled, or queue is full.
func (b *MemoryBus) Publish(ctx context.Context, event events.Event) error {
	if b.closed.Load() {
		return ErrBusClosed
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case b.queue <- event:
		return nil
	default:
		return ErrQueueFull
	}
}

// Subscribe registers a handler for a specific event type.
// Returns an idempotent unsubscribe function.
func (b *MemoryBus) Subscribe(eventType events.EventType, handler events.Handler) func() {
	id := b.nextSubID.Add(1)

	b.mu.Lock()
	if b.specific[eventType] == nil {
		b.specific[eventType] = make(map[uint64]events.Handler)
	}
	b.specific[eventType][id] = handler
	b.mu.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
			b.mu.Lock()
			delete(b.specific[eventType], id)
			if len(b.specific[eventType]) == 0 {
				delete(b.specific, eventType)
			}
			b.mu.Unlock()
		})
	}
}

// SubscribeAll registers a handler for all event types.
// Returns an idempotent unsubscribe function.
func (b *MemoryBus) SubscribeAll(handler events.Handler) func() {
	id := b.nextSubID.Add(1)

	b.mu.Lock()
	b.wildcards[id] = handler
	b.mu.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
			b.mu.Lock()
			delete(b.wildcards, id)
			b.mu.Unlock()
		})
	}
}

// Close stops the bus, closes the queue, and waits for worker goroutines to complete.
func (b *MemoryBus) Close() error {
	if b.closed.CompareAndSwap(false, true) {
		close(b.done)
		close(b.queue)
		b.wg.Wait()
	}
	return nil
}

func (b *MemoryBus) worker() {
	defer b.wg.Done()

	for evt := range b.queue {
		b.dispatch(evt)
	}
}

func (b *MemoryBus) dispatch(evt events.Event) {
	b.mu.RLock()
	handlers := make([]events.Handler, 0, len(b.wildcards)+len(b.specific[evt.Type]))

	for _, h := range b.wildcards {
		handlers = append(handlers, h)
	}
	for _, h := range b.specific[evt.Type] {
		handlers = append(handlers, h)
	}
	b.mu.RUnlock()

	ctx := context.Background()
	for _, handler := range handlers {
		b.invokeHandler(ctx, handler, evt)
	}
}

func (b *MemoryBus) invokeHandler(ctx context.Context, handler events.Handler, evt events.Event) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("eventbus: panic recovered in event handler",
				"event_id", evt.ID,
				"event_type", string(evt.Type),
				"panic", fmt.Sprintf("%v", r),
			)
		}
	}()

	if err := handler(ctx, evt); err != nil {
		slog.Warn("eventbus: handler returned error",
			"event_id", evt.ID,
			"event_type", string(evt.Type),
			"error", err,
		)
	}
}
