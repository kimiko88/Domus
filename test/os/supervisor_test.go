package os_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMS04_Supervisor_FilesExist verifies that all files for Milestone MS-OS-04
// (containerd runtime configuration, supervisor service, and Go daemon entrypoint) exist.
func TestMS04_Supervisor_FilesExist(t *testing.T) {
	requiredFiles := []string{
		"os/configs/containerd-config.toml",
		"os/systemd/containerd.service",
		"os/systemd/domus-supervisor.service",
		"cmd/domus-supervisor/main.go",
		"docs/os/MS-OS-04-SUPERVISOR-AND-CONTAINERD.md",
	}

	baseDir := findRepoRoot(t)

	for _, relPath := range requiredFiles {
		fullPath := filepath.Join(baseDir, relPath)
		info, err := os.Stat(fullPath)
		if err != nil {
			t.Errorf("expected supervisor/containerd file missing: %s (err: %v)", relPath, err)
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

// TestMS04_ContainerdConfig_Directives verifies containerd configuration has
// cgroups v2, persistent OCI root on /data, and volatile state in /run.
func TestMS04_ContainerdConfig_Directives(t *testing.T) {
	baseDir := findRepoRoot(t)
	cfgPath := filepath.Join(baseDir, "os", "configs", "containerd-config.toml")

	content, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("failed to read containerd-config.toml: %v", err)
	}

	cfg := string(content)

	requiredTokens := []struct {
		token string
		desc  string
	}{
		{"root = \"/data/containerd/root\"", "Persistent image and layer storage on /data"},
		{"state = \"/run/containerd\"", "Volatile runtime state on tmpfs"},
		{"address = \"/run/containerd/containerd.sock\"", "Default Unix domain socket"},
		{"SystemdCgroup = true", "Systemd cgroups v2 integration"},
		{"overlayfs", "OverlayFS storage snapshotter"},
	}

	for _, req := range requiredTokens {
		if !strings.Contains(cfg, req.token) {
			t.Errorf("containerd-config.toml missing directive %q: %s", req.token, req.desc)
		}
	}
}

// TestMS04_SupervisorService_Dependencies verifies domus-supervisor.service
// correctly orders after containerd, data.mount, and domus-storage-init.
func TestMS04_SupervisorService_Dependencies(t *testing.T) {
	baseDir := findRepoRoot(t)
	unitPath := filepath.Join(baseDir, "os", "systemd", "domus-supervisor.service")

	content, err := os.ReadFile(unitPath)
	if err != nil {
		t.Fatalf("failed to read domus-supervisor.service: %v", err)
	}

	unit := string(content)

	requiredDirectives := []string{
		"Requires=containerd.service data.mount",
		"After=containerd.service data.mount domus-storage-init.service",
		"ExecStart=/usr/bin/domus-supervisor",
		"Restart=always",
	}

	for _, req := range requiredDirectives {
		if !strings.Contains(unit, req) {
			t.Errorf("domus-supervisor.service missing directive: %s", req)
		}
	}
}

// TestMS04_ContainerdService_StorageDependency verifies containerd.service
// strictly depends on data.mount before starting.
func TestMS04_ContainerdService_StorageDependency(t *testing.T) {
	baseDir := findRepoRoot(t)
	unitPath := filepath.Join(baseDir, "os", "systemd", "containerd.service")

	content, err := os.ReadFile(unitPath)
	if err != nil {
		t.Fatalf("failed to read containerd.service: %v", err)
	}

	unit := string(content)

	requiredDirectives := []string{
		"Requires=data.mount",
		"After=network.target local-fs.target data.mount",
		"ExecStart=/usr/bin/containerd --config /etc/containerd/config.toml",
		"Delegate=yes",
	}

	for _, req := range requiredDirectives {
		if !strings.Contains(unit, req) {
			t.Errorf("containerd.service missing directive: %s", req)
		}
	}
}
