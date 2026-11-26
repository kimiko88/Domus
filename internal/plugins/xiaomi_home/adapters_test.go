package xiaomi_home

import (
	"context"
	"testing"
	"time"
	"github.com/stretchr/testify/require"
)

func TestLocalAdapterLifecycle(t *testing.T) {
	adapter, err := NewLocalAdapter("miio_test", 2*time.Second)
	require.NoError(t, err)
	require.Equal(t, "xiaomi_miio_transport", adapter.Type())
	require.Error(t, adapter.HealthCheck(context.Background()))
	require.NoError(t, adapter.Start(context.Background()))
	client, err := adapter.Client()
	require.NoError(t, err)
	require.NotNil(t, client)
	require.NoError(t, adapter.HealthCheck(context.Background()))
	require.NoError(t, adapter.Stop(context.Background()))
	_, err = adapter.Client()
	require.Error(t, err)
}

func TestCloudAdapterLifecycleAndClientValidation(t *testing.T) {
	adapter, err := NewCloudAdapter("integration_xiaomi_home_cloud", 12*time.Second)
	require.NoError(t, err)
	require.Equal(t, CloudPluginType, adapter.Type())
	require.NoError(t, adapter.Start(context.Background()))
	client, err := adapter.NewClient(CloudConfig{
		ClientID: "123456789012",
		ClientSecret: "app-secret",
		RedirectURL: "https://domus.example/callback",
		Region: "eu",
	})
	require.NoError(t, err)
	require.NotNil(t, client)
	require.NoError(t, adapter.Stop(context.Background()))
	_, err = adapter.NewClient(CloudConfig{
		ClientID:"123456789012", ClientSecret:"app-secret",
		RedirectURL:"https://domus.example/callback", Region:"eu",
	})
	require.Error(t, err)
}
