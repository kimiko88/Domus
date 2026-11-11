// Package eventbus_test tests the in-memory event bus implementation
// adhering to the events.Bus domain contract (TDD red-to-green phase).
package eventbus_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryBus_PublishAndSubscribeSpecific(t *testing.T) {
	bus := eventbus.NewMemoryBus(eventbus.Config{
		BufferSize: 100,
		Workers:    2,
	})
	defer bus.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var received events.Event
	var wg sync.WaitGroup
	wg.Add(1)

	unsub := bus.Subscribe(events.EntityStateChanged, func(c context.Context, e events.Event) error {
		received = e
		wg.Done()
		return nil
	})
	defer unsub()

	// Publish another event type — should NOT trigger subscriber
	err := bus.Publish(ctx, events.Event{
		ID:         "evt-ignored",
		Type:       events.DeviceDiscovered,
		OccurredAt: time.Now(),
		Source:     "test",
	})
	require.NoError(t, err)

	targetEvent := events.Event{
		ID:         "evt-123",
		Type:       events.EntityStateChanged,
		Payload:    map[string]any{"state": "on"},
		OccurredAt: time.Now().Truncate(time.Millisecond),
		Source:     "zigbee2mqtt",
	}

	err = bus.Publish(ctx, targetEvent)
	require.NoError(t, err)

	wg.Wait()

	assert.Equal(t, targetEvent.ID, received.ID)
	assert.Equal(t, targetEvent.Type, received.Type)
	assert.Equal(t, targetEvent.Source, received.Source)
	assert.Equal(t, targetEvent.Payload, received.Payload)
}

func TestMemoryBus_SubscribeAll(t *testing.T) {
	bus := eventbus.NewMemoryBus(eventbus.Config{
		BufferSize: 100,
		Workers:    2,
	})
	defer bus.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var count int32
	var wg sync.WaitGroup
	wg.Add(3)

	unsub := bus.SubscribeAll(func(c context.Context, e events.Event) error {
		atomic.AddInt32(&count, 1)
		wg.Done()
		return nil
	})
	defer unsub()

	evtTypes := []events.EventType{
		events.SystemStarted,
		events.DeviceOnline,
		events.CommandRequested,
	}

	for _, et := range evtTypes {
		err := bus.Publish(ctx, events.Event{
			ID:         "id-" + string(et),
			Type:       et,
			OccurredAt: time.Now(),
		})
		require.NoError(t, err)
	}

	wg.Wait()
	assert.Equal(t, int32(3), atomic.LoadInt32(&count))
}

func TestMemoryBus_UnsubscribeIsIdempotent(t *testing.T) {
	bus := eventbus.NewMemoryBus(eventbus.Config{
		BufferSize: 100,
		Workers:    2,
	})
	defer bus.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var count int32

	unsub := bus.Subscribe(events.SceneActivated, func(c context.Context, e events.Event) error {
		atomic.AddInt32(&count, 1)
		return nil
	})

	// Unsubscribe immediately
	unsub()
	unsub() // idempotent call

	err := bus.Publish(ctx, events.Event{
		ID:         "evt-scene-1",
		Type:       events.SceneActivated,
		OccurredAt: time.Now(),
	})
	require.NoError(t, err)

	// Allow event loop to process
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, int32(0), atomic.LoadInt32(&count), "Handler must not be invoked after unsubscribe")
}

func TestMemoryBus_ConcurrentPublish(t *testing.T) {
	bus := eventbus.NewMemoryBus(eventbus.Config{
		BufferSize: 1000,
		Workers:    4,
	})
	defer bus.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	const numEvents = 200
	var receivedCount int32
	var wg sync.WaitGroup
	wg.Add(numEvents)

	unsub := bus.Subscribe(events.EntityStateChanged, func(c context.Context, e events.Event) error {
		atomic.AddInt32(&receivedCount, 1)
		wg.Done()
		return nil
	})
	defer unsub()

	var pubWg sync.WaitGroup
	for i := 0; i < numEvents; i++ {
		pubWg.Add(1)
		go func(idx int) {
			defer pubWg.Done()
			_ = bus.Publish(ctx, events.Event{
				ID:         "evt-concurrent",
				Type:       events.EntityStateChanged,
				OccurredAt: time.Now(),
			})
		}(i)
	}

	pubWg.Wait()
	wg.Wait()

	assert.Equal(t, int32(numEvents), atomic.LoadInt32(&receivedCount))
}

