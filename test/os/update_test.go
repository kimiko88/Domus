package os_test

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMS03_Update_FilesExist verifies all files for Milestone MS-OS-03
// (Dual-Slot A/B Update Engine with RAUC, U-Boot bootcounter, and x.509 PKI) are present.
func TestMS03_Update_FilesExist(t *testing.T) {
	requiredFiles := []string{
		"os/rauc/system.conf",
		"os/rauc/domus-os.raucm",
		"os/rauc/certs/ca.cert.pem",
		"os/rauc/certs/development.cert.pem",
		"os/rauc/certs/development.key.pem",
		"os/board/raspberrypi4-64/uboot.env.txt",
		"os/board/raspberrypi4-64/fw_env.config",
		"os/scripts/generate-rauc-keys.sh",
		"os/scripts/build-rauc-bundle.sh",
		"os/scripts/domus-update-agent.sh",
		"docs/os/MS-OS-03-AB-UPDATE-ENGINE.md",
	}

	baseDir := findRepoRoot(t)

	for _, relPath := range requiredFiles {
		fullPath := filepath.Join(baseDir, relPath)
		info, err := os.Stat(fullPath)
		if err != nil {
			t.Errorf("expected update engine file missing: %s (err: %v)", relPath, err)
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

// TestMS03_RaucSystemConf_ValidSlots verifies that system.conf specifies both slot A
// and slot B rootfs partitions, uboot bootloader, and x.509 keyring.
func TestMS03_RaucSystemConf_ValidSlots(t *testing.T) {
	baseDir := findRepoRoot(t)
	confPath := filepath.Join(baseDir, "os", "rauc", "system.conf")

	content, err := os.ReadFile(confPath)
	if err != nil {
		t.Fatalf("failed to read system.conf: %v", err)
	}

	cfg := string(content)

	requiredTokens := []struct {
		token string
		desc  string
	}{
		{"compatible=domus-os-rpi4", "Hardware/OS compatibility identifier"},
		{"bootloader=uboot", "U-Boot bootloader integration"},
		{"[slot.rootfs.0]", "Slot 0 definition"},
		{"device=/dev/disk/by-partlabel/domus-root-a", "Slot A partition device"},
		{"bootname=a", "Slot A bootname matching U-Boot slot variable"},
		{"[slot.rootfs.1]", "Slot 1 definition"},
		{"device=/dev/disk/by-partlabel/domus-root-b", "Slot B partition device"},
		{"bootname=b", "Slot B bootname matching U-Boot slot variable"},
		{"type=raw", "Raw squashfs image write mode"},
		{"path=/etc/rauc/ca.cert.pem", "Keyring CA certificate path"},
		{"statusfile=/data/system/rauc/status", "Persistent status file on /data partition"},
	}

	for _, req := range requiredTokens {
		if !strings.Contains(cfg, req.token) {
			t.Errorf("system.conf missing token %q: %s", req.token, req.desc)
		}
	}
}

// TestMS03_X509_CertificateValidity parses ca.cert.pem and development.cert.pem
// to verify they are valid, well-formed x.509 PEM certificates.
func TestMS03_X509_CertificateValidity(t *testing.T) {
	baseDir := findRepoRoot(t)

	certFiles := []string{
		"os/rauc/certs/ca.cert.pem",
		"os/rauc/certs/development.cert.pem",
	}

	for _, cf := range certFiles {
		fullPath := filepath.Join(baseDir, cf)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			t.Fatalf("failed to read cert file %s: %v", cf, err)
		}

		block, _ := pem.Decode(data)
		if block == nil {
			t.Fatalf("file %s is not valid PEM format", cf)
		}
		if block.Type != "CERTIFICATE" {
			t.Errorf("file %s block type is %s, expected CERTIFICATE", cf, block.Type)
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			t.Errorf("failed to parse x.509 certificate %s: %v", cf, err)
			continue
		}

		// Verify subject and issuer are non-empty
		if cert.Subject.CommonName == "" {
			t.Errorf("certificate %s has empty CommonName", cf)
		}
	}
}

// TestMS03_RaucManifest_Structure verifies domus-os.raucm template syntax.
func TestMS03_RaucManifest_Structure(t *testing.T) {
	baseDir := findRepoRoot(t)
	manifestPath := filepath.Join(baseDir, "os", "rauc", "domus-os.raucm")

	content, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("failed to read domus-os.raucm: %v", err)
	}

	m := string(content)

	requiredTokens := []string{
		"[package]",
		"compatible=domus-os-rpi4",
		"version=",
		"[image.rootfs]",
		"filename=rootfs.squashfs",
	}

	for _, token := range requiredTokens {
		if !strings.Contains(m, token) {
			t.Errorf("domus-os.raucm missing required token: %s", token)
		}
	}
}

// TestMS03_UBootEnv_Bootcounter verifies default uboot.env.txt contains
// A/B variables and bootcounter limits.
func TestMS03_UBootEnv_Bootcounter(t *testing.T) {
	baseDir := findRepoRoot(t)
	envPath := filepath.Join(baseDir, "os", "board", "raspberrypi4-64", "uboot.env.txt")

	content, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("failed to read uboot.env.txt: %v", err)
	}

	envStr := string(content)

	requiredVars := []string{
		"boot_slot=a",
		"bootcount=0",
		"bootlimit=3",
		"upgrade_available=0",
	}

	for _, v := range requiredVars {
		if !strings.Contains(envStr, v) {
			t.Errorf("uboot.env.txt missing variable: %s", v)
		}
	}
}

// TestMS03_FwEnvConfig_PartitionTarget verifies fw_env.config targets the boot partition.
func TestMS03_FwEnvConfig_PartitionTarget(t *testing.T) {
	baseDir := findRepoRoot(t)
	cfgPath := filepath.Join(baseDir, "os", "board", "raspberrypi4-64", "fw_env.config")

	content, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("failed to read fw_env.config: %v", err)
	}

	cfg := string(content)

	if !strings.Contains(cfg, "/boot/uboot.env") && !strings.Contains(cfg, "domus-boot") {
		t.Errorf("fw_env.config must configure U-Boot environment file location")
	}
}

// TestMS03_UpdateAgent_CLICommands verifies domus-update-agent.sh handles
// install, status, mark-good, and rollback commands.
func TestMS03_UpdateAgent_CLICommands(t *testing.T) {
	baseDir := findRepoRoot(t)
	scriptPath := filepath.Join(baseDir, "os", "scripts", "domus-update-agent.sh")

	content, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("failed to read domus-update-agent.sh: %v", err)
	}

	script := string(content)

	requiredCommands := []string{
		"install",
		"status",
		"mark-good",
		"rollback",
		"rauc status",
	}

	for _, cmd := range requiredCommands {
		if !strings.Contains(script, cmd) {
			t.Errorf("domus-update-agent.sh missing command handler for: %s", cmd)
		}
	}
}

// TestMS03_BuildBundleScript_Flags verifies build-rauc-bundle.sh has signing parameters.
func TestMS03_BuildBundleScript_Flags(t *testing.T) {
	baseDir := findRepoRoot(t)
	scriptPath := filepath.Join(baseDir, "os", "scripts", "build-rauc-bundle.sh")

	content, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("failed to read build-rauc-bundle.sh: %v", err)
	}

	script := string(content)

	requiredTokens := []string{
		"rauc bundle",
		"--cert",
		"--key",
		".raucb",
		"sha256sum",
	}

	for _, token := range requiredTokens {
		if !strings.Contains(script, token) {
			t.Errorf("build-rauc-bundle.sh missing token: %s", token)
		}
	}
}
