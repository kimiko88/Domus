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
	"strings"
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
	ErrAdapterNotFound       = errors.New("supervisor: adapter not found")
	ErrAdapterExists         = errors.New("supervisor: adapter already registered")
	ErrInvalidAdapter        = errors.New("supervisor: adapter must have a non-empty ID and type")
	ErrInvalidBreakerPolicy  = errors.New("supervisor: breaker policy values must be positive and max_backoff >= initial_backoff")
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
	adapter    Adapter
	policy     BreakerPolicy
	state      State
	failCount  int
	lastError  string
	lastProbe  time.Time
	nextProbe  time.Time
	healthyAt  time.Time
	stopFailed bool
}

// Supervisor monitors integration adapters, enforces circuit breaker backoffs,
// isolates panics, and publishes health transitions to the event bus.
type Supervisor struct {
	// lifecycleMu serializes Start/Stop/Register/Unregister/Restart/Replace.
	// Adapter I/O may block, so the state mutex is not used as a lifecycle gate.
	lifecycleMu sync.Mutex
	mu           sync.RWMutex
	entries      map[string]*supervisedEntry
	bus          events.Bus
	cancel       context.CancelFunc
	running      bool
	wg           sync.WaitGroup
}

// NewSupervisor constructs a new plugin Supervisor.
func NewSupervisor(bus events.Bus) *Supervisor {
	return &Supervisor{
		entries: make(map[string]*supervisedEntry),
		bus:     bus,
	}
}

// Register adds an adapter to the supervision tree.
func validateBreakerPolicy(policy BreakerPolicy) error {
	if policy.MaxFailures <= 0 || policy.InitialBackoff <= 0 ||
		policy.MaxBackoff < policy.InitialBackoff || policy.HealthCheckInterval <= 0 {
		return ErrInvalidBreakerPolicy
	}
	return nil
}

func (s *Supervisor) Register(adapter Adapter, policy ...BreakerPolicy) error {
	if adapter == nil || strings.TrimSpace(adapter.ID()) == "" || strings.TrimSpace(adapter.Type()) == "" {
		return ErrInvalidAdapter
	}

	pol := DefaultBreakerPolicy()
	if len(policy) > 0 {
		pol = policy[0]
	}
	if err := validateBreakerPolicy(pol); err != nil {
		return err
	}

	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()

	id := adapter.ID()
	if _, exists := s.entries[id]; exists {
		return ErrAdapterExists
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
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()

	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}
	// The supervisor lifetime is owned by Stop, not by the short-lived startup
	// request context. Individual adapter starts still use the supplied context.
	supCtx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.running = true
	for id, entry := range s.entries {
		if entry.stopFailed {
			if stopErr := s.safeStop(ctx, entry); stopErr != nil {
				entry.state = StateDegraded
				entry.lastError = stopErr.Error()
				slog.Warn("supervisor: adapter remains degraded after failed stop", "id", id, "error", stopErr)
				continue
			}
			entry.stopFailed = false
		}
		entry.state = StateStarting
		s.safeStart(ctx, entry)
		slog.Info("supervisor: adapter started", "id", id, "type", entry.adapter.Type(), "state", entry.state)
	}
	s.mu.Unlock()

	// Add while lifecycleMu is held so Stop cannot race Wait against Add.
	s.wg.Add(1)
	go s.monitorLoop(supCtx)
	return nil
}

// Stop gracefully shuts down all supervised adapters and halts monitoring.
func (s *Supervisor) Stop(ctx context.Context) error {
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()

	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil
	}
	if s.cancel != nil {
		s.cancel()
	}
	s.running = false
	s.mu.Unlock()

	s.wg.Wait()

	s.mu.Lock()
	var firstErr error
	for id, entry := range s.entries {
		if err := s.safeStop(ctx, entry); err != nil {
			entry.state = StateDegraded
			entry.stopFailed = true
			entry.lastError = err.Error()
			if firstErr == nil {
				firstErr = err
			}
		} else {
			entry.state = StateStopped
			entry.stopFailed = false
			entry.lastError = ""
			entry.failCount = 0
		}
		slog.Info("supervisor: adapter stopped", "id", id, "state", entry.state)
	}
	s.mu.Unlock()
	return firstErr
}

// StartAdapter starts one registered adapter. It is used when an integration is
// enabled after the supervisor tree is already running.
func (s *Supervisor) StartAdapter(ctx context.Context, id string) error {
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()

	s.mu.Lock()
	entry, ok := s.entries[id]
	if !ok {
		s.mu.Unlock()
		return ErrAdapterNotFound
	}
	if entry.state == StateHealthy || entry.state == StateStarting {
		s.mu.Unlock()
		return nil
	}
	if entry.stopFailed {
		if err := s.safeStop(ctx, entry); err != nil {
			entry.state = StateDegraded
			entry.lastError = err.Error()
			s.mu.Unlock()
			return fmt.Errorf("supervisor: adapter %s still cannot stop: %w", id, err)
		}
		entry.stopFailed = false
	}
	entry.state = StateStarting
	s.safeStart(ctx, entry)
	state, lastError := entry.state, entry.lastError
	s.mu.Unlock()
	if state != StateHealthy {
		return fmt.Errorf("supervisor: adapter %s failed to start: %s", id, lastError)
	}
	s.publishStateChange(id, StateHealthy, "")
	return nil
}

