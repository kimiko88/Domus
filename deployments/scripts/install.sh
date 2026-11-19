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
#      Go / Node toolchains and Go / npm download caches are kept between runs
#      under /opt/domus-toolchains and /var/cache/domus/build by default.
#   3. Installs binaries, web assets, config, systemd unit; starts the service.
#
# Tip: on a Pi with < 2 GB RAM, build on your PC and copy the results:
#   GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o domus ./cmd/domus
#   GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o domus-cli ./cmd/domus-cli
#   (cd web && npm ci && npm run build)
# then run this script on the Pi: it will just install what it finds.
#
# Environment overrides:
#   DOMUS_VERSION, DOMUS_HOST, DOMUS_PORT, DOMUS_COOKIE_SECURE, DOMUS_VAPID_SUBJECT, DOMUS_USER, DOMUS_GROUP
#   DOMUS_INSTALL_DIR (/usr/local/bin)   DOMUS_CONFIG_DIR (/etc/domus)
#   DOMUS_DATA_DIR (/var/lib/domus)      DOMUS_WEB_DIR (/usr/local/share/domus/web)
#   GO_VERSION (default: read from go.mod)   GO_INSTALL_DIR (/opt/domus-toolchains)   NODE_MAJOR (default: 22)
#   DOMUS_BUILD_CACHE_DIR (/var/cache/domus/build)   persistent Go and npm caches
#   INSTALL_MOSQUITTO=1   install the Mosquitto MQTT broker via apt
# ==============================================================================
set -euo pipefail

DOMUS_INSTALL_DIR="${DOMUS_INSTALL_DIR:-/usr/local/bin}"
DOMUS_CONFIG_DIR="${DOMUS_CONFIG_DIR:-/etc/domus}"
DOMUS_DATA_DIR="${DOMUS_DATA_DIR:-/var/lib/domus}"
DOMUS_WEB_DIR="${DOMUS_WEB_DIR:-/usr/local/share/domus/web}"
DOMUS_USER="${DOMUS_USER:-domus}"
DOMUS_GROUP="${DOMUS_GROUP:-domus}"
DOMUS_HOST_EXPLICIT=0
[[ -n "${DOMUS_HOST:-}" ]] && DOMUS_HOST_EXPLICIT=1
DOMUS_HOST="${DOMUS_HOST:-0.0.0.0}"
DOMUS_PORT="${DOMUS_PORT:-8080}"
DOMUS_COOKIE_SECURE_EXPLICIT="${DOMUS_COOKIE_SECURE+x}"
DOMUS_COOKIE_SECURE="${DOMUS_COOKIE_SECURE:-false}"
DOMUS_LAN_TLS_CERT_FILE_EXPLICIT="${DOMUS_LAN_TLS_CERT_FILE+x}"
DOMUS_LAN_TLS_KEY_FILE_EXPLICIT="${DOMUS_LAN_TLS_KEY_FILE+x}"
DOMUS_LAN_TLS_CERT_FILE="${DOMUS_LAN_TLS_CERT_FILE:-${DOMUS_CONFIG_DIR}/tls/lan.crt}"
DOMUS_LAN_TLS_KEY_FILE="${DOMUS_LAN_TLS_KEY_FILE:-${DOMUS_CONFIG_DIR}/tls/lan.key}"
REQUESTED_VAPID_SUBJECT="${DOMUS_VAPID_SUBJECT:-}"
if [[ "${DOMUS_COOKIE_SECURE}" != "true" && "${DOMUS_COOKIE_SECURE}" != "false" ]]; then
    printf '%s\n' 'DOMUS_COOKIE_SECURE must be true or false' >&2
    exit 1
fi
NODE_MAJOR="${NODE_MAJOR:-22}"
INSTALL_MOSQUITTO="${INSTALL_MOSQUITTO:-0}"
GO_INSTALL_DIR="${GO_INSTALL_DIR:-/opt/domus-toolchains}"
DOMUS_BUILD_CACHE_DIR="${DOMUS_BUILD_CACHE_DIR:-/var/cache/domus/build}"

BOLD=$'\033[1m'; GREEN=$'\033[0;32m'; CYAN=$'\033[0;36m'
YELLOW=$'\033[1;33m'; RED=$'\033[0;31m'; NC=$'\033[0m'
log_info()    { echo -e "${CYAN}[INFO]${NC} $*"; }
log_success() { echo -e "${GREEN}[ OK ]${NC} $*"; }
log_warn()    { echo -e "${YELLOW}[WARN]${NC} $*"; }
log_error()   { echo -e "${RED}[FAIL]${NC} $*" >&2; }

