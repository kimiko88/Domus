#!/usr/bin/env bash
# ─── Domus OS - RAUC Update Bundle Generator ──────────────────────────────────
# Creates a signed .raucb update bundle containing the immutable SquashFS rootfs.
# Usage:
#   ./os/scripts/build-rauc-bundle.sh --version <version> --rootfs <path> [options]
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
RAUC_DIR="${REPO_ROOT}/os/rauc"

VERSION="1.0.0"
ROOTFS_IMG=""
CERT_FILE="${RAUC_DIR}/certs/development.cert.pem"
KEY_FILE="${RAUC_DIR}/certs/development.key.pem"
OUTPUT_DIR="${REPO_ROOT}/build/bundles"

usage() {
    cat << EOF
Domus OS RAUC Update Bundle Generator

Usage:
  $(basename "$0") --version <version> --rootfs <path-to-rootfs.squashfs> [options]

Options:
  --version <ver>   Release version string (e.g. 1.1.0)
  --rootfs <path>   Path to built rootfs.squashfs image
  --cert <path>     Path to x.509 signing certificate (default: development.cert.pem)
  --key <path>      Path to private key for signing (default: development.key.pem)
  --out-dir <path>  Target output directory for .raucb bundle
  --help            Show this help message
EOF
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --version)
            VERSION="$2"
            shift 2
            ;;
        --rootfs)
            ROOTFS_IMG="$2"
            shift 2
            ;;
        --cert)
            CERT_FILE="$2"
            shift 2
            ;;
        --key)
            KEY_FILE="$2"
            shift 2
            ;;
        --out-dir)
            OUTPUT_DIR="$2"
            shift 2
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

if [ -z "${ROOTFS_IMG}" ]; then
    # Look for rootfs in default buildroot output directory
    DEFAULT_ROOTFS="${REPO_ROOT}/build/buildroot/buildroot-2024.02.5/output/images/rootfs.squashfs"
    if [ -f "${DEFAULT_ROOTFS}" ]; then
        ROOTFS_IMG="${DEFAULT_ROOTFS}"
    else
        echo "ERROR: --rootfs path is required and default image was not found."
        exit 1
    fi
fi

if [ ! -f "${CERT_FILE}" ] || [ ! -f "${KEY_FILE}" ]; then
    echo "ERROR: Signing certificate or key not found."
    echo "Run ./os/scripts/generate-rauc-keys.sh to generate development certificates."
    exit 1
fi

BUNDLE_TMP=$(mktemp -d)
trap 'rm -rf "${BUNDLE_TMP}"' EXIT

echo "=== Assembling RAUC Bundle: Domus OS v${VERSION} ==="
mkdir -p "${OUTPUT_DIR}"
BUNDLE_FILE="${OUTPUT_DIR}/domus-os-${VERSION}.raucb"

# Copy rootfs image into bundle staging
cp -f "${ROOTFS_IMG}" "${BUNDLE_TMP}/rootfs.squashfs"

# Generate manifest for this release
cat << EOF > "${BUNDLE_TMP}/manifest.raucm"
[package]
compatible=domus-os-rpi4
version=${VERSION}
description=Domus OS Release ${VERSION}

[image.rootfs]
filename=rootfs.squashfs
EOF

# Execute rauc bundle command (or host rauc)
RAUC_BIN="rauc"
if [ -x "${REPO_ROOT}/build/buildroot/buildroot-2024.02.5/output/host/bin/rauc" ]; then
    RAUC_BIN="${REPO_ROOT}/build/buildroot/buildroot-2024.02.5/output/host/bin/rauc"
fi

echo "Signing bundle with x.509 certificate..."
"${RAUC_BIN}" bundle \
    --cert="${CERT_FILE}" \
    --key="${KEY_FILE}" \
    "${BUNDLE_TMP}" \
    "${BUNDLE_FILE}"

# Generate SHA256 verification hash
(cd "${OUTPUT_DIR}" && sha256sum "$(basename "${BUNDLE_FILE}")" > "$(basename "${BUNDLE_FILE}").sha256")

echo "=== RAUC Bundle Generated Successfully ==="
echo "Bundle file: ${BUNDLE_FILE}"
echo "Checksum:    ${BUNDLE_FILE}.sha256"
