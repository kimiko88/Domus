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
	"github.com/domus-hub/domus/internal/plugins/imou_life"
	"github.com/stretchr/testify/require"
)

type imouPluginFixture struct {
	repos   *sqlite.Repositories
	cipher  *crypto.EnvelopeCipher
	manager *apppluginruntime.Manager
	handler *ImouLifeHandler
}

func newImouPluginFixture(t *testing.T) *imouPluginFixture {
	t.Helper()
	ctx := context.Background()
	db, err := sqlite.Open(config.DatabaseConfig{
		Path: ":memory:", WALMode: false, BusyTimeoutMs: 1000, MaxOpenConns: 1,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, sqlite.Migrate(ctx, db))

	cipher, err := crypto.NewEnvelopeCipher([]byte("imou-plugin-test-master-key"))
	require.NoError(t, err)
	repos := sqlite.NewRepositories(db)
	repos.Cipher = cipher
	manager, err := apppluginruntime.NewManager(repos, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = manager.Stop(context.Background()) })
	handler, err := NewImouLifeHandler(repos.Integrations, cipher)
	require.NoError(t, err)
	handler.WithPluginManager(manager)
	return &imouPluginFixture{repos: repos, cipher: cipher, manager: manager, handler: handler}
}

func seedImouPluginRecord(t *testing.T, f *imouPluginFixture) *imouStoredConfig {
	t.Helper()
	config := imouStoredConfig{
		AccessKey: "legacy-imou-project-key",
		SecretKey: "legacy-imou-project-secret",
		DataCenter: "eu",
		AuthorizationURL: "https://auth.example/imou/authorize",
		PublicBaseURL: "https://domus.example",
		CallbackSecret: strings.Repeat("a", 64),
		EventSecret: strings.Repeat("b", 64),
		PushEnabled: true,
		PushFlags: []string{"alarm", "status"},
		AdminAccessToken: "At_legacy-administrator-token",
		AdminExpiresAt: time.Now().UTC().Add(time.Hour).Truncate(time.Second),
		AccessToken: "Ot_legacy-user-access-token",
		RefreshToken: "legacy-user-refresh-token",
		ExpiresAt: time.Now().UTC().Add(time.Hour).Truncate(time.Second),
	}
	raw, err := json.Marshal(config)
	require.NoError(t, err)
	encrypted, err := f.cipher.Encrypt(string(raw))
	require.NoError(t, err)
	now := time.Now().UTC()
	require.NoError(t, f.repos.Integrations.Create(context.Background(), &integration.Integration{
		ID: imouIntegrationID,
		Type: imou_life.PluginType,
		Name: "Imou Life",
		Enabled: true,
		Status: "connected",
		ConfigEncrypted: []byte(encrypted),
		CreatedAt: now,
		UpdatedAt: now,
	}))
	return &config
}

func imouTestRequest() *http.Request {
	return httptest.NewRequest(http.MethodGet, "/api/v1/integrations/imou-life/status", nil)
}