# Apple Push rejects localhost and .local VAPID subjects. Require a
# contact URI with a fully-qualified, non-local hostname.
vapid_subject_is_valid() {
    local subject="$1" target host domain fqdn_re
    fqdn_re='^[[:alnum:]]([[:alnum:]-]*[[:alnum:]])?(\.[[:alnum:]]([[:alnum:]-]*[[:alnum:]])?)+$'
    case "$subject" in
        mailto:*)
            target="${subject#mailto:}"
            [[ "$target" == *@* && "$target" != *@*'@'* && "$target" != *[[:space:]]* ]] || return 1
            domain="${target##*@}"
            domain="$(printf '%s' "$domain" | tr '[:upper:]' '[:lower:]')"
            [[ "$domain" =~ $fqdn_re ]] || return 1
            [[ "$domain" != localhost && "$domain" != *.localhost && "$domain" != *.local ]]
            ;;
        https://*)
            host="${subject#https://}"
            host="${host%%/*}"
            host="${host%%\?*}"
            host="${host%%#*}"
            host="${host%%:*}"
            host="$(printf '%s' "$host" | tr '[:upper:]' '[:lower:]')"
            [[ "$host" =~ $fqdn_re ]] || return 1
            [[ "$host" != localhost && "$host" != *.localhost && "$host" != *.local ]]
            ;;
        *)
            return 1
            ;;
    esac
}

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
for bin in curl tar xz git openssl; do
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
        # /var/tmp = disk, not RAM-backed tmpfs: temporary build staging can be large.
        WORK_DIR="$(mktemp -d /var/tmp/domus-build.XXXXXX)"
    fi
}

sha256_check() { # <file> <expected-hash>
    local got; got="$(sha256sum "$1" | awk '{print $1}')"
    if [[ "${got}" != "$2" ]]; then
        log_error "Checksum mismatch for $(basename "$1")"; exit 1
    fi
}

