# Changelog

All notable changes to the Domus project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0] - 2026-10-03

### Added
- **User & RBAC Domain**: Domain models `User`, `Role` (admin, operator, viewer), `Session`, and validation rules in `internal/domain/user/`.
- **Argon2id Password Hashing**: Cryptographic password hashing and verification in `internal/infrastructure/crypto/` with configurable memory, iterations, and parallelism.
- **SQLite Repositories**:
  - `UserRepository`: User storage, retrieval by ID/username, and count checks.
  - `SessionRepository`: Session storage, retrieval, deletion, and batch purge of expired sessions.
  - `AuditRepository`: Structured audit log persistence tracking authentication and system events.
- **Authentication Application Service**:
  - `SetupInitialAdmin`: First-boot onboarding wizard preventing duplicate setups.
  - `Login`: Authentication with timing-attack mitigation and session creation.
  - `Logout`: Session revocation and audit tracking.
  - `ValidateSession`: Real-time session expiry and account status validation.
  - `RateLimiter`: In-memory sliding-window brute-force protection per client IP.
- **HTTP Transport & Middlewares**:
  - Endpoints: `POST /api/v1/auth/setup`, `GET /api/v1/auth/setup-status`, `POST /api/v1/auth/login`, `POST /api/v1/auth/logout`, `GET /api/v1/auth/me`.
  - Security middlewares: `Authenticate`, `RequireAuth`, `RequireRole`, `RateLimit`.
  - Cookie security: `HttpOnly`, `SameSite=Lax`, and CSRF token emission.
- **TDD Test Suites**:
  - Unit tests for Argon2id, RateLimiter, and middlewares.
  - Integration tests for SQLite user, session, and audit repositories.
  - E2E integration test covering setup wizard, credential validation, session cookie handling, logout, and audit logging.

## [0.1.0] - 2026-10-03

### Added
- **Repository Skeleton**: Modular monolith Go 1.23 codebase following hexagonal architecture (Ports & Adapters).
- **Configuration Engine**: Multi-tiered configuration loader supporting defaults, TOML configuration files, and `DOMUS_*` environment variable overrides.
- **SQLite Database with WAL**:
  - Embedded migrations using Go `embed.FS`.
  - Baseline database migration (`000001_init.up.sql` / `000001_init.down.sql`) provisioning all 12 domain tables.
  - Runtime enforcement of `PRAGMA foreign_keys = ON`, `PRAGMA busy_timeout = 5000`, and `PRAGMA journal_mode = WAL`.
- **In-Memory Event Bus**: Thread-safe, bounded, multi-worker event bus supporting typed event subscriptions, wildcard subscriptions, safe unsubscribe callbacks, and panic isolation.
- **HTTP Server & Routing**:
  - Chi v5 router with request ID, real IP, structured `slog` logger, and recovery middlewares.
  - Health check endpoint `/health` and `/api/v1/system/health` returning system version, uptime, and database connectivity.
- **Architecture Documentation & ADRs**:
  - ADR-001: Modular Monolith Architecture with Hexagonal Ports & Adapters.
  - ADR-002: SQLite with Write-Ahead Logging (WAL) and Pure Go Driver.
  - ADR-003: In-Memory Bounded Event Bus.
  - OpenAPI 3.1 REST API specification (`docs/openapi/domus.yaml`).
- **Deployments & Tooling**:
  - Multi-stage Dockerfile targeting ARM64 and AMD64 with zero-CGO static binaries.
  - Docker Compose development stack with Domus and Eclipse Mosquitto MQTT broker.
  - Systemd service unit (`domus.service`) for native Linux / Raspberry Pi OS installations.
  - Cross-platform `Makefile` with targets for build, test, lint, and run.
- **Automated Testing Suite (TDD)**:
  - Unit tests for configuration loader and event bus.
  - Integration tests for SQLite connection, PRAGMAs, and HTTP server.
  - End-to-end (E2E) integration test covering the entire running application stack.
