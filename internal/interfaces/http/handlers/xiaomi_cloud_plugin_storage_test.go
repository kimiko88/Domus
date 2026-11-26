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
	"github.com/domus-hub/domus/internal/plugins/xiaomi_home"
	"github.com/stretchr/testify/require"
)

type xiaomiCloudPluginFixture struct {
	repos   *sqlite.Repositories
	cipher  *crypto.EnvelopeCipher
	manager *apppluginruntime.Manager
	handler *XiaomiCloudHandler
}

func newXiaomiCloudPluginFixture(t *testing.T) *xiaomiCloudPluginFixture {
	t.Helper()
	ctx := context.Background()
	db, err := sqlite.Open(config.DatabaseConfig{
		Path: ":memory:", WALMode: false, BusyTimeoutMs: 1000, MaxOpenConns: 1,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, sqlite.Migrate(ctx, db))

	cipher, err := crypto.NewEnvelopeCipher([]byte("xiaomi-plugin-test-master-key"))
	require.NoError(t, err)
	repos := sqlite.NewRepositories(db)
	repos.Cipher = cipher
	manager, err := apppluginruntime.NewManager(repos, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = manager.Stop(context.Background()) })
	handler, err := NewXiaomiCloudHandler(repos.Integrations, repos.Devices, repos.Entities, cipher, nil)
	require.NoError(t, err)
	handler.WithPluginManager(manager)
	return &xiaomiCloudPluginFixture{repos: repos, cipher: cipher, manager: manager, handler: handler}
}

func seedXiaomiCloudPluginRecord(t *testing.T, f *xiaomiCloudPluginFixture) *xiaomiCloudConfig {
	t.Helper()
	config := xiaomiCloudConfig{
		ClientID: "123456789012",
		ClientSecret: "legacy-xiaomi-app-secret",
		Region: "eu",
		PublicBaseURL: "https://domus.example",
		RedirectURL: "https://domus.example/api/v1/integrations/xiaomi-home/cloud/callback/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		CallbackSecret: strings.Repeat("a", 64),
		EventSecret: strings.Repeat("b", 64),
		OAuthState: "legacy-oauth-state-0123456789abcdef",
		AccessToken: "legacy-xiaomi-access-token",
		RefreshToken: "legacy-xiaomi-refresh-token",
		MacKey: "legacy-xiaomi-mac-key",
		MacAlgorithm: "HmacSHA1",
		ExpiresAt: time.Now().UTC().Add(time.Hour).Truncate(time.Second),
		UserID: "xiaomi-account-12345",
		Nickname: "Xiaomi Home",
	}
	raw, err := json.Marshal(config)
	require.NoError(t, err)
	encrypted, err := f.cipher.Encrypt(string(raw))
	require.NoError(t, err)
	now := time.Now().UTC()
	require.NoError(t, f.repos.Integrations.Create(context.Background(), &integration.Integration{
		ID: xiaomiCloudIntegrationID,
		Type: xiaomi_home.CloudPluginType,
		Name: "Xiaomi Home Cloud",
		Enabled: true,
		Status: "connected",
		ConfigEncrypted: []byte(encrypted),
		CreatedAt: now,
		UpdatedAt: now,
	}))
	return &config
}

func TestXiaomiCloudManagerRestoresLegacyEncryptedIntegrationRecord(t *testing.T) {
	f := newXiaomiCloudPluginFixture(t)
	expected := seedXiaomiCloudPluginRecord(t, f)

	require.NoError(t, f.manager.Start(context.Background()))
	got, err := f.handler.loadConfig(context.Background())
	require.NoError(t, err)
	require.Equal(t, expected.ClientID, got.ClientID)
	require.Equal(t, expected.ClientSecret, got.ClientSecret)
	require.Equal(t, expected.AccessToken, got.AccessToken)
	require.Equal(t, expected.RefreshToken, got.RefreshToken)
	require.Equal(t, expected.UserID, got.UserID)

	instance, err := f.repos.Integrations.GetByID(context.Background(), xiaomiCloudIntegrationID)
	require.NoError(t, err)
	require.Equal(t, xiaomi_home.CloudPluginType, instance.Type)
	require.Equal(t, "connected", instance.Status)
	require.NotContains(t, string(instance.ConfigEncrypted), expected.ClientSecret)
	require.NotContains(t, string(instance.ConfigEncrypted), expected.AccessToken)

	adapter, ok := f.manager.GetAdapter(xiaomiCloudIntegrationID)
	require.True(t, ok, "the existing encrypted row must be restored as a plugin instance")
	cloudAdapter, ok := adapter.(*xiaomi_home.CloudAdapter)
	require.True(t, ok)
	require.NoError(t, cloudAdapter.HealthCheck(context.Background()))
	client, err := f.handler.newClient(got)
	require.NoError(t, err)
	require.NotNil(t, client)

	status := httptest.NewRecorder()
	f.handler.Status(status, httptest.NewRequest(http.MethodGet, "/status", nil))
	require.Equal(t, http.StatusOK, status.Code)
	require.NotContains(t, status.Body.String(), expected.ClientSecret)
	require.NotContains(t, status.Body.String(), expected.AccessToken)
	require.NotContains(t, status.Body.String(), expected.RefreshToken)
}

