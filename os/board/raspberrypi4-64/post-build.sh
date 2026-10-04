#!/usr/bin/env bash
# ─── Domus OS - Buildroot Post-Build Hook ─────────────────────────────────────
# Configures the target filesystem before creating the immutable SquashFS image.
set -euo pipefail

TARGET_DIR="${1}"
BOARD_DIR="$(dirname "${BASH_SOURCE[0]}")"

echo "=== Domus OS Post-Build: Configuring Target Filesystem in ${TARGET_DIR} ==="

# 1. Ensure required appliance mount points exist
mkdir -p "${TARGET_DIR}/data"
mkdir -p "${TARGET_DIR}/mnt"
mkdir -p "${TARGET_DIR}/run/domus"
mkdir -p "${TARGET_DIR}/var/volatile"
chmod 0755 "${TARGET_DIR}/data"

# 2. Configure /etc/os-release with Domus OS branding and metadata
cat << 'EOF' > "${TARGET_DIR}/etc/os-release"
NAME="Domus OS"
ID=domus-os
PRETTY_NAME="Domus OS 1.0 (RPi4 ARM64)"
VERSION="1.0.0"
VERSION_ID="1.0.0"
BUILD_ID="20261004"
VARIANT="Smart Home Appliance"
VARIANT_ID="appliance"
HOME_URL="https://domus-hub.org"
SUPPORT_URL="https://github.com/domus-hub/domus"
BUG_REPORT_URL="https://github.com/domus-hub/domus/issues"
EOF

# 3. Configure /etc/hostname
echo "domus" > "${TARGET_DIR}/etc/hostname"

# 4. Generate hardened /etc/fstab with read-only rootfs and tmpfs mount points
cat << 'EOF' > "${TARGET_DIR}/etc/fstab"
# /etc/fstab: Domus OS static file system information.
# <file system>                          <mount pt>       <type>   <options>                        <dump> <pass>
# Root filesystem is mounted read-only by the kernel/init
/dev/root                                /                auto     defaults,ro                      1      1

# Persistent user data partition
/dev/disk/by-partlabel/domus-data        /data            ext4     defaults,noatime,commit=30       0      2

# Volatile RAM filesystems (protect flash memory from wear)
tmpfs                                    /tmp             tmpfs    defaults,nosuid,nodev,size=256M  0      0
tmpfs                                    /run             tmpfs    defaults,nosuid,nodev,size=128M  0      0
tmpfs                                    /var/volatile    tmpfs    defaults,nosuid,nodev,size=128M  0      0
EOF

# 5. Link /var/log to /var/volatile/log or tmpfs if not already handled
mkdir -p "${TARGET_DIR}/var/volatile/log"
if [ ! -L "${TARGET_DIR}/var/log" ]; then
    rm -rf "${TARGET_DIR}/var/log"
    ln -s /var/volatile/log "${TARGET_DIR}/var/log"
fi

# 6. Embedded hardening: disable empty root password login over network
if [ -f "${TARGET_DIR}/etc/shadow" ]; then
    # Ensure root has no blank password
    sed -i 's/^root::/root:*:/' "${TARGET_DIR}/etc/shadow"
fi

# 7. Install custom config.txt and cmdline.txt into BINARIES_DIR staging if needed
if [ -n "${BINARIES_DIR:-}" ]; then
    cp -f "${BOARD_DIR}/config.txt" "${BINARIES_DIR}/config.txt"
    cp -f "${BOARD_DIR}/cmdline.txt" "${BINARIES_DIR}/cmdline.txt"
fi

# 8. Install systemd storage and mount units
SYSTEM_DIR="${BOARD_DIR}/../../systemd"
SYSTEMD_SYSTEM="${TARGET_DIR}/etc/systemd/system"
mkdir -p "${SYSTEMD_SYSTEM}"
mkdir -p "${SYSTEMD_SYSTEM}/local-fs.target.wants"
mkdir -p "${SYSTEMD_SYSTEM}/basic.target.wants"

