package events

import "context"

// Handler is a function called for each matching domain event.
// Implementations must be fast; expensive work should be dispatched to a goroutine.
// Returning an error logs the failure but does not stop other handlers.
type Handler func(ctx context.Context, event Event) error

// Bus is the internal event bus contract.
// All methods are safe for concurrent use.
type Bus interface {
	// Publish enqueues an event for async delivery to all matching subscribers.
	// Returns an error only when the queue is full or the context is cancelled.
	// A full-queue error is non-fatal: the caller may log it and continue.
	Publish(ctx context.Context, event Event) error

	// Subscribe registers a handler for a specific event type.
	// The returned function removes the handler; it is safe to call multiple times.
	Subscribe(eventType EventType, handler Handler) (unsubscribe func())

	// SubscribeAll registers a handler that receives every event type.
	// The returned function removes the handler.
	SubscribeAll(handler Handler) (unsubscribe func())
}
