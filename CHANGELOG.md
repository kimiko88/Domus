# Changelog

All notable changes to the Domus project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.6.0] - 2026-10-04

### Added
- **Vue 3 + Vite + TypeScript PWA Frontend (`web/`)**:
  - Modular, accessible, and responsive single-page progressive web application with offline support and service worker registration (`vite-plugin-pwa`).
  - Dark mode glassmorphic aesthetic tailored for smart home dashboards (`tokens.css`, `main.css`).
- **State Management & Communication Stores (Pinia)**:
  - `auth`: user authentication, first-boot wizard status check (`checkSetup`), login, admin initialization, and session validation.
  - `entities`: optimistic updates, service command dispatching, domain groupings (`lights`, `switches`, `sensors`, `binarySensors`), and live state patching.
  - `devices`: physical hardware tracking, online/offline counts, area grouping.
  - `areas`: room and spatial hierarchy management.
  - `ws`: real-time WebSocket connection state, automatic reconnection backoff, event router for state updates, and alert notifications.
  - `notifications`: reactive in-app toast stack with auto-dismiss timers.
- **Custom Composables**:
  - `useWebSocket`: robust WebSocket client with exponential backoff, heartbeat keepalive ping/pong, and typed subscription bus.
  - `useEntityCommand`: high-level entity interaction composable for optimistic toggles, brightness control, and thermostat adjustments.
- **Entity Card Components (`components/cards/`)**:
  - `LightCard`: ambient glow, on/off toggle, slider for brightness control with 0–100% feedback.
  - `SwitchCard`: power indicator (W), optimistic state switching, tactile feedback.
  - `SensorCard`: auto-adapts icon and formatting based on `device_class` (temperature, humidity, motion, battery level indicator).
  - `ClimateCard`: dual target/current temperature display, +/- 0.5 °C setpoint adjustment, HVAC mode indicators.
- **UI Base & Layout Components**:
  - `BaseButton`: primary, secondary, danger, and ghost variants with spinner loading states.
  - `BaseCard`: glassmorphic container with customizable header/footer and active state glows.
  - `BaseToast`: fixed stack notifications with slide-in animations.
  - `AppSidebar`: responsive navigation drawer with active route highlight and mobile backdrop.
  - `AppTopbar`: live WebSocket connection badge, page header, user profile, and quick logout.
- **Views & Routing**:
  - `DashboardView`: multi-stat overview tiles, category filter pills, dynamic card grid.
  - `AreasView`: card grid by room with entity counts and quick previews.
  - `DevicesView`: hardware list with online/offline filters and integration badges.
  - `IntegrationsView`: hardware and cloud adapter statuses with manual reconnect action.
  - `SettingsView`: user information, system metrics, and theme overview.
  - `LoginView` & `SetupView`: first boot wizard and credentials form.
  - Protected navigation guard with setup redirect and session authentication enforcement.
- **TDD Test Suites**:
  - 84 unit and component test cases across 27 test files executed with Vitest and Happy DOM.
  - Full TypeScript type checking and production bundle verification (`vue-tsc -b && vite build`).

## [0.5.0] - 2026-10-03

### Added
- **Automation Domain & DSL (`internal/domain/automation`)**:
  - `Automation` domain model with validation invariants (`name` required, `triggers` and `actions` required).
  - Flexible JSON DSL for `Trigger` (`state`, `numeric_state`, `event`, `time`), `Condition` (`state`, `numeric_state`, `time`, `and`, `or`, `not`), and `Action` (`service`, `delay`, `event`).
  - `AutomationRun` model recording execution status (`running`, `completed`, `failed`, `skipped`), trigger context, duration in milliseconds, and run logs.
- **Scene Domain (`internal/domain/scene`)**:
  - `Scene` domain model with multi-entity preset actions.
- **SQLite Repositories**:
  - `AutomationRepository`: CRUD for automations and full execution run history tracking in `automations` and `automation_runs` tables.
  - `SceneRepository`: CRUD for scene presets in `scenes` table.
  - Exposed via aggregate `Repositories.Automations` and `Repositories.Scenes`.
- **Automation Engine Runtime (`internal/application/automation`)**:
  - `Evaluator`: evaluates trigger matching and condition predicates (supporting numeric comparisons, state equality, time windows, and logical operators).
  - `Engine`: event-driven worker pool listening to `events.Bus`. Matches incoming events, enforces per-rule cooldown periods, checks conditions, runs actions asynchronously with panic recovery, logs execution runs, and publishes `automation.triggered`, `automation.run_completed`, and `automation.run_failed`.
  - Manual trigger capability via `TriggerAutomation(ctx, id)`.
- **Scene Application Service (`internal/application/scene`)**:
  - `Service.ApplyScene(ctx, id)`: translates scene definitions into sequential entity control commands and broadcasts `scene.activated`.
- **REST Endpoints for Automations & Scenes**:
  - `/api/v1/automations` (GET, POST): list and create automations.
  - `/api/v1/automations/{id}` (GET, PUT, DELETE): inspect, update, and remove rules.
  - `/api/v1/automations/{id}/trigger` (POST): manual trigger endpoint.
  - `/api/v1/automations/{id}/runs` (GET): execution history logs.
  - `/api/v1/scenes` (GET, POST): list and create scenes.
  - `/api/v1/scenes/{id}` (GET, PUT, DELETE): inspect, update, and remove scenes.
  - `/api/v1/scenes/{id}/apply` (POST): activate scene preset.
- **WebSocket Real-Time Broadcast Integration**:
  - Added subscriptions in `websocket.Hub` for `automation.triggered`, `automation.run_completed`, and `automation.run_failed`.
