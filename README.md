# Domus 🏠

> A lightweight, local-first, self-hosted smart home automation hub designed for Raspberry Pi 4.

Domus is built in modern idiomatic Go and designed as a high-performance, resource-efficient alternative to heavier home automation platforms. It runs entirely on your local network without external cloud dependencies.

---

## ⚡ Highlights

- **Local-First Core**: Automations, state evaluation, MQTT communication, and dashboards run locally without Internet access.
- **Ultra-Low Resource Footprint**: Written in pure Go with zero CGO dependencies; runs comfortably in ~20-30 MB RAM on Raspberry Pi 4 ARM64.
- **Embedded SQLite in WAL Mode**: Fast concurrent telemetry reads and atomic writes with foreign key safety and zero external database services.
- **In-Memory Event Bus**: Decoupled, typed domain events dispatched asynchronously through bounded worker pools with handler isolation.
- **Clean Architecture**: Modular monolith organized around Hexagonal Ports & Adapters.
- **TDD (Test-Driven Development)**: Comprehensive unit tests, integration tests, and full stack end-to-end (E2E) suites.

---

## 📁 Repository Structure

```
domus/
├── cmd/
│   ├── domus/              # Main hub entrypoint (server, eventbus, db wiring)
│   └── domus-cli/          # CLI utility for administration and migrations
│
├── internal/
│   ├── domain/             # Pure business models and port interfaces (zero dependencies)
│   │   └── events/         # Typed domain events and event bus interface
│   ├── infrastructure/     # Implementations of technical ports
│   │   ├── config/         # Multi-tiered TOML + environment variable loader
│   │   ├── eventbus/       # Bounded in-memory event bus
│   │   └── sqlite/         # SQLite driver (pure Go), PRAGMAs, embedded migrations
│   └── interfaces/         # Transport layer
│       └── http/           # Chi HTTP router, slog request logger, health handler
│
├── test/
│   └── e2e/                # End-to-end integration test suite
│
├── docs/
│   ├── adr/                # Architecture Decision Records
│   └── openapi/            # OpenAPI 3.1 REST API specification
│
├── deployments/
│   ├── docker/             # Multi-stage Dockerfile and Docker Compose dev stack
│   └── systemd/            # Systemd service unit for native Raspberry Pi installation
│
├── Makefile                # Build, test, lint, and run workflow targets
├── .golangci.yml           # Static analysis and linter configuration
└── domus.toml.example      # Reference configuration template
```

---

## 🚀 Getting Started

### Prerequisites

- **Go**: 1.23 or newer
- **Make** (optional, recommended)
- **Docker & Docker Compose** (optional, for containerized run)

### Running Locally

```bash
# 1. Clone repository
git clone https://github.com/domus-hub/domus.git
cd domus

# 2. Run test suite (unit + integration + e2e)
make test

# 3. Start development server
make run
```

The hub starts on `http://localhost:8080`. You can verify status:

```bash
curl http://localhost:8080/health
```

Output:
```json
{
  "database": "healthy",
  "status": "ok",
  "uptime_s": 3,
  "version": "1.0.0-dev"
}
```

---

## 🧪 Testing

Following strict **Test-Driven Development (TDD)**, the codebase includes:

- **Unit tests**: Configuration parsing, event bus concurrency, subscriber routing, unsubscription idempotency.
- **Integration tests**: SQLite connection initialization, PRAGMA configuration, embedded database migrations, HTTP router and middlewares.
- **E2E tests**: Full-stack application lifecycle validation from startup to shutdown.

```bash
# Run all tests
make test

# Run tests with coverage report
make test-cover

# Run static analysis linters
make lint
```

---

## 🐳 Docker & Deployments

### Development with Docker Compose

Spin up Domus along with an Eclipse Mosquitto MQTT broker:

```bash
docker compose -f deployments/docker/docker-compose.yml up --build
```

### Native Raspberry Pi (systemd)

Copy the compiled binary to `/usr/local/bin/domus`, copy `domus.toml.example` to `/etc/domus/domus.toml`, and enable the service:

```bash
sudo cp deployments/systemd/domus.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now domus
```

---

## 🗺️ Roadmap & Milestones

- [x] **MS-01: Foundation & Skeleton** (Repo structure, pure Go SQLite WAL, event bus, config loader, HTTP server, E2E test, Docker/Systemd)
- [x] **MS-02: Auth, RBAC & First API** (Argon2id, session cookies, CSRF, admin wizard, login/logout, audit log)
- [x] **MS-03: MQTT Generic & Entity Model** (MQTT adapter, device/entity domain repositories, WebSocket live feed, REST commands)
- [x] **MS-04: Zigbee2MQTT & ESPHome** (Zigbee2MQTT auto-discovery & mapper, ESPHome MQTT mapper, Cloud resilience circuit breaker, E2E suite)
- [x] **MS-05: Automation Engine v1** (JSON DSL, triggers, conditions, actions, execution runner, scenes, E2E suite)
- [ ] **MS-06: Frontend Core** (Vue 3, Vite, TypeScript, Pinia, PWA, dynamic dashboard)
- [ ] **MS-07: Frontend Automations & Scenes** (Visual automation builder, scene toggles)
- [ ] **MS-08: Telemetry History & Metrics** (State change retention, Chart.js time-series)
- [ ] **MS-09: Hardening & Backup/Restore** (Encrypted backups, secrets management)
- [ ] **MS-10: Final ARM64 Packaging & Release** (Production images, automated installer)

---

## 📜 License

Licensed under the MIT License.
