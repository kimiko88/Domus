package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	apppluginruntime "github.com/domus-hub/domus/internal/application/plugin_runtime"
	"github.com/domus-hub/domus/internal/domain/integration"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/crypto"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	"github.com/domus-hub/domus/internal/plugins/vesync"
	"github.com/stretchr/testify/require"
)

type vesyncPluginFixture struct {
	repos   *sqlite.Repositories
	cipher  *crypto.EnvelopeCipher
	manager *apppluginruntime.Manager
	handler *VeSyncHandler
}

func newVeSyncPluginFixture(t *testing.T) *vesyncPluginFixture {
	t.Helper()
	ctx := context.Background()
	db, err := sqlite.Open(config.DatabaseConfig{
		Path: ":memory:", WALMode: false, BusyTimeoutMs: 1000, MaxOpenConns: 1,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, sqlite.Migrate(ctx, db))

	cipher, err := crypto.NewEnvelopeCipher([]byte("vesync-plugin-test-master-key"))
	require.NoError(t, err)
	repos := sqlite.NewRepositories(db)
	repos.Cipher = cipher
	manager, err := apppluginruntime.NewManager(repos, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = manager.Stop(context.Background()) })

	handler, err := NewVeSyncHandler(repos.Integrations, cipher)
	require.NoError(t, err)
	handler.WithPluginManager(manager)
	return &vesyncPluginFixture{repos: repos, cipher: cipher, manager: manager, handler: handler}
}

func veSyncTestSession() vesync.Session {
	return vesync.Session{
		AccountID: "vesync-account-001",
		Token: "private-vesync-session-token",
		CountryCode: "IT",
		Region: "EU",
		TimeZone: "Europe/Rome",
	}
}

func seedVeSyncPluginRecord(t *testing.T, f *vesyncPluginFixture) vesync.Session {
	t.Helper()
	session := veSyncTestSession()
	raw, err := json.Marshal(vesyncStoredConfig{Session: session})
	require.NoError(t, err)
	encrypted, err := f.cipher.Encrypt(string(raw))
	require.NoError(t, err)
	now := time.Now().UTC()
	require.NoError(t, f.repos.Integrations.Create(context.Background(), &integration.Integration{
		ID: vesync.IntegrationID,
		Type: vesync.PluginType,
		Name: "VeSync Cloud",
		Enabled: true,
		Status: "connected",
		ConfigEncrypted: []byte(encrypted),
		CreatedAt: now,
		UpdatedAt: now,
	}))
	return session
}

func TestVeSyncPluginManagerRestoresLegacyEncryptedSession(t *testing.T) {
	f := newVeSyncPluginFixture(t)
	expected := seedVeSyncPluginRecord(t, f)

	require.NoError(t, f.manager.Start(context.Background()))
	got, err := f.handler.loadConfig(httptest.NewRequest(http.MethodGet, "/status", nil))
	require.NoError(t, err)
	require.Equal(t, expected, got.Session)

	adapter, ok := f.manager.GetAdapter(vesync.IntegrationID)
	require.True(t, ok, "the historical encrypted row must be restored by Plugin Manager")
	typed, ok := adapter.(*vesync.Adapter)
	require.True(t, ok)
	require.Equal(t, vesync.PluginType, typed.Type())
	require.NoError(t, typed.HealthCheck(context.Background()))

	record, err := f.repos.Integrations.GetByID(context.Background(), vesync.IntegrationID)
	require.NoError(t, err)
	require.Equal(t, vesync.PluginType, record.Type)
	require.NotContains(t, string(record.ConfigEncrypted), expected.Token)
	require.NotContains(t, string(record.ConfigEncrypted), expected.AccountID)

	response := httptest.NewRecorder()
	f.handler.Status(response, httptest.NewRequest(http.MethodGet, "/status", nil))
	require.Equal(t, http.StatusOK, response.Code)
	require.NotContains(t, response.Body.String(), expected.Token)
	require.NotContains(t, response.Body.String(), expected.AccountID)
	require.Contains(t, response.Body.String(), "EU")
}

