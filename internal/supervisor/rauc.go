package supervisor

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// CLIRaucController invokes the system RAUC utility via command line.
type CLIRaucController struct {
	raucBin string
}

// NewCLIRaucController creates a controller using the host's rauc binary.
func NewCLIRaucController(binaryPath string) *CLIRaucController {
	if binaryPath == "" {
		binaryPath = "rauc"
	}
	return &CLIRaucController{
		raucBin: binaryPath,
	}
}

// MarkGood notifies RAUC that the currently active slot booted successfully.
func (r *CLIRaucController) MarkGood(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, r.raucBin, "status", "mark-good")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("rauc status mark-good failed (%w): %s", err, string(out))
	}
	return nil
}

// GetStatus retrieves the current RAUC status.
func (r *CLIRaucController) GetStatus(ctx context.Context) (RaucStatus, error) {
	cmd := exec.CommandContext(ctx, r.raucBin, "status", "--output-format=shell")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return RaucStatus{}, fmt.Errorf("rauc status failed (%w): %s", err, string(out))
	}

	st := RaucStatus{Status: "ok"}
	lines := strings.Split(string(out), "\n")
	for _, l := range lines {
		parts := strings.SplitN(strings.TrimSpace(l), "=", 2)
		if len(parts) == 2 {
			k, v := parts[0], strings.Trim(parts[1], "\"")
			switch k {
			case "RAUC_SYSTEM_COMPATIBLE":
				st.Compatible = v
			case "RAUC_BOOT_SLOT":
				st.ActiveSlot = v
				st.BootedSlot = v
			}
		}
	}
	return st, nil
}

// Rollback marks the current slot as bad, prompting fallback on reboot.
func (r *CLIRaucController) Rollback(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, r.raucBin, "status", "mark-bad")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("rauc status mark-bad failed (%w): %s", err, string(out))
	}
	return nil
}
