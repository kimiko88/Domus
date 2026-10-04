package supervisor

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// MaintenanceStatus represents the current state of on-demand SSH access.
type MaintenanceStatus struct {
	Active           bool       `json:"active"`
	ExpiresAt        *time.Time `json:"expires_at,omitempty"`
	RemainingSeconds int64      `json:"remaining_seconds"`
	SSHPort          int        `json:"ssh_port"`
	KeyFingerprint   string     `json:"key_fingerprint,omitempty"`
}

// EnableMaintenanceRequest contains duration and public key options.
type EnableMaintenanceRequest struct {
	DurationMinutes int    `json:"duration_minutes"`
	AuthorizedKey   string `json:"authorized_key,omitempty"`
}

// ServiceManager abstracts host service lifecycle management.
type ServiceManager interface {
	StartSSH(ctx context.Context) error
	StopSSH(ctx context.Context) error
	IsSSHRunning(ctx context.Context) (bool, error)
}

// SystemdServiceManager manages sshd via systemctl or init script.
type SystemdServiceManager struct{}

// StartSSH starts the SSH daemon.
func (s *SystemdServiceManager) StartSSH(ctx context.Context) error {
	if _, err := exec.LookPath("systemctl"); err != nil {
		// systemctl not available on this host (e.g. dev/test environment)
		return nil
	}
	cmd := exec.CommandContext(ctx, "systemctl", "start", "sshd")
	return cmd.Run()
}

// StopSSH stops the SSH daemon.
func (s *SystemdServiceManager) StopSSH(ctx context.Context) error {
	if _, err := exec.LookPath("systemctl"); err != nil {
		// systemctl not available on this host (e.g. dev/test environment)
		return nil
	}
	cmd := exec.CommandContext(ctx, "systemctl", "stop", "sshd")
	return cmd.Run()
}

// IsSSHRunning checks if the SSH daemon is running.
func (s *SystemdServiceManager) IsSSHRunning(ctx context.Context) (bool, error) {
	if _, err := exec.LookPath("systemctl"); err != nil {
		return false, nil
	}
	cmd := exec.CommandContext(ctx, "systemctl", "is-active", "--quiet", "sshd")
	err := cmd.Run()
	return err == nil, nil
}

// MaintenanceManager controls controlled time-bounded SSH maintenance mode.
type MaintenanceManager struct {
	mu             sync.RWMutex
	sshDir         string
	serviceMgr     ServiceManager
	active         bool
	expiresAt      *time.Time
	keyFingerprint string
	timer          *time.Timer
}

// NewMaintenanceManager initializes the maintenance manager with SSH disabled by default.
func NewMaintenanceManager(sshDir string, serviceMgr ServiceManager) *MaintenanceManager {
	if serviceMgr == nil {
		serviceMgr = &SystemdServiceManager{}
	}
	return &MaintenanceManager{
		sshDir:     sshDir,
		serviceMgr: serviceMgr,
		active:     false,
	}
}

// SetServiceManager overrides the service manager implementation.
func (m *MaintenanceManager) SetServiceManager(serviceMgr ServiceManager) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.serviceMgr = serviceMgr
}

// GetStatus returns the current maintenance status and remaining time.
func (m *MaintenanceManager) GetStatus() MaintenanceStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var remaining int64
	if m.active && m.expiresAt != nil {
		rem := time.Until(*m.expiresAt).Seconds()
		if rem > 0 {
			remaining = int64(rem)
		} else {
			remaining = 0
		}
	}

	return MaintenanceStatus{
		Active:           m.active,
		ExpiresAt:        m.expiresAt,
		RemainingSeconds: remaining,
		SSHPort:          22,
		KeyFingerprint:   m.keyFingerprint,
	}
}

// Enable activates maintenance mode for the requested duration (clamped 1m-120m).
func (m *MaintenanceManager) Enable(ctx context.Context, req EnableMaintenanceRequest) (*MaintenanceStatus, error) {
	minutes := req.DurationMinutes
	if minutes <= 0 {
		minutes = 30 // default 30 minutes
	}
	if minutes > 120 {
		minutes = 120 // max 2 hours
	}

	return m.enableWithDuration(ctx, time.Duration(minutes)*time.Minute, req.AuthorizedKey)
}

func (m *MaintenanceManager) enableWithDuration(ctx context.Context, duration time.Duration, pubKey string) (*MaintenanceStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 1. Ensure SSH directory exists with strict permissions (0700)
	if err := os.MkdirAll(m.sshDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create ssh directory: %w", err)
	}

	// 2. Install authorized key if provided
	authKeyPath := filepath.Join(m.sshDir, "authorized_keys")
	trimmedKey := strings.TrimSpace(pubKey)
	fingerprint := ""

	if trimmedKey != "" {
		if !strings.HasPrefix(trimmedKey, "ssh-") && !strings.HasPrefix(trimmedKey, "ecdsa-") {
			return nil, errors.New("invalid public key format, expected ssh-* or ecdsa-*")
		}
		if err := os.WriteFile(authKeyPath, []byte(trimmedKey+"\n"), 0600); err != nil {
			return nil, fmt.Errorf("failed to write authorized_keys: %w", err)
		}
		hash := sha256.Sum256([]byte(trimmedKey))
		fingerprint = "SHA256:" + hex.EncodeToString(hash[:8])
	} else {
		// Empty / removed authorized_keys
		_ = os.Remove(authKeyPath)
	}

	// 3. Start SSH service
	if m.serviceMgr != nil {
		if err := m.serviceMgr.StartSSH(ctx); err != nil {
			return nil, fmt.Errorf("failed to start ssh service: %w", err)
		}
	}

	// 4. Set state and auto-expiration timer
	if m.timer != nil {
		m.timer.Stop()
	}

	exp := time.Now().Add(duration)
	m.active = true
	m.expiresAt = &exp
	m.keyFingerprint = fingerprint

	m.timer = time.AfterFunc(duration, func() {
		_, _ = m.Disable(context.Background())
	})

	remSeconds := int64(duration.Seconds())
	return &MaintenanceStatus{
		Active:           true,
		ExpiresAt:        m.expiresAt,
		RemainingSeconds: remSeconds,
		SSHPort:          22,
		KeyFingerprint:   m.keyFingerprint,
	}, nil
}

// Disable deactivates maintenance mode, wipes keys, and stops the SSH service.
func (m *MaintenanceManager) Disable(ctx context.Context) (*MaintenanceStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.timer != nil {
		m.timer.Stop()
		m.timer = nil
	}

	// 1. Wipe authorized keys file
	authKeyPath := filepath.Join(m.sshDir, "authorized_keys")
	_ = os.Remove(authKeyPath)

	// 2. Stop SSH service
	var stopErr error
	if m.serviceMgr != nil {
		stopErr = m.serviceMgr.StopSSH(ctx)
	}

	// 3. Reset internal state
	m.active = false
	m.expiresAt = nil
	m.keyFingerprint = ""

	status := &MaintenanceStatus{
		Active:           false,
		ExpiresAt:        nil,
		RemainingSeconds: 0,
		SSHPort:          22,
	}

	if stopErr != nil {
		return status, fmt.Errorf("failed to stop ssh service: %w", stopErr)
	}
	return status, nil
}
