package supervisor

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type mockServiceManager struct {
	sshRunning bool
	startCalls int
	stopCalls  int
}

func (m *mockServiceManager) StartSSH(ctx context.Context) error {
	m.sshRunning = true
	m.startCalls++
	return nil
}

func (m *mockServiceManager) StopSSH(ctx context.Context) error {
	m.sshRunning = false
	m.stopCalls++
	return nil
}

func (m *mockServiceManager) IsSSHRunning(ctx context.Context) (bool, error) {
	return m.sshRunning, nil
}

func TestMaintenanceManager_DefaultState(t *testing.T) {
	tempDir := t.TempDir()
	mockSvc := &mockServiceManager{}

	mgr := NewMaintenanceManager(tempDir, mockSvc)
	status := mgr.GetStatus()

	if status.Active {
		t.Fatal("expected maintenance mode to be disabled by default")
	}
	if status.RemainingSeconds != 0 {
		t.Fatalf("expected 0 remaining seconds, got %d", status.RemainingSeconds)
	}
	if status.ExpiresAt != nil {
		t.Fatal("expected nil ExpiresAt by default")
	}
	if status.SSHPort != 22 {
		t.Fatalf("expected SSH port 22, got %d", status.SSHPort)
	}
}

func TestMaintenanceManager_EnableAndDisable(t *testing.T) {
	tempDir := t.TempDir()
	mockSvc := &mockServiceManager{}

	mgr := NewMaintenanceManager(tempDir, mockSvc)
	ctx := context.Background()

	testPubKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIG... user@host"
	req := EnableMaintenanceRequest{
		DurationMinutes: 15,
		AuthorizedKey:   testPubKey,
	}

	// 1. Enable
	status, err := mgr.Enable(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error enabling maintenance: %v", err)
	}

	if !status.Active {
		t.Fatal("expected status to be active")
	}
	if status.RemainingSeconds <= 0 || status.RemainingSeconds > 15*60 {
		t.Fatalf("expected remaining seconds <= 900, got %d", status.RemainingSeconds)
	}
	if status.ExpiresAt == nil {
		t.Fatal("expected non-nil ExpiresAt")
	}
	if mockSvc.startCalls != 1 {
		t.Fatalf("expected 1 call to StartSSH, got %d", mockSvc.startCalls)
	}

	// Verify authorized_keys written
	authKeyFile := filepath.Join(tempDir, "authorized_keys")
	content, err := os.ReadFile(authKeyFile)
	if err != nil {
		t.Fatalf("failed to read authorized_keys: %v", err)
	}
	if string(content) != testPubKey+"\n" {
		t.Fatalf("unexpected authorized_keys content: %q", string(content))
	}

	// 2. Disable
	status, err = mgr.Disable(ctx)
	if err != nil {
		t.Fatalf("unexpected error disabling maintenance: %v", err)
	}

	if status.Active {
		t.Fatal("expected status to be inactive")
	}
	if status.RemainingSeconds != 0 {
		t.Fatalf("expected 0 remaining seconds, got %d", status.RemainingSeconds)
	}
	if mockSvc.stopCalls != 1 {
		t.Fatalf("expected 1 call to StopSSH, got %d", mockSvc.stopCalls)
	}

	// Verify authorized_keys file removed or empty
	if _, err := os.Stat(authKeyFile); !os.IsNotExist(err) {
		data, _ := os.ReadFile(authKeyFile)
		if len(data) > 0 {
			t.Fatalf("expected authorized_keys to be removed or empty, found %d bytes", len(data))
		}
	}
}

func TestMaintenanceManager_DurationClamping(t *testing.T) {
	tempDir := t.TempDir()
	mockSvc := &mockServiceManager{}
	mgr := NewMaintenanceManager(tempDir, mockSvc)
	ctx := context.Background()

	// Zero or negative duration defaults to 30 min
	status, err := mgr.Enable(ctx, EnableMaintenanceRequest{DurationMinutes: 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.RemainingSeconds > 30*60 || status.RemainingSeconds < 29*60 {
		t.Fatalf("expected approx 1800s remaining, got %d", status.RemainingSeconds)
	}
	_, _ = mgr.Disable(ctx)

	// Excess duration clamped to 120 min max
	status, err = mgr.Enable(ctx, EnableMaintenanceRequest{DurationMinutes: 500})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.RemainingSeconds > 120*60 || status.RemainingSeconds < 119*60 {
		t.Fatalf("expected clamped 7200s remaining, got %d", status.RemainingSeconds)
	}
	_, _ = mgr.Disable(ctx)
}

func TestMaintenanceManager_AutoExpire(t *testing.T) {
	tempDir := t.TempDir()
	mockSvc := &mockServiceManager{}
	mgr := NewMaintenanceManager(tempDir, mockSvc)
	ctx := context.Background()

	// Enable with very short custom duration for test
	mgr.enableWithDuration(ctx, 50*time.Millisecond, "ssh-ed25519 testkey")
	if !mgr.GetStatus().Active {
		t.Fatal("expected active initially")
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	status := mgr.GetStatus()
	if status.Active {
		t.Fatal("expected maintenance mode to auto-expire")
	}
	if mockSvc.stopCalls < 1 {
		t.Fatalf("expected StopSSH to be called on expiration, got %d calls", mockSvc.stopCalls)
	}
}
