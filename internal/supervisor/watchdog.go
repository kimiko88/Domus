// Package supervisor provides hardware watchdog supervision and appliance reliability controls.
package supervisor

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// Watchdog abstracts hardware watchdog pinging (/dev/watchdog on Linux appliance).
type Watchdog interface {
	Ping(ctx context.Context) error
	Close() error
}

// DeviceWatchdog interacts with the Linux /dev/watchdog character device.
type DeviceWatchdog struct {
	mu     sync.Mutex
	path   string
	file   *os.File
	active bool
}

// NewDeviceWatchdog creates a hardware watchdog client. If path is empty, defaults to /dev/watchdog.
func NewDeviceWatchdog(path string) *DeviceWatchdog {
	if path == "" {
		path = "/dev/watchdog"
	}
	return &DeviceWatchdog{
		path: path,
	}
}

// Open opens the hardware watchdog device.
func (w *DeviceWatchdog) Open() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.active && w.file != nil {
		return nil
	}

	f, err := os.OpenFile(w.path, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("watchdog: cannot open %s: %w", w.path, err)
	}
	w.file = f
	w.active = true
	return nil
}

// Ping writes a heartbeat byte to /dev/watchdog to reset the countdown timer.
func (w *DeviceWatchdog) Ping(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.active || w.file == nil {
		// Attempt to open device lazily
		f, err := os.OpenFile(w.path, os.O_WRONLY, 0)
		if err != nil {
			return fmt.Errorf("watchdog: device not open: %w", err)
		}
		w.file = f
		w.active = true
	}

	// Write 1 byte heartbeat to keep watchdog alive
	_, err := w.file.Write([]byte{0})
	if err != nil {
		return fmt.Errorf("watchdog: ping failed: %w", err)
	}
	return nil
}

// Close sends the Linux magic character 'V' to gracefully disarm the watchdog before closing.
func (w *DeviceWatchdog) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.active || w.file == nil {
		return nil
	}

	// Linux watchdog standard: writing 'V' safely disarms the watchdog on close
	_, _ = w.file.Write([]byte("V"))
	err := w.file.Close()
	w.file = nil
	w.active = false
	return err
}

// MockWatchdog provides an in-memory recorder for unit tests.
type MockWatchdog struct {
	mu        sync.Mutex
	PingCount int
	Closed    bool
	FailPing  bool
}

func NewMockWatchdog() *MockWatchdog {
	return &MockWatchdog{}
}

func (m *MockWatchdog) Ping(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.FailPing {
		return fmt.Errorf("mock watchdog ping failed")
	}
	m.PingCount++
	return nil
}

func (m *MockWatchdog) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Closed = true
	return nil
}

func (m *MockWatchdog) IsClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.Closed
}

func (m *MockWatchdog) GetPingCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.PingCount
}

// StartWatchdogLoop begins background pinging every interval as long as core is healthy.
func StartWatchdogLoop(ctx context.Context, w Watchdog, interval time.Duration, isHealthy func() bool) {
	if w == nil || interval <= 0 {
		return
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				_ = w.Close()
				return
			case <-ticker.C:
				if isHealthy != nil && !isHealthy() {
					log.Printf("[Watchdog] Core is unhealthy! Withholding watchdog ping to trigger system recovery/reboot.")
					continue
				}
				if err := w.Ping(ctx); err != nil {
					log.Printf("[Watchdog] Warning: failed to ping hardware watchdog: %v", err)
				}
			}
		}
	}()
}
