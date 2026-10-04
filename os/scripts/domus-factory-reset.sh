#!/usr/bin/env bash
# ─── Domus OS - Factory Reset Utility ─────────────────────────────────────────
# Clears all user data, databases, secrets, container layers, and extensions
# from the persistent /data partition, returning the appliance to factory state.
# The immutable SquashFS base OS remains completely untouched.
set -euo pipefail

FORCE=false
KEEP_NETWORK=false
DATA_ROOT="${DATA_ROOT:-/data}"

usage() {
    cat << EOF
Domus OS Factory Reset Utility

Usage:
  $(basename "$0") [options]

Options:
  --force           Execute factory reset immediately without interactive confirmation
  --keep-network    Preserve Wi-Fi and Ethernet IP configurations across reset
  --help            Show this help message

WARNING: This operation permanently erases all devices, automations, users,
database records, and installed applications!
EOF
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --force)
            FORCE=true
            shift
            ;;
        --keep-network)
            KEEP_NETWORK=true
            shift
            ;;
        --help|-h)
            usage
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

if [ "${FORCE}" != "true" ]; then
    echo "=========================================================="
    echo "===             DOMUS OS FACTORY RESET                ==="
    echo "=========================================================="
    echo "WARNING: You are about to permanently erase all user data,"
    echo "configurations, SQLite databases, and installed apps!"
    echo "This action CANNOT be undone."
    echo ""
    read -r -p "Are you sure you want to proceed? (Type 'RESET' to confirm): " CONFIRM
    if [ "${CONFIRM}" != "RESET" ]; then
        echo "Factory reset aborted by user."
        exit 1
    fi
fi

echo "=== Initiating Domus OS Factory Reset ==="

# 1. Stop Domus Supervisor and Core services if running
if command -v systemctl >/dev/null 2>&1; then
    echo "Stopping container engine and supervisor services..."
    systemctl stop domus-supervisor.service 2>/dev/null || true
    systemctl stop containerd.service 2>/dev/null || true
fi

# 2. Preserve network configuration if requested
TEMP_NET_BACKUP=""
if [ "${KEEP_NETWORK}" = "true" ] && [ -d "${DATA_ROOT}/system/network" ]; then
    echo "Preserving network configurations..."
    TEMP_NET_BACKUP=$(mktemp -d)
    cp -rf "${DATA_ROOT}/system/network/"* "${TEMP_NET_BACKUP}/" 2>/dev/null || true
fi

# 3. Purge user-level and runtime data directories
echo "Purging user data, database, and container storage..."
rm -rf "${DATA_ROOT}/core"
rm -rf "${DATA_ROOT}/supervisor"
rm -rf "${DATA_ROOT}/apps"
rm -rf "${DATA_ROOT}/containerd"
rm -rf "${DATA_ROOT}/logs"
rm -rf "${DATA_ROOT}/system/ssh"
rm -rf "${DATA_ROOT}/system/machine-id"

if [ "${KEEP_NETWORK}" != "true" ]; then
    rm -rf "${DATA_ROOT}/system/network"
fi

# 4. Flush file system buffers to ensure complete deletion on flash storage
sync

# 5. Restore network configuration if preserved
if [ "${KEEP_NETWORK}" = "true" ] && [ -n "${TEMP_NET_BACKUP}" ] && [ -d "${TEMP_NET_BACKUP}" ]; then
    mkdir -p "${DATA_ROOT}/system/network"
    cp -rf "${TEMP_NET_BACKUP}/"* "${DATA_ROOT}/system/network/" 2>/dev/null || true
    rm -rf "${TEMP_NET_BACKUP}"
fi

# 6. Re-provision empty directory hierarchy with proper permissions
echo "Re-provisioning clean storage skeleton..."
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [ -x "${SCRIPT_DIR}/domus-storage-init.sh" ]; then
    DATA_ROOT="${DATA_ROOT}" SKIP_RESIZE=true "${SCRIPT_DIR}/domus-storage-init.sh"
elif [ -x "/usr/sbin/domus-storage-init" ]; then
    DATA_ROOT="${DATA_ROOT}" SKIP_RESIZE=true /usr/sbin/domus-storage-init
fi

sync
echo "=== Domus OS Factory Reset Completed Successfully ==="

# 7. Reboot appliance if systemctl is functional
if command -v systemctl >/dev/null 2>&1 && [ "${DATA_ROOT}" = "/data" ]; then
    echo "Rebooting system into clean appliance state..."
    systemctl reboot || reboot
fi
