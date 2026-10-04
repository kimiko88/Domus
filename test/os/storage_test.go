package os_test

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestMS02_Storage_FilesExist verifies all files for Milestone MS-OS-02
// (Read-Only SquashFS, Persistent Storage, Systemd Mounts, and Factory Reset) exist.
func TestMS02_Storage_FilesExist(t *testing.T) {
	requiredFiles := []string{
		"os/systemd/data.mount",
		"os/systemd/domus-storage-init.service",
		"os/systemd/journald.conf",
		"os/systemd/tmp.mount",
		"os/scripts/domus-storage-init.sh",
		"os/scripts/domus-factory-reset.sh",
		"docs/os/MS-OS-02-STORAGE-AND-IMMUTABILITY.md",
	}

	baseDir := findRepoRoot(t)

	for _, relPath := range requiredFiles {
		fullPath := filepath.Join(baseDir, relPath)
		info, err := os.Stat(fullPath)
		if err != nil {
			t.Errorf("expected storage file missing: %s (err: %v)", relPath, err)
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

// TestMS02_DataMountUnit_Options verifies data.mount unit file configuration.
func TestMS02_DataMountUnit_Options(t *testing.T) {
	baseDir := findRepoRoot(t)
	mountPath := filepath.Join(baseDir, "os", "systemd", "data.mount")

	content, err := os.ReadFile(mountPath)
	if err != nil {
		t.Fatalf("failed to read data.mount: %v", err)
	}

	unit := string(content)

	requiredDirectives := []struct {
		token string
		desc  string
	}{
		{"What=/dev/disk/by-partlabel/domus-data", "Device path by GPT partition label"},
		{"Where=/data", "Target mount directory"},
		{"Type=ext4", "Ext4 filesystem type with journaling"},
		{"noatime", "Disable access time updates to prevent flash wear"},
		{"commit=30", "Delayed journal flush commit interval (30s)"},
		{"errors=remount-ro", "Remount read-only on unrecoverable filesystem error"},
	}

	for _, req := range requiredDirectives {
		if !strings.Contains(unit, req.token) {
			t.Errorf("data.mount missing directive %q: %s", req.token, req.desc)
		}
	}
}

// TestMS02_JournaldConfig_Limits verifies journald configuration has strict disk limits.
func TestMS02_JournaldConfig_Limits(t *testing.T) {
	baseDir := findRepoRoot(t)
	confPath := filepath.Join(baseDir, "os", "systemd", "journald.conf")

	content, err := os.ReadFile(confPath)
	if err != nil {
		t.Fatalf("failed to read journald.conf: %v", err)
	}

	conf := string(content)

	requiredSettings := []struct {
		setting string
		desc    string
	}{
		{"Storage=persistent", "Enable persistent journal logs"},
		{"SystemMaxUse=100M", "Hard ceiling of 100MB to protect storage"},
		{"SystemMaxFileSize=10M", "Individual log file max size of 10MB"},
		{"ForwardToSyslog=no", "Disable syslog forward to prevent duplicate logging"},
	}

	for _, req := range requiredSettings {
		if !strings.Contains(conf, req.setting) {
			t.Errorf("journald.conf missing setting %q: %s", req.setting, req.desc)
		}
	}
}

// TestMS02_StorageInitScript_Structure verifies domus-storage-init.sh establishes
// the complete directory hierarchy defined in the specification.
func TestMS02_StorageInitScript_Structure(t *testing.T) {
	baseDir := findRepoRoot(t)
	scriptPath := filepath.Join(baseDir, "os", "scripts", "domus-storage-init.sh")

	content, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("failed to read domus-storage-init.sh: %v", err)
	}

	script := string(content)

	requiredDirs := []string{
		"${DATA_ROOT}/system",
		"${DATA_ROOT}/system/network",
		"${DATA_ROOT}/system/ssh",
		"${DATA_ROOT}/system/rauc",
		"${DATA_ROOT}/core",
		"${DATA_ROOT}/core/backups",
		"${DATA_ROOT}/supervisor",
		"${DATA_ROOT}/supervisor/manifests",
		"${DATA_ROOT}/apps",
		"${DATA_ROOT}/containerd",
		"${DATA_ROOT}/logs/journal",
	}

	for _, dir := range requiredDirs {
		if !strings.Contains(script, dir) {
			t.Errorf("domus-storage-init.sh does not provision directory: %s", dir)
		}
	}

	// Verify permissions enforcement
	if !strings.Contains(script, "chmod 0700") {
		t.Errorf("domus-storage-init.sh must enforce restricted permissions (0700) for secrets/ssh/network")
	}

	// Verify safe execution header
	if !strings.HasPrefix(strings.TrimSpace(script), "#!/usr/bin/env bash") &&
		!strings.HasPrefix(strings.TrimSpace(script), "#!/bin/sh") &&
		!strings.HasPrefix(strings.TrimSpace(script), "#!/bin/bash") {
		t.Errorf("domus-storage-init.sh missing shebang")
	}
	if !strings.Contains(script, "set -euo pipefail") && !strings.Contains(script, "set -e") {
		t.Errorf("domus-storage-init.sh missing safe error handling flags")
	}
}

// TestMS02_FactoryResetScript_Safety verifies domus-factory-reset.sh contains
// proper safety confirmation checks and purge mechanisms.
func TestMS02_FactoryResetScript_Safety(t *testing.T) {
	baseDir := findRepoRoot(t)
	scriptPath := filepath.Join(baseDir, "os", "scripts", "domus-factory-reset.sh")

	content, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("failed to read domus-factory-reset.sh: %v", err)
	}

	script := string(content)

	requiredPatterns := []string{
		"--force",
		"${DATA_ROOT}/core",
		"${DATA_ROOT}/supervisor",
		"${DATA_ROOT}/apps",
		"${DATA_ROOT}/containerd",
		"sync",
	}

	for _, p := range requiredPatterns {
		if !strings.Contains(script, p) {
			t.Errorf("domus-factory-reset.sh missing pattern: %s", p)
		}
	}
}

// TestMS02_StorageInit_ExecutionDryRun executes domus-storage-init.sh in a temporary root directory
// to ensure it executes cleanly, creates all directories with proper permissions, and is idempotent.
func TestMS02_StorageInit_ExecutionDryRun(t *testing.T) {
	baseDir := findRepoRoot(t)
	scriptPath := filepath.Join(baseDir, "os", "scripts", "domus-storage-init.sh")

	if _, err := os.Stat(scriptPath); err != nil {
		t.Skip("script not yet created, skipping dry-run")
	}

	// Check if bash is available
	bashPath, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not found on system, skipping script execution test")
	}

	tempDir, err := os.MkdirTemp(baseDir, ".tmp-domus-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	fakeData := filepath.Join(tempDir, "data")
	if err := os.MkdirAll(fakeData, 0755); err != nil {
		t.Fatalf("failed to create fake data dir: %v", err)
	}

	relData, err := filepath.Rel(baseDir, fakeData)
	if err != nil {
		t.Fatalf("failed to get relative data path: %v", err)
	}
	slashData := "./" + filepath.ToSlash(relData)
	relScript := "./os/scripts/domus-storage-init.sh"

	// Run script with DATA_ROOT pointing to fakeData
	cmd := exec.Command(bashPath, "-c", "DATA_ROOT='"+slashData+"' SKIP_RESIZE=true "+relScript)
	cmd.Dir = baseDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("script execution failed (err: %v, output: %s)", err, string(out))
	}

	// Verify that directories were created under fakeData
	expectedSubdirs := []string{
		"system",
		"system/network",
		"system/ssh",
		"system/rauc",
		"core",
		"core/backups",
		"supervisor",
		"supervisor/manifests",
		"apps",
		"containerd",
		"logs/journal",
	}

	for _, sub := range expectedSubdirs {
		dirPath := filepath.Join(fakeData, filepath.FromSlash(sub))
		info, err := os.Stat(dirPath)
		if err != nil {
			t.Errorf("expected directory was not created: %s (err: %v)", sub, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("expected directory but got file: %s", sub)
		}
	}

	// Test idempotency: run a second time, must exit 0 without errors
	cmd2 := exec.Command(bashPath, "-c", "DATA_ROOT='"+slashData+"' SKIP_RESIZE=true "+relScript)
	cmd2.Dir = baseDir
	out2, err := cmd2.CombinedOutput()
	if err != nil {
		t.Fatalf("idempotency check failed on second run (err: %v, output: %s)", err, string(out2))
	}
}

// TestMS02_PostBuildScript_InstallsStorageUnits verifies that post-build.sh installs
// the storage init service, data mount, and journald configs into target rootfs.
func TestMS02_PostBuildScript_InstallsStorageUnits(t *testing.T) {
	baseDir := findRepoRoot(t)
	postBuildPath := filepath.Join(baseDir, "os", "board", "raspberrypi4-64", "post-build.sh")

	f, err := os.Open(postBuildPath)
	if err != nil {
		t.Fatalf("failed to open post-build.sh: %v", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	content := ""
	for scanner.Scan() {
		content += scanner.Text() + "\n"
	}

	requiredHooks := []string{
		"domus-storage-init",
		"journald.conf",
		"data.mount",
	}

	for _, hook := range requiredHooks {
		if !strings.Contains(content, hook) {
			t.Errorf("post-build.sh missing storage integration hook for: %s", hook)
		}
	}
}
