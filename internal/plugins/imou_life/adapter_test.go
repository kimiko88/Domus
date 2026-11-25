package imou_life

import (
	"context"
	"testing"
	"github.com/stretchr/testify/require"
)

func TestAdapterOwnsClientLifecycle(t *testing.T) {
	adapter, err := NewAdapter("imou_test", 12e9)
	require.NoError(t, err)
	require.Equal(t, "imou_life_transport", adapter.Type())
	require.Error(t, adapter.HealthCheck(context.Background()))
	require.NoError(t, adapter.Start(context.Background()))
	require.NoError(t, adapter.HealthCheck(context.Background()))
	client, err := adapter.NewClient(Config{AccessKey:"project-key", SecretKey:"project-secret", DataCenter:"eu"})
	require.NoError(t, err)
	require.NotNil(t, client)
	require.NoError(t, adapter.Stop(context.Background()))
	_, err = adapter.NewClient(Config{AccessKey:"project-key", SecretKey:"project-secret", DataCenter:"eu"})
	require.Error(t, err)
}

func TestAdapterRejectsInvalidTimeoutAndIdentity(t *testing.T) {
	_, err := NewAdapter("", 0)
	require.Error(t, err)
	_, err = NewAdapter("imou_test", 30e9)
	require.Error(t, err)
}
