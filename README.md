# Domus 🏠

> A lightweight, local-first, self-hosted smart home automation hub designed for Raspberry Pi 4.

Domus is built in modern idiomatic Go and Vue 3 + TypeScript, designed as a high-performance, resource-efficient alternative to heavier home automation platforms. It runs entirely on your local network without external cloud dependencies.

---

## ⚡ Highlights

- **Local-First Core**: Automations, state evaluation, MQTT communication, telemetry history, and dashboards run locally without Internet access.
- **Ultra-Low Resource Footprint**: Written in pure Go with zero CGO dependencies; runs comfortably in ~20-30 MB RAM on Raspberry Pi 4 ARM64.
- **Embedded SQLite in WAL Mode**: Fast concurrent telemetry reads and atomic writes with foreign key safety and zero external database services.
- **In-Memory Event Bus**: Decoupled, typed domain events dispatched asynchronously through bounded worker pools with handler isolation.
- **Rule Automation Engine & Scenes**: Powerful declarative JSON DSL supporting triggers (state, numeric thresholds, time, events), condition trees (AND/OR/NOT), and action workflows with cooldown protection.
- **Hardware Protocol Adapters**: First-class support for generic MQTT, Zigbee2MQTT auto-discovery, ESPHome telemetry, and graceful cloud circuit-breakers (Tuya, Xiaomi, eWeLink, VeSync).
- **Security Hardening**: Argon2id password hashing, session cookies with SameSite/HttpOnly, CSRF protection, rate limiting, and immutable audit logging.
- **Encrypted Backup & Restore**: AES-256-GCM encrypted backup archives with PBKDF2 key derivation and path-traversal-safe restore CLI.
- **Progressive Web App (PWA)**: Vue 3 + TypeScript + Vite + Pinia single-page application with glassmorphic dark UI, real-time WebSocket live sync, and mobile installability.
- **Multi-Language Frontend**: Full internationalization (i18n) supporting **Italiano 🇮🇹, English 🇬🇧, Français 🇫🇷, Español 🇪🇸, and Deutsch 🇩🇪**.
- **Interactive OpenAPI Documentation**: Embedded OpenAPI 3.1 specification and Swagger UI accessible directly at `/docs`.
- **TDD (Test-Driven Development)**: Comprehensive unit tests, repository integration tests, and full-stack end-to-end (E2E) suites across backend and frontend.

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
│   │   ├── area/           # Room and area spatial hierarchies
│   │   ├── audit/          # Security audit log entities
│   │   ├── automation/     # JSON DSL triggers, conditions, actions, and runs
│   │   ├── device/         # Physical hardware abstractions
│   │   ├── entity/         # Typed domain entities (lights, switches, sensors, climate)
│   │   ├── events/         # Typed domain events and event bus interface
│   │   ├── history/        # Telemetry historical state models
│   │   ├── integration/    # Hardware and cloud adapter models
│   │   ├── scene/          # Multi-device preset scenes
│   │   └── user/           # User authentication and RBAC roles
│   │
│   ├── application/        # Application services and use cases
│   │   ├── auth/           # Login, wizard setup, session management, rate limiter
│   │   ├── automation/     # Rule evaluator and asynchronous execution engine
│   │   ├── backup/         # AES-256-GCM backup and idempotent restore service
│   │   ├── history/        # Batched state recorder and retention cleanup worker
│   │   └── scene/          # Scene activation dispatcher
│   │
│   ├── infrastructure/     # Implementation of technical ports
│   │   ├── config/         # TOML + env variable loader, PBKDF2/AES-GCM secrets
│   │   ├── crypto/         # Argon2id password hasher
│   │   ├── eventbus/       # Bounded in-memory event bus
│   │   ├── mqtt/           # Paho MQTT v3.1.1 client wrapper with auto-reconnect
│   │   ├── sqlite/         # SQLite driver (pure Go), WAL pragma, embedded migrations
│   │   └── websocket/      # Real-time WebSocket broadcast hub
│   │
│   ├── interfaces/         # Transport layer
│   │   ├── http/           # Chi router, security headers, CORS, handlers
│   │   │   ├── handlers/   # REST API endpoint handlers
│   │   │   ├── middleware/ # Auth, CSRF, rate limit, security headers, logger
│   │   │   └── swagger/    # Embedded OpenAPI 3.1 YAML and Swagger UI
│   │   └── websocket/      # WebSocket client connection upgrade handler
│   │
│   └── plugins/            # Protocol integration adapters
│       ├── cloud/          # Circuit breaker and cloud provider stubs
│       ├── esphome_mqtt/   # ESPHome telemetry and command mapper
│       ├── mqtt_generic/   # Topic-to-entity MQTT generic adapter
│       └── zigbee2mqtt/    # Zigbee2MQTT device discovery and status mapper
│
├── web/                    # Frontend Vue 3 + TypeScript Progressive Web App
│   ├── src/
│   │   ├── api/            # Typed REST client and models
│   │   ├── components/     # Glassmorphic entity cards, modals, charts, layout
│   │   ├── locales/        # Translations: it.json, en.json, fr.json, es.json, de.json
│   │   ├── router/         # Vue router with navigation auth guards
│   │   ├── stores/         # Pinia stores (auth, entities, devices, automations, etc.)
│   │   └── views/          # Dashboard, Areas, Devices, Automations, Scenes, History, Audit, Settings
│   └── package.json
│
├── docs/
│   ├── adr/                # Architecture Decision Records (ADR-001 to ADR-006)
│   └── openapi/            # OpenAPI 3.1 YAML REST specification
│
├── deployments/
│   ├── docker/             # Multi-stage Dockerfile and Docker Compose stack
│   ├── systemd/            # Hardened Linux systemd service unit
│   └── scripts/            # Non-interactive install and uninstall automation scripts
│
├── Makefile                # Build, test, lint, and run workflow targets
├── .golangci.yml           # Static analysis and linter configuration
└── domus.toml.example      # Reference configuration template
```

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

The user interface supports 5 languages out-of-the-box:
- 🇮🇹 **Italiano** (Predefinito)
- 🇬🇧 **English**
- 🇫🇷 **Français**
- 🇪🇸 **Español**
- 🇩🇪 **Deutsch**

Language selection is persisted to local storage and dynamically reactive across all views, controls, navigation bars, and audit logs.

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

---

## 📜 Architecture Decision Records (ADRs)

- [ADR-001: Modular Monolith Architecture with Hexagonal Ports & Adapters](docs/adr/ADR-001-modular-monolith.md)
- [ADR-002: SQLite with Write-Ahead Logging (WAL) and Pure Go Driver](docs/adr/ADR-002-sqlite-wal.md)
- [ADR-003: In-Memory Bounded Event Bus](docs/adr/ADR-003-event-bus-in-memory.md)
- [ADR-004: Structured JSON DSL for Automations and Scenes](docs/adr/ADR-004-json-dsl-automations.md)
- [ADR-005: AES-256-GCM Secrets Encryption and Idempotent Backup/Restore](docs/adr/ADR-005-secrets-encryption-and-backup.md)
- [ADR-006: Packaging, Multi-Architecture Containerization, and Offline Deployment](docs/adr/ADR-006-packaging-and-offline-deployment.md)

---

## 📄 License

Licensed under the MIT License.