# Toolchains are intentionally persistent: reinstalling Domus must not redownload Node/Go.
toolchain_dir() {
    mkdir -p "${GO_INSTALL_DIR}"
    echo "${GO_INSTALL_DIR}"
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
    local tdir; tdir="$(toolchain_dir)"
    local candidate cached_major
    local candidates=( "${tdir}"/node-v"${NODE_MAJOR}".*-linux-"${NODE_ARCH}"/bin/node )
    for candidate in "${candidates[@]}"; do
        if [[ -x "${candidate}" ]]; then
            cached_major="$("${candidate}" -p 'process.versions.node.split(".")[0]' 2>/dev/null || true)"
            if [[ "${cached_major}" == "${NODE_MAJOR}" ]]; then
                local cached_version
                cached_version="$("${candidate}" -v)"
                log_info "Using cached Node.js ${cached_version} from $(dirname "${candidate}")"
                export PATH="$(dirname "${candidate}"):${PATH}"
                return
            fi
        fi
    done

    ensure_workdir
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
    # Keep module downloads and compiled package objects across installer runs.
    # Go verifies module checksums and downloads only modules absent from GOMODCACHE.
    export GOPATH="${DOMUS_BUILD_CACHE_DIR}/go"
    export GOMODCACHE="${GOPATH}/pkg/mod"
    export GOCACHE="${GOPATH}/build-cache"
    export GOTOOLCHAIN=local
    mkdir -p "${GOMODCACHE}" "${GOCACHE}"
    log_info "Go module cache : ${GOMODCACHE}"
    log_info "Go build cache  : ${GOCACHE}"
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
        # npm's content-addressed cache is persistent. Prefer a fully offline
        # install first; contact the registry only if this lockfile needs uncached packages.
        export npm_config_cache="${DOMUS_BUILD_CACHE_DIR}/npm" npm_config_update_notifier=false
        mkdir -p "${npm_config_cache}"
        if npm ci --offline --ignore-scripts --no-audit --no-fund; then
            log_success "Frontend dependencies restored entirely from the local npm cache"
        else
            log_warn "npm cache is incomplete for this lockfile; downloading only missing packages"
            npm ci --prefer-offline --ignore-scripts --no-audit --no-fund
        fi
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

# Add optional LAN listener defaults to older configurations without enabling it
# or replacing administrator-supplied values.
if ! grep -Fqx '[server.lan]' "${CONFIG_TARGET}"; then
    cat >> "${CONFIG_TARGET}" <<EOF

[server.lan]
enabled = false
host = "0.0.0.0"
port = 8081
tls_cert_file = "${DOMUS_LAN_TLS_CERT_FILE}"
tls_key_file = "${DOMUS_LAN_TLS_KEY_FILE}"
EOF
else
    ensure_lan_config_key() {
        local key="$1" value="$2"
        if ! awk -v target='[server.lan]' -v key="${key}" '
            /^\[/ { inside = ($0 == target) }
            inside && $0 ~ "^[[:space:]]*" key "[[:space:]]*=" { found = 1 }
            END { exit(found ? 0 : 1) }
        ' "${CONFIG_TARGET}"; then
            sed -i "/^\[server\.lan\]/a ${key} = ${value}" "${CONFIG_TARGET}"
        fi
    }
    ensure_lan_config_key enabled false
    ensure_lan_config_key host '"0.0.0.0"'
    ensure_lan_config_key port 8081
    ensure_lan_config_key tls_cert_file "\"\${DOMUS_LAN_TLS_CERT_FILE}\""
    ensure_lan_config_key tls_key_file "\"\${DOMUS_LAN_TLS_KEY_FILE}\""
fi
chown root:"${DOMUS_GROUP}" "${CONFIG_TARGET}"
chmod 0640 "${CONFIG_TARGET}"
log_info "LAN HTTPS configuration prepared; listener remains disabled by default"

# Resolve certificate paths from the effective TOML config. Explicit installer
# overrides are written into [server.lan]; otherwise existing custom paths win.
set_lan_config_path() {
    local key="$1" value="$2" tmp_file
    tmp_file="$(mktemp "${DOMUS_CONFIG_DIR}/.domus-toml.XXXXXX")"
    awk -v target='[server.lan]' -v key="${key}" -v value="${value}" '
        /^\[/ { inside = ($0 == target) }
        inside && $0 ~ "^[[:space:]]*" key "[[:space:]]*=" {
            print key " = \"" value "\""
            found = 1
            next
        }
        { print }
        END { if (!found) exit 1 }
    ' "${CONFIG_TARGET}" > "${tmp_file}" || {
        rm -f "${tmp_file}"
        log_error "Could not update server.lan.${key} in ${CONFIG_TARGET}"
        exit 1
    }
    cat "${tmp_file}" > "${CONFIG_TARGET}"
    rm -f "${tmp_file}"
}
read_lan_config_path() {
    local key="$1"
    awk -v target='[server.lan]' -v key="${key}" '
        /^\[/ { inside = ($0 == target) }
        inside && $0 ~ "^[[:space:]]*" key "[[:space:]]*=" {
            sub(/^[^=]*=[[:space:]]*/, "")
            gsub(/^["\047]|["\047]$/, "")
            print
            exit
        }
    ' "${CONFIG_TARGET}"
}
if [[ "${DOMUS_LAN_TLS_CERT_FILE_EXPLICIT}" == "x" ]]; then
    set_lan_config_path tls_cert_file "${DOMUS_LAN_TLS_CERT_FILE}"
else
    DOMUS_LAN_TLS_CERT_FILE="$(read_lan_config_path tls_cert_file)"
fi
if [[ "${DOMUS_LAN_TLS_KEY_FILE_EXPLICIT}" == "x" ]]; then
    set_lan_config_path tls_key_file "${DOMUS_LAN_TLS_KEY_FILE}"
else
    DOMUS_LAN_TLS_KEY_FILE="$(read_lan_config_path tls_key_file)"
fi
[[ -n "${DOMUS_LAN_TLS_CERT_FILE}" && -n "${DOMUS_LAN_TLS_KEY_FILE}" ]] || {
    log_error "server.lan TLS certificate and key paths must not be empty"
    exit 1
}
chown root:"${DOMUS_GROUP}" "${CONFIG_TARGET}"
chmod 0640 "${CONFIG_TARGET}"

# Prepare a persistent self-signed certificate for the optional LAN HTTPS listener.
# It is safe to create these files while the listener remains disabled.
# Existing pairs are preserved; incomplete or mismatched pairs fail closed.
ensure_lan_tls_certificate() {
    local cert_file="$1" key_file="$2" cert_dir key_dir tmp_dir
    local short_host fqdn san_file dns_index ip_index cert_pub key_pub interface address
    cert_dir="$(dirname "${cert_file}")"
    key_dir="$(dirname "${key_file}")"
    [[ "${cert_file}" != "${key_file}" ]] || {
        log_error "LAN TLS certificate and key must use different paths"
        exit 1
    }
    install -d -o root -g "${DOMUS_GROUP}" -m 0750 "${cert_dir}" "${key_dir}"

    if [[ -e "${cert_file}" || -e "${key_file}" ]]; then
        if [[ ! -s "${cert_file}" || ! -s "${key_file}" ]]; then
            log_error "An incomplete LAN TLS pair exists; restore both files. Existing files were not overwritten."
            exit 1
        fi
        openssl x509 -in "${cert_file}" -noout >/dev/null 2>&1 || {
            log_error "Existing LAN TLS certificate is invalid: ${cert_file}"
            exit 1
        }
        openssl pkey -in "${key_file}" -noout >/dev/null 2>&1 || {
            log_error "Existing LAN TLS private key is invalid: ${key_file}"
            exit 1
        }
        cert_pub="$(openssl x509 -in "${cert_file}" -pubkey -noout 2>/dev/null | openssl pkey -pubin -outform DER 2>/dev/null | openssl dgst -sha256)" || {
            log_error "Could not validate existing LAN TLS certificate"
            exit 1
        }
        key_pub="$(openssl pkey -in "${key_file}" -pubout -outform DER 2>/dev/null | openssl dgst -sha256)" || {
            log_error "Could not validate existing LAN TLS private key"
            exit 1
        }
        [[ -n "${cert_pub}" && "${cert_pub}" == "${key_pub}" ]] || {
            log_error "Existing LAN TLS certificate and private key do not match"
            exit 1
        }
        chown root:"${DOMUS_GROUP}" "${cert_file}" "${key_file}"
        chmod 0644 "${cert_file}"
        chmod 0640 "${key_file}"
        log_info "Preserving existing LAN TLS certificate and key"
        return 0
    fi

    short_host="$(hostname -s 2>/dev/null || hostname)"
    [[ -n "${short_host}" ]] || short_host="domus"
    fqdn="$(hostname -f 2>/dev/null || true)"
    tmp_dir="$(mktemp -d "${DOMUS_CONFIG_DIR}/.lan-tls.XXXXXX")"
    san_file="${tmp_dir}/openssl.cnf"
    {
        printf '%s\n' '[req]' 'prompt = no' 'distinguished_name = dn' 'x509_extensions = v3_req' '[dn]'
        printf 'CN = Domus LAN (%s)\n' "${short_host}"
        printf '%s\n' '[v3_req]' 'basicConstraints = critical,CA:FALSE' 'keyUsage = critical,digitalSignature,keyEncipherment' 'extendedKeyUsage = serverAuth' 'subjectAltName = @alt_names' '[alt_names]'
        dns_index=1
        printf 'DNS.%d = %s\n' "${dns_index}" "${short_host}"
        dns_index=$((dns_index + 1))
        printf 'DNS.%d = %s.local\n' "${dns_index}" "${short_host}"
        if [[ -n "${fqdn}" && "${fqdn}" != "${short_host}" && "${fqdn}" != "localhost" ]]; then
            dns_index=$((dns_index + 1))
            printf 'DNS.%d = %s\n' "${dns_index}" "${fqdn}"
        fi
        ip_index=0
        if command -v ip >/dev/null 2>&1; then
            while read -r interface address; do
                [[ -n "${address}" ]] || continue
                case "${interface}" in
                    tailscale*|docker*|br-*|veth*|virbr*|lo|wg*|tun*) continue ;;
                esac
                ip_index=$((ip_index + 1))
                printf 'IP.%d = %s\n' "${ip_index}" "${address}"
            done < <(ip -o addr show scope global 2>/dev/null | awk '{iface=$2; sub(/@.*/, "", iface); addr=$4; sub(/\/.*/, "", addr); print iface, addr}')
        fi
    } > "${san_file}"

    if ! openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:P-256 -sha256 -nodes \
        -days 825 -config "${san_file}" -extensions v3_req \
        -keyout "${tmp_dir}/lan.key" -out "${tmp_dir}/lan.crt" >/dev/null 2>&1; then
        rm -rf "${tmp_dir}"
        log_error "Could not generate the LAN TLS certificate with OpenSSL"
        exit 1
    fi
    install -o root -g "${DOMUS_GROUP}" -m 0644 "${tmp_dir}/lan.crt" "${cert_file}"
    install -o root -g "${DOMUS_GROUP}" -m 0640 "${tmp_dir}/lan.key" "${key_file}"
    rm -rf "${tmp_dir}"
    log_success "Created self-signed LAN TLS certificate: ${cert_file}"
    log_info "LAN TLS private key installed with root:${DOMUS_GROUP} ownership and mode 0640"
}

ensure_lan_tls_certificate "${DOMUS_LAN_TLS_CERT_FILE}" "${DOMUS_LAN_TLS_KEY_FILE}"

# Tailscale Serve can bind the Tailscale interface on the same TCP port that
# it proxies to localhost. In that case Domus must bind loopback, not 0.0.0.0.
TAILSCALE_SERVE_STATUS=""
TAILSCALE_SERVES_DOMUS=0
TAILSCALE_SERVE_URL=""
if command -v tailscale >/dev/null 2>&1; then
    TAILSCALE_SERVE_STATUS="$(tailscale serve status 2>/dev/null || true)"
    if [[ -n "${TAILSCALE_SERVE_STATUS}" ]] && printf '%s\n' "${TAILSCALE_SERVE_STATUS}" | awk -v port="${DOMUS_PORT}" '
        $0 ~ "^[[:space:]]*https?://[^[:space:]]+:" port "[[:space:]]+\\(tailnet only\\)[[:space:]]*$" {
            matching_service = 1
            next
        }
        matching_service && $0 ~ "proxy http://127\\.0\\.0\\.1:" port "(/)?[[:space:]]*$" {
            found = 1
            matching_service = 0
        }
        matching_service && $0 ~ "^[[:space:]]*https?://" {
            matching_service = 0
        }
        END { exit(found ? 0 : 1) }
    '; then
        TAILSCALE_SERVES_DOMUS=1
        TAILSCALE_SERVE_URL="$(printf '%s\n' "${TAILSCALE_SERVE_STATUS}" | grep -E "https?://[^[:space:]]+:${DOMUS_PORT} \\(tailnet only\\)" | head -n 1 | awk '{print $1}' || true)"
    fi
fi

# Re-read the effective host for both fresh and existing configurations.
cfg_host="$(awk -F= '/^[[:space:]]*host[[:space:]]*=/ {gsub(/[[:space:]"]/, "", $2); print $2; exit}' "${CONFIG_TARGET}")"

