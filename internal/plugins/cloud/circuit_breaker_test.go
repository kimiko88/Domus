// Package cloud_test provides unit and integration tests for cloud adapters and resilience patterns.
package cloud_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/plugins/cloud"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCircuitBreaker_StateTransitions(t *testing.T) {
	t.Run("successful calls remain in closed state", func(t *testing.T) {
		cb := cloud.NewCircuitBreaker(cloud.CircuitBreakerConfig{
			FailureThreshold: 3,
			ResetTimeout:     100 * time.Millisecond,
		})

		assert.Equal(t, cloud.StateClosed, cb.State())

		err := cb.Execute(context.Background(), func(ctx context.Context) error {
			return nil
		})
		require.NoError(t, err)
		assert.Equal(t, cloud.StateClosed, cb.State())
		assert.Equal(t, 0, cb.FailureCount())
	})

	t.Run("consecutive failures trip to open state and fail fast", func(t *testing.T) {
		cb := cloud.NewCircuitBreaker(cloud.CircuitBreakerConfig{
			FailureThreshold: 2,
			ResetTimeout:     100 * time.Millisecond,
		})

		customErr := errors.New("upstream cloud timeout")

		// First failure
		err := cb.Execute(context.Background(), func(ctx context.Context) error {
			return customErr
		})
		assert.Equal(t, customErr, err)
		assert.Equal(t, cloud.StateClosed, cb.State())
		assert.Equal(t, 1, cb.FailureCount())

		// Second failure trips breaker
		err = cb.Execute(context.Background(), func(ctx context.Context) error {
			return customErr
		})
		assert.Equal(t, customErr, err)
		assert.Equal(t, cloud.StateOpen, cb.State())

		// Subsequent calls fail fast without executing function
		executed := false
		err = cb.Execute(context.Background(), func(ctx context.Context) error {
			executed = true
			return nil
		})
		assert.ErrorIs(t, err, cloud.ErrCircuitBreakerOpen)
		assert.False(t, executed, "Operation must not be executed when circuit is open")
	})

	t.Run("transitions to half-open after reset timeout and recovers on success", func(t *testing.T) {
		cb := cloud.NewCircuitBreaker(cloud.CircuitBreakerConfig{
			FailureThreshold: 1,
			ResetTimeout:     50 * time.Millisecond,
		})

		// Trip to open
		_ = cb.Execute(context.Background(), func(ctx context.Context) error {
			return errors.New("cloud error")
		})
		assert.Equal(t, cloud.StateOpen, cb.State())

		// Wait for reset timeout
		time.Sleep(70 * time.Millisecond)

		// Next call should be allowed as half-open probe
		executed := false
		err := cb.Execute(context.Background(), func(ctx context.Context) error {
			executed = true
			return nil
		})
		require.NoError(t, err)
		assert.True(t, executed)
		// Should now be recovered to closed
		assert.Equal(t, cloud.StateClosed, cb.State())
		assert.Equal(t, 0, cb.FailureCount())
	})

	t.Run("reopens if half-open probe fails", func(t *testing.T) {
		cb := cloud.NewCircuitBreaker(cloud.CircuitBreakerConfig{
			FailureThreshold: 1,
			ResetTimeout:     50 * time.Millisecond,
		})

		_ = cb.Execute(context.Background(), func(ctx context.Context) error {
			return errors.New("cloud error")
		})
		assert.Equal(t, cloud.StateOpen, cb.State())

		time.Sleep(70 * time.Millisecond)

		// Probe fails
		probeErr := errors.New("probe failed")
		err := cb.Execute(context.Background(), func(ctx context.Context) error {
			return probeErr
		})
		assert.Equal(t, probeErr, err)
		assert.Equal(t, cloud.StateOpen, cb.State())

		// Immediately subsequent call is rejected
		err = cb.Execute(context.Background(), func(ctx context.Context) error {
			return nil
		})
		assert.ErrorIs(t, err, cloud.ErrCircuitBreakerOpen)
	})
}
