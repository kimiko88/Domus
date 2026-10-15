package supervisor

import (
	"context"
	"time"
)

// ContainerState represents the lifecycle status of a container.
type ContainerState string

const (
	ContainerStateRunning ContainerState = "running"
	ContainerStateStopped ContainerState = "stopped"
	ContainerStateFailed  ContainerState = "failed"
	ContainerStateUnknown ContainerState = "unknown"
)

// ContainerConfig defines container runtime properties.
type ContainerConfig struct {
	Image         string            `json:"image"`
	Command       []string          `json:"command,omitempty"`
	Environment   map[string]string `json:"environment,omitempty"`
	Mounts        []MountConfig     `json:"mounts,omitempty"`
	NetworkMode   string            `json:"network_mode,omitempty"`
	MemoryLimitMB int64             `json:"memory_limit_mb,omitempty"`
	CPULimit      float64           `json:"cpu_limit,omitempty"`
	ReadOnlyRoot  bool              `json:"read_only_root,omitempty"`
}

// MountConfig defines a bind mount from host to container.
type MountConfig struct {
	Source   string `json:"source"`
	Target   string `json:"target"`
	ReadOnly bool   `json:"read_only"`
}

// ContainerStatus represents the current status of a container.
type ContainerStatus struct {
	Name      string         `json:"name"`
	Image     string         `json:"image"`
	State     ContainerState `json:"state"`
	StartedAt time.Time      `json:"started_at"`
	ExitCode  int            `json:"exit_code"`
}

// RaucStatus represents the active slot and update state from RAUC.
type RaucStatus struct {
	Compatible string `json:"compatible"`
	ActiveSlot string `json:"active_slot"`
	BootedSlot string `json:"booted_slot"`
	Status     string `json:"status"`
}

// SupervisorStatus aggregates system, core, and update engine health.
type SupervisorStatus struct {
	Version       string                     `json:"version"`
	UptimeSeconds int64                      `json:"uptime_seconds"`
	CoreStatus    ContainerStatus            `json:"core_status"`
	CoreHealthy   bool                       `json:"core_healthy"`
	SlotConfirmed bool                       `json:"slot_confirmed"`
	ActiveSlot    string                     `json:"active_slot"`
	Containers    map[string]ContainerStatus `json:"containers"`
}

// ContainerRuntime abstracts container creation and management.
type ContainerRuntime interface {
	EnsureContainer(ctx context.Context, name string, cfg ContainerConfig) error
	StartContainer(ctx context.Context, name string) error
	StopContainer(ctx context.Context, name string, timeout time.Duration) error
	GetContainerStatus(ctx context.Context, name string) (ContainerStatus, error)
	ListContainers(ctx context.Context) ([]ContainerStatus, error)
}

// RaucController abstracts communication with the RAUC update agent.
type RaucController interface {
	MarkGood(ctx context.Context) error
	GetStatus(ctx context.Context) (RaucStatus, error)
	Rollback(ctx context.Context) error
}

// HealthChecker abstracts polling of services.
type HealthChecker interface {
	Check(ctx context.Context, url string) (bool, error)
}

// Config specifies runtime options for the Domus Supervisor.
type Config struct {
	Version             string        `json:"version"`
	SupervisorSocket    string        `json:"supervisor_socket"`
	ContainerdSocket    string        `json:"containerd_socket"`
	CoreImage           string        `json:"core_image"`
	CoreHealthURL       string        `json:"core_health_url"`
	DataDir             string        `json:"data_dir"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	HealthCheckTimeout  time.Duration `json:"health_check_timeout"`
	MaxHealthRetries    int           `json:"max_health_retries"`
	AutoConfirmSlot     bool          `json:"auto_confirm_slot"`
	RaucAgentPath       string        `json:"rauc_agent_path"`
	WatchdogDevicePath  string        `json:"watchdog_device_path"`
	WatchdogPingInterval time.Duration `json:"watchdog_ping_interval"`
	AutoRollbackOnFail  bool          `json:"auto_rollback_on_fail"`
}

// DefaultConfig returns reasonable default configuration values.
func DefaultConfig() Config {
	return Config{
		Version:              "1.0.0",
		SupervisorSocket:     "/run/domus/supervisor.sock",
		ContainerdSocket:     "/run/containerd/containerd.sock",
		CoreImage:            "ghcr.io/domus-hub/domus:latest",
		CoreHealthURL:        "http://127.0.0.1:8080/health",
		DataDir:              "/data",
		HealthCheckInterval:  500 * time.Millisecond,
		HealthCheckTimeout:   2 * time.Second,
		MaxHealthRetries:     15,
		AutoConfirmSlot:      true,
		RaucAgentPath:        "/usr/sbin/domus-update",
		WatchdogDevicePath:   "/dev/watchdog",
		WatchdogPingInterval: 10 * time.Second,
		AutoRollbackOnFail:   true,
	}
}
