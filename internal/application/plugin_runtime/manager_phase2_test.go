package plugin_runtime

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/plugins"
	"github.com/stretchr/testify/require"
)

type phase2AdapterConfig struct {
	Endpoint string `json:"endpoint"`
	APIToken string `json:"api_token"`
	FailStart bool `json:"fail_start"`
}

type phase2Adapter struct {
	id string
	config phase2AdapterConfig
}

func (a *phase2Adapter) ID() string { return a.id }
func (a *phase2Adapter) Type() string { return "phase2_bridge" }
func (a *phase2Adapter) Start(context.Context) error {
	if a.config.FailStart {
		return errors.New("injected adapter startup failure")
	}
	return nil
}
func (a *phase2Adapter) Stop(context.Context) error { return nil }
func (a *phase2Adapter) HealthCheck(context.Context) error { return nil }

func registerPhase2TestPlugin(t *testing.T, manager *Manager) {
	t.Helper()
	manifest := plugins.Manifest{
		APIVersion: plugins.ManifestAPIVersion,
		ID: "phase2_bridge",
		Name: "Phase 2 Bridge",
		Version: "1.0.0",
		MultiInstance: true,
		Config: []plugins.ConfigField{
			{Name: "endpoint", Type: "url", Required: true},
			{Name: "api_token", Type: "string", Required: true, Secret: true},
			{Name: "fail_start", Type: "boolean"},
		},
	}
	err := manager.Registry().Register(manifest, func(_ context.Context, id string, raw json.RawMessage, _ plugins.Dependencies) (plugins.Adapter, error) {
		config, err := plugins.DecodeConfig[phase2AdapterConfig](raw)
		if err != nil {
			return nil, err
		}
		return &phase2Adapter{id: id, config: config}, nil
	})
	require.NoError(t, err)
}

func TestManagerUpdateConfigCommitsEncryptedSettingsAndRollsBackFailedReplacement(t *testing.T) {
	manager, repo, cipher := newTestManager(t)
	registerPhase2TestPlugin(t, manager)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, manager.Start(ctx))
	defer manager.Stop(context.Background())

	const original = `{"endpoint":"https://old.example.test/api","api_token":"old-secret-value-0123456789","fail_start":false}`
	instance, err := manager.Create(ctx, "phase2_bridge_one", "phase2_bridge", "Phase 2 test", json.RawMessage(original), true)
	require.NoError(t, err)
	require.Equal(t, "connected", instance.Status)
	oldAdapterValue, ok := manager.GetAdapter("phase2_bridge_one")
	require.True(t, ok)
	oldAdapter := oldAdapterValue.(*phase2Adapter)

	const updated = `{"endpoint":"https://new.example.test/api","api_token":"new-secret-value-0123456789","fail_start":false}`
	updatedInstance, err := manager.UpdateConfig(ctx, "phase2_bridge_one", json.RawMessage(updated))
	require.NoError(t, err)
	require.Equal(t, "connected", updatedInstance.Status)
	newAdapterValue, ok := manager.GetAdapter("phase2_bridge_one")
	require.True(t, ok)
	newAdapter := newAdapterValue.(*phase2Adapter)
	require.NotSame(t, oldAdapter, newAdapter)
	require.Equal(t, "https://new.example.test/api", newAdapter.config.Endpoint)
	require.Equal(t, "new-secret-value-0123456789", newAdapter.config.APIToken)

	stored, err := repo.GetByID(ctx, "phase2_bridge_one")
	require.NoError(t, err)
	require.NotContains(t, string(stored.ConfigEncrypted), "new-secret-value")
	plain, err := cipher.Decrypt(string(stored.ConfigEncrypted))
	require.NoError(t, err)
	require.JSONEq(t, updated, plain)

	const invalidRuntimeUpdate = `{"endpoint":"https://broken.example.test/api","api_token":"bad-secret-value-0123456789","fail_start":true}`
	_, err = manager.UpdateConfig(ctx, "phase2_bridge_one", json.RawMessage(invalidRuntimeUpdate))
	require.Error(t, err)

	restoredValue, ok := manager.GetAdapter("phase2_bridge_one")
	require.True(t, ok)
	require.Same(t, newAdapter, restoredValue, "manager must retain the known-good adapter after a failed replacement")
	status, ok := manager.supervisor.GetStatus("phase2_bridge_one")
	require.True(t, ok)
	require.Equal(t, plugins.StateHealthy, status.State)

	stored, err = repo.GetByID(ctx, "phase2_bridge_one")
	require.NoError(t, err)
	plain, err = cipher.Decrypt(string(stored.ConfigEncrypted))
	require.NoError(t, err)
	require.JSONEq(t, updated, plain)
}

func TestManagerEnableFailureDoesNotPersistEnabledState(t *testing.T) {
	manager, repo, _ := newTestManager(t)
	registerPhase2TestPlugin(t, manager)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, manager.Start(ctx))
	defer manager.Stop(context.Background())

	config := json.RawMessage(`{"endpoint":"https://bridge.example.test/api","api_token":"secret-value-0123456789","fail_start":true}`)
	_, err := manager.Create(ctx, "phase2_disabled", "phase2_bridge", "Disabled bridge", config, false)
	require.NoError(t, err)
	require.Error(t, manager.SetEnabled(ctx, "phase2_disabled", true))

	stored, err := repo.GetByID(ctx, "phase2_disabled")
	require.NoError(t, err)
	require.False(t, stored.Enabled, "failed enable must leave the durable instance disabled")
	_, prepared := manager.GetAdapter("phase2_disabled")
	require.False(t, prepared, "failed enable must not leave a half-registered adapter")
}
