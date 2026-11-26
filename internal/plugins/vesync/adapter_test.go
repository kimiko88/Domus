package vesync

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAdapterOwnsVeSyncClientLifecycle(t *testing.T) {
	adapter, err := NewAdapter(IntegrationID, 8*time.Second)
	require.NoError(t, err)
	require.Equal(t, PluginType, adapter.Type())
	require.Equal(t, IntegrationID, adapter.ID())
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

func TestAdapterRejectsInvalidIdentityTimeoutAndSession(t *testing.T) {
	_, err := NewAdapter("vesync_transport", 8*time.Second)
	require.ErrorContains(t, err, "instance ID")

	_, err = NewAdapter(IntegrationID, 21*time.Second)
	require.ErrorContains(t, err, "timeout")

	_, err = NewAdapterWithSession(IntegrationID, Session{
		AccountID: "account-1",
	}, 8*time.Second)
	require.ErrorContains(t, err, "both account ID and token")

	_, err = NewAdapterWithSession(IntegrationID, Session{
		AccountID: "account-1", Token: "private-token", Region: "APAC",
	}, 8*time.Second)
	require.ErrorContains(t, err, "unsupported cloud region")

	_, err = NewAdapterWithSession(IntegrationID, Session{}, 8*time.Second)
	require.NoError(t, err, "a newly-installed but not yet connected instance is valid")
}
