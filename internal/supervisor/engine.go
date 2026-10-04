package supervisor

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// Engine is the central orchestration engine of the Domus Supervisor.
type Engine struct {
	mu            sync.RWMutex
	cfg           Config
	runtime       ContainerRuntime
	rauc          RaucController
	health        HealthChecker
	startTime     time.Time
	coreHealthy   bool
	slotConfirmed bool
	cancelHealth  context.CancelFunc
}

// NewEngine creates and returns a new supervisor Engine.
func NewEngine(cfg Config, runtime ContainerRuntime, rauc RaucController, health HealthChecker) *Engine {
	return &Engine{
		cfg:       cfg,
		runtime:   runtime,
		rauc:      rauc,
		health:    health,
		startTime: time.Now(),
	}
}

// Start launches the core container and begins background health monitoring.
func (e *Engine) Start(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// 1. Ensure core container configuration
	coreConfig := ContainerConfig{
		Image: e.cfg.CoreImage,
		Mounts: []MountConfig{
			{
				Source:   e.cfg.DataDir + "/core",
				Target:   "/data",
				ReadOnly: false,
			},
		},
		Environment: map[string]string{
			"DOMUS_DATA_DIR": "/data",
		},
		NetworkMode:  "host",
		ReadOnlyRoot: false,
	}

	if err := e.runtime.EnsureContainer(ctx, "domus-core", coreConfig); err != nil {
		return fmt.Errorf("failed to configure domus-core container: %w", err)
	}

	// 2. Start core container
	if err := e.runtime.StartContainer(ctx, "domus-core"); err != nil {
		return fmt.Errorf("failed to start domus-core container: %w", err)
	}

	// 3. Launch background healthcheck loop
	healthCtx, cancel := context.WithCancel(context.Background())
	e.cancelHealth = cancel
	go e.pollHealthcheck(healthCtx)

	return nil
}

// pollHealthcheck queries Core health endpoint and confirms boot slot upon success.
func (e *Engine) pollHealthcheck(ctx context.Context) {
	ticker := time.NewTicker(e.cfg.HealthCheckInterval)
	defer ticker.Stop()

	retries := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			retries++
			healthy, err := e.health.Check(ctx, e.cfg.CoreHealthURL)
			if err == nil && healthy {
				e.mu.Lock()
				e.coreHealthy = true
				if e.cfg.AutoConfirmSlot && !e.slotConfirmed {
					if raucErr := e.rauc.MarkGood(ctx); raucErr == nil {
						e.slotConfirmed = true
						log.Printf("[Supervisor] Domus Core healthy! A/B boot slot confirmed via RAUC.")
					} else {
						log.Printf("[Supervisor] Warning: RAUC mark-good failed: %v", raucErr)
					}
				}
				e.mu.Unlock()
				return
			}

			if retries >= e.cfg.MaxHealthRetries {
				e.mu.Lock()
				e.coreHealthy = false
				e.mu.Unlock()
				log.Printf("[Supervisor] Alert: Domus Core healthcheck failed after %d retries.", retries)
				return
			}
		}
	}
}

// RestartCore stops and restarts the Domus Core container.
func (e *Engine) RestartCore(ctx context.Context) error {
	if err := e.runtime.StopContainer(ctx, "domus-core", 5*time.Second); err != nil {
		return err
	}
	return e.runtime.StartContainer(ctx, "domus-core")
}

// Stop terminates background monitoring.
func (e *Engine) Stop(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.cancelHealth != nil {
		e.cancelHealth()
	}
	return nil
}

// GetStatus returns the current snapshot of the supervisor and managed containers.
func (e *Engine) GetStatus() SupervisorStatus {
	e.mu.RLock()
	defer e.mu.RUnlock()

	coreStatus, _ := e.runtime.GetContainerStatus(context.Background(), "domus-core")
	activeSlot := "a"
	if rStatus, err := e.rauc.GetStatus(context.Background()); err == nil && rStatus.ActiveSlot != "" {
		activeSlot = rStatus.ActiveSlot
	}

	containersMap := make(map[string]ContainerStatus)
	if list, err := e.runtime.ListContainers(context.Background()); err == nil {
		for _, c := range list {
			containersMap[c.Name] = c
		}
	}

	return SupervisorStatus{
		Version:       e.cfg.Version,
		UptimeSeconds: int64(time.Since(e.startTime).Seconds()),
		CoreStatus:    coreStatus,
		CoreHealthy:   e.coreHealthy,
		SlotConfirmed: e.slotConfirmed,
		ActiveSlot:    activeSlot,
		Containers:    containersMap,
	}
}
