package supervisor

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	appIDRegex     = regexp.MustCompile(`^[a-z0-9][a-z0-9-_]{1,63}$`)
	validDeviceReg = regexp.MustCompile(`^/dev/(ttyUSB[0-9]+|ttyACM[0-9]+|serial[0-9]*|gpiochip[0-9]*)$`)
)

// AppManifest represents a verified Domus App configuration.
type AppManifest struct {
	ID              string            `json:"id" yaml:"id"`
	Name            string            `json:"name" yaml:"name"`
	Version         string            `json:"version" yaml:"version"`
	Description     string            `json:"description" yaml:"description"`
	Publisher       string            `json:"publisher,omitempty" yaml:"publisher,omitempty"`
	License         string            `json:"license,omitempty" yaml:"license,omitempty"`
	MinimumDomusVer string            `json:"minimum_domus_version,omitempty" yaml:"minimum_domus_version,omitempty"`
	Image           string            `json:"image" yaml:"image"`
	ImageDigest     string            `json:"image_digest,omitempty" yaml:"image_digest,omitempty"`
	Architecture    []string          `json:"architecture,omitempty" yaml:"architecture,omitempty"`
	Entrypoint      []string          `json:"entrypoint,omitempty" yaml:"entrypoint,omitempty"`
	Environment     map[string]string `json:"environment,omitempty" yaml:"environment,omitempty"`
	Permissions     AppPermissions    `json:"permissions" yaml:"permissions"`
	Resources       AppResources      `json:"resources" yaml:"resources"`
	Healthcheck     *AppHealthcheck   `json:"healthcheck,omitempty" yaml:"healthcheck,omitempty"`
}

// AppPermissions defines declared hardware, network, and storage privileges.
type AppPermissions struct {
	Network NetworkPermission  `json:"network" yaml:"network"`
	Devices []DevicePermission `json:"devices,omitempty" yaml:"devices,omitempty"`
	MQTT    *MQTTPermission    `json:"mqtt,omitempty" yaml:"mqtt,omitempty"`
	Storage []StorageVolume    `json:"storage,omitempty" yaml:"storage,omitempty"`
}

// NetworkPermission configures isolation and port forwarding.
type NetworkPermission struct {
	Mode         string       `json:"mode" yaml:"mode"` // bridge, host, none
	InboundPorts []PortConfig `json:"inbound_ports,omitempty" yaml:"inbound_ports,omitempty"`
	Outbound     bool         `json:"outbound" yaml:"outbound"`
}

// PortConfig defines container port mapping.
type PortConfig struct {
	ContainerPort int    `json:"container_port" yaml:"container_port"`
	HostPort      int    `json:"host_port,omitempty" yaml:"host_port,omitempty"`
	Protocol      string `json:"protocol,omitempty" yaml:"protocol,omitempty"` // tcp, udp
	UIIngress     bool   `json:"ui_ingress,omitempty" yaml:"ui_ingress,omitempty"`
}

