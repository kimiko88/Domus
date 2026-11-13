#!/usr/bin/env bash
# ==============================================================================
# Domus — Installer (backend + frontend) for Raspberry Pi OS / Debian / Ubuntu
# Targets: linux/arm64 (Pi 3/4/5, 64-bit OS), linux/armv7 (32-bit OS), linux/amd64
#
# Usage (from a clone of the repo):
#   sudo ./deployments/scripts/install.sh
#
# What it does:
#   1. Gets the binaries:  prebuilt (./domus, ./domus-cli)  ->  or builds them
#   2. Gets the frontend:  prebuilt (./web/dist)            ->  or builds it
#      Missing Go / Node toolchains are downloaded into a temporary directory
#      and removed afterwards, so the Pi stays clean.
#   3. Installs binaries, web assets, config, systemd unit; starts the service.
#
# Tip: on a Pi with < 2 GB RAM, build on your PC and copy the results:
#   GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o domus ./cmd/domus
#   GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o domus-cli ./cmd/domus-cli
#   (cd web && npm ci && npm run build)
# then run this script on the Pi: it will just install what it finds.
#
# Environment overrides:
#   DOMUS_VERSION, DOMUS_HOST, DOMUS_PORT, DOMUS_COOKIE_SECURE, DOMUS_USER, DOMUS_GROUP
#   DOMUS_INSTALL_DIR (/usr/local/bin)   DOMUS_CONFIG_DIR (/etc/domus)
#   DOMUS_DATA_DIR (/var/lib/domus)      DOMUS_WEB_DIR (/usr/local/share/domus/web)
#   GO_VERSION (default: read from go.mod)   GO_INSTALL_DIR (/opt/domus-toolchains)   NODE_MAJOR (default: 22)
#   INSTALL_MOSQUITTO=1   install the Mosquitto MQTT broker via apt
#   KEEP_TOOLCHAINS=1     keep downloaded Node under /opt/domus-toolchains
# ==============================================================================
set -euo pipefail

DOMUS_INSTALL_DIR="${DOMUS_INSTALL_DIR:-/usr/local/bin}"
DOMUS_CONFIG_DIR="${DOMUS_CONFIG_DIR:-/etc/domus}"
DOMUS_DATA_DIR="${DOMUS_DATA_DIR:-/var/lib/domus}"
DOMUS_WEB_DIR="${DOMUS_WEB_DIR:-/usr/local/share/domus/web}"
DOMUS_USER="${DOMUS_USER:-domus}"
DOMUS_GROUP="${DOMUS_GROUP:-domus}"
DOMUS_HOST="${DOMUS_HOST:-0.0.0.0}"
DOMUS_PORT="${DOMUS_PORT:-8080}"
DOMUS_COOKIE_SECURE_EXPLICIT="${DOMUS_COOKIE_SECURE+x}"
DOMUS_COOKIE_SECURE="${DOMUS_COOKIE_SECURE:-false}"
if [[ "${DOMUS_COOKIE_SECURE}" != "true" && "${DOMUS_COOKIE_SECURE}" != "false" ]]; then
    printf '%s\n' 'DOMUS_COOKIE_SECURE must be true or false' >&2
    exit 1
fi
NODE_MAJOR="${NODE_MAJOR:-22}"
INSTALL_MOSQUITTO="${INSTALL_MOSQUITTO:-0}"
KEEP_TOOLCHAINS="${KEEP_TOOLCHAINS:-0}"
GO_INSTALL_DIR="${GO_INSTALL_DIR:-/opt/domus-toolchains}"

BOLD=$'\033[1m'; GREEN=$'\033[0;32m'; CYAN=$'\033[0;36m'
YELLOW=$'\033[1;33m'; RED=$'\033[0;31m'; NC=$'\033[0m'
log_info()    { echo -e "${CYAN}[INFO]${NC} $*"; }
log_success() { echo -e "${GREEN}[ OK ]${NC} $*"; }
log_warn()    { echo -e "${YELLOW}[WARN]${NC} $*"; }
log_error()   { echo -e "${RED}[FAIL]${NC} $*" >&2; }

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
WORK_DIR=""

