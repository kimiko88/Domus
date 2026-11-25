package plugins_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/domus-hub/domus/internal/plugins"
	"github.com/stretchr/testify/require"
)

type registryTestAdapter struct {
	id string
	kind string
}

func (a *registryTestAdapter) ID() string { return a.id }
func (a *registryTestAdapter) Type() string { return a.kind }
func (a *registryTestAdapter) Start(context.Context) error { return nil }
func (a *registryTestAdapter) Stop(context.Context) error { return nil }
func (a *registryTestAdapter) HealthCheck(context.Context) error { return nil }

type registryTestConfig struct {
	Host string `json:"host"`
	Port int `json:"port"`
}

func testManifest() plugins.Manifest {
	return plugins.Manifest{
		APIVersion: plugins.ManifestAPIVersion,
		ID: "sample_bridge",
		Name: "Sample bridge",
		Version: "1.2.3",
		Capabilities: []plugins.Capability{plugins.CapabilityDevicesRead, plugins.CapabilityNetworkLAN},
		Config: []plugins.ConfigField{
			{Name: "host", Type: "string", Required: true},
			{Name: "port", Type: "integer", Required: true},
		},
	}
}

func TestRegistryCreatesValidatedAdapter(t *testing.T) {
	registry := plugins.NewRegistry()
	require.NoError(t, registry.Register(testManifest(), func(ctx context.Context, id string, raw json.RawMessage, deps plugins.Dependencies) (plugins.Adapter, error) {
		cfg, err := plugins.DecodeConfig[registryTestConfig](raw)
		if err != nil { return nil, err }
		require.Equal(t, "192.0.2.10", cfg.Host)
		require.Equal(t, 502, cfg.Port)
		require.NotNil(t, deps.Logger)
		return &registryTestAdapter{id: id, kind: "sample_bridge"}, nil
	}))
	adapter, err := registry.Create(context.Background(), "sample_bridge", "kitchen_bridge", json.RawMessage(`{"host":"192.0.2.10","port":502}`), plugins.Dependencies{})
	require.NoError(t, err)
	require.Equal(t, "kitchen_bridge", adapter.ID())
	require.Equal(t, "sample_bridge", adapter.Type())
	require.Len(t, registry.List(), 1)
}

func TestRegistryRejectsDuplicateUnknownAndInvalidConfiguration(t *testing.T) {
	registry := plugins.NewRegistry()
	factory := func(context.Context, string, json.RawMessage, plugins.Dependencies) (plugins.Adapter, error) {
		return &registryTestAdapter{id: "unused", kind: "sample_bridge"}, nil
	}
	require.NoError(t, registry.Register(testManifest(), factory))
	require.ErrorIs(t, registry.Register(testManifest(), factory), plugins.ErrPluginExists)
	_, err := registry.Create(context.Background(), "missing", "instance_1", nil, plugins.Dependencies{})
	require.ErrorIs(t, err, plugins.ErrPluginNotFound)
	_, err = registry.Create(context.Background(), "sample_bridge", "instance_1", json.RawMessage(`{"host":"192.0.2.10"}`), plugins.Dependencies{})
	require.ErrorIs(t, err, plugins.ErrInvalidPluginConfig)
	_, err = registry.Create(context.Background(), "sample_bridge", "instance_1", json.RawMessage(`{"host":"192.0.2.10","port":"502"}`), plugins.Dependencies{})
	require.ErrorIs(t, err, plugins.ErrInvalidPluginConfig)
	_, err = registry.Create(context.Background(), "sample_bridge", "instance_1", json.RawMessage(`{"host":"192.0.2.10","port":502,"secret":"bad"}`), plugins.Dependencies{})
	require.ErrorIs(t, err, plugins.ErrInvalidPluginConfig)
}

func TestRegistryRejectsBadManifestAndFactoryMismatch(t *testing.T) {
	registry := plugins.NewRegistry()
	bad := testManifest()
	bad.Capabilities = append(bad.Capabilities, plugins.Capability("shell:root"))
	require.ErrorIs(t, registry.Register(bad, func(context.Context, string, json.RawMessage, plugins.Dependencies) (plugins.Adapter, error) {
		return nil, nil
	}), plugins.ErrInvalidManifest)

	require.NoError(t, registry.Register(testManifest(), func(context.Context, string, json.RawMessage, plugins.Dependencies) (plugins.Adapter, error) {
		return &registryTestAdapter{id: "wrong", kind: "other"}, nil
	}))
	_, err := registry.Create(context.Background(), "sample_bridge", "expected", json.RawMessage(`{"host":"192.0.2.10","port":502}`), plugins.Dependencies{})
	require.ErrorIs(t, err, plugins.ErrAdapterMismatch)
}

func TestDecodeConfigRejectsUnknownFields(t *testing.T) {
	_, err := plugins.DecodeConfig[registryTestConfig](json.RawMessage(`{"host":"x","port":502,"command":"curl | sh"}`))
	require.ErrorIs(t, err, plugins.ErrInvalidPluginConfig)
	_, err = plugins.DecodeConfig[registryTestConfig](json.RawMessage(`{"host":"x","port":502} {}`))
	require.Error(t, err)
}

func TestRegistryFactoryErrorIsWrapped(t *testing.T) {
	registry := plugins.NewRegistry()
	factoryErr := errors.New("device transport unavailable")
	require.NoError(t, registry.Register(testManifest(), func(context.Context, string, json.RawMessage, plugins.Dependencies) (plugins.Adapter, error) {
		return nil, factoryErr
	}))
	_, err := registry.Create(context.Background(), "sample_bridge", "instance_1", json.RawMessage(`{"host":"192.0.2.10","port":502}`), plugins.Dependencies{})
	require.ErrorIs(t, err, factoryErr)
}
