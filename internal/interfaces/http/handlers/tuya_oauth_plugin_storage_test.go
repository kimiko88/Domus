package handlers

import (
	"context"
	"encoding/json"
	"database/sql"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	apppluginruntime "github.com/domus-hub/domus/internal/application/plugin_runtime"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/crypto"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	"github.com/domus-hub/domus/internal/plugins/tuya_cloud"
	"github.com/stretchr/testify/require"
)

func newManagedTuyaOAuthTestHandler(t *testing.T) (*TuyaOAuthHandler, *apppluginruntime.Manager, *sql.DB, *crypto.EnvelopeCipher) {
	t.Helper()
	ctx := context.Background()
	db, err := sqlite.Open(config.DatabaseConfig{
		Path: ":memory:", WALMode: false, BusyTimeoutMs: 1000, MaxOpenConns: 1,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, sqlite.Migrate(ctx, db))

	cipher, err := crypto.NewEnvelopeCipher([]byte("phase3-tuya-oauth-test-master-key"))
	require.NoError(t, err)
	repos := sqlite.NewRepositories(db)
	repos.Cipher = cipher
	manager, err := apppluginruntime.NewManager(repos, nil)
	require.NoError(t, err)
	require.NoError(t, manager.Start(ctx))
	t.Cleanup(func() { _ = manager.Stop(context.Background()) })

	httpClient := &http.Client{Transport: tuyaOAuthTestTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`{"success":true,"result":{"uid":"unused","access_token":"unused","refresh_token":"unused","expire_time":7200}}`)),
			Header: make(http.Header),
		}, nil
	})}
	oauthClient, err := tuya_cloud.NewOAuthClient(tuya_cloud.OAuthConfig{
		ClientID: "oauth-app-client", Secret: "oauth-app-secret",
		Identifier: "domus.phase3.test", BaseURL: "https://openapi.tuyaeu.com",
		HTTPClient: httpClient,
	})
	require.NoError(t, err)
	handler, err := NewTuyaOAuthHandler(
		db, cipher, oauthClient, "oauth-app-client",
		"https://example.test/authorize", "https://example.test/callback", manager,
	)
	require.NoError(t, err)
	return handler, manager, db, cipher
}

func TestTuyaOAuthMigratesLegacyTokensIntoEncryptedPluginInstance(t *testing.T) {
	handler, manager, db, cipher := newManagedTuyaOAuthTestHandler(t)
	ctx := context.Background()
	const userID = "admin-user-1"
	const accessToken = "legacy-access-token-should-not-be-visible"
	const refreshToken = "legacy-refresh-token-should-not-be-visible"
	accessEnc, err := cipher.Encrypt(accessToken)
	require.NoError(t, err)
	refreshEnc, err := cipher.Encrypt(refreshToken)
	require.NoError(t, err)
	expiresAt := time.Now().Add(time.Hour).Unix()
	_, err = db.ExecContext(ctx,
		"INSERT INTO tuya_user_oauth_connections(user_id,tuya_uid,access_token_enc,refresh_token_enc,expires_at,updated_at) VALUES(?,?,?,?,?,?)",
		userID, "smart-life-uid-1", accessEnc, refreshEnc, expiresAt, time.Now().Unix(),
	)
	require.NoError(t, err)

	token, err := handler.validToken(ctx, userID)
	require.NoError(t, err)
	require.Equal(t, "smart-life-uid-1", token.UID)
	require.Equal(t, accessToken, token.AccessToken)
	require.Equal(t, refreshToken, token.RefreshToken)

	var remaining int
	require.NoError(t, db.QueryRowContext(ctx,
		"SELECT count(*) FROM tuya_user_oauth_connections WHERE user_id=?", userID,
	).Scan(&remaining))
	require.Zero(t, remaining, "legacy row must be deleted after successful encrypted migration")

	instance, err := manager.GetConfig(ctx, tuya_cloud.PluginType)
	require.NoError(t, err)
	var config map[string]any
	require.NoError(t, json.Unmarshal(instance, &config))
	// Decode nested JSON without turning token strings into public instance fields.
	var nested map[string]storedSmartLifeOAuth
	nestedRaw, err := json.Marshal(config[smartLifeOAuthConfigKey])
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(nestedRaw, &nested))
	require.Equal(t, accessToken, nested[userID].AccessToken)
	require.Equal(t, refreshToken, nested[userID].RefreshToken)

	records, err := manager.ListInstances(ctx)
	require.NoError(t, err)
	publicJSON, err := json.Marshal(records)
	require.NoError(t, err)
	require.NotContains(t, string(publicJSON), accessToken)
	require.NotContains(t, string(publicJSON), refreshToken)

	var encryptedConfig []byte
	require.NoError(t, db.QueryRowContext(ctx,
		"SELECT config_encrypted FROM integrations WHERE id=?", tuya_cloud.PluginType,
	).Scan(&encryptedConfig))
	require.NotContains(t, string(encryptedConfig), accessToken)
	require.NotContains(t, string(encryptedConfig), refreshToken)
}

