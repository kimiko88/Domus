package midea

import (
	"context"
	"testing"
	"github.com/stretchr/testify/require"
)

func TestRuntimeAdapterLifecycleOwnsProtocolDrivers(t *testing.T) {
	adapter, err := NewRuntimeAdapter("midea_a1")
	require.NoError(t, err)
	require.Equal(t, "midea_a1_protocol", adapter.Type())
	require.Error(t, adapter.HealthCheck(context.Background()))
	require.NoError(t, adapter.Start(context.Background()))
	core, err := adapter.Core()
	require.NoError(t, err)
	require.NotNil(t, core)
	v2, v3 := adapter.Drivers()
	require.NotNil(t, v2)
	require.NotNil(t, v3)
	require.NoError(t, adapter.HealthCheck(context.Background()))
	require.NoError(t, adapter.Stop(context.Background()))
	_, err = adapter.Core()
	require.Error(t, err)
}
