// Package cloud provides cloud integration adapters, resilience patterns, and offline degradation stubs.
package cloud

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	// ErrCircuitBreakerOpen is returned when calls are rejected because the circuit is open.
	ErrCircuitBreakerOpen = errors.New("cloud: circuit breaker is open (offline)")
)

// State represents the current lifecycle state of a circuit breaker.
type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

// String returns a human-readable representation of the circuit state.
func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreakerConfig controls thresholds and timing for the circuit breaker.
type CircuitBreakerConfig struct {
	FailureThreshold int           // Number of consecutive failures to trip to Open
	ResetTimeout     time.Duration // Duration to remain Open before entering Half-Open
}

// CircuitBreaker protects the local system from slow or failing cloud dependencies.
type CircuitBreaker struct {
	mu               sync.Mutex
	cfg              CircuitBreakerConfig
	state            State
	consecutiveFails int
	openedAt         time.Time
}

// NewCircuitBreaker creates a configured CircuitBreaker in the Closed state.
func NewCircuitBreaker(cfg CircuitBreakerConfig) *CircuitBreaker {
	if cfg.FailureThreshold <= 0 {
		cfg.FailureThreshold = 3
	}
	if cfg.ResetTimeout <= 0 {
		cfg.ResetTimeout = 30 * time.Second
	}
	return &CircuitBreaker{
		cfg:   cfg,
		state: StateClosed,
	}
}

// State returns the current State of the circuit breaker.
func (cb *CircuitBreaker) State() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == StateOpen && time.Since(cb.openedAt) > cb.cfg.ResetTimeout {
		return StateHalfOpen
	}
	return cb.state
}

// FailureCount returns the current count of consecutive failures.
func (cb *CircuitBreaker) FailureCount() int {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.consecutiveFails
}

// Execute wraps an operation, enforcing circuit state transitions.
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	cb.mu.Lock()
	now := time.Now()

	if cb.state == StateOpen {
		if now.Sub(cb.openedAt) > cb.cfg.ResetTimeout {
			cb.state = StateHalfOpen
		} else {
			cb.mu.Unlock()
			return ErrCircuitBreakerOpen
		}
	}
	cb.mu.Unlock()

	err := fn(ctx)

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.consecutiveFails++
		if cb.state == StateHalfOpen || cb.consecutiveFails >= cb.cfg.FailureThreshold {
			cb.state = StateOpen
			cb.openedAt = time.Now()
		}
		return err
	}

	// Success resets counters and returns to Closed
	cb.consecutiveFails = 0
	cb.state = StateClosed
	return nil
}