func TestMemoryBus_BufferFullReturnsError(t *testing.T) {
	// Block the sole worker inside a handler so the bounded queue can be
	// filled deterministically. Workers=0 is normalized to the default.
	bus := eventbus.NewMemoryBus(eventbus.Config{
		BufferSize:     1,
		Workers:        1,
		HandlerTimeout: time.Minute,
	})
	releaseHandler := make(chan struct{})
	handlerStarted := make(chan struct{})
	bus.Subscribe(events.SystemStarted, func(context.Context, events.Event) error {
		close(handlerStarted)
		<-releaseHandler
		return nil
	})
	defer func() {
		close(releaseHandler)
		_ = bus.Close()
	}()

	ctx := context.Background()
	require.NoError(t, bus.Publish(ctx, events.Event{
		ID: "evt-1", Type: events.SystemStarted,
	}))
	select {
	case <-handlerStarted:
	case <-time.After(time.Second):
		t.Fatal("worker did not start the blocking handler")
	}

	// The worker is occupied by evt-1, so evt-2 fills the only queue slot.
	require.NoError(t, bus.Publish(ctx, events.Event{
		ID: "evt-2", Type: events.SystemStarted,
	}))

	// evt-3 must be rejected rather than silently accepted.
	err := bus.Publish(ctx, events.Event{
		ID: "evt-3", Type: events.SystemStarted,
	})
	require.ErrorIs(t, err, eventbus.ErrQueueFull,
		"publishing to a full buffer must return ErrQueueFull")
}

func TestMemoryBus_TracePropagation(t *testing.T) {
	bus := eventbus.NewMemoryBus(eventbus.Config{
		BufferSize: 10,
		Workers:    1,
	})
	defer bus.Close()

	rootTrace := events.NewTraceContext()
	ctx := events.ContextWithTrace(context.Background(), rootTrace)

	receivedTraceChan := make(chan *events.TraceContext, 1)

	bus.Subscribe(events.EntityStateChanged, func(c context.Context, e events.Event) error {
		tc := events.TraceFromContext(c)
		receivedTraceChan <- tc
		return nil
	})

	err := bus.Publish(ctx, events.Event{
		ID:         "evt-trace-1",
		Type:       events.EntityStateChanged,
		OccurredAt: time.Now(),
	})
	require.NoError(t, err)

	select {
	case tc := <-receivedTraceChan:
		require.NotNil(t, tc)
		assert.Equal(t, rootTrace.TraceID, tc.TraceID)
		assert.Equal(t, rootTrace.SpanID, tc.SpanID)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for trace propagation in eventbus")
	}
}

func TestMemoryBus_SlowHandlerTimeoutDoesNotBlockWorkers(t *testing.T) {
	// Configure with short timeout of 50ms and 1 single worker
	bus := eventbus.NewMemoryBus(eventbus.Config{
		BufferSize:     10,
		Workers:        1,
		HandlerTimeout: 50 * time.Millisecond,
	})
	defer bus.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Slow handler that hangs for 300ms
	bus.Subscribe(events.DeviceOnline, func(c context.Context, e events.Event) error {
		select {
		case <-c.Done():
			return c.Err()
		case <-time.After(300 * time.Millisecond):
			return nil
		}
	})

	secondHandlerRan := make(chan struct{})
	bus.Subscribe(events.DeviceOffline, func(c context.Context, e events.Event) error {
		close(secondHandlerRan)
		return nil
	})

	// 1. Publish slow event
	start := time.Now()
	err := bus.Publish(ctx, events.Event{
		ID:   "evt-slow",
		Type: events.DeviceOnline,
	})
	require.NoError(t, err)

	// 2. Publish fast event immediately after
	err = bus.Publish(ctx, events.Event{
		ID:   "evt-fast",
		Type: events.DeviceOffline,
	})
	require.NoError(t, err)

	// Second handler must execute without waiting the full 300ms of the slow handler
	select {
	case <-secondHandlerRan:
		elapsed := time.Since(start)
		assert.Less(t, elapsed, 200*time.Millisecond, "Worker should unblock around 50ms timeout, not 300ms")
	case <-time.After(1 * time.Second):
		t.Fatal("second event timed out waiting for worker")
	}
}

func TestMemoryBus_AsyncQueueIsolation(t *testing.T) {
	bus := eventbus.NewMemoryBus(eventbus.Config{
		BufferSize:      10,
		Workers:         1,
		AsyncBufferSize: 10,
		AsyncWorkers:    2,
		HandlerTimeout:  1 * time.Second,
	})
	defer bus.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	slowAsyncFinished := make(chan struct{})
	fastCoreFinished := make(chan struct{})

	// Async subscriber sleeps for 100ms
	bus.SubscribeAsync(events.EntityStateChanged, func(c context.Context, e events.Event) error {
		time.Sleep(100 * time.Millisecond)
		close(slowAsyncFinished)
		return nil
	})

	// Core synchronous subscriber executes instantly
	bus.Subscribe(events.EntityStateChanged, func(c context.Context, e events.Event) error {
		close(fastCoreFinished)
		return nil
	})

	start := time.Now()
	err := bus.Publish(ctx, events.Event{
		ID:   "evt-mixed",
		Type: events.EntityStateChanged,
	})
	require.NoError(t, err)

	// Fast core handler must finish well before the slow async handler
	select {
	case <-fastCoreFinished:
		assert.Less(t, time.Since(start), 50*time.Millisecond, "Core handler should finish immediately")
	case <-time.After(500 * time.Millisecond):
		t.Fatal("fast core handler took too long")
	}

	select {
	case <-slowAsyncFinished:
		// Succeeded asynchronously
	case <-time.After(500 * time.Millisecond):
		t.Fatal("async handler failed to finish")
	}
}
