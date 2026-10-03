# ADR-001: Modular Monolith Architecture with Hexagonal Ports & Adapters

## Status
Accepted

## Date
2026-10-03

## Context
Domus is designed as a lightweight, reliable, local-first home automation hub targeting low-power ARM64 single-board computers (specifically Raspberry Pi 4). 
Microservices introduce substantial overhead in memory consumption, inter-process communication serialization, operational complexity, and multi-binary deployment failures on edge hardware. 
Conversely, an unstructured monolithic application quickly degrades into tight coupling between hardware protocols (MQTT, Zigbee), storage backends, and application logic.

## Decision
We adopt a **Modular Monolith** pattern organized around Hexagonal Architecture (Ports and Adapters):
1. **Domain Layer** (`internal/domain/`): Contains core business models, typed value objects, domain events, and repository interfaces (ports). It has zero dependencies on transport, external drivers, or database implementations.
2. **Application Layer** (`internal/application/`): Orchestrates business use cases, command handlers, and automation runners.
3. **Infrastructure Layer** (`internal/infrastructure/`): Implements ports for SQLite persistence, MQTT client wrapper, WebSocket hub, and in-memory event bus.
4. **Interfaces / Transport Layer** (`internal/interfaces/`): Exposes HTTP REST routes and WebSocket transports.
5. **Plugins Layer** (`internal/plugins/`): Houses protocol adapters (MQTT generic, Zigbee2MQTT, ESPHome) adhering to standard domain adapter contracts.

## Consequences
### Positive
- Single compiled binary with low memory footprint (~15-30 MB RAM idle).
- Fast in-memory communication across modules without network serialization overhead.
- Clear separation of concerns; adapters and persistence layers can be swapped or tested with mocks without touching domain logic.
- Simpler development, testing, and deployment lifecycle.

### Negative
- Enforcing modular boundaries requires discipline and code review to avoid cross-domain dependency leaks.
