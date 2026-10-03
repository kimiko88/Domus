// Package cloud provides integration adapters with circuit breaker resilience for cloud services.
package cloud

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/plugins"
	"github.com/google/uuid"
)

// Ensure Adapter implements plugins.Adapter.
var _ plugins.Adapter = (*Adapter)(nil)

// Config configures the cloud resilience adapter.
type Config struct {
	IntegrationID    string
	Provider         Provider
	FailureThreshold int
	ResetTimeout     time.Duration
}

// Adapter provides a circuit-breaker wrapped cloud integration.
type Adapter struct {
	cfg     Config
	cb      *CircuitBreaker
	bus     events.Bus
	mu      sync.Mutex
	running bool
}

// NewAdapter constructs a cloud resilience adapter.
func NewAdapter(cfg Config, bus events.Bus) *Adapter {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: cfg.FailureThreshold,
		ResetTimeout:     cfg.ResetTimeout,
	})
	return &Adapter{
		cfg: cfg,
		cb:  cb,
		bus: bus,
	}
}

// ID returns the unique integration ID.
func (a *Adapter) ID() string {
	return a.cfg.IntegrationID
}

// Type returns the protocol/vendor identifier.
func (a *Adapter) Type() string {
	if a.cfg.Provider != nil {
		return fmt.Sprintf("cloud_%s", a.cfg.Provider.Name())
	}
	return "cloud_generic"
}

// HealthCheck verifies connectivity and returns ErrCircuitBreakerOpen if degraded.
func (a *Adapter) HealthCheck(ctx context.Context) error {
	if a.cb.State() == StateOpen {
		return ErrCircuitBreakerOpen
	}
	if a.cfg.Provider == nil {
		return nil
	}
	return a.cb.Execute(ctx, func(c context.Context) error {
		return a.cfg.Provider.Ping(c)
	})
}

// Start marks the adapter as active.
func (a *Adapter) Start(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.running = true
	slog.Info("cloud adapter started", "integration_id", a.cfg.IntegrationID, "type", a.Type())
	return nil
}

// Stop terminates the adapter.
func (a *Adapter) Stop(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.running = false
	slog.Info("cloud adapter stopped", "integration_id", a.cfg.IntegrationID)
	return nil
}

// Execute wraps cloud command execution with the circuit breaker.
func (a *Adapter) Execute(ctx context.Context, externalID string, cmd string, params map[string]any) error {
	if a.cfg.Provider == nil {
		return ErrCloudOffline
	}

	err := a.cb.Execute(ctx, func(c context.Context) error {
		return a.cfg.Provider.ExecuteCommand(c, externalID, cmd, params)
	})

	if err != nil {
		if a.cb.State() == StateOpen && a.bus != nil {
			_ = a.bus.Publish(ctx, events.Event{
				ID:         uuid.NewString(),
				Type:       events.IntegrationStateChanged,
				OccurredAt: time.Now().UTC(),
				Source:     a.Type(),
				Payload: map[string]any{
					"integration_id": a.cfg.IntegrationID,
					"status":         "degraded",
					"error":          err.Error(),
				},
			})
		}
		return err
	}

	return nil
}
