// Package plugins provides hardware and protocol adapter lifecycle management,
// circuit breaker protection, and supervision trees.
package plugins

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
)

// State represents the operational state of an integration adapter within the supervisor.
type State string

const (
	StateStopped  State = "stopped"
	StateStarting State = "starting"
	StateHealthy  State = "healthy"
	StateDegraded State = "degraded"
	StateTripped  State = "tripped" // Circuit breaker open: backoff in progress
)

var (
	ErrAdapterNotFound = errors.New("supervisor: adapter not found")
	ErrAdapterExists   = errors.New("supervisor: adapter already registered")
)

// BreakerPolicy configures the fault-tolerance parameters for an adapter.
type BreakerPolicy struct {
	// MaxFailures is the consecutive failure threshold before the circuit breaker trips.
	MaxFailures int
	// InitialBackoff is the starting delay before probing a failed adapter.
	InitialBackoff time.Duration
	// MaxBackoff is the maximum delay cap between reconnection probes.
	MaxBackoff time.Duration
	// HealthCheckInterval is the period between active heartbeat probes.
	HealthCheckInterval time.Duration
}

// DefaultBreakerPolicy returns production-tuned parameters for local appliance adapters.
func DefaultBreakerPolicy() BreakerPolicy {
	return BreakerPolicy{
		MaxFailures:         5,
		InitialBackoff:      1 * time.Second,
		MaxBackoff:          60 * time.Second,
		HealthCheckInterval: 15 * time.Second,
	}
}

// AdapterStatus captures the real-time health and diagnostics for a supervised adapter.
type AdapterStatus struct {
	ID           string    `json:"id"`
	Type         string    `json:"type"`
	State        State     `json:"state"`
	FailCount    int       `json:"fail_count"`
	LastError    string    `json:"last_error,omitempty"`
	LastProbeAt  time.Time `json:"last_probe_at,omitempty"`
	NextProbeAt  time.Time `json:"next_probe_at,omitempty"`
	HealthySince time.Time `json:"healthy_since,omitempty"`
}

type supervisedEntry struct {
	adapter   Adapter
	policy    BreakerPolicy
	state     State
	failCount int
	lastError string
	lastProbe time.Time
	nextProbe time.Time
	healthyAt time.Time
}

// Supervisor monitors integration adapters, enforces circuit breaker backoffs,
// isolates panics, and publishes health transitions to the event bus.
type Supervisor struct {
	mu      sync.RWMutex
	entries map[string]*supervisedEntry
	bus     events.Bus
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// NewSupervisor constructs a new plugin Supervisor.
func NewSupervisor(bus events.Bus) *Supervisor {
	return &Supervisor{
		entries: make(map[string]*supervisedEntry),
		bus:     bus,
	}
}

// Register adds an adapter to the supervision tree.
func (s *Supervisor) Register(adapter Adapter, policy ...BreakerPolicy) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := adapter.ID()
	if _, exists := s.entries[id]; exists {
		return ErrAdapterExists
	}

	pol := DefaultBreakerPolicy()
	if len(policy) > 0 {
		pol = policy[0]
	}

	s.entries[id] = &supervisedEntry{
		adapter: adapter,
		policy:  pol,
		state:   StateStopped,
	}
	return nil
}

// Start boots up all registered adapters and begins background health supervision.
func (s *Supervisor) Start(ctx context.Context) error {
	s.mu.Lock()
	supCtx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel

	for id, entry := range s.entries {
		entry.state = StateStarting
		s.safeStart(ctx, entry)
		slog.Info("supervisor: adapter started", "id", id, "type", entry.adapter.Type(), "state", entry.state)
	}
	s.mu.Unlock()

	s.wg.Add(1)
	go s.monitorLoop(supCtx)

	return nil
}

// Stop gracefully shuts down all supervised adapters and halts monitoring.
func (s *Supervisor) Stop(ctx context.Context) error {
	s.mu.Lock()
	if s.cancel != nil {
		s.cancel()
	}
	s.mu.Unlock()

	s.wg.Wait()

	s.mu.Lock()
	defer s.mu.Unlock()

	var firstErr error
	for id, entry := range s.entries {
		if err := s.safeStop(ctx, entry); err != nil && firstErr == nil {
			firstErr = err
		}
		entry.state = StateStopped
		slog.Info("supervisor: adapter stopped", "id", id)
	}

	return firstErr
}

// GetStatus returns the current status of an adapter.
func (s *Supervisor) GetStatus(id string) (AdapterStatus, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.entries[id]
	if !ok {
		return AdapterStatus{}, false
	}

	return AdapterStatus{
		ID:           entry.adapter.ID(),
		Type:         entry.adapter.Type(),
		State:        entry.state,
		FailCount:    entry.failCount,
		LastError:    entry.lastError,
		LastProbeAt:  entry.lastProbe,
		NextProbeAt:  entry.nextProbe,
		HealthySince: entry.healthyAt,
	}, true
}

// ListStatus returns diagnostic statuses for all registered adapters.
func (s *Supervisor) ListStatus() []AdapterStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]AdapterStatus, 0, len(s.entries))
	for _, entry := range s.entries {
		list = append(list, AdapterStatus{
			ID:           entry.adapter.ID(),
			Type:         entry.adapter.Type(),
			State:        entry.state,
			FailCount:    entry.failCount,
			LastError:    entry.lastError,
			LastProbeAt:  entry.lastProbe,
			NextProbeAt:  entry.nextProbe,
			HealthySince: entry.healthyAt,
		})
	}
	return list
}

