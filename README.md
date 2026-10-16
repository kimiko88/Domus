# Domus 🏠

> A lightweight, local-first, self-hosted smart home automation hub designed for Raspberry Pi 4.

Domus is built in modern idiomatic Go and Vue 3 + TypeScript, designed as a high-performance, resource-efficient alternative to heavier home automation platforms. It runs entirely on your local network without external cloud dependencies.

---

## ⚡ Highlights

- **Local-First Core**: Automations, state evaluation, MQTT communication, telemetry history, and dashboards run locally without Internet access.
- **Ultra-Low Resource Footprint**: Written in pure Go with zero CGO dependencies; runs comfortably in ~20-30 MB RAM on Raspberry Pi 4 ARM64.
- **Embedded SQLite in WAL Mode**: Fast concurrent telemetry reads and atomic writes with foreign key safety and zero external database services.
- **In-Memory Event Bus**: Decoupled, typed domain events dispatched asynchronously through bounded worker pools with handler isolation.
- **Dynamic Energy Tariffs & Load Optimization**: Native ingestion of daily PUN (Italy), Nord Pool, and Octopus dynamic hourly prices with automated scheduling of flexible, power-hungry loads (heat pumps, EV charging, boilers) at lowest or negative cost hours.
- **Local AI Computer Vision NVR**: Zero-cloud RTSP/WebRTC camera streaming with real-time motion zones and lightweight local ONNX object detection (people, vehicles, packages) generating instant animated snapshots on push notifications.
- **In-Browser USB WebSerial Firmware Flasher**: One-click firmware flashing directly from the browser (Chrome, Edge) with automatic baud rate negotiation and Wi-Fi provisioning for ESP32, ESP8266, Tasmota, WLED, Sonoff, and ESPHome.
- **Visual Flowchart Automation Designer**: Interactive Node-RED-style SVG flowchart canvas for designing, inspecting, and dry-run simulating multi-branch automation pipelines.
- **High Availability (HA) Failover Dual-Hub**: Keepalived Virtual IP (VIP) failover with real-time SQLite WAL delta replication, sub-second peer heartbeat monitoring, and 1-click manual node promotion.
- **Multi-Room Always-On Voice Satellites**: On-device "Ehi Domus" wake-word detection, local WebSocket audio streaming, room-aware contextual command resolution ("spegni la luce" acts on the room where the speaker is located), and hardware privacy mute.
- **Zero-Cloud Brand Discovery & 1-Click LAN Pairing**: Native LAN mDNS/SSDP discovery and instant pairing for Philips Hue, Shelly Gen1/Gen2, TP-Link Kasa/Tapo, Nanoleaf, Yeelight, Tuya Local, and Zigbee2MQTT.
- **Cloud Voice Assistants (Optional & Privacy-First)**: Lambda OAuth2 bridge for Amazon Alexa Smart Home (v3) and Smart Home Actions for Google Home (`action.devices.SYNC`, `QUERY`, `EXECUTE`) while guaranteeing 100% local privacy by default through bearer token endpoints.
- **Time-Limited Shared Dashboards**: Cryptographically signed HMAC-SHA256 temporary tokens for sharing specific dashboards with guests or kiosk screens (`/share/:token`) without exposing admin credentials.
- **Industrial Automation Protocol Bridges**: Native Modbus TCP adapter for commercial solar inverters and heat pumps (coils, registers, float32, scaling factors), alongside a pure Go KNXnet/IP tunneling client for European wired installations (DPT 1.001, 5.001, 9.001, 14.056).
- **Distributed Tracing (OpenTelemetry)**: Full W3C `traceparent` context propagation (`00-<trace_id>-<span_id>-01`) across incoming HTTP requests, asynchronous internal Event Bus dispatches, and SQLite WAL database operations.
- **100% Native Mobile Companion Apps (iOS & Android)**: Full-featured native companion apps written in Swift 6 (SwiftUI, WidgetKit, CarPlay, ActivityKit) and Kotlin 2.x (Jetpack Compose, Glance widgets, Quick Settings Tiles, Biometrics) with zero mock/stub data, sub-millisecond mDNS discovery (`_domus._tcp.local`), and real-time bidirectional WebSocket live sync.
- **Outbound Webhook Dispatcher**: Asynchronous event dispatching engine with HMAC-SHA256 request signing (`X-Domus-Signature`), wildcard event pattern matching (e.g. `alarm.*`, `entity.state_changed`, `*`), bounded worker pool, automatic fail count tracking, and immediate testing endpoints.
- **Comprehensive API & Network Hardening**: Request body size limitation (`MaxBodySize` 4 MB limit) against DoS payload inflation, token bucket rate limiting on authentication and sensitive endpoints, strict CORS/CSP policies, and automatic CSRF header injection.
- **Security Alarm Control Panel & Zigbee Direct Binding**: Built-in multi-state security alarm system (Away, Home, Night, Disarmed) with entry/exit delays and Zigbee direct binding bypass that operates even if the hub is rebooting.
- **Shared Family Shopping List**: Collaborative real-time shopping list with department categorization and instant WebSocket synchronization across family members.
- **Security Hardening**: Argon2id password hashing, session cookies with SameSite/HttpOnly, CSRF protection, rate limiting, and immutable audit logging.
- **Encrypted Backup & Restore**: AES-256-GCM encrypted backup archives with PBKDF2 key derivation and path-traversal-safe restore CLI.
- **Progressive Web App (PWA)**: Vue 3 + TypeScript + Vite + Pinia single-page application with glassmorphic dark UI, real-time WebSocket live sync, and mobile installability.
- **Multi-Language Frontend**: Full internationalization (i18n) with 100% key and subkey parity across **Italiano 🇮🇹, English 🇬🇧, Français 🇫🇷, Español 🇪🇸, and Deutsch 🇩🇪**.
- **Interactive OpenAPI Documentation**: Embedded OpenAPI 3.1 specification and Swagger UI accessible directly at `/docs`.
- **TDD (Test-Driven Development)**: Comprehensive unit tests, repository integration tests, and full-stack end-to-end (E2E) suites across backend, web, iOS, and Android.

