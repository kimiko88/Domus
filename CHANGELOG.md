# Changelog

All notable changes to the Domus project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.2.0] - 2026-10-10

### Added
- **Dynamic Energy Tariffs & Flexible Load Management (MS-12)**:
  - **Tariff Ingestion Engine (`internal/domain/energy`, `internal/application/energy`)**:
    - Daily ingestion and caching of hourly day-ahead electricity prices from Italian PUN, Nord Pool, and Octopus Agile.
    - 24-hour interactive pricing curve visualization with low/medium/high threshold color-coding.
    - Automated scheduling engine for flexible, power-hungry loads (heat pumps, EV charging, storage water heaters) at minimum or negative cost windows.
    - REST endpoints: `GET /api/v1/energy/tariffs`, `GET /api/v1/energy/dynamic-tariffs`, `PUT /api/v1/energy/flexible-loads`, `POST /api/v1/energy/evaluate-tariffs`.
- **Local AI Computer Vision NVR & WebRTC (MS-12)**:
  - **NVR Pipeline (`internal/domain/camera`, `internal/application/camera`)**:
    - Lightweight Go-native RTSP ingestion and low-latency WebRTC video streaming.
    - Local ONNX object detection pipeline for zero-cloud classification (people, vehicles, packages, pets).
    - Customizable polygonal motion detection zones per camera feed.
    - Event clips generation and instant animated snapshot previews on mobile push notifications.
    - REST endpoints: `GET /api/v1/cameras`, `POST /api/v1/cameras`, `GET /api/v1/cameras/nvr/status`, `POST /api/v1/cameras/{id}/webrtc`, `GET /api/v1/cameras/{id}/zones`, `POST /api/v1/cameras/{id}/zones`, `POST /api/v1/cameras/{id}/detect`, `GET /api/v1/cameras/events`, `GET /api/v1/cameras/events/{id}/clip`.
- **In-Browser USB WebSerial Firmware Flasher (MS-12)**:
  - **WebSerial Flashing Suite (`web/src/components/modals/FirmwareFlasherModal.vue`, `internal/application/ota`)**:
    - Direct browser-to-MCU USB flashing using W3C WebSerial API (Chrome, Edge, Opera).
    - Built-in firmware catalog for ESP32, ESP8266, Tasmota, WLED, ESPHome, and Sonoff Zigbee Dongles.
    - Automatic baud rate negotiation, bootloader handshake, flash offset management, and real-time ANSI terminal logging.
    - Wi-Fi provisioning injecting network credentials and hub IP directly into firmware images prior to flashing.
    - REST endpoints: `GET /api/v1/ota/flasher/catalog`, `GET /api/v1/ota/flasher/{id}/manifest`, `POST /api/v1/ota/flasher/provision`.
- **Visual Flowchart Automation Designer (MS-12)**:
  - **SVG Flowchart Designer (`web/src/components/modals/AutomationFlowchartModal.vue`)**:
    - Interactive Node-RED-style SVG canvas integrated in the automations view.
    - Trigger, condition/gate, and action nodes with drag-and-drop connections.
    - Real-time dry-run execution simulator tracing event propagation paths.
    - Bidirectional serialization with Domus native JSON DSL.
- **Dual-Hub High Availability Failover (MS-12)**:
  - **HA & Replication Architecture (`internal/application/ecosystem`, `web/src/views/SettingsView.vue`)**:
    - Keepalived Virtual IP (VIP) sharing between primary and secondary hub nodes.
    - Real-time SQLite WAL delta streaming synchronization.
    - Continuous heartbeat monitoring with automated sub-2s failover detection.
    - Manual leader promotion via Settings view and REST API.
    - REST endpoints: `GET /api/v1/system/failover/status`, `POST /api/v1/system/failover/heartbeat`, `POST /api/v1/system/failover/promote`.