- **TDD Test Suites**:
  - Unit tests for automation and scene domain models and validation.
  - Integration tests for SQLite automation and scene repositories.
  - Unit tests for trigger & condition evaluator and engine execution runner.
  - Unit tests for scene application service.
  - HTTP handler tests covering all REST endpoints.
  - Full end-to-end integration test (`test/e2e/automation_scene_e2e_test.go`) validating the complete loop: MQTT telemetry $\to$ entity state update $\to$ event bus $\to$ automation evaluation $\to$ condition verification $\to$ hardware command dispatch $\to$ SQLite run recording $\to$ real-time WebSocket client notifications, plus scene creation and activation.

## [0.4.0] - 2026-10-03

### Added
- **Zigbee2MQTT Protocol Adapter (`internal/plugins/zigbee2mqtt`)**:
  - `DeviceMapper`: Parses Zigbee2MQTT `bridge/devices` payloads into Domus `Device` and typed `Entity` hierarchies (lights, switches, contact/occupancy binary sensors, temperature/humidity/battery/power sensors).
  - Outbound command translation: maps Domus `Command` (`turn_on`, `turn_off`, `toggle`, `set_brightness`, `set_color_temp`) to `zigbee2mqtt/<friendly_name>/set` JSON payloads.
  - `zigbee2mqtt.Adapter` implementing `plugins.Adapter`: auto-discovery, telemetry state ingestion, SQLite persistence, and event bus broadcast.
  - Bridge state tracking: monitors `bridge/state` ("online"/"offline") and emits `events.IntegrationStateChanged`.
- **ESPHome MQTT Integration Adapter (`internal/plugins/esphome_mqtt`)**:
  - `EntityMapper`: Parses `<node_name>/<domain>/<object_id>/state` topics, extracts node/domain/object, infers units of measurement (°C, %, W, V, A, dBm).
  - Outbound commands: translates control commands to `<node_name>/<domain>/<object_id>/command`.
  - Node LWT tracking: monitors `<node_name>/status` and emits `events.DeviceOnline` / `events.DeviceOffline`.
  - `esphome_mqtt.Adapter` implementing `plugins.Adapter`: on-the-fly auto-discovery and state synchronization.
- **Cloud Resilience & Graceful Degradation (`internal/plugins/cloud`)**:
  - `CircuitBreaker`: Thread-safe Closed / Open / Half-Open state machine with configurable failure threshold and reset cooldown.
  - Fast-failing (<1ms) when cloud API is down without blocking local event loops or HTTP requests.
  - Cloud provider stubs: `TuyaProvider`, `XiaomiProvider`, `EwelinkProvider`, and `VesyncProvider`.
  - `cloud.Adapter` implementing `plugins.Adapter`: degrades gracefully and emits `events.IntegrationStateChanged` with status `degraded`.
- **Real-Time Integration Status Forwarding**:
  - Subscribed `websocket.Hub` to `events.IntegrationStateChanged` so UI receives instant status updates when bridges or cloud adapters disconnect or degrade.
- **TDD Test Suites**:
  - Unit tests for Zigbee2MQTT device mapper and adapter.
  - Unit tests for ESPHome entity mapper and adapter.
  - Unit tests for Cloud CircuitBreaker and resilience adapter.
  - End-to-end integration test (`test/e2e/zigbee2mqtt_esphome_e2e_test.go`) validating concurrent Z2M, ESPHome, and Cloud adapters with live WebSocket frame verification.

## [0.3.0] - 2026-10-03

### Added
- **Hardware & Spatial Domains**:
  - `Area` domain model and SQLite repository for room and zone hierarchies.
  - `Integration` domain model and SQLite repository for plugin management.
  - `Device` domain model and SQLite repository tracking physical hardware and online statuses.
  - `Entity` domain model and SQLite repository for discrete telemetry points with typed state value objects (`LightState`, `SwitchState`, `SensorState`, `BinarySensorState`).
- **Generic MQTT Integration Adapter**:
  - `Client` interface with production Paho wrapper (`PahoClient`) and thread-safe testing mock (`MockClient`) supporting wildcard topic matching.
  - `mqtt_generic.Adapter` implementing `plugins.Adapter`.
  - Ingestion: translates incoming MQTT messages into entity state updates in SQLite and emits `events.EntityStateChanged` onto `events.Bus`.
  - Outbound commands: intercepts `events.CommandRequested` from event bus and dispatches payload to device MQTT command topics.
- **WebSocket Real-time Broadcast Hub**:
  - `websocket.Hub` managing active WebSocket client handles and real-time broadcasts.
  - Automatic event bus subscription forwarding `entity.state_changed`, `device.online`, `device.offline`, `scene.activated`, and `notification.created` to connected clients.
  - `websocket.Handler` upgrading connections with session validation and ping/pong keepalive.
- **REST Endpoints for Spaces & Hardware**:
  - `/api/v1/areas`: list and create areas.
  - `/api/v1/devices`: list and inspect devices with filtering.
  - `/api/v1/entities`: list and inspect entities with filtering.
  - `/api/v1/entities/{id}/command`: dispatch control commands triggering domain events and hardware execution.
  - `/ws`: real-time WebSocket connection endpoint.
- **TDD Test Suites**:
  - Unit tests for Area, Device, Entity, and Command HTTP handlers.
  - Integration tests for SQLite Area, Device, Entity, and Integration repositories.
  - Unit tests for WebSocket Hub and MQTT Generic Adapter.
  - E2E integration test: MQTT ingestion $\to$ SQLite persistence $\to$ EventBus dispatch $\to$ live WebSocket client frame delivery $\to$ REST command execution $\to$ MQTT publish.

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
