package vesync

import (
	"context"
	"testing"
	"time"
	"github.com/stretchr/testify/require"
)

func TestAdapterOwnsVeSyncClientLifecycle(t *testing.T) {
	adapter, err := NewAdapter("vesync_test", 8*time.Second)
	require.NoError(t, err)
	require.Equal(t, "vesync_transport", adapter.Type())
	require.Error(t, adapter.HealthCheck(context.Background()))
	require.NoError(t, adapter.Start(context.Background()))
	require.NoError(t, adapter.HealthCheck(context.Background()))
	client, err := adapter.Client()
	require.NoError(t, err)
	require.NotNil(t, client)
	require.NoError(t, adapter.Stop(context.Background()))
	_, err = adapter.Client()
	require.Error(t, err)
}