if (( TAILSCALE_SERVES_DOMUS )); then
    if (( DOMUS_HOST_EXPLICIT )) && [[ "${DOMUS_HOST}" != "127.0.0.1" ]]; then
        log_error "Tailscale Serve listens on TCP ${DOMUS_PORT} and proxies to 127.0.0.1:${DOMUS_PORT}."
        log_error "DOMUS_HOST=${DOMUS_HOST} conflicts with that setup. Re-run with DOMUS_HOST=127.0.0.1 or change the Tailscale Serve backend."
        exit 1
    fi

    desired_host="${DOMUS_HOST}"
    if (( ! DOMUS_HOST_EXPLICIT )); then
        desired_host="127.0.0.1"
    fi

    if [[ "${cfg_host}" != "${desired_host}" ]]; then
        if grep -Eq '^[[:space:]]*host[[:space:]]*=' "${CONFIG_TARGET}"; then
            sed -i -E "s|^[[:space:]]*host[[:space:]]*=.*|host = \"${desired_host}\"|" "${CONFIG_TARGET}"
        elif grep -q '^\[server\]' "${CONFIG_TARGET}"; then
            sed -i "/^\[server\]/a host = \"${desired_host}\"" "${CONFIG_TARGET}"
        else
            log_error "Cannot set server.host: [server] section is missing in ${CONFIG_TARGET}"
            exit 1
        fi
        chown root:"${DOMUS_GROUP}" "${CONFIG_TARGET}"
        chmod 0640 "${CONFIG_TARGET}"
    fi

    cfg_host="${desired_host}"
    DOMUS_HOST="${desired_host}"
    log_warn "Detected Tailscale Serve on port ${DOMUS_PORT} forwarding to localhost. Domus will bind to ${cfg_host} so it does not collide with tailscaled."
    log_info "Tailscale Serve remains the remote entry point; direct LAN access is not available with this loopback binding."