func TestImouPluginManagerRestoresLegacyEncryptedIntegrationRecord(t *testing.T) {
	f := newImouPluginFixture(t)
	expected := seedImouPluginRecord(t, f)

	require.NoError(t, f.manager.Start(context.Background()))
	got, err := f.handler.loadConfig(imouTestRequest())
	require.NoError(t, err)
	require.Equal(t, expected.AccessKey, got.AccessKey)
	require.Equal(t, expected.SecretKey, got.SecretKey)
	require.Equal(t, expected.AccessToken, got.AccessToken)
	require.Equal(t, expected.RefreshToken, got.RefreshToken)
	require.Equal(t, expected.AdminAccessToken, got.AdminAccessToken)
	require.Equal(t, expected.PushFlags, got.PushFlags)

	record, err := f.repos.Integrations.GetByID(context.Background(), imouIntegrationID)
	require.NoError(t, err)
	require.Equal(t, imou_life.PluginType, record.Type)
	require.Equal(t, "connected", record.Status)
	for _, secret := range []string{expected.SecretKey, expected.AccessToken, expected.RefreshToken, expected.AdminAccessToken} {
		require.NotContains(t, string(record.ConfigEncrypted), secret)
	}

	adapter, ok := f.manager.GetAdapter(imouIntegrationID)
	require.True(t, ok, "the encrypted legacy row should be restored as a managed plugin instance")
	imouAdapter, ok := adapter.(*imou_life.Adapter)
	require.True(t, ok)
	require.NoError(t, imouAdapter.HealthCheck(context.Background()))
	client, err := f.handler.newClient(imou_life.Config{
		AccessKey: got.AccessKey, SecretKey: got.SecretKey, DataCenter: got.DataCenter,
	})
	require.NoError(t, err)
	require.NotNil(t, client)

	status := httptest.NewRecorder()
	f.handler.Status(status, imouTestRequest())
	require.Equal(t, http.StatusOK, status.Code)
	for _, secret := range []string{expected.SecretKey, expected.AccessToken, expected.RefreshToken, expected.AdminAccessToken} {
		require.NotContains(t, status.Body.String(), secret)
	}
}

func TestImouConfigureCreatesEncryptedManagedInstance(t *testing.T) {
	f := newImouPluginFixture(t)
	require.NoError(t, f.manager.Start(context.Background()))

	body := `{
		"access_key":"new-imou-project-key",
		"secret_key":"new-imou-project-secret",
		"data_center":"eu",
		"authorization_url":"https://auth.example/imou/authorize",
		"public_base_url":"https://domus.example"
	}`
	request := httptest.NewRequest(http.MethodPut, "/config", strings.NewReader(body))
	response := httptest.NewRecorder()
	f.handler.Configure(response, request)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.NotContains(t, response.Body.String(), "new-imou-project-secret")

	record, err := f.repos.Integrations.GetByID(context.Background(), imouIntegrationID)
	require.NoError(t, err)
	require.Equal(t, imou_life.PluginType, record.Type)
	require.True(t, record.Enabled)
	require.NotContains(t, string(record.ConfigEncrypted), "new-imou-project-secret")

	stored, err := f.manager.GetConfig(context.Background(), imouIntegrationID)
	require.NoError(t, err)
	var saved imouStoredConfig
	require.NoError(t, json.Unmarshal(stored, &saved))
	require.Equal(t, "new-imou-project-key", saved.AccessKey)
	require.Equal(t, "new-imou-project-secret", saved.SecretKey)
	require.Equal(t, "eu", saved.DataCenter)
	require.Len(t, saved.CallbackSecret, 64)
	require.Len(t, saved.EventSecret, 64)

	adapter, ok := f.manager.GetAdapter(imouIntegrationID)
	require.True(t, ok)
	require.NoError(t, adapter.HealthCheck(context.Background()))

	status := httptest.NewRecorder()
	f.handler.Status(status, imouTestRequest())
	require.Equal(t, http.StatusOK, status.Code)
	require.NotContains(t, status.Body.String(), saved.SecretKey)
}

