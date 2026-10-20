// Package eventbus implements an in-memory, bounded, concurrent event bus
// fulfilling the domain events.Bus interface with dedicated queues and timeout protection.
package eventbus

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
)

// ErrBusClosed is returned when publishing to an already closed bus.
var ErrBusClosed = errors.New("eventbus: bus is closed")

// ErrQueueFull is returned when the internal event buffer has reached capacity.
var ErrQueueFull = errors.New("eventbus: queue is full")

// Config defines the operational parameters for MemoryBus.
type Config struct {
	// BufferSize is the capacity of the primary event queue.
	BufferSize int
	// Workers is the number of background worker goroutines consuming the primary queue.
	Workers int
	// HandlerTimeout is the maximum duration allocated to each handler execution before timing out.
	// Defaults to 3 seconds if not set.
	HandlerTimeout time.Duration
	// AsyncBufferSize is the capacity of the separate async handler queue.
	// Defaults to 1024 if not set.
	AsyncBufferSize int
	// AsyncWorkers is the number of background workers consuming the async queue.
	// Defaults to 4 if not set.
	AsyncWorkers int
}

type subscriberEntry struct {
	id      uint64
	handler events.Handler
	isAsync bool
}

type asyncJob struct {
	ctx     context.Context
	handler events.Handler
	event   events.Event
}

// MemoryBus is a thread-safe, bounded, in-memory implementation of events.Bus.
// It features separate queues for core events and asynchronous I/O-bound handlers,
// as well as per-handler execution timeouts to protect the core hub from lagging subscribers.
type MemoryBus struct {
	cfg Config

	mu        sync.RWMutex
	specific  map[events.EventType]map[uint64]subscriberEntry
	wildcards map[uint64]subscriberEntry
	nextSubID atomic.Uint64

	queue      chan events.Event
	asyncQueue chan asyncJob
	done       chan struct{}
	closed     atomic.Bool
	wg         sync.WaitGroup
}

// NewMemoryBus constructs and starts a new MemoryBus instance.
func NewMemoryBus(cfg Config) *MemoryBus {
	if cfg.BufferSize <= 0 {
		cfg.BufferSize = 1024
	}
	if cfg.Workers <= 0 {
		cfg.Workers = 4
	}
	if cfg.HandlerTimeout <= 0 {
		cfg.HandlerTimeout = 3 * time.Second
	}
	if cfg.AsyncBufferSize <= 0 {
		cfg.AsyncBufferSize = 1024
	}
	if cfg.AsyncWorkers <= 0 {
		cfg.AsyncWorkers = 4
	}

	b := &MemoryBus{
		cfg:        cfg,
		specific:   make(map[events.EventType]map[uint64]subscriberEntry),
		wildcards:  make(map[uint64]subscriberEntry),
		queue:      make(chan events.Event, cfg.BufferSize),
		asyncQueue: make(chan asyncJob, cfg.AsyncBufferSize),
		done:       make(chan struct{}),
	}

	// Start core dispatcher workers
	for i := 0; i < cfg.Workers; i++ {
		b.wg.Add(1)
		go b.coreWorker()
	}

	// Start dedicated async handler workers
	for i := 0; i < cfg.AsyncWorkers; i++ {
		b.wg.Add(1)
		go b.asyncWorker()
	}

	return b
}

