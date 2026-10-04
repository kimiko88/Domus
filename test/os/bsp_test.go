package os_test

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMS01_BSP_FilesExist verifies that all required files for Milestone MS-OS-01
// (Buildroot Board Support Package for Raspberry Pi 4 ARM64) are present in the repository.
func TestMS01_BSP_FilesExist(t *testing.T) {
	requiredFiles := []string{
		"os/configs/domus_rpi4_defconfig",
		"os/board/raspberrypi4-64/genimage.cfg",
		"os/board/raspberrypi4-64/config.txt",
		"os/board/raspberrypi4-64/cmdline.txt",
		"os/board/raspberrypi4-64/boot.cmd",
		"os/board/raspberrypi4-64/post-build.sh",
		"os/board/raspberrypi4-64/post-image.sh",
		"os/build-container/Dockerfile",
		"os/scripts/build.sh",
		"docs/os/MS-OS-01-BSP-GUIDE.md",
	}

	// Locate repo root relative to test file or working directory
	baseDir := findRepoRoot(t)

	for _, relPath := range requiredFiles {
		fullPath := filepath.Join(baseDir, relPath)
		info, err := os.Stat(fullPath)
		if err != nil {
			t.Errorf("expected BSP file missing: %s (err: %v)", relPath, err)
			continue
		}
		if info.IsDir() {
			t.Errorf("expected file but got directory: %s", relPath)
		}
		if info.Size() == 0 {
			t.Errorf("file is empty: %s", relPath)
		}
	}
}

// TestMS01_Defconfig_Validation verifies that domus_rpi4_defconfig contains all
// critical Buildroot configuration keys for Raspberry Pi 4 ARM64, systemd, and A/B partitions.
func TestMS01_Defconfig_Validation(t *testing.T) {
	baseDir := findRepoRoot(t)
	defconfigPath := filepath.Join(baseDir, "os", "configs", "domus_rpi4_defconfig")

	content, err := os.ReadFile(defconfigPath)
	if err != nil {
		t.Fatalf("failed to read defconfig: %v", err)
	}

	configStr := string(content)

	requiredConfigs := []struct {
		key      string
		expected string
		desc     string
	}{
		{"BR2_aarch64", "y", "Target 64-bit ARM architecture"},
		{"BR2_cortex_a72", "y", "Cortex-A72 CPU tuning for BCM2711"},
		{"BR2_INIT_SYSTEMD", "y", "systemd as init system"},
		{"BR2_PACKAGE_RPI_FIRMWARE", "y", "Raspberry Pi firmware binaries"},
		{"BR2_TARGET_UBOOT", "y", "U-Boot bootloader integration"},
		{"BR2_TARGET_UBOOT_BOARD_DEFCONFIG", "\"rpi_4\"", "U-Boot board configuration for RPi4"},
		{"BR2_TARGET_ROOTFS_SQUASHFS", "y", "SquashFS read-only root filesystem"},
		{"BR2_PACKAGE_HOST_GENIMAGE", "y", "Host genimage tool for partition image generation"},
		{"BR2_PACKAGE_HOST_DOSFSTOOLS", "y", "Host dosfstools for FAT32 boot partition"},
		{"BR2_PACKAGE_HOST_MTOOLS", "y", "Host mtools for FAT filesystem manipulation"},
		{"BR2_PACKAGE_HOST_UBOOT_TOOLS", "y", "Host mkimage for boot.scr compilation"},
		{"BR2_PACKAGE_E2FSPROGS", "y", "e2fsprogs for ext4 data partition"},
		{"BR2_PACKAGE_DOSFSTOOLS", "y", "dosfstools for boot partition management"},
	}

	for _, req := range requiredConfigs {
		pattern := req.key + "=" + req.expected
		if !strings.Contains(configStr, pattern) {
			t.Errorf("missing or incorrect configuration %q (expected %s): %s", req.key, req.expected, req.desc)
		}
	}

	// Verify post-build and post-image script paths
	if !strings.Contains(configStr, "post-build.sh") {
		t.Errorf("defconfig must specify post-build.sh hook")
	}
	if !strings.Contains(configStr, "post-image.sh") {
		t.Errorf("defconfig must specify post-image.sh hook")
	}
}

// TestMS01_Genimage_PartitionScheme verifies that genimage.cfg defines the exact
// 4-partition GPT layout (BOOT FAT32, ROOT_A SquashFS, ROOT_B SquashFS, DATA ext4).
func TestMS01_Genimage_PartitionScheme(t *testing.T) {
	baseDir := findRepoRoot(t)
	genimagePath := filepath.Join(baseDir, "os", "board", "raspberrypi4-64", "genimage.cfg")

	content, err := os.ReadFile(genimagePath)
	if err != nil {
		t.Fatalf("failed to read genimage.cfg: %v", err)
	}

	cfg := string(content)

	// Check partition table type
	if !strings.Contains(cfg, `partition-table-type = "gpt"`) {
		t.Errorf("genimage.cfg must specify GPT partition table: partition-table-type = \"gpt\"")
	}

	// Check output image name
	if !strings.Contains(cfg, `domus-os-rpi4.img`) {
		t.Errorf("genimage.cfg must generate domus-os-rpi4.img")
	}

	// Verify partition definitions
	requiredPartitions := []string{
		"domus-boot",
		"domus-root-a",
		"domus-root-b",
		"domus-data",
	}

	for _, part := range requiredPartitions {
		if !strings.Contains(cfg, `partition `+part) {
			t.Errorf("genimage.cfg missing partition definition: %s", part)
		}
	}

	// Verify filesystem types
	if !strings.Contains(cfg, "image boot.vfat") {
		t.Errorf("genimage.cfg missing boot.vfat definition")
	}
	if !strings.Contains(cfg, "image data.ext4") {
		t.Errorf("genimage.cfg missing data.ext4 definition")
	}
}