---

## 📁 Repository Structure

```
domus/
├── cmd/
│   ├── domus/              # Main hub entrypoint (server, event bus, db wiring)
│   └── domus-cli/          # CLI utility for administration, backup, and restore
│
├── internal/
│   ├── domain/             # Pure business models and port interfaces (zero dependencies)
│   │   ├── alarm/          # Alarm control panel states and PIN validation
│   │   ├── area/           # Room and area spatial hierarchies
│   │   ├── audit/          # Security audit log entities
│   │   ├── automation/     # JSON DSL triggers, conditions, actions, and runs
│   │   ├── camera/         # Cameras, motion zones, AI detection events
│   │   ├── chore/          # Chores and recurrence schedules
│   │   ├── cloud_assistant/# Alexa Smart Home & Google Actions models
│   │   ├── dashboard/      # Custom widget dashboards & share token models
│   │   ├── device/         # Physical hardware abstractions
│   │   ├── energy/         # Energy tariffs, load shedding, flexible loads
│   │   ├── entity/         # Typed domain entities (lights, switches, sensors, climate)
│   │   ├── events/         # Typed domain events, bus interface & W3C trace context
│   │   ├── floor/          # Multi-level floor plans and interactive markers
│   │   ├── helper/         # Virtual helpers (input booleans, numbers, timers)
│   │   ├── history/        # Telemetry historical state models
│   │   ├── integration/    # Hardware and cloud adapter models
│   │   ├── mesh/           # Zigbee mesh topology and direct bindings
│   │   ├── scene/          # Multi-device preset scenes
│   │   ├── shopping/       # Shared family shopping list items
│   │   └── user/           # User authentication and RBAC roles
│   │
│   ├── application/        # Application services and use cases
│   │   ├── alarm/          # Alarm panel state machine service
│   │   ├── assistant/      # Local LLM voice assistant & satellite dispatcher
│   │   ├── auth/           # Login, wizard setup, session management, rate limiter, family invites
│   │   ├── automation/     # Rule evaluator and asynchronous execution engine
│   │   ├── backup/         # AES-256-GCM backup and idempotent restore service
│   │   ├── brand/          # Zero-cloud LAN discovery and 1-click pairing service
│   │   ├── camera/         # Camera streams, zones, and local ONNX vision pipeline
│   │   ├── cloud_assistant/# Alexa Directive & Google Smart Home fulfillment service
│   │   ├── dashboard/      # Dashboard CRUD & temporary HMAC-SHA256 share service
│   │   ├── energy/         # Tariffs evaluator & flexible load scheduler
│   │   ├── history/        # Batched state recorder and retention cleanup worker
│   │   ├── mesh/           # Zigbee direct binding manager
│   │   ├── ota/            # Firmware catalog and WebSerial provisioning
│   │   ├── scene/          # Scene activation dispatcher
│   │   └── shopping/       # Real-time shopping list service
│   │
│   ├── infrastructure/     # Implementation of technical ports
│   │   ├── config/         # TOML + env variable loader, PBKDF2/AES-GCM secrets
│   │   ├── crypto/         # Argon2id password hasher
│   │   ├── eventbus/       # Bounded in-memory event bus with trace context propagation
│   │   ├── mqtt/           # Paho MQTT v3.1.1 client wrapper with auto-reconnect
│   │   ├── sqlite/         # SQLite driver (pure Go), WAL pragma, embedded migrations & query tracing
│   │   └── websocket/      # Real-time WebSocket broadcast hub
│   │
│   ├── interfaces/         # Transport layer
│   │   ├── http/           # Chi router, security headers, CORS, handlers
│   │   │   ├── handlers/   # REST API endpoint handlers (cloud, dashboard share, alarm, etc.)
│   │   │   ├── middleware/ # W3C TraceContext, Auth, RBAC, CSRF, rate limit, security headers, logger
│   │   │   └── swagger/    # Embedded OpenAPI 3.1 YAML and Swagger UI
│   │   └── websocket/      # WebSocket client connection upgrade handler
│   │
│   └── plugins/            # Protocol integration adapters
│       ├── ble_proxy/      # Bluetooth Low Energy proxy & beacon tracker
│       ├── cloud/          # Circuit breaker and cloud provider stubs
│       ├── esphome_mqtt/   # ESPHome telemetry and command mapper
│       ├── homekit/        # Native Apple HomeKit accessory bridge
│       ├── knx/            # European wired KNXnet/IP UDP tunneling client & DPT codec
│       ├── matter/         # Matter/Thread cluster commissioner & node manager
│       ├── modbus_tcp/     # Industrial Modbus TCP adapter (inverters, heat pumps, energy meters)
│       ├── mqtt_generic/   # Topic-to-entity MQTT generic adapter
│       ├── tuya_local/     # Local LAN protocol for Tuya devices (zero cloud)
│       ├── wasm/           # Sandboxed WebAssembly plugin runtime
│       ├── wyoming/        # Wyoming protocol integration for voice satellites
│       ├── zigbee2mqtt/    # Zigbee2MQTT device discovery and status mapper
│       └── zwave_js/       # Z-Wave JS WebSocket driver (switches, sensors, meters)
│
├── web/                    # Frontend Vue 3 + TypeScript Progressive Web App
│   ├── src/
│   │   ├── api/            # Typed REST client and models (with CSRF auto-injection)
│   │   ├── components/     # Modals (Flowchart, Flasher, Satellites), cards, charts
│   │   ├── locales/        # 100% parity: it.json, en.json, fr.json, es.json, de.json
│   │   ├── router/         # Vue router with navigation auth guards
│   │   ├── stores/         # Pinia stores (auth, family, entities, devices, automations, etc.)
│   │   └── views/          # Dashboard, Energy, Cameras, Shopping, Integrations, Automations, etc.
│   └── package.json
│
├── mobile/                 # 100% Native Mobile Companion Apps (Zero WebViews/Mocks)
│   ├── ios/                # DomusKit (Swift 6, SwiftUI, WidgetKit, CarPlay, ActivityKit)
│   └── android/            # com.domus.app (Kotlin 2.x, Jetpack Compose, Glance, TileService)
│
├── os/                     # Domus OS Buildroot BSP & Embedded Linux Appliance (RPi4 ARM64)
│   ├── configs/            # Buildroot defconfigs (domus_rpi4_defconfig)
│   ├── board/              # Board configurations, genimage GPT, config.txt, cmdline.txt, boot.cmd
│   ├── build-container/    # Hermetic Docker build container for reproducible builds
│   └── scripts/            # Build and validation scripts (build.sh)
│
├── docs/
│   ├── adr/                # Architecture Decision Records (ADR-001 to ADR-013)
│   ├── openapi/            # OpenAPI 3.1 YAML REST specification
│   ├── os/                 # Domus OS BSP and Appliance guides (MS-OS-01)
│   └── USER-MANUAL.md      # Comprehensive user and administrator manual (IT)
│
├── deployments/
│   ├── docker/             # Multi-stage Dockerfile and Docker Compose stack
│   ├── systemd/            # Hardened Linux systemd service unit
│   └── scripts/            # Non-interactive install and uninstall automation scripts
│
├── Makefile                # Build, test, lint, run and Domus OS targets
├── .golangci.yml           # Static analysis and linter configuration
└── domus.toml.example      # Reference configuration template
```

