package midea

import (
	"context"
	"testing"
	"github.com/stretchr/testify/require"
)

func TestRuntimeAdapterLifecycleOwnsProtocolDrivers(t *testing.T) {
	adapter, err := NewRuntimeAdapter(IntegrationID)
	require.NoError(t, err)
	require.Equal(t, PluginType, adapter.Type())
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

func TestManagedRuntimeAdapterRestoresOnlyValidatedLANDevices(t *testing.T) {
	device := Device{
		ID: "123456789", Name: "Basement dehumidifier", TypeCode: "A1",
		Kind: KindDehumidifier, Protocol: ProtocolMSmartV2,
		Host: "192.168.1.50", Port: 6444,
	}
	adapter, err := NewRuntimeAdapterWithDevices(IntegrationID, []A1LANConfig{{
		Device: device, ApplianceID: 123456789, MessageVersion: 3,
	}})
	require.NoError(t, err)
	require.Equal(t, IntegrationID, adapter.ID())
	require.Equal(t, PluginType, adapter.Type())
	require.Error(t, adapter.HealthCheck(context.Background()))
	require.NoError(t, adapter.Start(context.Background()))
	core, err := adapter.Core()
	require.NoError(t, err)
	require.Len(t, core.ListDevices(), 1)
	require.Equal(t, "123456789", core.ListDevices()[0].ID)
	require.NoError(t, adapter.Stop(context.Background()))
}

func TestManagedRuntimeAdapterValidatesStableIDAndDeviceSecrets(t *testing.T) {
	_, err := NewRuntimeAdapterWithDevices("other_instance", nil)
	require.ErrorContains(t, err, "stable integration instance ID")

	device := Device{
		ID: "123456789", Name: "Comfee dehumidifier", TypeCode: "A1",
		Kind: KindDehumidifier, Protocol: ProtocolMSmartV3,
		Host: "192.168.1.50", Port: 6444,
	}
	_, err = NewRuntimeAdapterWithDevices(IntegrationID, []A1LANConfig{{
		Device: device, ApplianceID: 123456789,
		TokenHex: "short", KeyHex: "short",
	}})
	require.Error(t, err)

	_, err = NewRuntimeAdapterWithDevices(IntegrationID, make([]A1LANConfig, maxManagedDevices+1))
	require.ErrorContains(t, err, "at most")
}