cleanup() {
    local rc=$?
    if [[ -n "${WORK_DIR}" && -d "${WORK_DIR}" ]]; then
        rm -rf "${WORK_DIR}"
    fi
    if [[ ${rc} -ne 0 ]]; then
        log_error "Installation failed (exit code ${rc}). Check the messages above and the service logs."
    fi
}
trap cleanup EXIT

# ── 0. Preconditions ───────────────────────────────────────────────────────────
if [[ "$(id -u)" -ne 0 ]]; then
    log_error "Run as root:  sudo $0"
    exit 1
fi
if [[ ! -f "${PROJECT_ROOT}/go.mod" ]]; then
    log_error "go.mod not found in ${PROJECT_ROOT}. Run the script from a full clone of the repository."
    exit 1
fi

echo -e "${BOLD}=== Domus installer ===${NC}"

# Version: env > git tag > go.mod-independent fallback
if [[ -z "${DOMUS_VERSION:-}" ]]; then
    DOMUS_VERSION="$(git -c safe.directory='*' -C "${PROJECT_ROOT}" describe --tags --always --dirty 2>/dev/null || echo "1.0.0")"
fi
BUILD_TIME="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

# ── 1. Architecture / board detection ──────────────────────────────────────────
case "$(uname -m)" in
    x86_64|amd64)  GOARCH_T="amd64"; GO_DL_ARCH="amd64";  NODE_ARCH="x64"   ;;
    aarch64|arm64) GOARCH_T="arm64"; GO_DL_ARCH="arm64";  NODE_ARCH="arm64" ;;
    armv7l|armhf)  GOARCH_T="arm";   GO_DL_ARCH="armv6l"; NODE_ARCH="armv7l" ;;
    *) log_error "Unsupported architecture: $(uname -m)"; exit 1 ;;
esac
BOARD=""
[[ -r /proc/device-tree/model ]] && BOARD="$(tr -d '\0' < /proc/device-tree/model)"
log_info "Architecture : $(uname -m) (${GOARCH_T})${BOARD:+ — ${BOARD}}"
log_info "Version      : ${DOMUS_VERSION}"

MEM_MB=$(awk '/MemTotal/ {printf "%d", $2/1024}' /proc/meminfo)
if (( MEM_MB < 1500 )); then
    log_warn "Only ${MEM_MB} MB RAM: compiling here may fail. Consider building on a PC (see header of this script)."
fi

# ── 2. System packages ─────────────────────────────────────────────────────────
need_pkgs=()
for bin in curl tar xz git; do
    command -v "${bin}" >/dev/null 2>&1 || need_pkgs+=("$([[ ${bin} == xz ]] && echo xz-utils || echo "${bin}")")
done
[[ -f /etc/ssl/certs/ca-certificates.crt ]] || need_pkgs+=(ca-certificates)
if [[ "${INSTALL_MOSQUITTO}" == "1" ]]; then
    command -v mosquitto >/dev/null 2>&1 || need_pkgs+=(mosquitto)
fi

# Receipt OCR uses Poppler for PDF text/raster extraction and Tesseract for
# scanned PDFs and camera images. Keep the runtime dependencies aligned with
# the Docker image so the feature also works on native Raspberry Pi installs.
if ! command -v pdftotext >/dev/null 2>&1 || ! command -v pdftoppm >/dev/null 2>&1 || ! command -v pdfinfo >/dev/null 2>&1; then
    need_pkgs+=(poppler-utils)
fi
if ! command -v tesseract >/dev/null 2>&1; then
    need_pkgs+=(tesseract-ocr tesseract-ocr-ita tesseract-ocr-eng)
else
    if ! tesseract --list-langs 2>/dev/null | grep -qx ita; then
        need_pkgs+=(tesseract-ocr-ita)
    fi
    if ! tesseract --list-langs 2>/dev/null | grep -qx eng; then
        need_pkgs+=(tesseract-ocr-eng)
    fi
