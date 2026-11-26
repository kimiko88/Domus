package handlers

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	apppluginruntime "github.com/domus-hub/domus/internal/application/plugin_runtime"
	"github.com/domus-hub/domus/internal/domain/integration"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/crypto"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	"github.com/domus-hub/domus/internal/plugins/midea"
	"github.com/stretchr/testify/require"
)

type mideaPluginFixture struct {
	repos   *sqlite.Repositories
	cipher  *crypto.EnvelopeCipher
	manager *apppluginruntime.Manager
	handler *MideaHandler
}

func newMideaPluginFixture(t *testing.T) *mideaPluginFixture {
	t.Helper()
	ctx := context.Background()
	db, err := sqlite.Open(config.DatabaseConfig{
		Path: ":memory:", WALMode: false, BusyTimeoutMs: 1000, MaxOpenConns: 1,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, sqlite.Migrate(ctx, db))

	cipher, err := crypto.NewEnvelopeCipher([]byte("midea-plugin-test-master-key"))
	require.NoError(t, err)
	repos := sqlite.NewRepositories(db)
	repos.Cipher = cipher
	manager, err := apppluginruntime.NewManager(repos, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = manager.Stop(context.Background()) })

	handler, err := NewMideaHandler(repos.Integrations, cipher, repos.Devices, repos.Entities)
	require.NoError(t, err)
	handler.WithPluginManager(manager)
	require.NoError(t, manager.Start(ctx))
	return &mideaPluginFixture{repos: repos, cipher: cipher, manager: manager, handler: handler}
}

func mideaPluginTestDevice(id, host string, protocol midea.Protocol) midea.A1LANConfig {
	device := midea.Device{
		ID: id, Name: "Comfee basement dehumidifier", Brand: "Comfee",
		Model: "MDDF16WF", TypeCode: "A1", Kind: midea.KindDehumidifier,
		Protocol: protocol, Host: host, Port: 6444,
	}
	config := midea.A1LANConfig{Device: device, ApplianceID: 123456789, MessageVersion: 3}
	if protocol == midea.ProtocolMSmartV3 {
		config.TokenHex = strings.Repeat("a", 128)
		config.KeyHex = strings.Repeat("b", 64)
	}
	return config
}

func TestMideaPluginManagerRestoresEncryptedLegacyDeviceConfiguration(t *testing.T) {
	ctx := context.Background()
	f := newMideaPluginFixtureWithoutStart(t)

	config := mideaStoredConfig{Devices: []midea.A1LANConfig{
		mideaPluginTestDevice("123456789", "192.168.1.50", midea.ProtocolMSmartV3),
	}}
	raw, err := json.Marshal(config)
	require.NoError(t, err)
	encrypted, err := f.cipher.Encrypt(string(raw))
	require.NoError(t, err)
	now := time.Now().UTC()
	require.NoError(t, f.repos.Integrations.Create(ctx, &integration.Integration{
		ID: midea.IntegrationID, Type: midea.PluginType,
		Name: "Midea / Comfee SmartHome (LAN)", Enabled: true,
		Status: "connected", ConfigEncrypted: []byte(encrypted),
		CreatedAt: now, UpdatedAt: now,
	}))

	handler, err := NewMideaHandler(f.repos.Integrations, f.cipher, f.repos.Devices, f.repos.Entities)
	require.NoError(t, err)
	handler.WithPluginManager(f.manager)
	require.Empty(t, handler.adapter.ListDevices(), "attaching Plugin Manager must discard the unmanaged constructor adapter")
	require.NoError(t, f.manager.Start(ctx))
	adapter, ok := f.manager.GetAdapter(midea.IntegrationID)
	require.True(t, ok)
	runtime, ok := adapter.(*midea.RuntimeAdapter)
	require.True(t, ok)
	require.NoError(t, handler.WithRuntimeAdapter(runtime))
	require.NoError(t, runtime.HealthCheck(ctx))

	core, err := runtime.Core()
	require.NoError(t, err)
	require.Len(t, core.ListDevices(), 1)
	require.Equal(t, "123456789", core.ListDevices()[0].ID)

	record, err := f.repos.Integrations.GetByID(ctx, midea.IntegrationID)
	require.NoError(t, err)
	require.NotContains(t, string(record.ConfigEncrypted), config.Devices[0].TokenHex)
	require.NotContains(t, string(record.ConfigEncrypted), config.Devices[0].KeyHex)
	require.NotContains(t, string(record.ConfigEncrypted), "192.168.1.50")

	instances, err := f.manager.ListInstances(ctx)
	require.NoError(t, err)
	publicJSON, err := json.Marshal(instances)
	require.NoError(t, err)
	require.NotContains(t, string(publicJSON), config.Devices[0].TokenHex)
	require.NotContains(t, string(publicJSON), config.Devices[0].KeyHex)

	for _, manifest := range f.manager.ListTypes() {
		if manifest.ID == midea.PluginType {
			require.Empty(t, manifest.Config, "internal device mappings must not be exposed in the catalog")
		}
	}
	require.Len(t, handler.adapter.ListDevices(), 1)
	require.NotNil(t, handler.adapterSnapshot(), "enabled managed instance must be used for requests")
	require.NoError(t, f.manager.SetEnabled(ctx, midea.IntegrationID, false))
	require.Nil(t, handler.adapterSnapshot(), "disabling the managed instance must not fall back to an unmanaged adapter")
}

