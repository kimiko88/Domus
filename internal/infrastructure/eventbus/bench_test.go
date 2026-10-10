package eventbus_test

import (
	"context"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
)

func BenchmarkEventBus_Publish(b *testing.B) {
	bus := eventbus.NewMemoryBus(eventbus.Config{
		BufferSize: 4096,
		Workers:    4,
	})
	defer bus.Close()

	bus.Subscribe(events.EntityStateChanged, func(ctx context.Context, evt events.Event) error {
		return nil
	})

	ctx := context.Background()
	evt := events.Event{
		ID:         "bench-evt",
		Type:       events.EntityStateChanged,
		OccurredAt: time.Now(),
		Source:     "bench",
		Payload:    map[string]any{"state": "ON"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bus.Publish(ctx, evt)
	}
}

func BenchmarkEventBus_ConcurrentPublish(b *testing.B) {
	bus := eventbus.NewMemoryBus(eventbus.Config{
		BufferSize: 16384,
		Workers:    8,
	})
	defer bus.Close()

	bus.Subscribe(events.EntityStateChanged, func(ctx context.Context, evt events.Event) error {
		return nil
	})

	ctx := context.Background()
	evt := events.Event{
		ID:         "bench-evt",
		Type:       events.EntityStateChanged,
		OccurredAt: time.Now(),
		Source:     "bench",
		Payload:    map[string]any{"state": "ON"},
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = bus.Publish(ctx, evt)
		}
	})
}
