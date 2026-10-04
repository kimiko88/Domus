# ─── Domus OS - U-Boot Boot Script (A/B Redundancy & Failover) ───────────────
# This script is compiled to boot.scr by mkimage and executed by U-Boot on RPi4.

echo "================================================="
echo "===            Domus OS Bootloader            ==="
echo "================================================="

# Set environment defaults if not already present
if test -z "${boot_slot}"; then
    setenv boot_slot "a"
fi

if test -z "${bootcount}"; then
    setenv bootcount 0
fi

if test -z "${bootlimit}"; then
    setenv bootlimit 3
fi

echo "Active Boot Slot: ${boot_slot}"
echo "Boot Attempts: ${bootcount} / Limit: ${bootlimit}"

# Check for bootcounter exceed (kernel panic or service failure on previous tries)
if test ${bootcount} -ge ${bootlimit}; then
    echo "WARNING: Bootlimit reached! Triggering automatic A/B slot failover..."
    if test "${boot_slot}" = "a"; then
        setenv boot_slot "b"
    else
        setenv boot_slot "a"
    fi
    setenv bootcount 0
    saveenv
    echo "Switched to fallback slot: ${boot_slot}"
fi

# Increment boot counter (will be reset to 0 by domus-supervisor via RAUC upon healthcheck OK)
setexpr bootcount ${bootcount} + 1
saveenv

# Determine target rootfs partition by label
if test "${boot_slot}" = "a"; then
    setenv rootpart "domus-root-a"
else
    setenv rootpart "domus-root-b"
fi

echo "Booting rootfs from partition: ${rootpart}"

# Set kernel boot arguments with selected root partition
setenv bootargs "console=serial0,115200 console=tty1 root=/dev/disk/by-partlabel/${rootpart} rootfstype=squashfs ro rootwait quiet audit=0 cgroup_enable=cpuset cgroup_memory=1 cgroup_enable=memory systemd.show_status=auto"

# Load Linux Kernel Image and Device Tree into RAM addresses
# Addresses for RPi 4 64-bit standard memory layout:
# kernel_addr_r = 0x00080000 / 0x00200000, fdt_addr_r = 0x02600000
echo "Loading Linux Kernel and Device Tree from boot partition..."
load mmc 0:1 ${kernel_addr_r} Image
load mmc 0:1 ${fdt_addr_r} bcm2711-rpi-4-b.dtb

# Execute AArch64 Linux kernel
echo "Executing Domus OS Linux Kernel..."
booti ${kernel_addr_r} - ${fdt_addr_r}