// Publish enqueues an event for asynchronous delivery to matching subscribers.
// Returns an error if the bus is closed, context is canceled, or queue is full.
func (b *MemoryBus) Publish(ctx context.Context, event events.Event) error {
	if b.closed.Load() {
		return ErrBusClosed
	}

	if event.TraceParent == "" {
		if tc := events.TraceFromContext(ctx); tc != nil {
			event.TraceParent = tc.String()
		}
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

// Subscribe registers a standard core handler for a specific event type.
// Returns an idempotent unsubscribe function.
func (b *MemoryBus) Subscribe(eventType events.EventType, handler events.Handler) func() {
	return b.registerSubscriber(eventType, handler, false)
}

// SubscribeAsync registers an asynchronous handler for a specific event type.
// Async handlers are dispatched to a separate queue with an independent worker pool,
// ensuring slow operations (e.g. external network I/O, webhooks, push notifications) never block the core loop.
func (b *MemoryBus) SubscribeAsync(eventType events.EventType, handler events.Handler) func() {
	return b.registerSubscriber(eventType, handler, true)
}

// SubscribeAll registers a handler for all event types.
// Returns an idempotent unsubscribe function.
func (b *MemoryBus) SubscribeAll(handler events.Handler) func() {
	return b.registerWildcard(handler, false)
}

// SubscribeAllAsync registers an asynchronous wildcard handler for all event types.
func (b *MemoryBus) SubscribeAllAsync(handler events.Handler) func() {
	return b.registerWildcard(handler, true)
}

func (b *MemoryBus) registerSubscriber(eventType events.EventType, handler events.Handler, isAsync bool) func() {
	id := b.nextSubID.Add(1)

	b.mu.Lock()
	if b.specific[eventType] == nil {
		b.specific[eventType] = make(map[uint64]subscriberEntry)
	}
	b.specific[eventType][id] = subscriberEntry{id: id, handler: handler, isAsync: isAsync}
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

func (b *MemoryBus) registerWildcard(handler events.Handler, isAsync bool) func() {
	id := b.nextSubID.Add(1)

	b.mu.Lock()
	b.wildcards[id] = subscriberEntry{id: id, handler: handler, isAsync: isAsync}
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

// Close stops the bus, closes all internal queues, and waits for worker goroutines to complete.
func (b *MemoryBus) Close() error {
	if b.closed.CompareAndSwap(false, true) {
		close(b.done)
		close(b.queue)
		close(b.asyncQueue)
		b.wg.Wait()
	}
	return nil
}

func (b *MemoryBus) coreWorker() {
	defer b.wg.Done()

	for evt := range b.queue {
		b.dispatch(evt)
	}
}

func (b *MemoryBus) asyncWorker() {
	defer b.wg.Done()

	for job := range b.asyncQueue {
		b.invokeHandlerWithTimeout(job.ctx, job.handler, job.event)
	}
}

func (b *MemoryBus) dispatch(evt events.Event) {
	b.mu.RLock()
	coreHandlers := make([]events.Handler, 0, len(b.wildcards)+len(b.specific[evt.Type]))
	asyncHandlers := make([]events.Handler, 0, len(b.wildcards)+len(b.specific[evt.Type]))

	for _, sub := range b.wildcards {
		if sub.isAsync {
			asyncHandlers = append(asyncHandlers, sub.handler)
		} else {
			coreHandlers = append(coreHandlers, sub.handler)
		}
	}
	for _, sub := range b.specific[evt.Type] {
		if sub.isAsync {
			asyncHandlers = append(asyncHandlers, sub.handler)
		} else {
			coreHandlers = append(coreHandlers, sub.handler)
		}
	}
	b.mu.RUnlock()

	ctx := context.Background()
	if evt.TraceParent != "" {
		if tc, err := events.ParseTraceParent(evt.TraceParent); err == nil {
			ctx = events.ContextWithTrace(ctx, tc)
		}
	}

	// 1. Offload async handlers immediately to separate asyncQueue without blocking core
	for _, handler := range asyncHandlers {
		job := asyncJob{
			ctx:     ctx,
			handler: handler,
			event:   evt,
		}
		select {
		case <-b.done:
			return
		case b.asyncQueue <- job:
		default:
			slog.Warn("eventbus: async queue full, dropping async event handler",
				"event_id", evt.ID,
				"event_type", string(evt.Type),
			)
		}
	}

	// 2. Execute core handlers with strict timeout protection
	for _, handler := range coreHandlers {
		b.invokeHandlerWithTimeout(ctx, handler, evt)
	}
}

func (b *MemoryBus) invokeHandlerWithTimeout(ctx context.Context, handler events.Handler, evt events.Event) {
	hCtx, cancel := context.WithTimeout(ctx, b.cfg.HandlerTimeout)
	defer cancel()

	done := make(chan struct{})
	var handlerErr error
	var panicked any

	go func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = r
			}
			close(done)
		}()
		handlerErr = handler(hCtx, evt)
	}()

	select {
	case <-done:
		if panicked != nil {
			slog.Error("eventbus: panic recovered in event handler",
				"event_id", evt.ID,
				"event_type", string(evt.Type),
				"panic", fmt.Sprintf("%v", panicked),
			)
		} else if handlerErr != nil {
			slog.Warn("eventbus: handler returned error",
				"event_id", evt.ID,
				"event_type", string(evt.Type),
				"error", handlerErr,
			)
		}
	case <-hCtx.Done():
		slog.Error("eventbus: handler execution timed out, unblocking dispatcher worker",
			"event_id", evt.ID,
			"event_type", string(evt.Type),
			"timeout", b.cfg.HandlerTimeout.String(),
		)
	}
}
