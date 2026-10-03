# ADR-002: SQLite with Write-Ahead Logging (WAL) and Pure Go Driver

## Status
Accepted

## Date
2026-10-03

## Context
Domus requires embedded storage capable of persisting devices, entities, automations, state history, and audit logs on edge devices without requiring an external database service (like PostgreSQL or MySQL) running on the Raspberry Pi.
Furthermore, SQLite databases using standard rollback journals suffer from lock contention between frequent sensor telemetry writes and simultaneous dashboard reads. Additionally, CGO dependencies complicate cross-compilation for ARM64.

## Decision
1. Use **SQLite** as the primary persistence engine for v1.
2. Use **`modernc.org/sqlite`**, a pure Go transpilation of SQLite that eliminates CGO requirements and cross-compilation toolchains.
3. Configure **Write-Ahead Logging (`PRAGMA journal_mode = WAL;`)** on startup:
   - Readers do not block writers.
   - Writers do not block readers.
4. Set **`PRAGMA busy_timeout = 5000;`** to gracefully handle lock contention under burst workloads.
5. Set **`PRAGMA foreign_keys = ON;`** to guarantee relational integrity.
6. Set **`PRAGMA synchronous = NORMAL;`** to balance durability with flash storage (SD card / SSD) write endurance on single-board computers.
7. Maintain repository interfaces in domain packages to allow future optional adapters (e.g., PostgreSQL for multi-node deployments).

## Consequences
### Positive
- Zero external database daemon needed; single file storage.
- High concurrent read performance suitable for real-time dashboards.
- Cross-platform builds (`GOOS=linux GOARCH=arm64`) run natively without cross-compilers.
- Low memory consumption.

### Negative
- Single writer limit requires managing write transactions efficiently and maintaining bounded connection pools.
