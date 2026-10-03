# ADR-003: In-Memory Bounded Event Bus

## Status
Accepted

## Date
2026-10-03

## Context
Domus is an event-driven system where sensor telemetry, device discoveries, command executions, and automation triggers must be broadcast to multiple decoupled subsystems (automation engine, WebSocket live feed, state history recorder, audit logger).
External messaging brokers like Redis Pub/Sub, RabbitMQ, or NATS add unnecessary memory and operational overhead on a single-node Raspberry Pi installation.

## Decision
1. Define a strict domain interface `events.Bus` in `internal/domain/events/bus.go` that knows nothing about transport layers.
2. Implement an in-memory event bus (`internal/infrastructure/eventbus/memory_bus.go`) using Go channels and a bounded worker pool.
3. Protect subscriber maps with read/write mutexes and provide thread-safe, idempotent unsubscription callbacks.
4. Provide panic isolation for individual event handlers to prevent a faulty subscriber from crashing the event bus or neighboring components.
5. Provide bounded queue capacity with non-blocking error returns or context timeouts to prevent unbounded memory growth under sudden event spikes.

## Consequences
### Positive
- Sub-microsecond event dispatch latency.
- Zero external runtime infrastructure dependencies.
- Handlers run isolated from publisher failure modes.

### Negative
- Events queued in memory are not preserved across hard process crashes (critical states are persisted in SQLite by repositories before or upon event emission).
