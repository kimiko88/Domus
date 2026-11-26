package http

import (
	"context"
	"encoding/hex"
	"testing"
	"time"

	apppluginruntime "github.com/domus-hub/domus/internal/application/plugin_runtime"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/crypto"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	"github.com/domus-hub/domus/internal/plugins/broadlink"
	"github.com/stretchr/testify/require"
)

func newBroadlinkMigrationManager(t *testing.T) (*apppluginruntime.Manager, *sqlite.Repositories, *crypto.EnvelopeCipher) {
	t.Helper()
	ctx := context.Background()
	db, err := sqlite.Open(config.DatabaseConfig{Path: ":memory:", WALMode: false, BusyTimeoutMs: 1000, MaxOpenConns: 1})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, sqlite.Migrate(ctx, db))
	cipher, err := crypto.NewEnvelopeCipher([]byte("broadlink-migration-test-key"))
	require.NoError(t, err)
	repos := sqlite.NewRepositories(db)
	repos.Cipher = cipher
	manager, err := apppluginruntime.NewManager(repos, nil)
	require.NoError(t, err)
	return manager, repos, cipher
}

func createLegacyBroadlinkDevice(t *testing.T, repos *sqlite.Repositories, cipher *crypto.EnvelopeCipher) {
	t.Helper()
	ctx := context.Background()
	_, err := repos.DB.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS broadlink_devices (
		mac TEXT PRIMARY KEY, name TEXT NOT NULL, ip TEXT NOT NULL,
		device_type INTEGER NOT NULL, device_id INTEGER NOT NULL,
		aes_key_enc TEXT NOT NULL, updated_at INTEGER NOT NULL
	)`)
	require.NoError(t, err)
	_, err = repos.DB.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS broadlink_commands (
		mac TEXT NOT NULL, name TEXT NOT NULL, medium TEXT NOT NULL,
		code_base64 TEXT NOT NULL, created_at INTEGER NOT NULL,
		PRIMARY KEY(mac, name, medium)
	)`)
	require.NoError(t, err)
	encryptedKey, err := cipher.Encrypt(hex.EncodeToString([]byte("0123456789abcdef")))
	require.NoError(t, err)
	_, err = repos.DB.ExecContext(ctx, `INSERT INTO broadlink_devices(mac,name,ip,device_type,device_id,aes_key_enc,updated_at)
		VALUES(?,?,?,?,?,?,?)`, "aa:bb:cc:dd:ee:ff", "Living room RM", "192.168.1.50", 0x2737, 1234, encryptedKey, time.Now().Unix())
	require.NoError(t, err)
}

func TestRestoreLegacyBroadlinkInstanceAdoptsOnlyPopulatedStorage(t *testing.T) {
	manager, repos, cipher := newBroadlinkMigrationManager(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	require.NoError(t, restoreLegacyBroadlinkInstance(ctx, manager, repos.DB))
	instances, err := manager.ListInstances(ctx)
	require.NoError(t, err)
	require.Empty(t, instances, "fresh installations should not get an unsolicited BroadLink instance")

	createLegacyBroadlinkDevice(t, repos, cipher)
	require.NoError(t, restoreLegacyBroadlinkInstance(ctx, manager, repos.DB))
	instances, err = manager.ListInstances(ctx)
	require.NoError(t, err)
	require.Len(t, instances, 1)
	require.Equal(t, broadlink.DefaultIntegrationID, instances[0].ID)
	require.Equal(t, broadlink.PluginType, instances[0].Type)
	require.True(t, instances[0].Enabled)

	stored, err := repos.Integrations.GetByID(ctx, broadlink.DefaultIntegrationID)
	require.NoError(t, err)
	require.NotContains(t, string(stored.ConfigEncrypted), "broadlink")
	plain, err := cipher.Decrypt(string(stored.ConfigEncrypted))
	require.NoError(t, err)
	require.JSONEq(t, `{}`, plain)

	require.NoError(t, manager.Start(ctx))
	adapter, ok := manager.GetAdapter(broadlink.DefaultIntegrationID)
	require.True(t, ok)
	typed, ok := adapter.(*broadlink.Adapter)
	require.True(t, ok)
	require.Len(t, typed.ListDevices(), 1, "paired devices must survive adoption")
	require.NoError(t, typed.HealthCheck(ctx))
	require.NoError(t, manager.Stop(context.Background()))
}

func TestRestoreLegacyBroadlinkInstanceDoesNotReenableDisabledInstance(t *testing.T) {
	manager, repos, cipher := newBroadlinkMigrationManager(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := manager.Create(ctx, broadlink.DefaultIntegrationID, broadlink.PluginType, "BroadLink Local", []byte(`{}`), false)
	require.NoError(t, err)
	createLegacyBroadlinkDevice(t, repos, cipher)
	require.NoError(t, restoreLegacyBroadlinkInstance(ctx, manager, repos.DB))
	require.NoError(t, manager.Start(ctx))
	defer manager.Stop(context.Background())

	instances, err := manager.ListInstances(ctx)
	require.NoError(t, err)
	require.Len(t, instances, 1)
	require.False(t, instances[0].Enabled)
	_, ok := manager.GetAdapter(broadlink.DefaultIntegrationID)
	require.False(t, ok, "a disabled BroadLink instance must not be prepared or started")
}