- **Multi-Room Always-On Voice Satellites (MS-12)**:
  - **Voice & Satellite Architecture (`internal/application/assistant`, `web/src/components/modals/VoiceAssistantModal.vue`)**:
    - Support for micro-satellites (Raspberry Pi Zero 2W, ESP32-S3 Box) with local "Ehi Domus" wake-word detection.
    - Audio streaming over local WebSocket to the hub for transcription and intent parsing.
    - Room-aware spatial context: automatic command resolution targeting entities in the satellite's room.
    - Hardware privacy mute switch support and visual LED indicators.
    - REST endpoints: `POST /api/v1/assistant/chat`, `GET /api/v1/assistant/config`, `PUT /api/v1/assistant/config`, `GET /api/v1/assistant/satellites`, `POST /api/v1/assistant/satellites`, `POST /api/v1/assistant/satellites/{id}/stream`.
- **Zero-Cloud Brand Discovery & 1-Click LAN Pairing (MS-12)**:
  - **Brand Ecosystem (`internal/application/brand`, `web/src/views/IntegrationsView.vue`)**:
    - Local network discovery via mDNS, SSDP, and broadcast UDP packets.
    - Native integration adapters for Philips Hue, Shelly (Gen1/Gen2/Gen3), TP-Link Kasa/Tapo, Nanoleaf, Yeelight, Tuya Local, and Zigbee2MQTT.
    - 1-Click pairing UI in Integrations view mapping physical device endpoints into typed Domus entities.
    - REST endpoints: `GET /api/v1/integrations/catalog`, `GET /api/v1/integrations/discovery/devices`, `POST /api/v1/integrations/discovery/scan`, `POST /api/v1/integrations/discovery/pair`.
- **Security Alarm Control Panel & Zigbee Direct Binding (MS-12)**:
  - **Security Panel & Mesh (`internal/domain/alarm`, `internal/application/alarm`, `internal/application/mesh`)**:
    - Four security armed modes (Disarmed, Arm Home, Arm Night, Arm Away) with PIN validation and entry/exit delays.
    - Zigbee direct binding manager enabling sensors to bind directly to actuators, working autonomously even when the hub is offline.
    - REST endpoints: `GET /api/v1/alarm`, `POST /api/v1/alarm/arm`, `POST /api/v1/alarm/disarm`, `POST /api/v1/alarm/trigger`, `PUT /api/v1/alarm/config`, `GET /api/v1/devices/bindings`, `POST /api/v1/devices/bindings`, `DELETE /api/v1/devices/bindings/{id}`.
- **Shared Family Shopping List (MS-12)**:
  - **Collaborative Shopping (`internal/domain/shopping`, `internal/application/shopping`, `web/src/views/ShoppingListView.vue`)**:
    - Real-time collaborative shopping checklist with WebSocket synchronization.
    - Automatic grocery department categorization and one-click archiving of completed items.
    - REST endpoints: `GET /api/v1/shopping`, `POST /api/v1/shopping`, `POST /api/v1/shopping/{id}/toggle`, `DELETE /api/v1/shopping/{id}`, `POST /api/v1/shopping/clear-completed`.
- **Full Frontend Localization Parity (i18n)**:
  - 100% key and subkey parity across all 5 supported languages (`it`, `en`, `fr`, `es`, `de`).
  - Complete coverage across all views, dialogs, charts, flasher terminal, and navigation elements.

## [1.1.0] - 2026-10-04

