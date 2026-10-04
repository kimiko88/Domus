#!/usr/bin/env bash
# ==============================================================================
# Domus — Automated Non-Interactive Installation Script
# Targets: Linux ARM64 (Raspberry Pi 4 / Debian / Ubuntu / Alpine) & x86_64
# ==============================================================================
set -euo pipefail

# Configuration parameters with sensible defaults (can be overridden via ENV)
DOMUS_VERSION="${DOMUS_VERSION:-1.0.0}"
DOMUS_INSTALL_DIR="${DOMUS_INSTALL_DIR:-/usr/local/bin}"
DOMUS_CONFIG_DIR="${DOMUS_CONFIG_DIR:-/etc/domus}"
DOMUS_DATA_DIR="${DOMUS_DATA_DIR:-/var/lib/domus}"
DOMUS_USER="${DOMUS_USER:-domus}"
DOMUS_GROUP="${DOMUS_GROUP:-domus}"
DOMUS_PORT="${DOMUS_PORT:-8080}"
NON_INTERACTIVE="${NON_INTERACTIVE:-1}"

# Text formatting
BOLD='\033[1m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${CYAN}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

# 1. Verify Root Privileges
if [[ "$(id -u)" -ne 0 ]]; then
    log_error "This installer must be run as root (or via sudo)."
    exit 1
fi

echo -e "${BOLD}===================================================================${NC}"
echo -e "${BOLD}     Domus — Local-First Smart Home Automation Hub Installer       ${NC}"
echo -e "${BOLD}===================================================================${NC}"
log_info "Installer target version : ${DOMUS_VERSION}"
log_info "Binary directory         : ${DOMUS_INSTALL_DIR}"
log_info "Configuration directory  : ${DOMUS_CONFIG_DIR}"
log_info "Data & DB directory      : ${DOMUS_DATA_DIR}"

# 2. Architecture Detection
ARCH="$(uname -m)"
case "${ARCH}" in
    x86_64|amd64)
        TARGET_ARCH="amd64"
        ;;
    aarch64|arm64)
        TARGET_ARCH="arm64"
        ;;
    armv7l|armhf)
        TARGET_ARCH="armv7"
        ;;
    *)
        log_error "Unsupported hardware architecture: ${ARCH}"
        exit 1
        ;;
esac
log_info "Detected system architecture: ${ARCH} (${TARGET_ARCH})"

# 3. Create Dedicated System User and Group
if ! getent group "${DOMUS_GROUP}" >/dev/null 2>&1; then
    log_info "Creating system group '${DOMUS_GROUP}'..."
    groupadd --system "${DOMUS_GROUP}"
fi

if ! id -u "${DOMUS_USER}" >/dev/null 2>&1; then
    log_info "Creating system user '${DOMUS_USER}'..."
    useradd --system \
        --gid "${DOMUS_GROUP}" \
        --home-dir "${DOMUS_DATA_DIR}" \
        --shell /usr/sbin/nologin \
        --comment "Domus Smart Home Hub" \
        "${DOMUS_USER}"
fi

# 4. Prepare Directory Tree & Set Permissions
log_info "Configuring directory permissions..."
mkdir -p "${DOMUS_CONFIG_DIR}" "${DOMUS_DATA_DIR}" "${DOMUS_INSTALL_DIR}"

chown -R "${DOMUS_USER}:${DOMUS_GROUP}" "${DOMUS_DATA_DIR}" "${DOMUS_CONFIG_DIR}"
chmod 0750 "${DOMUS_DATA_DIR}"
chmod 0755 "${DOMUS_CONFIG_DIR}"

# 5. Install Binaries
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

if [[ -f "${PROJECT_ROOT}/domus" && -f "${PROJECT_ROOT}/domus-cli" ]]; then
    log_info "Installing pre-built binaries from project tree..."
    install -m 0755 "${PROJECT_ROOT}/domus" "${DOMUS_INSTALL_DIR}/domus"
    install -m 0755 "${PROJECT_ROOT}/domus-cli" "${DOMUS_INSTALL_DIR}/domus-cli"
elif command -v go >/dev/null 2>&1 && [[ -f "${PROJECT_ROOT}/go.mod" ]]; then
    log_info "Go toolchain detected. Compiling release binaries locally for ${TARGET_ARCH}..."
    (
        cd "${PROJECT_ROOT}"
        CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X 'main.Version=${DOMUS_VERSION}'" -o "${DOMUS_INSTALL_DIR}/domus" ./cmd/domus
        CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X 'main.Version=${DOMUS_VERSION}'" -o "${DOMUS_INSTALL_DIR}/domus-cli" ./cmd/domus-cli
    )
