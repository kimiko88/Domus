#!/usr/bin/env bash
# ==============================================================================
# Domus — Clean Uninstallation Script
# ==============================================================================
set -euo pipefail

DOMUS_INSTALL_DIR="${DOMUS_INSTALL_DIR:-/usr/local/bin}"
DOMUS_CONFIG_DIR="${DOMUS_CONFIG_DIR:-/etc/domus}"
DOMUS_DATA_DIR="${DOMUS_DATA_DIR:-/var/lib/domus}"
PURGE_DATA="${PURGE_DATA:-0}"

# Text formatting
BOLD='\033[1m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

if [[ "$(id -u)" -ne 0 ]]; then
    echo -e "${RED}[ERROR]${NC} This uninstaller must be run as root (or via sudo)." >&2
    exit 1
fi

echo -e "${BOLD}===================================================================${NC}"
echo -e "${BOLD}                  Domus Uninstallation Tool                        ${NC}"
echo -e "${BOLD}===================================================================${NC}"

# 1. Stop and disable systemd service
if command -v systemctl >/dev/null 2>&1; then
    if systemctl is-active --quiet domus.service 2>/dev/null; then
        echo -e "${CYAN}[INFO]${NC} Stopping domus service..."
        systemctl stop domus.service
    fi

    if systemctl is-enabled --quiet domus.service 2>/dev/null; then
        echo -e "${CYAN}[INFO]${NC} Disabling domus service..."
        systemctl disable domus.service
    fi

    if [[ -f /etc/systemd/system/domus.service ]]; then
        echo -e "${CYAN}[INFO]${NC} Removing systemd service unit..."
        rm -f /etc/systemd/system/domus.service
        systemctl daemon-reload
    fi
fi

# 2. Remove binaries
echo -e "${CYAN}[INFO]${NC} Removing binaries..."
rm -f "${DOMUS_INSTALL_DIR}/domus" "${DOMUS_INSTALL_DIR}/domus-cli"

# 3. Purge data if requested
if [[ "${PURGE_DATA}" -eq 1 ]]; then
    echo -e "${YELLOW}[WARN]${NC} PURGE_DATA=1 specified. Removing data and configuration..."
    rm -rf "${DOMUS_DATA_DIR}" "${DOMUS_CONFIG_DIR}"
    echo -e "${GREEN}[SUCCESS]${NC} Data and configuration removed."
else
    echo -e "${CYAN}[INFO]${NC} Preserving database and configuration at ${DOMUS_DATA_DIR} and ${DOMUS_CONFIG_DIR}."
    echo -e "${CYAN}[INFO]${NC} To remove them manually, run: sudo rm -rf ${DOMUS_DATA_DIR} ${DOMUS_CONFIG_DIR}"
fi

echo ""
echo -e "${GREEN}${BOLD}Domus uninstallation completed successfully.${NC}"
