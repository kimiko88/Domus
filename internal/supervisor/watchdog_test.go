package supervisor_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/supervisor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockRaucWithRollback struct {
	mu            sync.Mutex
	markedGood    bool
	rollbackCount int
	status        supervisor.RaucStatus
}

func (m *MockRaucWithRollback) MarkGood(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.markedGood = true
	return nil
}

func (m *MockRaucWithRollback) GetStatus(ctx context.Context) (supervisor.RaucStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.status, nil
}

func (m *MockRaucWithRollback) Rollback(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rollbackCount++
	return nil
}

type MockFailingHealthChecker struct{}

func (f *MockFailingHealthChecker) Check(ctx context.Context, url string) (bool, error) {
	return false, nil // always fails
}

func TestSupervisor_WatchdogAndAutoRollback(t *testing.T) {
	tempDir := t.TempDir()

	cfg := supervisor.DefaultConfig()
	cfg.DataDir = tempDir
	cfg.HealthCheckInterval = 20 * time.Millisecond
	cfg.MaxHealthRetries = 3
	cfg.AutoConfirmSlot = true
	cfg.AutoRollbackOnFail = true
	cfg.WatchdogPingInterval = 30 * time.Millisecond

	runtime := newMockRuntime()
	rauc := &MockRaucWithRollback{
		status: supervisor.RaucStatus{
			Status:     "ok",
			ActiveSlot: "system1",
			BootedSlot: "system1",
		},
	}
	health := &MockFailingHealthChecker{}

	engine := supervisor.NewEngine(cfg, runtime, rauc, health)

	mockWd := supervisor.NewMockWatchdog()
	engine.SetWatchdog(mockWd)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	require.NoError(t, engine.Start(ctx))

	// Wait for healthcheck to fail 3 times and trigger auto-rollback
	assert.Eventually(t, func() bool {
		rauc.mu.Lock()
		defer rauc.mu.Unlock()
		return rauc.rollbackCount == 1
	}, 1*time.Second, 20*time.Millisecond)

	assert.False(t, engine.IsCoreHealthy())

	// Stop engine and verify watchdog close
	_ = engine.Stop(ctx)
	cancel()

	assert.Eventually(t, func() bool {
		return mockWd.IsClosed()
	}, 1*time.Second, 20*time.Millisecond)
}
