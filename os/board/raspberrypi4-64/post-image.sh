#!/usr/bin/env bash
# ─── Domus OS - Buildroot Post-Image Hook ─────────────────────────────────────
# Compiles U-Boot boot script and generates the complete GPT flashable image.
set -euo pipefail

BOARD_DIR="$(dirname "${BASH_SOURCE[0]}")"
BINARIES_DIR="${1:-output/images}"
GENIMAGE_CFG="${BOARD_DIR}/genimage.cfg"
GENIMAGE_TMP="${BUILD_DIR:-output/build}/genimage.tmp"

echo "=== Domus OS Post-Image: Processing binaries in ${BINARIES_DIR} ==="

# 1. Compile U-Boot boot.cmd into boot.scr using mkimage
if command -v mkimage >/dev/null 2>&1; then
    echo "Compiling U-Boot script with host mkimage..."
    mkimage -A arm64 -O linux -T script -C none -a 0 -e 0 \
        -n "Domus OS Boot Script" \
        -d "${BOARD_DIR}/boot.cmd" \
        "${BINARIES_DIR}/boot.scr"
elif [ -x "${HOST_DIR:-output/host}/bin/mkimage" ]; then
    echo "Compiling U-Boot script with Buildroot host-mkimage..."
    "${HOST_DIR}/bin/mkimage" -A arm64 -O linux -T script -C none -a 0 -e 0 \
        -n "Domus OS Boot Script" \
        -d "${BOARD_DIR}/boot.cmd" \
        "${BINARIES_DIR}/boot.scr"
else
    echo "WARNING: mkimage tool not found; copying raw boot.cmd as fallback boot.scr"
    cp -f "${BOARD_DIR}/boot.cmd" "${BINARIES_DIR}/boot.scr"
fi

# 2. Ensure config.txt and cmdline.txt are present in BINARIES_DIR
cp -f "${BOARD_DIR}/config.txt" "${BINARIES_DIR}/config.txt"
cp -f "${BOARD_DIR}/cmdline.txt" "${BINARIES_DIR}/cmdline.txt"

# 3. Clean up genimage temporary directory
rm -rf "${GENIMAGE_TMP}"
mkdir -p "${GENIMAGE_TMP}"

# 4. Invoke genimage to assemble the final GPT raw disk image
echo "Running genimage to build domus-os-rpi4.img..."
GENIMAGE_BIN="genimage"
if [ -x "${HOST_DIR:-output/host}/bin/genimage" ]; then
    GENIMAGE_BIN="${HOST_DIR}/bin/genimage"
fi

"${GENIMAGE_BIN}" \
    --rootpath "${TARGET_DIR:-output/target}" \
    --tmppath "${GENIMAGE_TMP}" \
    --inputpath "${BINARIES_DIR}" \
    --outputpath "${BINARIES_DIR}" \
    --config "${GENIMAGE_CFG}"

# 5. Generate SHA256 checksum for the assembled image if present
FINAL_IMAGE="${BINARIES_DIR}/domus-os-rpi4.img"
if [ -f "${FINAL_IMAGE}" ]; then
    echo "Generating SHA256 checksum for ${FINAL_IMAGE}..."
    (cd "${BINARIES_DIR}" && sha256sum "domus-os-rpi4.img" > "domus-os-rpi4.img.sha256")
    echo "Generated: ${FINAL_IMAGE}.sha256"
fi

echo "=== Domus OS Post-Image: Image Assembly Completed Successfully ==="