func newMideaPluginFixtureWithoutStart(t *testing.T) *mideaPluginFixture {
	t.Helper()
	ctx := context.Background()
	db, err := sqlite.Open(config.DatabaseConfig{
		Path: ":memory:", WALMode: false, BusyTimeoutMs: 1000, MaxOpenConns: 1,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, sqlite.Migrate(ctx, db))
	cipher, err := crypto.NewEnvelopeCipher([]byte("midea-plugin-test-master-key"))
	require.NoError(t, err)
	repos := sqlite.NewRepositories(db)
	repos.Cipher = cipher
	manager, err := apppluginruntime.NewManager(repos, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = manager.Stop(context.Background()) })
	return &mideaPluginFixture{repos: repos, cipher: cipher, manager: manager}
}

func TestMideaPluginManagerCreateReplaceAndDisconnectKeepsStableInstance(t *testing.T) {
	ctx := context.Background()
	f := newMideaPluginFixture(t)
	firstConfig := mideaStoredConfig{Devices: []midea.A1LANConfig{
		mideaPluginTestDevice("123456789", "192.168.1.50", midea.ProtocolMSmartV3),
	}}
	f.handler.mu.Lock()
	f.handler.config = firstConfig
	require.NoError(t, f.handler.saveConfigLocked(ctx, "connected"))
	f.handler.mu.Unlock()

	firstAdapter, ok := f.manager.GetAdapter(midea.IntegrationID)
	require.True(t, ok)
	firstRuntime, ok := firstAdapter.(*midea.RuntimeAdapter)
	require.True(t, ok)
	require.NoError(t, firstRuntime.HealthCheck(ctx))
	firstCore, err := firstRuntime.Core()
	require.NoError(t, err)
	require.Len(t, firstCore.ListDevices(), 1)

	rotated := mideaPluginTestDevice("123456789", "192.168.1.50", midea.ProtocolMSmartV3)
	rotated.TokenHex = strings.Repeat("c", 128)
	second := mideaPluginTestDevice("123456790", "192.168.1.51", midea.ProtocolMSmartV2)
	f.handler.mu.Lock()
	f.handler.config = mideaStoredConfig{Devices: []midea.A1LANConfig{rotated, second}}
	require.NoError(t, f.handler.saveConfigLocked(ctx, "connected"))
	f.handler.mu.Unlock()

	secondAdapter, ok := f.manager.GetAdapter(midea.IntegrationID)
	require.True(t, ok)
	secondRuntime, ok := secondAdapter.(*midea.RuntimeAdapter)
	require.True(t, ok)
	require.NotSame(t, firstRuntime, secondRuntime, "a configuration mutation must replace the runtime adapter")
	require.Error(t, firstRuntime.HealthCheck(ctx), "the previous runtime must be stopped after replacement")
	secondCore, err := secondRuntime.Core()
	require.NoError(t, err)
	require.Len(t, secondCore.ListDevices(), 2)

	persisted, err := f.manager.GetConfig(ctx, midea.IntegrationID)
	require.NoError(t, err)
	var restored mideaStoredConfig
	require.NoError(t, json.Unmarshal(persisted, &restored))
	require.Equal(t, rotated.TokenHex, restored.Devices[0].TokenHex)

	f.handler.mu.Lock()
	f.handler.config = mideaStoredConfig{}
	require.NoError(t, f.handler.saveConfigLocked(ctx, "disconnected"))
	f.handler.mu.Unlock()

	_, ok = f.manager.GetAdapter(midea.IntegrationID)
	require.False(t, ok, "disconnect must stop and remove the runtime adapter")
	persisted, err = f.manager.GetConfig(ctx, midea.IntegrationID)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(persisted, &restored))
	require.Empty(t, restored.Devices, "disconnect must remove device credentials from managed config")

	record, err := f.repos.Integrations.GetByID(ctx, midea.IntegrationID)
	require.NoError(t, err)
	require.False(t, record.Enabled)
	require.Equal(t, "disconnected", record.Status)
	require.Equal(t, midea.PluginType, record.Type)
}