func TestXiaomiCloudConfigureCreatesEncryptedManagerInstance(t *testing.T) {
	f := newXiaomiCloudPluginFixture(t)
	require.NoError(t, f.manager.Start(context.Background()))

	body := `{
		"client_id":"123456789012",
		"client_secret":"new-xiaomi-app-secret",
		"region":"eu",
		"public_base_url":"https://domus.example"
	}`
	request := httptest.NewRequest(http.MethodPut, "/config", strings.NewReader(body))
	response := httptest.NewRecorder()
	f.handler.Configure(response, request)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.NotContains(t, response.Body.String(), "new-xiaomi-app-secret")

	record, err := f.repos.Integrations.GetByID(context.Background(), xiaomiCloudIntegrationID)
	require.NoError(t, err)
	require.Equal(t, xiaomi_home.CloudPluginType, record.Type)
	require.True(t, record.Enabled)
	require.NotContains(t, string(record.ConfigEncrypted), "new-xiaomi-app-secret")

	stored, err := f.manager.GetConfig(context.Background(), xiaomiCloudIntegrationID)
	require.NoError(t, err)
	var config xiaomiCloudConfig
	require.NoError(t, json.Unmarshal(stored, &config))
	require.Equal(t, "123456789012", config.ClientID)
	require.Equal(t, "new-xiaomi-app-secret", config.ClientSecret)
	require.Equal(t, "eu", config.Region)
	require.NotEmpty(t, config.CallbackSecret)
	require.NotEmpty(t, config.EventSecret)
	require.Contains(t, config.RedirectURL, config.CallbackSecret)

	adapter, ok := f.manager.GetAdapter(xiaomiCloudIntegrationID)
	require.True(t, ok)
	require.NoError(t, adapter.HealthCheck(context.Background()))

	statusResponse := httptest.NewRecorder()
	f.handler.Status(statusResponse, httptest.NewRequest(http.MethodGet, "/status", nil))
	require.Equal(t, http.StatusOK, statusResponse.Code)
	require.NotContains(t, statusResponse.Body.String(), config.ClientSecret)
	if config.AccessToken != "" {
		require.NotContains(t, statusResponse.Body.String(), config.AccessToken)
	}
}

func TestXiaomiCloudSaveConfigReEnablesDisabledPluginInstance(t *testing.T) {
	f := newXiaomiCloudPluginFixture(t)
	seedXiaomiCloudPluginRecord(t, f)
	require.NoError(t, f.manager.Start(context.Background()))
	require.NoError(t, f.manager.SetEnabled(context.Background(), xiaomiCloudIntegrationID, false))
	_, ok := f.manager.GetAdapter(xiaomiCloudIntegrationID)
	require.False(t, ok)

	config := &xiaomiCloudConfig{
		ClientID: "123456789012",
		ClientSecret: "legacy-xiaomi-app-secret",
		Region: "eu",
		PublicBaseURL: "https://domus.example",
		RedirectURL: "https://domus.example/api/v1/integrations/xiaomi-home/cloud/callback/" + strings.Repeat("a", 64),
		CallbackSecret: strings.Repeat("a", 64),
		EventSecret: strings.Repeat("b", 64),
	}
	require.NoError(t, f.handler.saveConfig(context.Background(), *config, "disconnected"))
	record, err := f.repos.Integrations.GetByID(context.Background(), xiaomiCloudIntegrationID)
	require.NoError(t, err)
	require.True(t, record.Enabled, "saving configuration must preserve the legacy auto-enable behavior")
	adapter, ok := f.manager.GetAdapter(xiaomiCloudIntegrationID)
	require.True(t, ok)
	require.NoError(t, adapter.HealthCheck(context.Background()))
}

func TestXiaomiCloudManagedUpdatesUseCurrentAdapterAndDisconnectKeepsAppConfig(t *testing.T) {
	f := newXiaomiCloudPluginFixture(t)
	seedXiaomiCloudPluginRecord(t, f)
	require.NoError(t, f.manager.Start(context.Background()))

	previous, ok := f.manager.GetAdapter(xiaomiCloudIntegrationID)
	require.True(t, ok)
	config, err := f.handler.loadConfig(context.Background())
	require.NoError(t, err)
	config.OAuthState = "pending-oauth-state-0123456789abcdef"
	config.AccessToken = "rotated-access-token"
	config.RefreshToken = "rotated-refresh-token"
	config.ExpiresAt = time.Now().UTC().Add(time.Hour).Truncate(time.Second)
	require.NoError(t, f.handler.saveConfig(context.Background(), *config, "connected"))

	current, ok := f.manager.GetAdapter(xiaomiCloudIntegrationID)
	require.True(t, ok)
	require.NotSame(t, previous, current, "atomic config updates replace the supervised adapter")
	require.NoError(t, current.HealthCheck(context.Background()))
	client, err := f.handler.newClient(config)
	require.NoError(t, err, "the handler must resolve the current adapter after replacement")
	require.NotNil(t, client)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/disconnect", nil)
	f.handler.Disconnect(response, request)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	stored, err := f.manager.GetConfig(context.Background(), xiaomiCloudIntegrationID)
	require.NoError(t, err)
	var after xiaomiCloudConfig
	require.NoError(t, json.Unmarshal(stored, &after))
	require.Equal(t, "123456789012", after.ClientID)
	require.Equal(t, "legacy-xiaomi-app-secret", after.ClientSecret)
	require.Empty(t, after.AccessToken)
	require.Empty(t, after.RefreshToken)
	require.Empty(t, after.OAuthState)
	require.Empty(t, after.UserID)
	require.Empty(t, after.Nickname)
	require.Empty(t, after.MacKey)
	require.Empty(t, after.MacAlgorithm)
}