fi
if (( ${#need_pkgs[@]} > 0 )); then
    if command -v apt-get >/dev/null 2>&1; then
        log_info "Installing packages: ${need_pkgs[*]}"
        DEBIAN_FRONTEND=noninteractive apt-get update -qq
        DEBIAN_FRONTEND=noninteractive apt-get install -y -qq --no-install-recommends "${need_pkgs[@]}"
    else
        log_error "Missing tools (${need_pkgs[*]}) and apt-get not available. Install them manually."
        exit 1
    fi
fi

# ── helpers ────────────────────────────────────────────────────────────────────
ensure_workdir() {
    if [[ -z "${WORK_DIR}" ]]; then
        # /var/tmp = disk, not RAM-backed tmpfs: toolchains + caches are large.
        WORK_DIR="$(mktemp -d /var/tmp/domus-build.XXXXXX)"
    fi
}

sha256_check() { # <file> <expected-hash>
    local got; got="$(sha256sum "$1" | awk '{print $1}')"
    if [[ "${got}" != "$2" ]]; then
        log_error "Checksum mismatch for $(basename "$1")"; exit 1
    fi
}

# Prints the toolchain directory. Callers must run ensure_workdir first
# (it cannot be called here: $(...) would lose the WORK_DIR assignment).
toolchain_dir() {
    local d="${WORK_DIR}/toolchains"
    [[ "${KEEP_TOOLCHAINS}" == "1" ]] && d="/opt/domus-toolchains"
    mkdir -p "${d}"
    echo "${d}"
}

install_go() {
    local want="${GO_VERSION:-$(awk '/^go[[:space:]]/ {print $2; exit}' "${PROJECT_ROOT}/go.mod")}"
    [[ "${want}" =~ ^[0-9]+\.[0-9]+(\.[0-9]+)?$ ]] || { log_error "Cannot determine Go version from go.mod"; exit 1; }
    # Go release archives use a patch version (e.g. 1.26.0).
    [[ "${want}" =~ ^[0-9]+\.[0-9]+$ ]] && want="${want}.0"

    local go_root="${GO_INSTALL_DIR}/go-${want}"
    if [[ -x "${go_root}/bin/go" ]]; then
        local cached_version
        cached_version="$("${go_root}/bin/go" env GOVERSION 2>/dev/null | sed 's/^go//')"
        if [[ -n "${cached_version}" ]] && ver_ge "${cached_version}" "${want}"; then
            log_info "Using previously downloaded Go ${cached_version} from ${go_root}"
            export PATH="${go_root}/bin:${PATH}"
            return
        fi
        log_warn "Cached Go at ${go_root} is unusable or too old; replacing it"
        rm -rf "${go_root}"
    fi

    ensure_workdir
    local name="go${want}.linux-${GO_DL_ARCH}.tar.gz"
    log_info "Downloading Go ${want} (${GO_DL_ARCH}) into ${GO_INSTALL_DIR}..."
    mkdir -p "${GO_INSTALL_DIR}"
    curl -fsSL "https://dl.google.com/go/${name}" -o "${WORK_DIR}/${name}"
    local sum; sum="$(curl -fsSL "https://dl.google.com/go/${name}.sha256" | awk '{print $1}')"
    sha256_check "${WORK_DIR}/${name}" "${sum}"

    local unpack_dir="${WORK_DIR}/go-${want}"
    mkdir -p "${unpack_dir}"
    tar -C "${unpack_dir}" --strip-components=1 -xzf "${WORK_DIR}/${name}"
    mkdir -p "${go_root}"
    cp -a "${unpack_dir}/." "${go_root}/"
    rm -f "${WORK_DIR}/${name}"
    export PATH="${go_root}/bin:${PATH}"
}

install_node() {
    ensure_workdir
    local tdir; tdir="$(toolchain_dir)"
    local base="https://nodejs.org/dist/latest-v${NODE_MAJOR}.x"
    log_info "Resolving Node.js ${NODE_MAJOR}.x (${NODE_ARCH})..."
    curl -fsSL "${base}/SHASUMS256.txt" -o "${WORK_DIR}/SHASUMS256.txt"
    local line file sum
    line="$(grep -E "node-v[0-9.]+-linux-${NODE_ARCH}\.tar\.xz$" "${WORK_DIR}/SHASUMS256.txt" | head -1)"
    [[ -n "${line}" ]] || { log_error "No Node ${NODE_MAJOR}.x build for linux-${NODE_ARCH}"; exit 1; }
    sum="${line%% *}"; file="${line##* }"
    if [[ ! -x "${tdir}/${file%.tar.xz}/bin/node" ]]; then
        log_info "Downloading ${file}..."
        curl -fsSL "${base}/${file}" -o "${WORK_DIR}/${file}"
        sha256_check "${WORK_DIR}/${file}" "${sum}"
        tar -C "${tdir}" -xJf "${WORK_DIR}/${file}"
        rm -f "${WORK_DIR}/${file}"
    fi
    export PATH="${tdir}/${file%.tar.xz}/bin:${PATH}"
}

# ── 3. Backend binaries ────────────────────────────────────────────────────────
STAGE_BIN=""
if [[ -f "${PROJECT_ROOT}/domus" && -f "${PROJECT_ROOT}/domus-cli" ]]; then
    log_info "Using prebuilt binaries from ${PROJECT_ROOT}"
    STAGE_BIN="${PROJECT_ROOT}"
else
    ensure_workdir
    mkdir -p "${WORK_DIR}/bin"
    # Prefer an already installed Go (including the standard /usr/local/go location).
    # Only download a toolchain when no usable installation meets go.mod.
    need_go="${GO_VERSION:-$(awk '/^go[[:space:]]/ {print $2; exit}' "${PROJECT_ROOT}/go.mod")}"
    [[ "${need_go}" =~ ^[0-9]+\.[0-9]+$ ]] && need_go="${need_go}.0"
    ver_ge() { [[ "$(printf '%s\n%s\n' "$2" "$1" | sort -V | head -1)" == "$2" ]]; } # $1 >= $2

    if ! command -v go >/dev/null 2>&1 && [[ -x /usr/local/go/bin/go ]]; then
        export PATH="/usr/local/go/bin:${PATH}"
    fi

    have_go=""
    if command -v go >/dev/null 2>&1; then
        have_go="$(go env GOVERSION 2>/dev/null | sed 's/^go//')"
    fi
    if [[ -n "${have_go}" ]] && ver_ge "${have_go}" "${need_go}"; then
        log_success "Using installed Go ${have_go}; no Go download or reinstall needed"
    else
        [[ -n "${have_go}" ]] && log_warn "Installed Go ${have_go} is older than required ${need_go}"
        install_go
    fi
    # Caches live in the temp dir (no SD-card clutter); GOTOOLCHAIN=local = never auto-download another Go.
    export GOCACHE="${WORK_DIR}/gocache" GOPATH="${WORK_DIR}/gopath" GOTOOLCHAIN=local
    log_info "Compiling backend ($(go version | awk '{print $3}'))..."
    (
        cd "${PROJECT_ROOT}"
        export CGO_ENABLED=0 GOOS=linux GOARCH="${GOARCH_T}"
        [[ "${GOARCH_T}" == "arm" ]] && export GOARM=7
        LD="-s -w -X main.Version=${DOMUS_VERSION} -X main.BuildTime=${BUILD_TIME}"
        go build -trimpath -ldflags="${LD}" -o "${WORK_DIR}/bin/domus"     ./cmd/domus
        go build -trimpath -ldflags="${LD}" -o "${WORK_DIR}/bin/domus-cli" ./cmd/domus-cli
    )
    STAGE_BIN="${WORK_DIR}/bin"
fi

# ── 4. Frontend ────────────────────────────────────────────────────────────────
STAGE_WEB=""
if [[ -f "${PROJECT_ROOT}/web/dist/index.html" ]]; then
    log_info "Using prebuilt frontend from web/dist"
    STAGE_WEB="${PROJECT_ROOT}/web/dist"
else
    ensure_workdir
    node_ok=0
    if command -v node >/dev/null 2>&1; then
        nmaj="$(node -p 'process.versions.node.split(".")[0]')"
        (( nmaj == 20 || nmaj >= 22 )) && node_ok=1
    fi
    (( node_ok )) || install_node
    log_info "Building frontend (Node $(node -v))..."
    # Build in a copy so the repo stays clean and root never owns node_modules in it.
    mkdir -p "${WORK_DIR}/web"
    tar -C "${PROJECT_ROOT}/web" --exclude=node_modules --exclude=dist -cf - . | tar -C "${WORK_DIR}/web" -xf -
    (
        cd "${WORK_DIR}/web"
        export npm_config_cache="${WORK_DIR}/npm-cache" npm_config_update_notifier=false
        npm ci --ignore-scripts --no-audit --no-fund
        npm run build
    )
    [[ -f "${WORK_DIR}/web/dist/index.html" ]] || { log_error "Frontend build produced no dist/index.html"; exit 1; }
    STAGE_WEB="${WORK_DIR}/web/dist"
fi

# ── 5. User, directories ───────────────────────────────────────────────────────
getent group "${DOMUS_GROUP}" >/dev/null 2>&1 || groupadd --system "${DOMUS_GROUP}"
if ! id -u "${DOMUS_USER}" >/dev/null 2>&1; then
    log_info "Creating system user '${DOMUS_USER}'"
    useradd --system --gid "${DOMUS_GROUP}" --home-dir "${DOMUS_DATA_DIR}" \
            --shell /usr/sbin/nologin --comment "Domus Smart Home Hub" "${DOMUS_USER}"
fi
mkdir -p "${DOMUS_CONFIG_DIR}" "${DOMUS_DATA_DIR}" "${DOMUS_INSTALL_DIR}"
chown "${DOMUS_USER}:${DOMUS_GROUP}" "${DOMUS_DATA_DIR}"
chmod 0750 "${DOMUS_DATA_DIR}"
chown root:"${DOMUS_GROUP}" "${DOMUS_CONFIG_DIR}"
chmod 0750 "${DOMUS_CONFIG_DIR}"

# ── 6. Stop service (if upgrading), install files ──────────────────────────────
HAVE_SYSTEMD=0
if command -v systemctl >/dev/null 2>&1 && [[ -d /run/systemd/system ]]; then HAVE_SYSTEMD=1; fi
if (( HAVE_SYSTEMD )) && systemctl is-active --quiet domus.service 2>/dev/null; then
    log_info "Stopping running domus service for upgrade"
    systemctl stop domus.service
fi

install -m 0755 "${STAGE_BIN}/domus"     "${DOMUS_INSTALL_DIR}/domus"
install -m 0755 "${STAGE_BIN}/domus-cli" "${DOMUS_INSTALL_DIR}/domus-cli"
log_success "Binaries installed in ${DOMUS_INSTALL_DIR}"

# Frontend: copy next to the target, then swap (never leaves a half-written UI)
mkdir -p "$(dirname "${DOMUS_WEB_DIR}")"
rm -rf "${DOMUS_WEB_DIR}.new"
cp -a "${STAGE_WEB}" "${DOMUS_WEB_DIR}.new"
rm -rf "${DOMUS_WEB_DIR}.old"
[[ -d "${DOMUS_WEB_DIR}" ]] && mv "${DOMUS_WEB_DIR}" "${DOMUS_WEB_DIR}.old"
mv "${DOMUS_WEB_DIR}.new" "${DOMUS_WEB_DIR}"
rm -rf "${DOMUS_WEB_DIR}.old"
chown -R root:root "${DOMUS_WEB_DIR}"
find "${DOMUS_WEB_DIR}" -type d -exec chmod 0755 {} + ; find "${DOMUS_WEB_DIR}" -type f -exec chmod 0644 {} +
log_success "Frontend installed in ${DOMUS_WEB_DIR}"

# ── 7. Configuration (never overwrite an existing one) ─────────────────────────
CONFIG_TARGET="${DOMUS_CONFIG_DIR}/domus.toml"
if [[ ! -f "${CONFIG_TARGET}" ]]; then
    if [[ -f "${PROJECT_ROOT}/domus.toml.example" ]]; then
        cp "${PROJECT_ROOT}/domus.toml.example" "${CONFIG_TARGET}"
        # Absolute paths + chosen port instead of the dev defaults
        sed -i -E \
            -e "s|^[[:space:]]*host[[:space:]]*=.*|host = \"${DOMUS_HOST}\"|" \
            -e "s|^port[[:space:]]*=.*|port = ${DOMUS_PORT}|" \
            -e "s|^cookie_secure[[:space:]]*=.*|cookie_secure = ${DOMUS_COOKIE_SECURE}|" \
            -e "s|^path[[:space:]]*=.*|path = \"${DOMUS_DATA_DIR}/domus.db\"|" \
            -e "s|^web_dir[[:space:]]*=.*|web_dir = \"${DOMUS_WEB_DIR}\"|" \
            "${CONFIG_TARGET}"
        # Older templates have no web_dir line: add it under [server]
        grep -q '^web_dir' "${CONFIG_TARGET}" || \
            sed -i "/^\[server\]/a web_dir = \"${DOMUS_WEB_DIR}\"" "${CONFIG_TARGET}"
    else
        cat > "${CONFIG_TARGET}" <<EOF
[server]
host = "${DOMUS_HOST}"
port = ${DOMUS_PORT}
cookie_secure = ${DOMUS_COOKIE_SECURE}
web_dir = "${DOMUS_WEB_DIR}"

[database]
path = "${DOMUS_DATA_DIR}/domus.db"
wal_mode = true
busy_timeout_ms = 5000

[logging]
level = "info"
format = "text"
EOF
    fi
    chown root:"${DOMUS_GROUP}" "${CONFIG_TARGET}"; chmod 0640 "${CONFIG_TARGET}"
    log_success "Created ${CONFIG_TARGET}"
else
    log_info "Existing configuration preserved: ${CONFIG_TARGET}"
    # An explicit DOMUS_COOKIE_SECURE override is intentional and may update
    # this one security setting without disturbing other existing preferences.
    if [[ "${DOMUS_COOKIE_SECURE_EXPLICIT}" == "x" ]]; then
        if grep -Eq '^[[:space:]]*cookie_secure[[:space:]]*=' "${CONFIG_TARGET}"; then
            sed -i -E "s|^[[:space:]]*cookie_secure[[:space:]]*=.*|cookie_secure = ${DOMUS_COOKIE_SECURE}|" "${CONFIG_TARGET}"
        elif grep -q '^\[server\]' "${CONFIG_TARGET}"; then
            sed -i "/^\[server\]/a cookie_secure = ${DOMUS_COOKIE_SECURE}" "${CONFIG_TARGET}"
        else
            log_error "Existing config has no [server] section; cannot apply DOMUS_COOKIE_SECURE safely"
            exit 1
        fi
        chown root:"${DOMUS_GROUP}" "${CONFIG_TARGET}"
        chmod 0640 "${CONFIG_TARGET}"
    fi
    # Upgrade missing installer-managed paths without overwriting user settings.
    if ! grep -Eq '^[[:space:]]*web_dir[[:space:]]*=' "${CONFIG_TARGET}"; then
        if grep -q '^\[server\]' "${CONFIG_TARGET}"; then
            sed -i "/^\[server\]/a web_dir = \"${DOMUS_WEB_DIR}\"" "${CONFIG_TARGET}"
            log_success "Added missing server.web_dir to existing configuration"
        else
            log_error "Existing config has no [server] section; cannot configure web_dir safely"
            exit 1
        fi
        chown root:"${DOMUS_GROUP}" "${CONFIG_TARGET}"
        chmod 0640 "${CONFIG_TARGET}"
    fi
    # Use the port and host the user actually configured for the health check.
    cfg_port="$(awk -F= '/^[[:space:]]*port[[:space:]]*=/ {gsub(/[[:space:]]/,"",$2); print $2; exit}' "${CONFIG_TARGET}")"
    [[ "${cfg_port}" =~ ^[0-9]+$ ]] && DOMUS_PORT="${cfg_port}"
    cfg_host="$(awk -F= '/^[[:space:]]*host[[:space:]]*=/ {gsub(/[[:space:]"]/,"",$2); print $2; exit}' "${CONFIG_TARGET}")"
    if [[ "${cfg_host}" == "127.0.0.1" || "${cfg_host}" == "localhost" ]]; then
        log_warn "Server is bound to ${cfg_host}; LAN access requires host = \"0.0.0.0\" in ${CONFIG_TARGET}"
    fi