fi

# Check only sockets that overlap Domus' configured bind address. A Tailscale
# listener on a Tailscale IPv4/IPv6 address does not conflict with 127.0.0.1.
port_conflict_info() {
    local host="$1" port="$2" listeners
    listeners="$(ss -H -ltnp "sport = :${port}" 2>/dev/null || true)"
    case "${host}" in
        127.0.0.1)
            awk -v suffix=":${port}" '$4 == ("127.0.0.1" suffix) || $4 == ("0.0.0.0" suffix) || $4 == ("[::]" suffix) || $4 == ("*" suffix) { print }' <<< "${listeners}"
            ;;
        localhost)
            awk -v suffix=":${port}" '$4 == ("127.0.0.1" suffix) || $4 == ("[::1]" suffix) || $4 == ("0.0.0.0" suffix) || $4 == ("[::]" suffix) || $4 == ("*" suffix) { print }' <<< "${listeners}"
            ;;
        *)
            printf '%s\n' "${listeners}"
            ;;
    esac
}

# Optional secrets and environment overrides. Preserve an existing file.
ENV_FILE="${DOMUS_CONFIG_DIR}/domus.env"
if [[ ! -f "${ENV_FILE}" ]]; then
    cat > "${ENV_FILE}" <<'EOF'
# Optional runtime secrets. Remove the leading # and set strong random values.
# DOMUS_CLOUD_ASSISTANT_TOKEN=
# DOMUS_FAILOVER_HEARTBEAT_TOKEN=
#
# Browser Web Push (VAPID P-256). The installer generates and stores these
# keys automatically on first install; never rotate them while subscriptions exist.
# DOMUS_VAPID_PUBLIC_KEY=
# DOMUS_VAPID_PRIVATE_KEY=
# DOMUS_VAPID_SUBJECT=mailto:admin@example.com
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

