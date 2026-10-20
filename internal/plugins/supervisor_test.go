package plugins_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/domus-hub/domus/internal/plugins"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockAdapter struct {
	id        string
	pType     string
	failCheck atomic.Bool
	panic     atomic.Bool
}

func (m *mockAdapter) ID() string   { return m.id }
func (m *mockAdapter) Type() string { return m.pType }
func (m *mockAdapter) Start(ctx context.Context) error {
	if m.panic.Load() {
		panic("hardware adapter driver explosion")
	}
	return nil
}
func (m *mockAdapter) Stop(ctx context.Context) error { return nil }
func (m *mockAdapter) HealthCheck(ctx context.Context) error {
	if m.panic.Load() {
		panic("healthcheck panic")
	}
	if m.failCheck.Load() {
		return errors.New("modbus tcp connection refused")
	}
	return nil
}

func TestSupervisor_LifecycleAndHealthyCheck(t *testing.T) {
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 50, Workers: 1})
	defer bus.Close()

	sup := plugins.NewSupervisor(bus)

	ad1 := &mockAdapter{id: "modbus_inverter", pType: "modbus_tcp"}
	err := sup.Register(ad1, plugins.BreakerPolicy{
		MaxFailures:         3,
		InitialBackoff:      50 * time.Millisecond,
		MaxBackoff:          200 * time.Millisecond,
		HealthCheckInterval: 50 * time.Millisecond,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = sup.Start(ctx)
	require.NoError(t, err)
	defer sup.Stop(ctx)

	st, ok := sup.GetStatus("modbus_inverter")
	require.True(t, ok)
	assert.Equal(t, plugins.StateHealthy, st.State)
	assert.Equal(t, 0, st.FailCount)
}

func TestSupervisor_CircuitBreakerTrippingAndRecovery(t *testing.T) {
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 50, Workers: 1})
	defer bus.Close()

	sup := plugins.NewSupervisor(bus)

	ad := &mockAdapter{id: "knx_gateway", pType: "knx"}
	err := sup.Register(ad, plugins.BreakerPolicy{
		MaxFailures:         2,
		InitialBackoff:      50 * time.Millisecond,
		MaxBackoff:          100 * time.Millisecond,
		HealthCheckInterval: 20 * time.Millisecond,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	require.NoError(t, sup.Start(ctx))
	defer sup.Stop(ctx)

	// Inject failure
	ad.failCheck.Store(true)

	// Wait for probes to trip breaker
	assert.Eventually(t, func() bool {
		st, ok := sup.GetStatus("knx_gateway")
		return ok && st.State == plugins.StateTripped
	}, 1500*time.Millisecond, 20*time.Millisecond, "Circuit breaker should trip after max failures")

	st, ok := sup.GetStatus("knx_gateway")
	require.True(t, ok)
	assert.GreaterOrEqual(t, st.FailCount, 2)
	assert.Contains(t, st.LastError, "connection refused")

	// Restore health
	ad.failCheck.Store(false)

	// Wait for breaker probe to recover
	assert.Eventually(t, func() bool {
		st, ok := sup.GetStatus("knx_gateway")
		return ok && st.State == plugins.StateHealthy
	}, 1500*time.Millisecond, 20*time.Millisecond, "Adapter should recover to healthy after backoff probe succeeds")

	st, _ = sup.GetStatus("knx_gateway")
	assert.Equal(t, 0, st.FailCount)
	assert.Empty(t, st.LastError)
}

func TestSupervisor_PanicIsolation(t *testing.T) {
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 50, Workers: 1})
	defer bus.Close()

	sup := plugins.NewSupervisor(bus)

	panickyAdapter := &mockAdapter{id: "corrupted_zwave", pType: "zwave_js"}
	panickyAdapter.panic.Store(true)

	err := sup.Register(panickyAdapter)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start should NOT panic or crash the process
	assert.NotPanics(t, func() {
		_ = sup.Start(ctx)
	})
	defer sup.Stop(ctx)

	st, ok := sup.GetStatus("corrupted_zwave")
	require.True(t, ok)
	assert.Equal(t, plugins.StateDegraded, st.State)
	assert.Contains(t, st.LastError, "panic in Start")
}
