#!/usr/bin/env bash
# ─── Domus OS - Update Agent CLI Utility ──────────────────────────────────────
# Controls dual-slot A/B system updates, health confirmation, and rollbacks.
# Executable: /usr/sbin/domus-update
set -euo pipefail

usage() {
    cat << EOF
Domus OS Update Agent

Usage:
  $(basename "$0") <command> [arguments]

Commands:
  install <bundle>   Install a signed .raucb update bundle (local file or HTTPS URL)
  status             Show current slot status, bootcounter, and update health
  mark-good          Mark the currently running slot as healthy (confirms update)
  rollback           Mark current slot bad and trigger immediate rollback to alternate slot
  help               Show this help message
EOF
}

CMD="${1:-help}"

case "${CMD}" in
    install)
        BUNDLE="${2:-}"
        if [ -z "${BUNDLE}" ]; then
            echo "ERROR: Missing update bundle path or URL."
            usage
            exit 1
        fi

        # Check if argument is a remote URL
        if [[ "${BUNDLE}" =~ ^https?:// ]]; then
            echo "Downloading update bundle from: ${BUNDLE}..."
            TMP_BUNDLE="/tmp/update.raucb"
            curl -fsSL --progress-bar -o "${TMP_BUNDLE}" "${BUNDLE}"
            BUNDLE="${TMP_BUNDLE}"
        fi

        if [ ! -f "${BUNDLE}" ]; then
            echo "ERROR: Bundle file not found: ${BUNDLE}"
            exit 1
        fi

        echo "=== Installing Update Bundle via RAUC ==="
        rauc install "${BUNDLE}"
        echo "Update installed successfully. Reboot system to activate new slot."
        ;;

    status)
        echo "=== Domus OS A/B Update Status ==="
        if command -v rauc >/dev/null 2>&1; then
            rauc status --detailed
        else
            echo "rauc command not available on this host."
        fi
        ;;

    mark-good)
        echo "=== Confirming Active Slot as Healthy ==="
        if command -v rauc >/dev/null 2>&1; then
            rauc status mark-good
            echo "Slot confirmed healthy. U-Boot bootcounter reset."
        else
            echo "rauc command not available on this host."
        fi
        ;;

    rollback)
        echo "=== Initiating Automatic Rollback to Alternate Slot ==="
        if command -v rauc >/dev/null 2>&1; then
            rauc status mark-bad
            echo "Active slot marked bad. Rebooting system to fallback slot..."
            systemctl reboot || reboot
        else
            echo "rauc command not available on this host."
        fi
        ;;

    help|-h|--help)
        usage
        exit 0
        ;;

    *)
        echo "Unknown command: ${CMD}"
        usage
        exit 1
        ;;
esac