# The application applies DOMUS_SERVER_HOST after reading domus.toml, so keep
# the systemd environment file aligned with the loopback binding as well.
if (( TAILSCALE_SERVES_DOMUS )); then
    if grep -Eq '^[[:space:]]*DOMUS_SERVER_HOST[[:space:]]*=' "${ENV_FILE}"; then
        sed -i -E "s|^[[:space:]]*DOMUS_SERVER_HOST[[:space:]]*=.*|DOMUS_SERVER_HOST=${cfg_host}|" "${ENV_FILE}"
    else
        printf 'DOMUS_SERVER_HOST=%s\n' "${cfg_host}" >> "${ENV_FILE}"
    fi
    chown root:"${DOMUS_GROUP}" "${ENV_FILE}"
    chmod 0640 "${ENV_FILE}"
    log_info "Set DOMUS_SERVER_HOST=${cfg_host} for the Tailscale Serve upstream"
fi

# Generate and persist one stable VAPID P-256 keypair. Never rotate these keys
# on upgrades: browser subscriptions are bound to the public key.
env_file_value() {
    local key="$1" value
    value="$(awk -F= -v key="$key" '$1 == key { sub(/^[^=]*=/, ""); print; exit }' "${ENV_FILE}")"
    value="$(printf '%s' "${value}" | tr -d '\r')"
    if [[ "${value}" == \"*\" ]]; then
        value="${value:1:${#value}-2}"
    fi
    printf '%s' "${value}"
}

hex_to_base64url() {
    local escaped
    escaped="$(printf '%s' "$1" | sed 's/../\\x&/g')"
    printf '%b' "${escaped}" | openssl base64 -A | tr '+/' '-_' | tr -d '='
}

generate_vapid_keypair() {
    local key_file key_dump private_hex public_hex
    key_file="$(mktemp "${DOMUS_CONFIG_DIR}/.vapid-key.XXXXXX")"
    chmod 0600 "${key_file}"
    if ! openssl ecparam -name prime256v1 -genkey -noout -out "${key_file}" >/dev/null 2>&1; then
        rm -f "${key_file}"
        log_error "Could not generate a VAPID P-256 keypair with OpenSSL"
        exit 1
    fi
    key_dump="$(openssl ec -in "${key_file}" -text -noout 2>/dev/null)" || {
        rm -f "${key_file}"
        log_error "Could not read the generated VAPID keypair"
        exit 1
    }
    rm -f "${key_file}"

    private_hex="$(printf '%s\n' "${key_dump}" | awk '
        /^[[:space:]]*priv:/ { section = "private"; next }
        /^[[:space:]]*pub:/  { section = ""; next }
        section == "private" {
            gsub(/[^[:xdigit:]]/, "")
            printf "%s", $0
        }')"
    public_hex="$(printf '%s\n' "${key_dump}" | awk '
        /^[[:space:]]*pub:/ { section = "public"; next }
        /^[[:space:]]*(ASN1 OID|NIST CURVE|Field Type):/ { section = ""; next }
        section == "public" {
            gsub(/[^[:xdigit:]]/, "")
            printf "%s", $0
        }')"

    # OpenSSL prints a P-256 private scalar as at most 32 bytes; left-pad if needed.
    if (( ${#private_hex} < 64 )); then
        private_hex="$(printf '%064s' "${private_hex}" | tr ' ' '0')"
    fi
    if [[ ${#private_hex} -ne 64 || ${#public_hex} -ne 130 || "${public_hex}" != 04* ]]; then
        log_error "OpenSSL returned an unexpected P-256 key format; VAPID keys were not written"
        exit 1
    fi

    DOMUS_VAPID_PRIVATE_KEY="$(hex_to_base64url "${private_hex}")"
    DOMUS_VAPID_PUBLIC_KEY="$(hex_to_base64url "${public_hex}")"
    [[ ${#DOMUS_VAPID_PRIVATE_KEY} -eq 43 && ${#DOMUS_VAPID_PUBLIC_KEY} -eq 87 ]] || {
        log_error "Generated VAPID keys have unexpected lengths"
        exit 1
    }
}

stored_vapid_public="$(env_file_value DOMUS_VAPID_PUBLIC_KEY)"
stored_vapid_private="$(env_file_value DOMUS_VAPID_PRIVATE_KEY)"
stored_vapid_subject="$(env_file_value DOMUS_VAPID_SUBJECT)"

# A partial keypair is unsafe: do not silently mix keys or replace an existing key.
if [[ -n "${stored_vapid_public}" && -z "${stored_vapid_private}" ]] ||
   [[ -z "${stored_vapid_public}" && -n "${stored_vapid_private}" ]]; then
    log_error "Only one VAPID key is present in ${ENV_FILE}; restore both DOMUS_VAPID_PUBLIC_KEY and DOMUS_VAPID_PRIVATE_KEY together."
    exit 1
fi

if [[ -n "${stored_vapid_public}" ]]; then
    DOMUS_VAPID_PUBLIC_KEY="${stored_vapid_public}"
    DOMUS_VAPID_PRIVATE_KEY="${stored_vapid_private}"
elif [[ -n "${DOMUS_VAPID_PUBLIC_KEY:-}" || -n "${DOMUS_VAPID_PRIVATE_KEY:-}" ]]; then
    # Explicit key overrides are supported, but the public/private pair is atomic.
    if [[ -z "${DOMUS_VAPID_PUBLIC_KEY:-}" || -z "${DOMUS_VAPID_PRIVATE_KEY:-}" ]]; then
        log_error "Set both DOMUS_VAPID_PUBLIC_KEY and DOMUS_VAPID_PRIVATE_KEY when supplying VAPID keys manually."
        exit 1
    fi
else
    log_info "Generating VAPID P-256 keypair (this happens only on first install)"
    generate_vapid_keypair
fi

if [[ ! "${DOMUS_VAPID_PUBLIC_KEY}" =~ ^[A-Za-z0-9_-]{87}$ ||
      ! "${DOMUS_VAPID_PRIVATE_KEY}" =~ ^[A-Za-z0-9_-]{43}$ ]]; then
    log_error "VAPID keys have an invalid Base64URL format or length"
    exit 1
fi

DOMUS_VAPID_SUBJECT="${stored_vapid_subject:-${DOMUS_VAPID_SUBJECT:-mailto:domus@localhost}}"
case "${DOMUS_VAPID_SUBJECT}" in
    mailto:*|https://*) ;;
    *) log_error "DOMUS_VAPID_SUBJECT must be a mailto: or https:// contact URI"; exit 1 ;;
esac

# Append only missing variables. Existing environment values are never overwritten.
if [[ -z "${stored_vapid_public}" ]]; then
    printf 'DOMUS_VAPID_PUBLIC_KEY=%s\n' "${DOMUS_VAPID_PUBLIC_KEY}" >> "${ENV_FILE}"
fi
if [[ -z "${stored_vapid_private}" ]]; then
    printf 'DOMUS_VAPID_PRIVATE_KEY=%s\n' "${DOMUS_VAPID_PRIVATE_KEY}" >> "${ENV_FILE}"
fi
if [[ -z "${stored_vapid_subject}" ]]; then
    printf 'DOMUS_VAPID_SUBJECT=%s\n' "${DOMUS_VAPID_SUBJECT}" >> "${ENV_FILE}"
fi
chown root:"${DOMUS_GROUP}" "${ENV_FILE}"
chmod 0640 "${ENV_FILE}"
log_success "VAPID Web Push keys are configured in ${ENV_FILE} (private key is not displayed)"

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

    # The previous domus.service instance was stopped above. Refuse to start
    # a second instance when another process still owns the configured port.
    # This avoids reporting a false-positive health check against that process.
    if command -v ss >/dev/null 2>&1; then
        port_listener="$(port_conflict_info "${cfg_host}" "${DOMUS_PORT}")"
        if [[ -n "${port_listener}" ]]; then
            log_error "TCP port ${DOMUS_PORT} is already in use; Domus was not started."
            printf '%s\n' "${port_listener}" >&2
            log_error "Identify the owner with: sudo ss -ltnp 'sport = :${DOMUS_PORT}'"
            log_error "Stop the old service/process that owns the port, then rerun the installer."
            exit 1
        fi
    else
        log_warn "'ss' is unavailable; cannot preflight TCP port ownership."
    fi

    log_info "Starting domus.service"
    systemctl restart domus.service

    log_info "Waiting for health endpoint (up to 30 s)..."
    healthy=0
    listener_info=""
    for _ in $(seq 1 30); do
        service_owns_port=1
        if command -v ss >/dev/null 2>&1; then
            main_pid="$(systemctl show domus.service --property=MainPID --value)"
            listener_info="$(ss -H -ltnp "sport = :${DOMUS_PORT}" 2>/dev/null || true)"
            if [[ ! "${main_pid}" =~ ^[1-9][0-9]*$ ]] || ! grep -Fq "pid=${main_pid}," <<< "${listener_info}"; then
                service_owns_port=0
            fi
        fi
        if (( service_owns_port )) && curl -fsS "http://127.0.0.1:${DOMUS_PORT}/health" >/dev/null 2>&1; then
            healthy=1
            break
        fi
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
        if [[ -n "${listener_info}" ]]; then
            log_error "Current TCP listener on port ${DOMUS_PORT}:"
            printf '%s\n' "${listener_info}" >&2
        fi
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
if [[ "${cfg_host}" == "127.0.0.1" || "${cfg_host}" == "localhost" ]]; then
    echo -e "Web UI    : ${CYAN}http://127.0.0.1:${DOMUS_PORT}/${NC} (solo locale)"
    if (( TAILSCALE_SERVES_DOMUS )) && [[ -n "${TAILSCALE_SERVE_URL}" ]]; then
        echo -e "Tailscale : ${CYAN}${TAILSCALE_SERVE_URL}/${NC} (tailnet only)"
    else
        echo -e "Remote UI : configure a reverse proxy to 127.0.0.1:${DOMUS_PORT} for remote access"
    fi
    echo -e "API docs  : ${CYAN}http://127.0.0.1:${DOMUS_PORT}/docs${NC}"
else
    echo -e "Web UI    : ${CYAN}http://${LAN_IP:-localhost}:${DOMUS_PORT}/${NC}   (or http://$(hostname).local:${DOMUS_PORT}/)"
    echo -e "API docs  : ${CYAN}http://${LAN_IP:-localhost}:${DOMUS_PORT}/docs${NC}"
fi
echo -e "Logs      : ${BOLD}journalctl -u domus -f${NC}"
echo -e "Status    : ${BOLD}systemctl status domus${NC}"
echo -e "CLI       : ${BOLD}domus-cli help${NC}"