// DevicePermission defines an allowed hardware device node.
type DevicePermission struct {
	Path        string `json:"path" yaml:"path"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Optional    bool   `json:"optional,omitempty" yaml:"optional,omitempty"`
}

// MQTTPermission defines scoped MQTT publish/subscribe access.
type MQTTPermission struct {
	ClientID  string   `json:"client_id,omitempty" yaml:"client_id,omitempty"`
	Publish   []string `json:"publish,omitempty" yaml:"publish,omitempty"`
	Subscribe []string `json:"subscribe,omitempty" yaml:"subscribe,omitempty"`
}

// StorageVolume defines an isolated persistent data mount.
type StorageVolume struct {
	Path       string `json:"path" yaml:"path"`
	VolumeName string `json:"volume_name" yaml:"volume_name"`
	QuotaMB    int64  `json:"quota_mb,omitempty" yaml:"quota_mb,omitempty"`
}

// AppResources specifies CPU and RAM boundaries.
type AppResources struct {
	MemoryLimitMB int64   `json:"memory_limit_mb" yaml:"memory_limit_mb"`
	CPULimit      float64 `json:"cpu_limit,omitempty" yaml:"cpu_limit,omitempty"`
	RestartPolicy string  `json:"restart_policy,omitempty" yaml:"restart_policy,omitempty"`
}

// AppHealthcheck defines container health monitoring.
type AppHealthcheck struct {
	Type            string `json:"type" yaml:"type"` // http, tcp, command
	URL             string `json:"url,omitempty" yaml:"url,omitempty"`
	IntervalSeconds int    `json:"interval_seconds,omitempty" yaml:"interval_seconds,omitempty"`
	TimeoutSeconds  int    `json:"timeout_seconds,omitempty" yaml:"timeout_seconds,omitempty"`
	Retries         int    `json:"retries,omitempty" yaml:"retries,omitempty"`
}

// ParseAndValidateManifest parses raw YAML/JSON and enforces strict sandboxing rules.
func ParseAndValidateManifest(data []byte) (*AppManifest, error) {
	var m AppManifest
	// Attempt YAML decode (which naturally handles JSON as a subset)
	if err := yaml.Unmarshal(data, &m); err != nil {
		// Fallback attempt standard JSON
		if jsonErr := json.Unmarshal(data, &m); jsonErr != nil {
			return nil, fmt.Errorf("failed to parse manifest format: %w", err)
		}
	}

	// 1. Validate ID
	if !appIDRegex.MatchString(m.ID) {
		return nil, fmt.Errorf("invalid app ID %q: must match regex ^[a-z0-9][a-z0-9-_]{1,63}$", m.ID)
	}

	// 2. Validate mandatory metadata
	if strings.TrimSpace(m.Name) == "" {
		return nil, errors.New("app name is required")
	}
	if strings.TrimSpace(m.Version) == "" {
		return nil, errors.New("app version is required")
	}
	if strings.TrimSpace(m.Image) == "" {
		return nil, errors.New("app image is required")
	}

	// 3. Validate network mode
	switch m.Permissions.Network.Mode {
	case "bridge", "host", "none":
	default:
		return nil, fmt.Errorf("invalid network mode %q: must be bridge, host, or none", m.Permissions.Network.Mode)
	}

	// 4. Validate device permissions (strict sandboxing)
	for _, dev := range m.Permissions.Devices {
		if !validDeviceReg.MatchString(dev.Path) {
			return nil, fmt.Errorf("forbidden device path %q: only /dev/ttyUSB*, /dev/ttyACM*, /dev/serial*, and /dev/gpiochip* are permitted", dev.Path)
		}
		// Explicitly reject disk devices and kernel memory
		if strings.HasPrefix(dev.Path, "/dev/sd") || strings.HasPrefix(dev.Path, "/dev/mmcblk") ||
			strings.HasPrefix(dev.Path, "/dev/nvme") || strings.HasPrefix(dev.Path, "/dev/mem") {
			return nil, fmt.Errorf("security violation: storage and memory devices cannot be mapped (%s)", dev.Path)
		}
	}

	// 5. Validate storage volumes
	for _, vol := range m.Permissions.Storage {
		if strings.TrimSpace(vol.Path) == "" || strings.TrimSpace(vol.VolumeName) == "" {
			return nil, errors.New("storage volume must specify both path and volume_name")
		}
		if vol.Path == "/" || vol.Path == "/etc" || vol.Path == "/bin" || vol.Path == "/sbin" {
			return nil, fmt.Errorf("forbidden storage mount path: %s", vol.Path)
		}
	}

	// 6. Validate resource limits
	if m.Resources.MemoryLimitMB < 32 || m.Resources.MemoryLimitMB > 3072 {
		return nil, fmt.Errorf("memory_limit_mb (%d) must be between 32 and 3072 MB", m.Resources.MemoryLimitMB)
	}
	if m.Resources.CPULimit < 0 || m.Resources.CPULimit > 4.0 {
		return nil, fmt.Errorf("cpu_limit (%.1f) must be between 0.1 and 4.0 cores", m.Resources.CPULimit)
	}

	return &m, nil
}