func TestImouManagedTokenRotationReplacesAdapterAndDisconnectKeepsProjectSettings(t *testing.T) {
	f := newImouPluginFixture(t)
	seedImouPluginRecord(t, f)
	require.NoError(t, f.manager.Start(context.Background()))

	previous, ok := f.manager.GetAdapter(imouIntegrationID)
	require.True(t, ok)
	config, err := f.handler.loadConfig(imouTestRequest())
	require.NoError(t, err)
	config.AccessToken = "Ot_rotated-user-access-token"
	config.RefreshToken = "rotated-user-refresh-token"
	config.ExpiresAt = time.Now().UTC().Add(time.Hour).Truncate(time.Second)
	config.AdminAccessToken = "At_rotated-administrator-token"
	config.AdminExpiresAt = time.Now().UTC().Add(time.Hour).Truncate(time.Second)
	require.NoError(t, f.handler.saveConfig(context.Background(), *config, "connected"))

	current, ok := f.manager.GetAdapter(imouIntegrationID)
	require.True(t, ok)
	require.NotSame(t, previous, current, "managed config updates must replace the supervised adapter")
	require.NoError(t, current.HealthCheck(context.Background()))
	client, err := f.handler.newClient(imou_life.Config{
		AccessKey: config.AccessKey, SecretKey: config.SecretKey, DataCenter: config.DataCenter,
	})
	require.NoError(t, err, "the handler must resolve the current adapter after replacement")
	require.NotNil(t, client)

	response := httptest.NewRecorder()
	f.handler.Disconnect(response, httptest.NewRequest(http.MethodPost, "/disconnect", nil))
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	stored, err := f.manager.GetConfig(context.Background(), imouIntegrationID)
	require.NoError(t, err)
	var after imouStoredConfig
	require.NoError(t, json.Unmarshal(stored, &after))
	require.Equal(t, "legacy-imou-project-key", after.AccessKey)
	require.Equal(t, "legacy-imou-project-secret", after.SecretKey)
	require.Equal(t, "https://auth.example/imou/authorize", after.AuthorizationURL)
	require.Equal(t, "https://domus.example", after.PublicBaseURL)
	require.Equal(t, strings.Repeat("a", 64), after.CallbackSecret)
	require.Equal(t, strings.Repeat("b", 64), after.EventSecret)
	require.True(t, after.PushEnabled)
	require.Empty(t, after.AccessToken)
	require.Empty(t, after.RefreshToken)
	require.Empty(t, after.ExpiresAt)
	require.Equal(t, "At_rotated-administrator-token", after.AdminAccessToken,
		"disconnecting the delegated user account must not erase the separate project token")
	require.NotEmpty(t, after.AdminExpiresAt)
}

func TestImouCatalogCreatedInstanceBootstrapsPrivateCallbacksAndPreservesThemOnEdit(t *testing.T) {
	f := newImouPluginFixture(t)
	require.NoError(t, f.manager.Start(context.Background()))

	publicConfig := json.RawMessage(`{
		"access_key":"catalog-imou-key",
		"secret_key":"catalog-imou-secret",
		"data_center":"eu",
		"authorization_url":"https://auth.example/imou/authorize",
		"public_base_url":"https://first.domus.example"
	}`)
	_, err := f.manager.Create(
		context.Background(),
		imouIntegrationID,
		imou_life.PluginType,
		"Imou Life",
		publicConfig,
		true,
	)
	require.NoError(t, err)

	status := httptest.NewRecorder()
	f.handler.Status(status, imouTestRequest())
	require.Equal(t, http.StatusOK, status.Code, status.Body.String())
	var statusBody map[string]any
	require.NoError(t, json.Unmarshal(status.Body.Bytes(), &statusBody))
	require.Equal(t, true, statusBody["configured"])

	config, err := f.handler.loadConfig(imouTestRequest())
	require.NoError(t, err)
	require.Len(t, config.CallbackSecret, 64)
	require.Len(t, config.EventSecret, 64)

	updatedPublicConfig := json.RawMessage(`{
		"access_key":"catalog-imou-key",
		"secret_key":"catalog-imou-secret",
		"data_center":"eu",
		"authorization_url":"https://auth.example/imou/authorize",
		"public_base_url":"https://second.domus.example"
	}`)
	_, err = f.manager.UpdateConfig(context.Background(), imouIntegrationID, updatedPublicConfig)
	require.NoError(t, err)

	updated, err := f.handler.loadConfig(imouTestRequest())
	require.NoError(t, err)
	require.Equal(t, config.CallbackSecret, updated.CallbackSecret)
	require.Equal(t, config.EventSecret, updated.EventSecret)
	require.Equal(t, "https://second.domus.example", updated.PublicBaseURL)
}