### Added
- **Multi-Client Family Management & Granular RBAC**:
  - **Domain & Persistence Layer (`internal/domain/user`, `internal/infrastructure/sqlite`)**:
    - `FamilyInvite` domain entity supporting token validation, pre-assigned roles (`admin`, `operator`, `viewer`), and configurable expiration.
    - Database migration `000002_family_invites` creating SQLite table with indexes.
    - SQLite repositories for `InviteRepository` and extensions to `UserRepository` (`List`, `UpdateRole`, `Delete`) and `SessionRepository` (`DeleteByUserID`).
  - **Application Service (`internal/application/auth/family_service.go`)**:
    - Invite generation (`CreateFamilyInvite`), token lookup (`GetInviteByToken`), invite revocation (`RevokeFamilyInvite`), and registration (`RegisterWithInvite`).
    - Family member governance (`ListFamilyMembers`, `UpdateFamilyMemberRole`, `RemoveFamilyMember`) with safeguards preventing demoting or removing the last administrator or self-deletion.
  - **HTTP Endpoints & Router RBAC Enforcement (`internal/interfaces/http/handlers/family_handler.go`, `server.go`)**:
    - Public invite endpoints: `GET /api/v1/auth/invite/{token}` and `POST /api/v1/auth/register`.
    - Family governance endpoints: `GET /api/v1/family/members`, `PUT /api/v1/family/members/{id}/role`, `DELETE /api/v1/family/members/{id}`, `GET /api/v1/family/invites`, `POST /api/v1/family/invites`, `DELETE /api/v1/family/invites/{id}`.
    - Granular RBAC middleware enforcement across all mutating endpoints:
      - `operator` and `admin` can send commands (`POST /entities/{id}/command`), apply scenes (`POST /scenes/{id}/apply`), and trigger automations (`POST /automations/{id}/trigger`).
      - `viewer` has read-only access (403 Forbidden on commands and triggers).
      - `admin` only can alter configuration (areas, scenes, automations, family members, invites).
  - **Concurrent Multi-Client Sessions**:
    - Validated simultaneous WebSocket connections and API calls across multiple clients and family members with independent buffering and real-time event distribution.
  - **Frontend UI & Localization (`web/`)**:
    - `FamilyView.vue`: Family members list with role badges, role selector, member deletion, invite modal, role description cards, and pending invites table with 1-click link copying.
    - `RegisterView.vue`: Dedicated registration flow supporting token extraction from query parameters, invitation validation, role banner, and immediate authenticated session creation.
    - Complete 5-language localization (`it`, `en`, `fr`, `es`, `de`) with 100% key parity.

## [1.0.0] - 2026-10-04

### Added
- **ARM64 & Production Packaging (MS-10)**:
  - Multi-stage Dockerfile compiling web assets (`web-builder`) and pure Go binaries (`builder`) for `linux/arm64` and `linux/amd64` into a hardened, non-root Alpine runtime.
  - Automated, non-interactive installation script (`deployments/scripts/install.sh`) supporting bare-metal Raspberry Pi 4 and Linux SBCs with automated architecture detection, system user creation, directory permissions, and systemd service registration.
  - Uninstallation script (`deployments/scripts/uninstall.sh`) with safe service teardown and optional data purge flag (`PURGE_DATA=1`).
  - Hardened systemd service unit (`deployments/systemd/domus.service`) with Linux sandboxing (`ProtectSystem=strict`, `ProtectHome=true`, `PrivateTmp=true`, `PrivateDevices=true`, `ReadWritePaths=/var/lib/domus`).
- **Embedded OpenAPI 3.1 & Interactive Swagger UI**:
  - `internal/interfaces/http/swagger`: Embedded OpenAPI 3.1 specification (`domus.yaml`) via `go:embed`.
  - Self-contained Swagger UI explorer served at `/docs` and `/swagger`.
  - Raw YAML specification served at `/api/openapi.yaml`.
- **5-Language Frontend Localization (i18n)**:
  - Complete translation catalogs for **Italiano 🇮🇹, English 🇬🇧, Français 🇫🇷, Español 🇪🇸, and Deutsch 🇩🇪**.
  - Parity test suite verifying 100% key and subkey consistency across all 5 languages (`src/locales/i18n.test.ts`).
  - Dynamic language selector in `SettingsView` and quick toggle in `AppTopbar` with `localStorage` persistence.
