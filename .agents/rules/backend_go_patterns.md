# Backend Go Guidelines & Patterns

These rules apply when developing Go services, domain models, or HTTP/WebSocket interfaces in `cmd/`, `internal/`, and `test/`.

## 1. Package Structure & Dependency Inversion
- `internal/domain`: Pure structs, interfaces, and sentinel errors. Never import `infrastructure`, `interfaces`, or external libraries.
- `internal/application`: Orchestrates use cases and domain entities.
- `internal/infrastructure`: Implements persistence, networking, and hardware bridges.
- `internal/interfaces`: Exposes HTTP routes via `chi.Router` and WebSocket handlers.

## 2. Concurrency & Reliability
- Always respect `context.Context` cancellation and deadlines in database queries and network calls.
- SQLite must always be accessed with write serialization (`MaxOpenConns(1)` or mutex guarded writes) and WAL mode enabled.
- Avoid goroutine leaks: every spawned goroutine must have an exit trigger (context cancellation or channel closure).

## 3. API Contract Best Practices
- Return consistent JSON envelopes with HTTP status codes.
- Do not expose internal database error details to client responses; log with `slog.Error` and return user-friendly messages.
- Support both entity-oriented endpoints (`/api/v1/entities/{id}/command`) and service-oriented endpoints (`/api/v1/services/{domain}/{service}`) to ensure multi-client compatibility.