if [ -f "${SYSTEM_DIR}/data.mount" ]; then
    cp -f "${SYSTEM_DIR}/data.mount" "${SYSTEMD_SYSTEM}/data.mount"
    ln -sf "/etc/systemd/system/data.mount" "${SYSTEMD_SYSTEM}/local-fs.target.wants/data.mount"
fi

if [ -f "${SYSTEM_DIR}/tmp.mount" ]; then
    cp -f "${SYSTEM_DIR}/tmp.mount" "${SYSTEMD_SYSTEM}/tmp.mount"
    ln -sf "/etc/systemd/system/tmp.mount" "${SYSTEMD_SYSTEM}/local-fs.target.wants/tmp.mount"
fi

if [ -f "${SYSTEM_DIR}/domus-storage-init.service" ]; then
    cp -f "${SYSTEM_DIR}/domus-storage-init.service" "${SYSTEMD_SYSTEM}/domus-storage-init.service"
    ln -sf "/etc/systemd/system/domus-storage-init.service" "${SYSTEMD_SYSTEM}/basic.target.wants/domus-storage-init.service"
fi

# 9. Install journald flash-protection configuration
if [ -f "${SYSTEM_DIR}/journald.conf" ]; then
    mkdir -p "${TARGET_DIR}/etc/systemd/journald.conf.d"
    cp -f "${SYSTEM_DIR}/journald.conf" "${TARGET_DIR}/etc/systemd/journald.conf.d/00-domus-limits.conf"
    cp -f "${SYSTEM_DIR}/journald.conf" "${TARGET_DIR}/etc/systemd/journald.conf"
fi

# 10. Install storage initialization and factory reset utilities
SCRIPTS_DIR="${BOARD_DIR}/../../scripts"
mkdir -p "${TARGET_DIR}/usr/sbin"
if [ -f "${SCRIPTS_DIR}/domus-storage-init.sh" ]; then
    cp -f "${SCRIPTS_DIR}/domus-storage-init.sh" "${TARGET_DIR}/usr/sbin/domus-storage-init"
    chmod 0755 "${TARGET_DIR}/usr/sbin/domus-storage-init"
fi

if [ -f "${SCRIPTS_DIR}/domus-factory-reset.sh" ]; then
    cp -f "${SCRIPTS_DIR}/domus-factory-reset.sh" "${TARGET_DIR}/usr/sbin/domus-factory-reset"
    chmod 0755 "${TARGET_DIR}/usr/sbin/domus-factory-reset"
fi

# 11. Install RAUC update configuration, keyring, and U-Boot tools config
RAUC_DIR="${BOARD_DIR}/../../rauc"
mkdir -p "${TARGET_DIR}/etc/rauc"
if [ -f "${RAUC_DIR}/system.conf" ]; then
    cp -f "${RAUC_DIR}/system.conf" "${TARGET_DIR}/etc/rauc/system.conf"
fi

if [ -f "${RAUC_DIR}/certs/ca.cert.pem" ]; then
    cp -f "${RAUC_DIR}/certs/ca.cert.pem" "${TARGET_DIR}/etc/rauc/ca.cert.pem"
    chmod 0644 "${TARGET_DIR}/etc/rauc/ca.cert.pem"
fi

if [ -f "${BOARD_DIR}/fw_env.config" ]; then
    cp -f "${BOARD_DIR}/fw_env.config" "${TARGET_DIR}/etc/fw_env.config"
fi

if [ -f "${SCRIPTS_DIR}/domus-update-agent.sh" ]; then
    cp -f "${SCRIPTS_DIR}/domus-update-agent.sh" "${TARGET_DIR}/usr/sbin/domus-update"
    chmod 0755 "${TARGET_DIR}/usr/sbin/domus-update"
fi

echo "=== Domus OS Post-Build: Target Filesystem Configured Successfully ==="


