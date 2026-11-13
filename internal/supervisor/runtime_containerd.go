package supervisor

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ContainerdRuntime manages containers via containerd's control interface.
type ContainerdRuntime struct {
	socketPath string
	ctrBin     string
	namespace  string
}

// NewContainerdRuntime initializes the containerd runtime driver.
func NewContainerdRuntime(socketPath string) *ContainerdRuntime {
	if socketPath == "" {
		socketPath = "/run/containerd/containerd.sock"
	}
	return &ContainerdRuntime{
		socketPath: socketPath,
		ctrBin:     "ctr",
		namespace:  "domus",
	}
}

func (c *ContainerdRuntime) args(subcmd string, extra ...string) []string {
	base := []string{"--address", c.socketPath, "--namespace", c.namespace, subcmd}
	return append(base, extra...)
}

// EnsureContainer pulls the image and creates the container if not present.
func (c *ContainerdRuntime) EnsureContainer(ctx context.Context, name string, cfg ContainerConfig) error {
	// Check if container already exists
	checkCmd := exec.CommandContext(ctx, c.ctrBin, c.args("container", "info", name)...)
	if err := checkCmd.Run(); err == nil {
		return nil // Container already registered
	}

	// Pull image if necessary
	pullCmd := exec.CommandContext(ctx, c.ctrBin, c.args("image", "pull", cfg.Image)...)
	_ = pullCmd.Run()

	// Prepare create arguments
	createArgs := c.args("container", "create")
	for _, m := range cfg.Mounts {
		roFlag := "rw"
		if m.ReadOnly {
			roFlag = "ro"
		}
		createArgs = append(createArgs, "--mount", fmt.Sprintf("type=bind,src=%s,dst=%s,options=rbind:%s", m.Source, m.Target, roFlag))
	}

	for k, v := range cfg.Environment {
		createArgs = append(createArgs, "--env", fmt.Sprintf("%s=%s", k, v))
	}

	if cfg.ReadOnlyRoot {
		createArgs = append(createArgs, "--read-only")
	}

	createArgs = append(createArgs, cfg.Image, name)
	createArgs = append(createArgs, cfg.Command...)

	cmd := exec.CommandContext(ctx, c.ctrBin, createArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("container create failed (%w): %s", err, string(out))
	}
	return nil
}

// StartContainer launches the container task in background mode.
func (c *ContainerdRuntime) StartContainer(ctx context.Context, name string) error {
	// Create and start the task
	cmd := exec.CommandContext(ctx, c.ctrBin, c.args("task", "start", "-d", name)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		// If task already exists, attempt to resume or re-run
		return fmt.Errorf("task start failed (%w): %s", err, string(out))
	}
	return nil
}

// StopContainer terminates the running container task.
func (c *ContainerdRuntime) StopContainer(ctx context.Context, name string, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	termCmd := exec.CommandContext(ctx, c.ctrBin, c.args("task", "kill", "-s", "SIGTERM", name)...)
	termErr := termCmd.Run()
	status, statusErr := c.GetContainerStatus(ctx, name)
	if statusErr != nil {
		return fmt.Errorf("failed to inspect task %s after SIGTERM: %w", name, statusErr)
	}
	if termErr != nil && status.State == ContainerStateRunning {
		return fmt.Errorf("failed to send SIGTERM to task %s: %w", name, termErr)
	}

	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for status.State == ContainerStateRunning {
		select {
		case <-ctx.Done():
			return fmt.Errorf("stopping task %s: %w", name, ctx.Err())
		case <-deadline.C:
			killCmd := exec.CommandContext(ctx, c.ctrBin, c.args("task", "kill", "-s", "SIGKILL", name)...)
			if out, err := killCmd.CombinedOutput(); err != nil {
				return fmt.Errorf("failed to force-kill task %s (%w): %s", name, err, strings.TrimSpace(string(out)))
			}
			// Give containerd a bounded period to report task termination.
			forceDeadline := time.NewTimer(2 * time.Second)
			defer forceDeadline.Stop()
			for {
				status, statusErr = c.GetContainerStatus(ctx, name)
				if statusErr != nil {
					return fmt.Errorf("failed to inspect task %s after SIGKILL: %w", name, statusErr)
				}
				if status.State != ContainerStateRunning {
					break
				}
				select {
				case <-ctx.Done():
					return fmt.Errorf("waiting for task %s termination: %w", name, ctx.Err())
				case <-forceDeadline.C:
					return fmt.Errorf("task %s is still running after SIGKILL", name)
				case <-ticker.C:
				}
			}
		case <-ticker.C:
			status, statusErr = c.GetContainerStatus(ctx, name)
			if statusErr != nil {
				return fmt.Errorf("failed to inspect task %s while stopping: %w", name, statusErr)
			}
		}
	}

	deleteCmd := exec.CommandContext(ctx, c.ctrBin, c.args("task", "delete", name)...)
	if out, err := deleteCmd.CombinedOutput(); err != nil {
		// A task that was already absent is stopped; otherwise surface the
		// runtime failure instead of claiming that cleanup succeeded.
		status, statusErr = c.GetContainerStatus(ctx, name)
		if statusErr != nil || status.State == ContainerStateRunning {
			return fmt.Errorf("failed to delete task %s (%w): %s", name, err, strings.TrimSpace(string(out)))
		}
	}
	return nil
}

// GetContainerStatus queries the state of the container and task.
func (c *ContainerdRuntime) GetContainerStatus(ctx context.Context, name string) (ContainerStatus, error) {
	cmd := exec.CommandContext(ctx, c.ctrBin, c.args("task", "ls")...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return ContainerStatus{Name: name, State: ContainerStateUnknown}, err
	}

	lines := strings.Split(string(out), "\n")
	for _, l := range lines {
		fields := strings.Fields(l)
		if len(fields) >= 3 && fields[0] == name {
			state := ContainerStateRunning
			if fields[2] == "STOPPED" {
				state = ContainerStateStopped
			}
			return ContainerStatus{
				Name:  name,
				State: state,
			}, nil
		}
	}
	return ContainerStatus{Name: name, State: ContainerStateStopped}, nil
}

// ListContainers returns all registered containers in the domus namespace.
func (c *ContainerdRuntime) ListContainers(ctx context.Context) ([]ContainerStatus, error) {
	cmd := exec.CommandContext(ctx, c.ctrBin, c.args("container", "ls", "-q")...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	names := strings.Fields(string(out))
	var result []ContainerStatus
	for _, n := range names {
		st, _ := c.GetContainerStatus(ctx, n)
		result = append(result, st)
	}
	return result, nil
}