// TestMS01_ConfigTxt_RPi4Firmware verifies config.txt contains essential RPi4 64-bit parameters.
func TestMS01_ConfigTxt_RPi4Firmware(t *testing.T) {
	baseDir := findRepoRoot(t)
	configTxtPath := filepath.Join(baseDir, "os", "board", "raspberrypi4-64", "config.txt")

	content, err := os.ReadFile(configTxtPath)
	if err != nil {
		t.Fatalf("failed to read config.txt: %v", err)
	}

	txt := string(content)

	requiredParams := []string{
		"arm_64bit=1",
		"kernel=u-boot.bin",
		"enable_uart=1",
	}

	for _, p := range requiredParams {
		if !strings.Contains(txt, p) {
			t.Errorf("config.txt missing required parameter: %s", p)
		}
	}
}

// TestMS01_CmdlineTxt_KernelParams verifies cmdline.txt kernel command line settings.
func TestMS01_CmdlineTxt_KernelParams(t *testing.T) {
	baseDir := findRepoRoot(t)
	cmdlinePath := filepath.Join(baseDir, "os", "board", "raspberrypi4-64", "cmdline.txt")

	content, err := os.ReadFile(cmdlinePath)
	if err != nil {
		t.Fatalf("failed to read cmdline.txt: %v", err)
	}

	line := strings.TrimSpace(string(content))

	// Ensure it is a single line (kernel cmdline requirement)
	if strings.Contains(line, "\n") {
		t.Errorf("cmdline.txt must be a single line without newline breaks")
	}

	requiredTokens := []string{
		"rootfstype=squashfs",
		"ro",
		"rootwait",
		"cgroup_enable=cpuset",
		"cgroup_memory=1",
	}

	for _, token := range requiredTokens {
		if !strings.Contains(line, token) {
			t.Errorf("cmdline.txt missing required token: %s", token)
		}
	}
}

// TestMS01_BootCmd_Failover verifies that boot.cmd implements U-Boot A/B slot selection
// and bootcounter failover logic.
func TestMS01_BootCmd_Failover(t *testing.T) {
	baseDir := findRepoRoot(t)
	bootCmdPath := filepath.Join(baseDir, "os", "board", "raspberrypi4-64", "boot.cmd")

	content, err := os.ReadFile(bootCmdPath)
	if err != nil {
		t.Fatalf("failed to read boot.cmd: %v", err)
	}

	script := string(content)

	requiredKeywords := []string{
		"boot_slot",
		"bootcount",
		"bootlimit",
		"domus-root-a",
		"domus-root-b",
		"booti",
	}

	for _, kw := range requiredKeywords {
		if !strings.Contains(script, kw) {
			t.Errorf("boot.cmd missing keyword/logic for: %s", kw)
		}
	}
}

// TestMS01_ShellScripts_HeaderSafety verifies post-build.sh, post-image.sh, and build.sh
// start with standard POSIX safe error flags (set -e or set -euo pipefail).
func TestMS01_ShellScripts_HeaderSafety(t *testing.T) {
	baseDir := findRepoRoot(t)
	scripts := []string{
		"os/board/raspberrypi4-64/post-build.sh",
		"os/board/raspberrypi4-64/post-image.sh",
		"os/scripts/build.sh",
	}

	for _, s := range scripts {
		fullPath := filepath.Join(baseDir, s)
		f, err := os.Open(fullPath)
		if err != nil {
			t.Fatalf("failed to open script %s: %v", s, err)
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		hasShebang := false
		hasSafeFlags := false

		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, "#!") {
				hasShebang = true
			}
			if strings.HasPrefix(line, "set -e") {
				hasSafeFlags = true
				break
			}
		}
		if err := scanner.Err(); err != nil {
			t.Errorf("scanner error reading script %s: %v", s, err)
		}

		if !hasShebang {
			t.Errorf("script %s missing shebang line (#!/usr/bin/env bash or #!/bin/sh)", s)
		}
		if !hasSafeFlags {
			t.Errorf("script %s missing safe execution flags (set -e...)", s)
		}
	}
}

func findRepoRoot(t *testing.T) string {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("cannot get working directory: %v", err)
	}

	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("repo root not found starting from %s", wd)
		}
		dir = parent
	}
}