// Unregister stops and removes one adapter from the supervision tree.
func (s *Supervisor) Unregister(ctx context.Context, id string) error {
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()

	s.mu.Lock()
	entry, ok := s.entries[id]
	if !ok {
		s.mu.Unlock()
		return ErrAdapterNotFound
	}
	// Do not drop the only reference to a possibly running adapter before Stop
	// succeeds. A timeout/error leaves it registered and visibly degraded so a
	// later retry or supervisor shutdown can stop it safely.
	if err := s.safeStop(ctx, entry); err != nil {
		entry.state = StateDegraded
		entry.stopFailed = true
		entry.lastError = err.Error()
		s.mu.Unlock()
		s.publishStateChange(id, StateDegraded, err.Error())
		return err
	}
	delete(s.entries, id)
	entry.state = StateStopped
	entry.stopFailed = false
	entry.lastError = ""
	s.mu.Unlock()
	s.publishStateChange(id, StateStopped, "")
	return nil
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
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()

	s.mu.Lock()
	entry, ok := s.entries[id]
	if !ok {
		s.mu.Unlock()
		return ErrAdapterNotFound
	}
	if err := s.safeStop(ctx, entry); err != nil {
		entry.state = StateDegraded
		entry.stopFailed = true
		entry.lastError = err.Error()
		s.mu.Unlock()
		return fmt.Errorf("supervisor: adapter %s failed to stop before restart: %w", id, err)
	}
	entry.stopFailed = false
	entry.state = StateStarting
	entry.failCount = 0
	entry.lastError = ""
	s.safeStart(ctx, entry)
	state, lastError := entry.state, entry.lastError
	s.mu.Unlock()

	if state != StateHealthy {
		return fmt.Errorf("supervisor: adapter %s failed to restart: %s", id, lastError)
	}
	s.publishStateChange(id, StateHealthy, "")
	return nil
}

// Replace atomically swaps the adapter instance behind an existing supervised
// ID. If the replacement fails to start, it attempts to restore the old adapter
// before returning an error. The existing entry remains registered throughout.
func (s *Supervisor) Replace(ctx context.Context, id string, replacement Adapter) error {
	if replacement == nil || strings.TrimSpace(replacement.ID()) != id || strings.TrimSpace(replacement.Type()) == "" {
		return ErrInvalidAdapter
	}

	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()

	s.mu.Lock()
	entry, ok := s.entries[id]
	if !ok {
		s.mu.Unlock()
		return ErrAdapterNotFound
	}
	if entry.adapter.Type() != replacement.Type() {
		s.mu.Unlock()
		return fmt.Errorf("%w: replacement type differs from registered adapter", ErrInvalidAdapter)
	}
	if err := s.safeStop(ctx, entry); err != nil {
		entry.state = StateDegraded
		entry.stopFailed = true
		entry.lastError = err.Error()
		s.mu.Unlock()
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		cleanupErr := s.safeStop(cleanupCtx, &supervisedEntry{adapter: replacement})
		cancel()
		return errors.Join(
			fmt.Errorf("supervisor: adapter %s failed to stop before replacement: %w", id, err),
			cleanupErr,
		)
	}
	entry.stopFailed = false

	if !s.running {
		entry.adapter = replacement
		entry.state = StateStopped
		entry.failCount = 0
		entry.lastError = ""
		entry.lastProbe = time.Time{}
		entry.nextProbe = time.Time{}
		entry.healthyAt = time.Time{}
		s.mu.Unlock()
		return nil
	}

	candidate := *entry
	candidate.adapter = replacement
	candidate.state = StateStarting
	candidate.failCount = 0
	candidate.lastError = ""
	candidate.lastProbe = time.Time{}
	candidate.nextProbe = time.Time{}
	candidate.healthyAt = time.Time{}
	s.safeStart(ctx, &candidate)
	if candidate.state == StateHealthy {
		*entry = candidate
		s.mu.Unlock()
		s.publishStateChange(id, StateHealthy, "")
		return nil
	}

	startErr := fmt.Errorf("supervisor: replacement for adapter %s failed to start: %s", id, candidate.lastError)
	_ = s.safeStop(ctx, &candidate)

	rollback := *entry
	rollback.state = StateStarting
	rollback.failCount = 0
	rollback.lastError = ""
	s.safeStart(ctx, &rollback)
	*entry = rollback
	if rollback.state != StateHealthy {
		rollbackErr := fmt.Errorf("supervisor: original adapter %s could not be restored: %s", id, rollback.lastError)
		s.mu.Unlock()
		s.publishStateChange(id, StateDegraded, rollback.lastError)
		return errors.Join(startErr, rollbackErr)
	}
	s.mu.Unlock()
	s.publishStateChange(id, StateHealthy, "")
	return startErr
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
		if entry.state == StateStopped || entry.state == StateStarting || entry.stopFailed {
			continue
		}
		// If circuit breaker is tripped, only probe if nextProbe time has elapsed.
		if entry.state == StateTripped && now.Before(entry.nextProbe) {
			continue
		}
		// Healthy and degraded adapters both observe the configured cadence. This
		// prevents a failing adapter from being hammered every 50 ms.
		if (entry.state == StateHealthy || entry.state == StateDegraded) &&
			!entry.lastProbe.IsZero() && now.Sub(entry.lastProbe) < entry.policy.HealthCheckInterval {
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
	jitterRange := int64(backoff / 10)
	if jitterRange <= 0 {
		return backoff
	}
	jitter := time.Duration(rand.Int64N(jitterRange))
	return backoff + jitter
}

func (s *Supervisor) safeStart(ctx context.Context, entry *supervisedEntry) {
	defer func() {
		if r := recover(); r != nil {
			entry.state = StateDegraded
			entry.failCount = 1
			entry.lastProbe = time.Now()
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
		entry.lastError = ""
		entry.healthyAt = time.Now()
	}
	entry.lastProbe = time.Now()
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