---

## ⚡ Domus OS: Purpose-Built Embedded Linux Appliance

Domus can run as a standalone, immutable Linux appliance on **Raspberry Pi 4 Model B (ARM64)**, built with **Buildroot 2024.02 LTS**, **Linux 6.6 LTS**, and **U-Boot**:

- **Dual-slot A/B updates** with automated rollback on failure.
- **Immutable SquashFS** read-only root filesystem.
- **Dedicated persistent `/data` partition** for SQLite WAL, configurations, and apps.
- **Reproducible build pipeline** executable natively or via hermetic Docker container.

```bash
# Validate BSP configuration and partition scheme
make os-validate

# Run BSP automated test suite
make os-test

# Build complete raw flashable disk image (.img) via Docker
make os-build
```

See [MS-OS-01-BSP-GUIDE.md](docs/os/MS-OS-01-BSP-GUIDE.md) for full instructions.

---

## 🚀 Quick Start & Installation

### Option 1: Automated Non-Interactive Script (Raspberry Pi & Linux)

The fastest way to install Domus as a production systemd service on Raspberry Pi OS or Debian/Ubuntu:

```bash
# Clone the repository
git clone https://github.com/domus-hub/domus.git
cd domus

# Run the automated installer with root privileges
sudo ./deployments/scripts/install.sh
```

