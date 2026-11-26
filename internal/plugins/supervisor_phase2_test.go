package plugins_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/plugins"
	"github.com/stretchr/testify/require"
)

type phase2LifecycleAdapter struct {
	id string
	kind string
	startFail atomic.Bool
	stopFail atomic.Bool
	starts atomic.Int32
	stops atomic.Int32
}

func (a *phase2LifecycleAdapter) ID() string { return a.id }
func (a *phase2LifecycleAdapter) Type() string { return a.kind }
func (a *phase2LifecycleAdapter) Start(context.Context) error {
	a.starts.Add(1)
	if a.startFail.Load() { return errors.New("injected start failure") }
	return nil
}
func (a *phase2LifecycleAdapter) Stop(context.Context) error {
	a.stops.Add(1)
	if a.stopFail.Load() { return errors.New("injected stop failure") }
	return nil
}
func (a *phase2LifecycleAdapter) HealthCheck(context.Context) error { return nil }

func TestSupervisorStartStopAreIdempotent(t *testing.T) {
	sup := plugins.NewSupervisor(nil)
	adapter := &phase2LifecycleAdapter{id: "phase2_idempotent", kind: "phase2_test"}
	require.NoError(t, sup.Register(adapter))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	require.NoError(t, sup.Start(ctx))
	require.NoError(t, sup.Start(ctx))
	require.EqualValues(t, 1, adapter.starts.Load())
	require.NoError(t, sup.Stop(ctx))
	require.NoError(t, sup.Stop(ctx))
	require.EqualValues(t, 1, adapter.stops.Load())

	require.NoError(t, sup.Start(ctx))
	require.EqualValues(t, 2, adapter.starts.Load())
	require.NoError(t, sup.Stop(ctx))
	require.EqualValues(t, 2, adapter.stops.Load())
}

func TestSupervisorUnregisterRetainsAdapterWhenStopFails(t *testing.T) {
	sup := plugins.NewSupervisor(nil)
	adapter := &phase2LifecycleAdapter{id: "phase2_stop_failure", kind: "phase2_test"}
	require.NoError(t, sup.Register(adapter))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, sup.Start(ctx))

	adapter.stopFail.Store(true)
	require.Error(t, sup.Unregister(ctx, adapter.ID()))
	status, ok := sup.GetStatus(adapter.ID())
	require.True(t, ok, "a failed stop must not orphan a potentially running adapter")
	require.Equal(t, plugins.StateDegraded, status.State)

	adapter.stopFail.Store(false)
	require.NoError(t, sup.Unregister(ctx, adapter.ID()))
	_, ok = sup.GetStatus(adapter.ID())
	require.False(t, ok)
	require.NoError(t, sup.Stop(ctx))
}

func TestSupervisorReplaceRollsBackWhenReplacementStartFails(t *testing.T) {
	sup := plugins.NewSupervisor(nil)
	original := &phase2LifecycleAdapter{id: "phase2_replace", kind: "phase2_test"}
	replacement := &phase2LifecycleAdapter{id: "phase2_replace", kind: "phase2_test"}
	replacement.startFail.Store(true)
	require.NoError(t, sup.Register(original))
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	require.NoError(t, sup.Start(ctx))

	require.Error(t, sup.Replace(ctx, original.ID(), replacement))
	status, ok := sup.GetStatus(original.ID())
	require.True(t, ok)
	require.Equal(t, plugins.StateHealthy, status.State)
	require.EqualValues(t, 2, original.starts.Load(), "the previous adapter must be restored after failed replacement")
	require.EqualValues(t, 1, original.stops.Load())
	require.EqualValues(t, 1, replacement.starts.Load())
	require.EqualValues(t, 1, replacement.stops.Load())

	replacement.startFail.Store(false)
	require.NoError(t, sup.Replace(ctx, original.ID(), replacement))
	status, ok = sup.GetStatus(original.ID())
	require.True(t, ok)
	require.Equal(t, plugins.StateHealthy, status.State)
	require.EqualValues(t, 2, replacement.starts.Load())
	require.NoError(t, sup.Stop(ctx))
}

func TestSupervisorRejectsInvalidAdapterAndPolicy(t *testing.T) {
	sup := plugins.NewSupervisor(nil)
	require.ErrorIs(t, sup.Register(nil), plugins.ErrInvalidAdapter)
	adapter := &phase2LifecycleAdapter{id: "phase2_bad_policy", kind: "phase2_test"}
	require.ErrorIs(t, sup.Register(adapter, plugins.BreakerPolicy{}), plugins.ErrInvalidBreakerPolicy)
	require.NoError(t, sup.Register(adapter))
}


func TestSupervisorRetriesFailedStopBeforeStartingAgain(t *testing.T) {
	sup := plugins.NewSupervisor(nil)
	adapter := &phase2LifecycleAdapter{id: "phase2_stop_retry", kind: "phase2_test"}
	require.NoError(t, sup.Register(adapter))
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	require.NoError(t, sup.Start(ctx))
	adapter.stopFail.Store(true)
	require.Error(t, sup.Stop(ctx))
	status, ok := sup.GetStatus(adapter.ID())
	require.True(t, ok)
	require.Equal(t, plugins.StateDegraded, status.State)

	adapter.stopFail.Store(false)
	require.NoError(t, sup.Start(ctx))
	require.EqualValues(t, 2, adapter.starts.Load())
	require.EqualValues(t, 2, adapter.stops.Load(), "Start must retry and confirm Stop before calling Start again")
	status, ok = sup.GetStatus(adapter.ID())
	require.True(t, ok)
	require.Equal(t, plugins.StateHealthy, status.State)
	require.NoError(t, sup.Stop(ctx))
}
