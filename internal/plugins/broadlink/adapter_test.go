package broadlink

import (
	"context"
	"testing"

	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/stretchr/testify/require"
)

func TestBroadlinkAdapterSupervisorLifecycle(t *testing.T) {
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 32, Workers: 1})
	defer bus.Close()
	adapter := NewAdapter("broadlink_test", bus)
	ctx := context.Background()
	require.Error(t, adapter.HealthCheck(ctx))
	require.NoError(t, adapter.Start(ctx))
	require.NoError(t, adapter.HealthCheck(ctx))
	require.NoError(t, adapter.Start(ctx))
	require.NoError(t, adapter.Stop(ctx))
	require.Error(t, adapter.HealthCheck(ctx))
	require.NoError(t, adapter.Start(ctx), "adapter should be restartable")
	require.NoError(t, adapter.HealthCheck(ctx))
	require.NoError(t, adapter.Stop(ctx))
}