func TestTuyaOAuthManagedTokensAreIsolatedPerAdminAndDisconnectPreservesOthers(t *testing.T) {
	handler, manager, _, _ := newManagedTuyaOAuthTestHandler(t)
	ctx := context.Background()

	first := tuya_cloud.OAuthToken{UID: "uid-first", AccessToken: "access-first", RefreshToken: "refresh-first", ExpireTime: 3600}
	second := tuya_cloud.OAuthToken{UID: "uid-second", AccessToken: "access-second", RefreshToken: "refresh-second", ExpireTime: 3600}
	require.NoError(t, handler.storeToken(ctx, "admin-one", first))
	require.NoError(t, handler.storeToken(ctx, "admin-two", second))

	loadedFirst, _, err := handler.loadManagedToken(ctx, "admin-one")
	require.NoError(t, err)
	require.Equal(t, first.UID, loadedFirst.UID)
	require.Equal(t, first.AccessToken, loadedFirst.AccessToken)

	loadedSecond, _, err := handler.loadManagedToken(ctx, "admin-two")
	require.NoError(t, err)
	require.Equal(t, second.UID, loadedSecond.UID)
	require.Equal(t, second.AccessToken, loadedSecond.AccessToken)

	require.NoError(t, handler.clearManagedToken(ctx, "admin-one"))
	_, _, err = handler.loadManagedToken(ctx, "admin-one")
	require.ErrorIs(t, err, errTuyaOAuthNotConnected)

	stillConnected, _, err := handler.loadManagedToken(ctx, "admin-two")
	require.NoError(t, err)
	require.Equal(t, second.AccessToken, stillConnected.AccessToken)

	// A normal configuration edit through the public Manager API must retain
	// hidden OAuth credentials and account tokens that are not in the public form.
	_, err = manager.UpdateConfig(ctx, tuya_cloud.PluginType, json.RawMessage(`{"base_url":"https://openapi.tuyaeu.com"}`))
	require.NoError(t, err)
	instance, err := manager.GetConfig(ctx, tuya_cloud.PluginType)
	require.NoError(t, err)
	require.Contains(t, string(instance), "admin-two")
	require.NotContains(t, string(instance), "admin-one")
	var pluginConfig map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(instance, &pluginConfig))
	require.Empty(t, pluginConfig["client_id"], "OAuth app credentials must not be copied into OpenAPI client_id")
	require.Empty(t, pluginConfig["secret"], "OAuth app secret must not be copied into OpenAPI secret")
	var oauthClientID, oauthSecret string
	require.NoError(t, json.Unmarshal(pluginConfig["oauth_client_id"], &oauthClientID))
	require.NoError(t, json.Unmarshal(pluginConfig["oauth_secret"], &oauthSecret))
	require.Equal(t, "oauth-app-client", oauthClientID)
	require.Equal(t, "oauth-app-secret", oauthSecret)
}


func TestTuyaOAuthConcurrentAccountWritesDoNotLoseEntries(t *testing.T) {
	handler, manager, _, _ := newManagedTuyaOAuthTestHandler(t)
	ctx := context.Background()
	tokens := map[string]tuya_cloud.OAuthToken{
		"admin-concurrent-a": {UID: "uid-a", AccessToken: "access-a", RefreshToken: "refresh-a", ExpireTime: 3600},
		"admin-concurrent-b": {UID: "uid-b", AccessToken: "access-b", RefreshToken: "refresh-b", ExpireTime: 3600},
	}
	errs := make(chan error, len(tokens))
	for userID, token := range tokens {
		go func(userID string, token tuya_cloud.OAuthToken) {
			errs <- handler.storeToken(ctx, userID, token)
		}(userID, token)
	}
	for range tokens {
		require.NoError(t, <-errs)
	}
	for userID, expected := range tokens {
		got, _, err := handler.loadManagedToken(ctx, userID)
		require.NoError(t, err)
		require.Equal(t, expected.AccessToken, got.AccessToken)
	}

	config, err := manager.GetConfig(ctx, tuya_cloud.PluginType)
	require.NoError(t, err)
	var document map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(config, &document))
	var accounts map[string]storedSmartLifeOAuth
	require.NoError(t, json.Unmarshal(document[smartLifeOAuthConfigKey], &accounts))
	require.Len(t, accounts, 2)
}
