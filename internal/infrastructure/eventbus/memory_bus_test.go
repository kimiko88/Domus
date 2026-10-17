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
	// A bus with buffer size 1 and 0 workers won't consume
	bus := eventbus.NewMemoryBus(eventbus.Config{
		BufferSize: 1,
		Workers:    0,
	})
	defer bus.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// First event fills the buffer
	err := bus.Publish(ctx, events.Event{
		ID:   "evt-1",
		Type: events.SystemStarted,
	})
	require.NoError(t, err)

	// Second event should return context deadline or buffer full error
	err = bus.Publish(ctx, events.Event{
		ID:   "evt-2",
		Type: events.SystemStarted,
	})
	assert.Error(t, err, "Publishing to full buffer must return an error")
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
