#!/usr/bin/env bash
# ─── Domus OS - Build Runner ──────────────────────────────────────────────────
# Automates Buildroot cross-compilation for Raspberry Pi 4 ARM64.
# Usage:
#   ./os/scripts/build.sh [--validate-only] [--docker] [--clean] [--defconfig]
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
OS_DIR="${REPO_ROOT}/os"
BUILDROOT_VERSION="2024.02.5"
BUILDROOT_TAR="buildroot-${BUILDROOT_VERSION}.tar.xz"
BUILDROOT_URL="https://buildroot.org/downloads/${BUILDROOT_TAR}"
BUILD_WORK_DIR="${REPO_ROOT}/build/buildroot"
DOCKER_IMAGE="domus-os-builder:latest"

MODE="build"
USE_DOCKER=false

usage() {
    cat << EOF
Domus OS Build Automation Tool

Usage:
  $(basename "$0") [options]

Options:
  --validate-only   Verify BSP files, configurations, and syntax without building
  --docker          Execute the build inside the hermetic Docker build container
  --clean           Remove all previous build artifacts and temporary files
  --help            Show this help message

Examples:
  $(basename "$0") --validate-only
  $(basename "$0") --docker
  $(basename "$0")
EOF
}

# Parse command line options
while [[ $# -gt 0 ]]; do
    case "$1" in
        --validate-only)
            MODE="validate"
            shift
            ;;
        --docker)
            USE_DOCKER=true
            shift
            ;;
        --clean)
            MODE="clean"
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

# Validation mode: check file presence and integrity
validate_bsp() {
    echo "=== Validating Domus OS BSP Configuration ==="

    local required_files=(
        "${OS_DIR}/configs/domus_rpi4_defconfig"
        "${OS_DIR}/board/raspberrypi4-64/genimage.cfg"
        "${OS_DIR}/board/raspberrypi4-64/config.txt"
        "${OS_DIR}/board/raspberrypi4-64/cmdline.txt"
        "${OS_DIR}/board/raspberrypi4-64/boot.cmd"
        "${OS_DIR}/board/raspberrypi4-64/post-build.sh"
        "${OS_DIR}/board/raspberrypi4-64/post-image.sh"
        "${OS_DIR}/build-container/Dockerfile"
    )

    for f in "${required_files[@]}"; do
        if [ ! -f "$f" ]; then
            echo "ERROR: Missing required BSP file: $f"
            return 1
        fi
        echo "  [OK] $(basename "$f")"
    done

    # Verify bash scripts have execute permissions or valid syntax
    bash -n "${OS_DIR}/board/raspberrypi4-64/post-build.sh"
    bash -n "${OS_DIR}/board/raspberrypi4-64/post-image.sh"
    bash -n "${OS_DIR}/scripts/build.sh"
    echo "  [OK] Shell scripts syntax validated successfully."

    # Validate defconfig critical flags
    local defconfig="${OS_DIR}/configs/domus_rpi4_defconfig"
    grep -q "BR2_aarch64=y" "$defconfig" || { echo "ERROR: BR2_aarch64=y missing"; return 1; }
    grep -q "BR2_cortex_a72=y" "$defconfig" || { echo "ERROR: BR2_cortex_a72=y missing"; return 1; }
    grep -q "BR2_INIT_SYSTEMD=y" "$defconfig" || { echo "ERROR: BR2_INIT_SYSTEMD=y missing"; return 1; }
    grep -q "BR2_TARGET_ROOTFS_SQUASHFS=y" "$defconfig" || { echo "ERROR: SquashFS missing"; return 1; }
    echo "  [OK] Defconfig parameters verified."

    # Validate genimage.cfg GPT table and partition names
    local genimage="${OS_DIR}/board/raspberrypi4-64/genimage.cfg"
    grep -q 'partition-table-type = "gpt"' "$genimage" || { echo "ERROR: GPT missing in genimage"; return 1; }
    grep -q 'partition domus-boot' "$genimage" || { echo "ERROR: domus-boot partition missing"; return 1; }
    grep -q 'partition domus-root-a' "$genimage" || { echo "ERROR: domus-root-a partition missing"; return 1; }
    grep -q 'partition domus-root-b' "$genimage" || { echo "ERROR: domus-root-b partition missing"; return 1; }
    grep -q 'partition domus-data' "$genimage" || { echo "ERROR: domus-data partition missing"; return 1; }
    echo "  [OK] Genimage GPT partition layout verified."

    echo "=== Domus OS BSP Validation PASSED ==="
    return 0
}

# Clean mode
clean_build() {
    echo "Cleaning build artifacts in ${BUILD_WORK_DIR}..."
    rm -rf "${BUILD_WORK_DIR}"
    echo "Clean completed."
}

# Run build inside Docker
run_in_docker() {
    echo "=== Preparing Docker Build Environment for Domus OS ==="
    docker build -t "${DOCKER_IMAGE}" "${OS_DIR}/build-container"
    
    echo "=== Launching Buildroot Build inside Docker ==="
    docker run --rm -it \
        -v "${REPO_ROOT}:/workspace" \
        -w "/workspace" \
        "${DOCKER_IMAGE}" \
        ./os/scripts/build.sh
}

# Native build execution
run_build() {
    validate_bsp

    mkdir -p "${BUILD_WORK_DIR}"
    cd "${BUILD_WORK_DIR}"

    # Download Buildroot tarball if not present
    if [ ! -d "buildroot-${BUILDROOT_VERSION}" ]; then
        if [ ! -f "${BUILDROOT_TAR}" ]; then
            echo "Downloading Buildroot ${BUILDROOT_VERSION}..."
            wget -q --show-progress -c "${BUILDROOT_URL}"
        fi
        echo "Extracting Buildroot ${BUILDROOT_VERSION}..."
        tar -xf "${BUILDROOT_TAR}"
    fi

    local BR_DIR="${BUILD_WORK_DIR}/buildroot-${BUILDROOT_VERSION}"
    cd "${BR_DIR}"

    # Copy board files and defconfig into Buildroot tree
    echo "Configuring Buildroot with domus_rpi4_defconfig..."
    mkdir -p board/raspberrypi4-64
    cp -rf "${OS_DIR}/board/raspberrypi4-64/"* board/raspberrypi4-64/
    cp -f "${OS_DIR}/configs/domus_rpi4_defconfig" configs/domus_rpi4_defconfig

    make domus_rpi4_defconfig
    echo "Starting compilation (this may take 20-40 minutes on initial run)..."
    make -j"$(nproc)"

    echo "=== Build Completed Successfully ==="
    echo "Output image: ${BR_DIR}/output/images/domus-os-rpi4.img"
    echo "SHA256 sum:   ${BR_DIR}/output/images/domus-os-rpi4.img.sha256"
}

# Dispatch based on arguments
if [ "${MODE}" = "validate" ]; then
    validate_bsp
    exit 0
elif [ "${MODE}" = "clean" ]; then
    clean_build
    exit 0
elif [ "${USE_DOCKER}" = true ]; then
    run_in_docker
    exit 0
else
    run_build
    exit 0
fi