- **Architecture Decision Records (ADRs)**:
  - ADR-004: Structured JSON DSL for Automations and Scenes.
  - ADR-005: AES-256-GCM Secrets Encryption and Idempotent Backup/Restore.
  - ADR-006: Packaging, Multi-Architecture Containerization, and Offline Deployment.
- **Documentation & Static SPA Serving**:
  - Comprehensive `README.md` with complete installation instructions, CLI usage, and architecture overview.
  - Configurable static SPA serving via `web_dir` setting and `DOMUS_SERVER_WEB_DIR` environment variable with fallback to embedded documentation.

## [0.9.0] - 2026-10-04

### Added
- **Security Hardening & Encrypted Backup/Restore (MS-09)**:
  - **Secrets Encryption (`internal/infrastructure/config/secrets.go`)**:
    - AES-256-GCM symmetric encryption for configuration secrets, tokens, and backup archives.
    - PBKDF2-HMAC-SHA256 key derivation with 100,000 iterations and cryptographically random 16-byte salt.
  - **Backup & Restore Application Service (`internal/application/backup/`)**:
    - Generates `.tar.gz` (plain) or `.tar.gz.enc` (AES-256-GCM encrypted) archives containing SQLite database, TOML configuration, and signed `backup_manifest.json`.
    - Idempotent restore engine with zip-slip / path-traversal attack protection ensuring files are extracted safely within target bounds.
  - **Administration CLI (`cmd/domus-cli/main.go`)**:
    - Subcommands `backup` and `restore` with passphrase flag support.
    - CLI commands for inspecting archive metadata and verifying integrity.
  - **HTTP Security Middlewares (`internal/interfaces/http/middleware/security.go`)**:
    - Security headers middleware: `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `X-XSS-Protection: 1; mode=block`, `Referrer-Policy: strict-origin-when-cross-origin`, `Strict-Transport-Security: max-age=31536000`, `Content-Security-Policy`.
    - Configurable Cross-Origin Resource Sharing (CORS) middleware.
  - **Security Audit Log View & Routing (`web/src/views/AuditView.vue`)**:
    - Dedicated audit log page displaying security and authentication events, actor types, timestamps, and expandable JSON payload inspector.
    - Exposed `GET /api/v1/audit` endpoint requiring authentication.

## [0.8.0] - 2026-10-04

### Added
- **State History & Telemetry Storage (MS-08)**:
  - **History Domain (`internal/domain/history`)**:
    - `StateHistory` domain model with validation invariants and `Repository` interface supporting `SaveBatch`, `ListByEntity`, and `PurgeOlderThan`.
  - **SQLite History Repository (`internal/infrastructure/sqlite/history_repo.go`)**:
    - High-performance transactional batch persistence into `state_history` table.
    - Chronological indexing and query filtering by entity ID/key, `from`, `to`, and `limit`.
    - Retention purges for historical state snapshots older than configured cutoff.
    - Added `PurgeOlderThan` in `audit.Repository` and `PurgeRunsOlderThan` in `automation.Repository`.
  - **Application Services (`internal/application/history/`)**:
    - `Recorder`: Asynchronous batched state history recorder subscribing to `events.EntityStateChanged` with buffered queue, periodic interval flush, batch size threshold, and safe drain on shutdown.
    - `RetentionService`: Background scheduled worker purging expired state history, audit logs, and automation runs according to `RetentionConfig` policies.
  - **HTTP Endpoints & Server Wiring**:
    - `GET /api/v1/entities/{id}/history`: Historical state logs query supporting both UUIDs and entity keys, with time range filtering and clamping limits.
    - `GET /api/v1/system/metrics`: Operational metrics endpoint reporting `uptime_s`, `entity_count`, `device_count`, `mqtt_connected`, `ws_clients`, and `db_size_bytes`.
- **Frontend Localization Architecture (i18n)**:
  - Robust, easily extensible localization system configured with `vue-i18n`.
  - Complete translation catalogs for Italian (`it.json`, default) and English (`en.json`).
  - Modular structure allowing new languages to be plugged in simply by adding a new JSON catalog.
  - Multi-language selectors in `SettingsView` and quick language toggle pill in `AppTopbar` with automatic `localStorage` persistence.
  - Localized navigation items, entity domains, history controls, and settings.
- **Frontend History & Metrics Components**:
  - `useHistoryStore` (`web/src/stores/history.ts`): Pinia store for fetching and caching historical telemetry.
  - `useSystemStore` (`web/src/stores/system.ts`): Pinia store for querying live system metrics and formatting human-readable uptime.
  - `SensorHistoryChart.vue` (`web/src/components/charts/`): Interactive Chart.js component with smooth bezier curves, neon cyan gradient fill, dark slate gridlines, and unit tooltips.
  - `HistoryView.vue` (`web/src/views/`): Full historical explorer featuring entity selector (prioritizing sensors), time range selectors (1h, 24h, 7d, 30d), summary stats cards (Current, Min, Max, Avg), and tabular data log.
  - `SettingsView.vue` (`web/src/views/`): Enhanced settings dashboard featuring live system metrics (with on-demand refresh) and language switcher.
- **TDD Test Suites**:
  - 136 frontend tests passing across 41 test files in Vitest.
  - All backend unit, integration, and end-to-end tests passing in Go.
  - Production PWA bundle verified (`vue-tsc -b && vite build`).

## [0.7.0] - 2026-10-04

### Added
- **Frontend Automations & Scenes (MS-07)**:
  - **API Types & Data Models (`web/src/api/types.ts`)**:
    - Full TypeScript typings for `Automation`, `Trigger`, `Condition`, `Action`, `AutomationRun`, `Scene`, `SceneAction`, and respective API response wrappers.
  - **Pinia State Management Stores**:
    - `automations`: Full CRUD lifecycle (`fetchAutomations`, `createAutomation`, `updateAutomation`, `deleteAutomation`), reactive live toggle (`toggleAutomation`), test execution trigger (`triggerAutomation`), and execution run logs query (`fetchRuns`).
    - `scenes`: Full CRUD operations (`fetchScenes`, `createScene`, `updateScene`, `deleteScene`), and one-click manual scene activation (`applyScene`).
  - **Card Components (`web/src/components/cards/`)**:
    - `SceneCard`: Glassmorphic scene card featuring ambient icon glow, one-click "Attiva" action button with pending feedback, edit/delete actions, and configured action counter badge.
    - `AutomationCard`: Feature-rich automation card with live toggle switch, execution mode badge (`single`, `restart`, `queued`, `parallel`), cooldown indicators, last run status badge, manual test trigger button, and run history viewer toggle.
  - **Modals & DSL Editors (`web/src/components/modals/`)**:
    - `SceneEditorModal`: Multi-action preset designer with dynamic action rows (entity selector, service command, and JSON parameters payload) and client-side validation.
    - `AutomationEditorModal`: Complete automation builder featuring metadata fields, execution mode selector, cooldown seconds configuration, and a validated JSON DSL editor with syntax validation, formatting helper, quick presets (Motion Light, Night Schedule), and domain invariant checks (`name`, `triggers`, `actions`).
    - `AutomationRunsModal`: Full execution history drawer with status badges (`completed`, `failed`, `running`, `skipped`), duration in milliseconds, trigger context details, error diagnostics, and step execution logs.
  - **Views (`web/src/views/`)**:
    - `ScenesView`: Replaced placeholder with rich scenes catalog, search filtering, quick creation workflow, and toast feedback.
    - `AutomationsView`: Replaced placeholder with search filter, status pills (`Tutte`, `Attive`, `Disabilitate`), creation modal integration, card grid, and runs history inspection.
  - **TDD Test Suites**:
    - 36 test files with 120 unit and component tests passing.
    - Full type checking and production PWA build verified (`vue-tsc -b && vite build`).

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