fi

# Optional secrets and environment overrides. Preserve an existing file.
ENV_FILE="${DOMUS_CONFIG_DIR}/domus.env"
if [[ ! -f "${ENV_FILE}" ]]; then
    cat > "${ENV_FILE}" <<'EOF'
# Optional runtime secrets. Remove the leading # and set strong random values.
# DOMUS_CLOUD_ASSISTANT_TOKEN=
# DOMUS_FAILOVER_HEARTBEAT_TOKEN=
#
# Optional geofence overrides (normally configure these in domus.toml):
# DOMUS_PRESENCE_HOME_LATITUDE=
# DOMUS_PRESENCE_HOME_LONGITUDE=
# DOMUS_PRESENCE_GEOFENCE_RADIUS_M=
# DOMUS_PRESENCE_GEOFENCE_ENABLED=
EOF
    log_success "Created optional environment file ${ENV_FILE}"
fi
chown root:"${DOMUS_GROUP}" "${ENV_FILE}"
chmod 0640 "${ENV_FILE}"

# ── 8. systemd unit (generated from the repo template) ─────────────────────────
if (( HAVE_SYSTEMD )); then
    TEMPLATE="${PROJECT_ROOT}/deployments/systemd/domus.service"
    [[ -f "${TEMPLATE}" ]] || { log_error "Missing ${TEMPLATE}"; exit 1; }
    sed -e "s|@DOMUS_USER@|${DOMUS_USER}|g" \
        -e "s|@DOMUS_GROUP@|${DOMUS_GROUP}|g" \
        -e "s|@DOMUS_DATA_DIR@|${DOMUS_DATA_DIR}|g" \
        -e "s|@DOMUS_CONFIG_DIR@|${DOMUS_CONFIG_DIR}|g" \
        -e "s|@DOMUS_INSTALL_DIR@|${DOMUS_INSTALL_DIR}|g" \
        -e "s|@DOMUS_WEB_DIR@|${DOMUS_WEB_DIR}|g" \
        "${TEMPLATE}" > /etc/systemd/system/domus.service
    chmod 0644 /etc/systemd/system/domus.service
    systemctl daemon-reload
    systemctl enable domus.service >/dev/null
    log_info "Starting domus.service"
    systemctl restart domus.service

    log_info "Waiting for health endpoint (up to 30 s)..."
    healthy=0
    for _ in $(seq 1 30); do
        if curl -fsS "http://127.0.0.1:${DOMUS_PORT}/health" >/dev/null 2>&1; then healthy=1; break; fi
        sleep 1
    done
    if (( healthy )); then
        log_success "Domus is up"
        ui_code="$(curl -sS -o /dev/null -w '%{http_code}' "http://127.0.0.1:${DOMUS_PORT}/" || true)"
        if [[ "${ui_code}" != "200" ]]; then
            log_error "Frontend health check failed: GET / returned ${ui_code} (expected 200)."
            log_error "Recent service logs:"
            journalctl -u domus -n 100 --no-pager >&2 || true
            log_warn "Stopping Domus because the web interface is not available."
            systemctl stop domus.service || true
            exit 1
        fi
        log_success "Web interface responds with HTTP 200"
    else
        log_error "Domus did not become healthy within 30 seconds."
        log_error "Recent service logs:"
        journalctl -u domus -n 100 --no-pager >&2 || true
        log_warn "Stopping the unhealthy service. Fix the reported error, then run: systemctl start domus"
        systemctl stop domus.service || true
        exit 1
    fi
else
    log_warn "systemd not found: run manually  DOMUS_SERVER_WEB_DIR=${DOMUS_WEB_DIR} ${DOMUS_INSTALL_DIR}/domus -config ${CONFIG_TARGET}"
fi

LAN_IP="$(hostname -I 2>/dev/null | awk '{print $1}')"
echo
echo -e "${GREEN}${BOLD}=== Domus installed ===${NC}"
echo -e "Web UI    : ${CYAN}http://${LAN_IP:-localhost}:${DOMUS_PORT}/${NC}   (or http://$(hostname).local:${DOMUS_PORT}/)"
echo -e "API docs  : ${CYAN}http://${LAN_IP:-localhost}:${DOMUS_PORT}/docs${NC}"
echo -e "Logs      : ${BOLD}journalctl -u domus -f${NC}"
echo -e "Status    : ${BOLD}systemctl status domus${NC}"
echo -e "CLI       : ${BOLD}domus-cli help${NC}"