The script automatically detects system architecture (`arm64`, `amd64`), provisions unprivileged system user `domus`, creates directories (`/var/lib/domus`, `/etc/domus`), compiles or copies binaries, installs the systemd unit, and enables the service.

To uninstall:
```bash
sudo ./deployments/scripts/uninstall.sh
```

---

### Option 2: Docker Compose (Recommended for Containerized Environments)

Run Domus along with an Eclipse Mosquitto MQTT broker:

```bash
docker compose -f deployments/docker/docker-compose.yml up -d --build
```

The hub will be reachable at `http://localhost:8080`.

---

### Option 3: Local Development (Go + Node)

#### Prerequisites
- **Go**: 1.23 or newer
- **Node.js**: 20 or newer
- **Make** (optional, recommended)

```bash
# 1. Start the Go backend
make run

# 2. In another terminal, start the web frontend dev server
cd web
npm install
npm run dev
```

Navigate to `http://localhost:5173` for the frontend with hot-reload, or `http://localhost:8080` for the backend.

---

## 📖 API Documentation (Embedded Swagger UI)

Domus ships with an embedded OpenAPI 3.1 specification and self-contained Swagger UI.
Once the server is running, explore and test the endpoints directly in your browser:

- **Swagger UI Explorer**: [http://localhost:8080/docs](http://localhost:8080/docs) (or `/swagger`)
- **Raw OpenAPI YAML Spec**: [http://localhost:8080/api/openapi.yaml](http://localhost:8080/api/openapi.yaml)
- **Health Check**: [http://localhost:8080/health](http://localhost:8080/health)

---

## 🔐 Administration & Backup CLI (`domus-cli`)

Domus includes a dedicated administration tool for backups and disaster recovery:

```bash
# Display CLI version and help
domus-cli help

# Create a plain .tar.gz backup
domus-cli backup -db /var/lib/domus/domus.db -config /etc/domus/domus.toml -out /backups

# Create an encrypted AES-256-GCM backup (.tar.gz.enc)
domus-cli backup -db /var/lib/domus/domus.db -out /backups -passphrase "MySuperSecretKey"

# Restore from plain backup
domus-cli restore -archive /backups/domus-backup-20261004-120000.tar.gz -out /var/lib/domus

# Restore from encrypted backup
domus-cli restore -archive /backups/domus-backup-20261004-120000.tar.gz.enc -out /var/lib/domus -passphrase "MySuperSecretKey"
```

---

## 🌍 Frontend Localization (i18n)

The user interface supports 5 languages out-of-the-box with 100% key parity:
- 🇮🇹 **Italiano** (Predefinito)
- 🇬🇧 **English**
- 🇫🇷 **Français**
- 🇪🇸 **Español**
- 🇩🇪 **Deutsch**

Language selection is persisted to local storage and dynamically reactive across all views, controls, navigation bars, modals, and audit logs.

---

## 🧪 Testing & Quality Assurance

Domus is built using strict **Test-Driven Development (TDD)**:

```bash
# Run all Go backend tests (unit, integration, and E2E)
go test ./... -v

# Run all Vue 3 / Pinia frontend tests (Vitest)
cd web && npm run test:run

# Verify production frontend build & TypeScript types
cd web && npm run build
```

---

## 🗺️ Roadmap & Milestones (100% Completed)

- [x] **MS-01: Foundation & Skeleton** — Pure Go SQLite in WAL mode, bounded in-memory event bus, TOML loader, HTTP server, E2E suite, Docker/Systemd.
- [x] **MS-02: Auth, RBAC & First API** — Argon2id hashing, session cookies, CSRF protection, admin onboarding wizard, audit repository.
- [x] **MS-03: MQTT Generic & Entity Model** — Entity, Device, Area, Integration models, MQTT adapter, WebSocket live feed, REST commands.
- [x] **MS-04: Zigbee2MQTT & ESPHome** — Z2M discovery and state mapper, ESPHome MQTT mapper, Cloud resilience circuit breaker.
- [x] **MS-05: Automation Engine v1** — Structured JSON DSL, trigger/condition/action evaluator, runner pool, scenes CRUD & activator.
- [x] **MS-06: Frontend Core** — Vue 3 + TypeScript + Vite + Pinia PWA, live dashboard, responsive glassmorphic cards.
- [x] **MS-07: Frontend Automations & Scenes** — Visual DSL automation builder, execution runs drawer, scene manager.
- [x] **MS-08: Telemetry History & Metrics** — Batched state history recorder, scheduled retention worker, Chart.js time-series, system metrics.
- [x] **MS-09: Hardening & Backup/Restore** — Security headers (HSTS, CSP, XSS), CORS, audit log viewer, AES-256-GCM encrypted backup/restore CLI.
- [x] **MS-10: Final ARM64 Packaging & Release** — Multi-stage ARM64 Dockerfile, non-interactive installation scripts, embedded OpenAPI/Swagger UI, complete ADRs, 5-language localization.
- [x] **MS-11: Multi-Client Family & Granular RBAC** — Concurrent multi-client sessions, creator-defined roles (admin, operator, viewer), secure invite links (`/register?token=...`), and family member governance.
- [x] **MS-12: Advanced Native Ecosystem & Strategic Innovations** — Dynamic Energy Tariffs (PUN / Nord Pool / Octopus) & Flexible Loads Scheduler, Local AI Computer Vision NVR with ONNX pipeline, In-Browser USB WebSerial Firmware Flasher, Visual Flowchart Automation Designer, Dual-Hub High Availability Failover, Multi-Room Voice Satellites ("Ehi Domus"), Zero-Cloud 1-Click Brand Discovery (Hue, Shelly, Tapo, Kasa, Nanoleaf, Yeelight, Tuya), Security Alarm Control Panel, Zigbee Direct Mesh Binding, Shared Family Shopping List, and complete 5-language localization parity across 100% of views.

---

## 📖 Documentazione

- **[Manuale Utente](docs/USER-MANUAL.md)** — Guida completa dall'installazione all'uso quotidiano (IT)
- **[Swagger UI](http://localhost:8080/docs)** — Documentazione API REST interattiva (quando il server è attivo)
- **[OpenAPI YAML](docs/openapi/domus.yaml)** — Specifica OpenAPI 3.1 completa

---

## 📜 Architecture Decision Records (ADRs)

### Domus Core
- [ADR-001: Modular Monolith Architecture with Hexagonal Ports & Adapters](docs/adr/ADR-001-modular-monolith.md)
- [ADR-002: SQLite with Write-Ahead Logging (WAL) and Pure Go Driver](docs/adr/ADR-002-sqlite-wal.md)
- [ADR-003: In-Memory Bounded Event Bus](docs/adr/ADR-003-event-bus-in-memory.md)
- [ADR-004: Structured JSON DSL for Automations and Scenes](docs/adr/ADR-004-json-dsl-automations.md)
- [ADR-005: AES-256-GCM Secrets Encryption and Idempotent Backup/Restore](docs/adr/ADR-005-secrets-encryption-and-backup.md)
- [ADR-006: Packaging, Multi-Architecture Containerization, and Offline Deployment](docs/adr/ADR-006-packaging-and-offline-deployment.md)

### Domus OS
- [ADR-007: Buildroot for Domus OS](docs/adr/ADR-007-buildroot-for-domus-os.md)
- [ADR-008: containerd Runtime](docs/adr/ADR-008-containerd-runtime.md)
- [ADR-009: Atomic A/B Updates with RAUC](docs/adr/ADR-009-atomic-ab-updates-rauc.md)
- [ADR-010: SquashFS and Stateless Root Filesystem](docs/adr/ADR-010-squashfs-and-stateless-rootfs.md)
- [ADR-011: Persistent Data Partition](docs/adr/ADR-011-persistent-data-partition.md)
- [ADR-012: Embedded Security and Hardening](docs/adr/ADR-012-embedded-security-and-hardening.md)
- [ADR-013: Supervisor and App Sandboxing](docs/adr/ADR-013-supervisor-and-app-sandboxing.md)

---

## 📄 License

Licensed under the MIT License.
