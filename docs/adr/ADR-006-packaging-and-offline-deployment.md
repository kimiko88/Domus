# ADR-006: Packaging, Multi-Architecture Containerization, and Offline Deployment

## Status
Accepted

## Date
2026-10-04

## Context
Domus targets both bare-metal single-board computers (specifically Raspberry Pi 4 running Debian/Raspberry Pi OS) and containerized edge server environments (Docker Compose, Kubernetes on k3s).
Installation must be deterministic, non-interactive, capable of functioning entirely offline, and respect modern Linux security sandboxing standards (systemd hardening, unprivileged users, strict filesystem mounts).

## Decision
We establish a dual deployment strategy supported by automated toolchains:
1. **Zero-CGO Static Binaries**:
   - Binaries are built with `CGO_ENABLED=0` using `modernc.org/sqlite` (pure Go SQLite driver).
   - This eliminates glibc/musl toolchain mismatch across Linux distributions and enables frictionless cross-compilation for `linux/arm64` and `linux/amd64`.
2. **Systemd Service Unit Hardening**:
   - A dedicated system user `domus:domus` runs the process without interactive shell.
   - Comprehensive security directives: `ProtectSystem=strict`, `ProtectHome=true`, `PrivateTmp=true`, `PrivateDevices=true`, `ProtectKernelTunables=true`, `ProtectControlGroups=true`.
   - Explicit filesystem whitelisting: `ReadWritePaths=/var/lib/domus` and `ReadOnlyPaths=/etc/domus`.
3. **Multi-Stage Container Image**:
   - Stage 1 compiles web frontend assets (`web/dist`) using Vite and TypeScript.
   - Stage 2 compiles pure Go binaries (`domus` and `domus-cli`).
   - Stage 3 provisions a minimal Alpine runtime image running as non-root user `domus` with embedded healthchecks.
4. **Embedded OpenAPI & Swagger Documentation**:
   - The OpenAPI 3.1 specification (`domus.yaml`) is embedded directly in the server binary via `go:embed`.
   - Swagger UI is served natively on `/docs` and `/swagger`, eliminating external documentation dependencies.

## Consequences
### Positive
- One-line non-interactive installation on Raspberry Pi OS via `deployments/scripts/install.sh`.
- Minimal attack surface in both bare-metal and container environments.
- Offline-ready: all assets, documentation, schemas, and database migrations are self-contained in the deployment bundle.