// Restart forces an immediate restart of a specific adapter.
func (s *Supervisor) Restart(ctx context.Context, id string) error {
	s.mu.Lock()
	entry, ok := s.entries[id]
	if !ok {
		s.mu.Unlock()
		return ErrAdapterNotFound
	}
	s.mu.Unlock()

	_ = s.safeStop(ctx, entry)

	s.mu.Lock()
	entry.state = StateStarting
	entry.failCount = 0
	entry.lastError = ""
	s.safeStart(ctx, entry)
	s.mu.Unlock()

	return nil
}

func (s *Supervisor) monitorLoop(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runProbes(ctx)
		}
	}
}

func (s *Supervisor) runProbes(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	for _, entry := range s.entries {
		// If circuit breaker is tripped, only probe if nextProbe time has elapsed
		if entry.state == StateTripped && now.Before(entry.nextProbe) {
			continue
		}

		// Check if interval has passed
		if entry.state == StateHealthy && now.Sub(entry.lastProbe) < entry.policy.HealthCheckInterval {
			continue
		}

		entry.lastProbe = now
		probeErr := s.safeHealthCheck(ctx, entry)

		if probeErr == nil {
			// Succeeded
			if entry.state != StateHealthy {
				slog.Info("supervisor: adapter recovered to healthy",
					"id", entry.adapter.ID(),
					"type", entry.adapter.Type(),
					"prev_state", entry.state,
				)
				entry.state = StateHealthy
				entry.failCount = 0
				entry.lastError = ""
				entry.healthyAt = now
				s.publishStateChange(entry.adapter.ID(), StateHealthy, "")
			}
		} else {
			// Failed probe
			entry.failCount++
			entry.lastError = probeErr.Error()

			if entry.failCount >= entry.policy.MaxFailures {
				// Trip circuit breaker
				entry.state = StateTripped
				backoff := s.calculateBackoff(entry.policy, entry.failCount)
				entry.nextProbe = now.Add(backoff)

				slog.Warn("supervisor: circuit breaker TRIPPED for adapter",
					"id", entry.adapter.ID(),
					"fail_count", entry.failCount,
					"backoff", backoff,
					"error", probeErr,
				)
				s.publishStateChange(entry.adapter.ID(), StateTripped, probeErr.Error())
			} else {
				entry.state = StateDegraded
				slog.Warn("supervisor: adapter probe degraded",
					"id", entry.adapter.ID(),
					"fail_count", entry.failCount,
					"error", probeErr,
				)
				s.publishStateChange(entry.adapter.ID(), StateDegraded, probeErr.Error())
			}
		}
	}
}

func (s *Supervisor) calculateBackoff(pol BreakerPolicy, failCount int) time.Duration {
	// Exponential backoff: initial * 2^(failCount - MaxFailures) + jitter
	exponent := failCount - pol.MaxFailures
	if exponent < 0 {
		exponent = 0
	}
	if exponent > 6 {
		exponent = 6 // avoid overflow
	}

	multiplier := math.Pow(2, float64(exponent))
	backoff := time.Duration(float64(pol.InitialBackoff) * multiplier)

	if backoff > pol.MaxBackoff {
		backoff = pol.MaxBackoff
	}

	// Add 10% random jitter to avoid thundering herd across adapters
	jitter := time.Duration(rand.Int64N(int64(backoff / 10)))
	return backoff + jitter
}

func (s *Supervisor) safeStart(ctx context.Context, entry *supervisedEntry) {
	defer func() {
		if r := recover(); r != nil {
			entry.state = StateDegraded
			entry.lastError = fmt.Sprintf("panic in Start: %v", r)
			slog.Error("supervisor: panic recovered in adapter Start", "id", entry.adapter.ID(), "panic", r)
		}
	}()

	startCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := entry.adapter.Start(startCtx); err != nil {
		entry.state = StateDegraded
		entry.lastError = err.Error()
		entry.failCount = 1
	} else {
		entry.state = StateHealthy
		entry.failCount = 0
		entry.healthyAt = time.Now()
	}
}

func (s *Supervisor) safeStop(ctx context.Context, entry *supervisedEntry) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in Stop: %v", r)
			slog.Error("supervisor: panic recovered in adapter Stop", "id", entry.adapter.ID(), "panic", r)
		}
	}()

	stopCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return entry.adapter.Stop(stopCtx)
}

func (s *Supervisor) safeHealthCheck(ctx context.Context, entry *supervisedEntry) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in HealthCheck: %v", r)
			slog.Error("supervisor: panic recovered in adapter HealthCheck", "id", entry.adapter.ID(), "panic", r)
		}
	}()

	probeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return entry.adapter.HealthCheck(probeCtx)
}

func (s *Supervisor) publishStateChange(id string, state State, errMsg string) {
	if s.bus == nil {
		return
	}
	_ = s.bus.Publish(context.Background(), events.Event{
		Type: events.IntegrationStateChanged,
		Payload: map[string]any{
			"id":         id,
			"state":      string(state),
			"last_error": errMsg,
		},
	})
}