else
    log_warn "No local binaries or Go toolchain found in current directory."
    log_info "Attempting to locate binaries in PATH or /usr/local/bin..."
    if [[ ! -x "${DOMUS_INSTALL_DIR}/domus" ]]; then
        log_error "Binary '${DOMUS_INSTALL_DIR}/domus' not found. Please compile or download release binaries before running installer."
        exit 1
    fi
fi

# 6. Install Default Configuration
CONFIG_TARGET="${DOMUS_CONFIG_DIR}/domus.toml"
if [[ ! -f "${CONFIG_TARGET}" ]]; then
    if [[ -f "${PROJECT_ROOT}/domus.toml.example" ]]; then
        log_info "Installing reference configuration template..."
        cp "${PROJECT_ROOT}/domus.toml.example" "${CONFIG_TARGET}"
        chown "${DOMUS_USER}:${DOMUS_GROUP}" "${CONFIG_TARGET}"
        chmod 0640 "${CONFIG_TARGET}"
    else
        log_warn "domus.toml.example not found. Creating minimal default configuration..."
        cat <<EOF > "${CONFIG_TARGET}"
[server]
host = "0.0.0.0"
port = ${DOMUS_PORT}
cookie_secure = false

[database]
path = "${DOMUS_DATA_DIR}/domus.db"
wal_mode = true
busy_timeout_ms = 5000

[logging]
level = "info"
format = "text"
EOF
        chown "${DOMUS_USER}:${DOMUS_GROUP}" "${CONFIG_TARGET}"
        chmod 0640 "${CONFIG_TARGET}"
    fi
else
    log_info "Existing configuration preserved at ${CONFIG_TARGET}"
fi

# 7. Install Systemd Service Unit
SYSTEMD_UNIT="/etc/systemd/system/domus.service"
if command -v systemctl >/dev/null 2>&1; then
    log_info "Configuring systemd service..."
    if [[ -f "${PROJECT_ROOT}/deployments/systemd/domus.service" ]]; then
        cp "${PROJECT_ROOT}/deployments/systemd/domus.service" "${SYSTEMD_UNIT}"
    else
        cat <<EOF > "${SYSTEMD_UNIT}"
[Unit]
Description=Domus Local-First Smart Home Hub
Documentation=https://github.com/domus-hub/domus
After=network.target mosquitto.service
Wants=mosquitto.service

[Service]
Type=simple
User=${DOMUS_USER}
Group=${DOMUS_GROUP}
WorkingDirectory=${DOMUS_DATA_DIR}
ExecStart=${DOMUS_INSTALL_DIR}/domus -config ${CONFIG_TARGET}
Restart=always
RestartSec=5s

# Security Sandboxing
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
PrivateDevices=true
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true
ReadWritePaths=${DOMUS_DATA_DIR}
ReadOnlyPaths=${DOMUS_CONFIG_DIR}

StandardOutput=journal
StandardError=journal
SyslogIdentifier=domus

[Install]
WantedBy=multi-user.target
EOF
    fi

    chmod 0644 "${SYSTEMD_UNIT}"
    systemctl daemon-reload
    systemctl enable domus.service
    log_info "Starting domus service..."
    systemctl restart domus.service

    # 8. Health Check Verification
    log_info "Verifying hub health endpoint..."
    HEALTHY=0
    for i in {1..10}; do
        if curl -s -f "http://127.0.0.1:${DOMUS_PORT}/health" >/dev/null 2>&1; then
            HEALTHY=1
            break
        fi
        sleep 1
    done

    if [[ "${HEALTHY}" -eq 1 ]]; then
        log_success "Domus hub is healthy and actively serving requests!"
    else
        log_warn "Hub did not respond to health check within 10 seconds. Check logs with: journalctl -u domus -f"
    fi
else
    log_warn "systemd not detected on this system. Service unit was not installed."
    log_info "You can start Domus manually via: ${DOMUS_INSTALL_DIR}/domus -config ${CONFIG_TARGET}"
fi

echo ""
echo -e "${GREEN}${BOLD}===================================================================${NC}"
echo -e "${GREEN}${BOLD}       Domus Installation Completed Successfully! 🎉              ${NC}"
echo -e "${GREEN}${BOLD}===================================================================${NC}"
echo -e "Web Interface     : ${CYAN}http://localhost:${DOMUS_PORT}${NC}"
echo -e "Interactive Docs  : ${CYAN}http://localhost:${DOMUS_PORT}/docs${NC}"
echo -e "Health Check      : ${CYAN}http://localhost:${DOMUS_PORT}/health${NC}"
echo -e "Service Status    : ${BOLD}systemctl status domus${NC}"
echo -e "Service Logs      : ${BOLD}journalctl -u domus -f${NC}"
echo -e "Administration CLI: ${BOLD}domus-cli help${NC}"
echo ""