func TestVeSyncManagedSessionCreateDisconnectAndReconnect(t *testing.T) {
	f := newVeSyncPluginFixture(t)
	require.NoError(t, f.manager.Start(context.Background()))

	session := veSyncTestSession()
	require.NoError(t, f.handler.saveConfig(context.Background(), vesyncStoredConfig{Session: session}, "connected"))

	raw, err := f.manager.GetConfig(context.Background(), vesync.IntegrationID)
	require.NoError(t, err)
	var saved vesyncStoredConfig
	require.NoError(t, json.Unmarshal(raw, &saved))
	require.Equal(t, session, saved.Session)

	record, err := f.repos.Integrations.GetByID(context.Background(), vesync.IntegrationID)
	require.NoError(t, err)
	require.Equal(t, vesync.PluginType, record.Type)
	require.True(t, record.Enabled)
	require.NotContains(t, string(record.ConfigEncrypted), session.Token)

	first, ok := f.manager.GetAdapter(vesync.IntegrationID)
	require.True(t, ok)
	require.NoError(t, first.HealthCheck(context.Background()))
	firstTyped, ok := first.(*vesync.Adapter)
	require.True(t, ok)

	// Public provider settings are retained when the private account session is
	// replaced through the account-linking flow.
	_, err = f.manager.UpdateConfig(context.Background(), vesync.IntegrationID, json.RawMessage(`{"http_timeout":"12s"}`))
	require.NoError(t, err)
	rotatedSession := session
	rotatedSession.Token = "private-vesync-rotated-session-token"
	require.NoError(t, f.handler.saveConfig(context.Background(), vesyncStoredConfig{Session: rotatedSession}, "connected"))
	managedConfig, err := f.manager.GetConfig(context.Background(), vesync.IntegrationID)
	require.NoError(t, err)
	var managedValues map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(managedConfig, &managedValues))
	require.JSONEq(t, `"12s"`, string(managedValues["http_timeout"]))
	currentClient, err := f.handler.clientForRequest(context.Background(), false)
	require.NoError(t, err, "handler must resolve the current adapter after config replacement")
	require.NotNil(t, currentClient)
	_, staleErr := firstTyped.Client()
	require.Error(t, staleErr, "replaced adapter must be stopped")

	require.NoError(t, f.handler.saveConfig(context.Background(), vesyncStoredConfig{}, "disconnected"))
	_, ok = f.manager.GetAdapter(vesync.IntegrationID)
	require.False(t, ok, "disconnect must disable the managed adapter")

	raw, err = f.manager.GetConfig(context.Background(), vesync.IntegrationID)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &saved))
	require.Empty(t, saved.Session.Token)
	require.Empty(t, saved.Session.AccountID)

	record, err = f.repos.Integrations.GetByID(context.Background(), vesync.IntegrationID)
	require.NoError(t, err)
	require.False(t, record.Enabled)
	require.Equal(t, "disconnected", record.Status)

	reconnected := veSyncTestSession()
	reconnected.Token = "private-vesync-new-session-token"
	require.NoError(t, f.handler.saveConfig(context.Background(), vesyncStoredConfig{Session: reconnected}, "connected"))

	adapter, ok := f.manager.GetAdapter(vesync.IntegrationID)
	require.True(t, ok, "a new account login must re-enable the stable instance")
	require.NoError(t, adapter.HealthCheck(context.Background()))
	reconnectedClient, err := f.handler.clientForRequest(context.Background(), false)
	require.NoError(t, err)
	require.NotNil(t, reconnectedClient)
	raw, err = f.manager.GetConfig(context.Background(), vesync.IntegrationID)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &saved))
	require.Equal(t, reconnected, saved.Session)
}


func TestVeSyncStatusDistinguishesConfiguredFromDisabledRuntime(t *testing.T) {
	f := newVeSyncPluginFixture(t)
	seedVeSyncPluginRecord(t, f)
	require.NoError(t, f.manager.Start(context.Background()))
	require.NoError(t, f.manager.SetEnabled(context.Background(), vesync.IntegrationID, false))

	response := httptest.NewRecorder()
	f.handler.Status(response, httptest.NewRequest(http.MethodGet, "/status", nil))
	require.Equal(t, http.StatusOK, response.Code)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &payload))
	require.Equal(t, true, payload["configured"], "the encrypted account session must remain configured")
	require.Equal(t, false, payload["connected"], "a disabled plugin adapter is not connected")

	stored, err := f.manager.GetConfig(context.Background(), vesync.IntegrationID)
	require.NoError(t, err)
	var config vesyncStoredConfig
	require.NoError(t, json.Unmarshal(stored, &config))
	require.Equal(t, "private-vesync-session-token", config.Session.Token, "manual disable must not erase account credentials")
}

func TestVeSyncPublicPluginCatalogDoesNotExposeSession(t *testing.T) {
	f := newVeSyncPluginFixture(t)
	types := f.manager.ListTypes()
	var found bool
	for _, manifest := range types {
		if manifest.ID != vesync.PluginType {
			continue
		}
		found = true
		require.False(t, manifest.Internal)
		for _, field := range manifest.Config {
			require.NotEqual(t, "session", field.Name)
			require.NotContains(t, strings.ToLower(field.Name), "token")
		}
	}
	require.True(t, found, "VeSync must be visible in the public plugin catalog")
}


func TestVeSyncBootstrapClientCannotBypassAnEnabledMissingAdapter(t *testing.T) {
	t.Run("first login without an instance", func(t *testing.T) {
		f := newVeSyncPluginFixture(t)
		client, err := f.handler.clientForRequest(context.Background(), true)
		require.NoError(t, err)
		require.NotNil(t, client)
	})

	t.Run("explicitly disabled instance", func(t *testing.T) {
		f := newVeSyncPluginFixture(t)
		_, err := f.manager.Create(context.Background(), vesync.IntegrationID, vesync.PluginType, "VeSync Cloud", json.RawMessage(`{}`), false)
		require.NoError(t, err)
		client, err := f.handler.clientForRequest(context.Background(), true)
		require.NoError(t, err)
		require.NotNil(t, client)
		_, err = f.handler.clientForRequest(context.Background(), false)
		require.Error(t, err, "normal cloud calls must not use the bootstrap client")
	})

	t.Run("enabled but not restored instance", func(t *testing.T) {
		f := newVeSyncPluginFixture(t)
		seedVeSyncPluginRecord(t, f)
		_, ok := f.manager.GetAdapter(vesync.IntegrationID)
		require.False(t, ok, "the manager has not started and has not restored the enabled row")
		client, err := f.handler.clientForRequest(context.Background(), true)
		require.ErrorContains(t, err, "enabled managed cloud adapter is unavailable")
		require.Nil(t, client, "an enabled persisted integration must not be bypassed with a direct client")
	})
}
