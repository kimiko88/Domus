#!/usr/bin/env bash
# ─── Domus OS - Persistent Storage Initialization ────────────────────────────
# Runs on boot via domus-storage-init.service to provision /data directory
# layout, initialize machine-id, redirect journal logs, and auto-expand storage.
set -euo pipefail

DATA_ROOT="${DATA_ROOT:-/data}"
SKIP_RESIZE="${SKIP_RESIZE:-false}"

echo "=== Domus OS: Initializing Persistent Storage at ${DATA_ROOT} ==="

# 1. Automatic partition resize on first boot (if storage media is larger than image)
if [ "${SKIP_RESIZE}" != "true" ] && command -v resize2fs >/dev/null 2>&1; then
    DATA_DEV="/dev/disk/by-partlabel/domus-data"
    if [ -b "${DATA_DEV}" ]; then
        echo "Attempting online ext4 expansion for ${DATA_DEV}..."
        resize2fs "${DATA_DEV}" 2>/dev/null || true
    fi
fi

# 2. Provision directory skeleton with principle of least privilege permissions
mkdir -p "${DATA_ROOT}/system"
chmod 0755 "${DATA_ROOT}/system"

mkdir -p "${DATA_ROOT}/system/network"
chmod 0700 "${DATA_ROOT}/system/network"

mkdir -p "${DATA_ROOT}/system/ssh"
chmod 0700 "${DATA_ROOT}/system/ssh"

mkdir -p "${DATA_ROOT}/system/rauc"
chmod 0755 "${DATA_ROOT}/system/rauc"

mkdir -p "${DATA_ROOT}/core"
chmod 0750 "${DATA_ROOT}/core"

mkdir -p "${DATA_ROOT}/core/backups"
chmod 0700 "${DATA_ROOT}/core/backups"

mkdir -p "${DATA_ROOT}/supervisor"
chmod 0750 "${DATA_ROOT}/supervisor"

mkdir -p "${DATA_ROOT}/supervisor/manifests"
chmod 0755 "${DATA_ROOT}/supervisor/manifests"

mkdir -p "${DATA_ROOT}/apps"
chmod 0755 "${DATA_ROOT}/apps"

mkdir -p "${DATA_ROOT}/containerd"
chmod 0711 "${DATA_ROOT}/containerd"

mkdir -p "${DATA_ROOT}/logs/journal"
chmod 2755 "${DATA_ROOT}/logs/journal"

# 3. Persistent Machine ID handling
PERSISTENT_MACHINE_ID="${DATA_ROOT}/system/machine-id"
SYSTEM_MACHINE_ID="/etc/machine-id"

if [ -f "${PERSISTENT_MACHINE_ID}" ] && [ -s "${PERSISTENT_MACHINE_ID}" ]; then
    # Restore persistent machine-id onto system if /etc is writable or via bind
    if [ -w "${SYSTEM_MACHINE_ID}" ]; then
        cp -f "${PERSISTENT_MACHINE_ID}" "${SYSTEM_MACHINE_ID}"
    fi
elif [ -f "${SYSTEM_MACHINE_ID}" ] && [ -s "${SYSTEM_MACHINE_ID}" ]; then
    # Save newly generated machine-id to persistent storage
    cp -f "${SYSTEM_MACHINE_ID}" "${PERSISTENT_MACHINE_ID}"
    chmod 0444 "${PERSISTENT_MACHINE_ID}"
fi

# 4. Redirect systemd journald logs to persistent storage partition
JOURNAL_SYMLINK="/var/log/journal"
if [ ! -L "${JOURNAL_SYMLINK}" ] && [ ! -d "${JOURNAL_SYMLINK}" ]; then
    mkdir -p "/var/log"
    ln -sf "${DATA_ROOT}/logs/journal" "${JOURNAL_SYMLINK}"
fi

echo "=== Domus OS: Storage Initialization Completed Successfully ==="
exit 0
