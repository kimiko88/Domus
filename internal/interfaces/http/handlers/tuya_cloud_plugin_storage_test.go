package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	domusmw "github.com/domus-hub/domus/internal/interfaces/http/middleware"
	"github.com/domus-hub/domus/internal/domain/user"
	"github.com/domus-hub/domus/internal/plugins/tuya_cloud"
	"github.com/stretchr/testify/require"
)

// managedTuyaAdminRequest attaches the same identity populated by session auth.
func managedTuyaAdminRequest(method, path, body string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	return req.WithContext(domusmw.WithUser(req.Context(), &user.User{
		ID: "tuya-admin-1", Username: "admin", Role: user.RoleAdmin, Enabled: true,
	}))
}

func TestTuyaCloudWithStorageMigratesLegacyConfigIntoPluginManager(t *testing.T) {
	_, manager, db, cipher := newManagedTuyaOAuthTestHandler(t)
	ctx := context.Background()

	_, err := db.Exec(`CREATE TABLE tuya_cloud_config (
		id TEXT PRIMARY KEY, client_id TEXT NOT NULL, secret_enc TEXT NOT NULL,
		uid TEXT NOT NULL, endpoint TEXT NOT NULL, updated_at INTEGER NOT NULL
	)`)
	require.NoError(t, err)
	const clientID = "legacy-openapi-client"
	const secret = "legacy-openapi-secret-not-plaintext"
	secretEnc, err := cipher.Encrypt(secret)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx,
		"INSERT INTO tuya_cloud_config(id,client_id,secret_enc,uid,endpoint,updated_at) VALUES('default',?,?,?,?,?)",
		clientID, secretEnc, "legacy-smart-life-uid", "https://openapi.tuyaeu.com", time.Now().Unix(),
	)
	require.NoError(t, err)

	handler := NewTuyaCloudHandler(nil, manager).WithStorage(db, cipher)
	stored, err := manager.GetConfig(ctx, tuya_cloud.PluginType)
	require.NoError(t, err)
	var config map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(stored, &config))
	var gotClientID, gotSecret, gotUID string
	require.NoError(t, json.Unmarshal(config["client_id"], &gotClientID))
	require.NoError(t, json.Unmarshal(config["secret"], &gotSecret))
	require.NoError(t, json.Unmarshal(config["uid"], &gotUID))
	require.Equal(t, clientID, gotClientID)
	require.Equal(t, secret, gotSecret)
	require.Equal(t, "legacy-smart-life-uid", gotUID)

	var rows int
	require.NoError(t, db.QueryRow("SELECT count(*) FROM tuya_cloud_config WHERE id='default'").Scan(&rows))
	require.Zero(t, rows, "legacy row must be deleted only after encrypted Plugin Manager persistence succeeds")

	var encrypted []byte
	require.NoError(t, db.QueryRow("SELECT config_encrypted FROM integrations WHERE id=?", tuya_cloud.PluginType).Scan(&encrypted))
	require.NotContains(t, string(encrypted), secret, "the persisted integration blob must not contain the plaintext secret")
	require.NotNil(t, handler.getClient(), "handler must be rebound to the manager-owned adapter after migration")
	require.Equal(t, clientID, handler.getClient().Config().ClientID)
}

func TestTuyaCloudManagedGetConfigDoesNotExposeSecret(t *testing.T) {
	_, manager, db, cipher := newManagedTuyaOAuthTestHandler(t)
	ctx := context.Background()
	raw, err := json.Marshal(map[string]any{
		"client_id": "configured-client-id",
		"secret": "must-never-be-returned",
		"base_url": "https://openapi.tuyaeu.com",
		"uid": "smart-life-uid",
	})
	require.NoError(t, err)
	_, err = manager.Create(ctx, tuya_cloud.PluginType, tuya_cloud.PluginType, "Tuya Cloud OpenAPI", raw, true)
	require.NoError(t, err)

	handler := NewTuyaCloudHandler(nil, manager).WithStorage(db, cipher)
	rec := httptest.NewRecorder()
	handler.GetConfig(rec, httptest.NewRequest(http.MethodGet, "/config", nil))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var response map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Equal(t, true, response["configured"])
	require.Equal(t, "configured-client-id", response["client_id"])
	require.Equal(t, "plugin_manager", response["source"])
	require.NotContains(t, rec.Body.String(), "must-never-be-returned")
}

func TestTuyaCloudSaveConfigPersistsThroughPluginManager(t *testing.T) {
	_, manager, db, cipher := newManagedTuyaOAuthTestHandler(t)
	handler := NewTuyaCloudHandler(nil, manager).WithStorage(db, cipher)
	request := managedTuyaAdminRequest(http.MethodPost, "/config", `{
		"client_id":"new-openapi-client",
		"client_secret":"new-openapi-secret",
		"uid":"new-smart-life-uid",
		"endpoint":"https://openapi.tuyaeu.com"
	}`)
	rec := httptest.NewRecorder()
	handler.SaveConfig(rec, request)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	stored, err := manager.GetConfig(context.Background(), tuya_cloud.PluginType)
	require.NoError(t, err)
	var config map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(stored, &config))
	var clientID, secret, uid string
	require.NoError(t, json.Unmarshal(config["client_id"], &clientID))
	require.NoError(t, json.Unmarshal(config["secret"], &secret))
	require.NoError(t, json.Unmarshal(config["uid"], &uid))
	require.Equal(t, "new-openapi-client", clientID)
	require.Equal(t, "new-openapi-secret", secret)
	require.Equal(t, "new-smart-life-uid", uid)
	require.NotContains(t, rec.Body.String(), secret)

	var legacyTableCount int
	require.NoError(t, db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='tuya_cloud_config'").Scan(&legacyTableCount))
	require.Zero(t, legacyTableCount, "managed installations must not create the legacy configuration table")
}

func TestTuyaCloudSaveConfigPreservesInternalOAuthAndUnexposedSchema(t *testing.T) {
	_, manager, db, cipher := newManagedTuyaOAuthTestHandler(t)
	ctx := context.Background()
	seed := map[string]any{
		"client_id": "old-openapi-client",
		"secret": "old-openapi-secret",
		"base_url": "https://openapi.tuyaeu.com",
		"uid": "old-uid",
		"schema": "tuya-project-schema",
		"oauth_client_id": "app-authorization-id",
		"oauth_secret": "app-authorization-secret",
		"oauth_identifier": "domus.test",
		"smart_life_oauth": map[string]any{
			"admin-one": map[string]any{
				"uid": "oauth-uid", "access_token": "access-token-preserved",
				"refresh_token": "refresh-token-preserved", "expires_at": time.Now().Add(time.Hour).Unix(),
			},
		},
	}
	raw, err := json.Marshal(seed)
	require.NoError(t, err)
	_, err = manager.Create(ctx, tuya_cloud.PluginType, tuya_cloud.PluginType, "Tuya Cloud OpenAPI", raw, true)
	require.NoError(t, err)

	handler := NewTuyaCloudHandler(nil, manager).WithStorage(db, cipher)
	request := managedTuyaAdminRequest(http.MethodPost, "/config", `{
		"client_id":"replacement-client",
		"client_secret":"replacement-secret",
		"uid":"",
		"endpoint":"https://openapi.tuyaeu.com"
	}`)
	rec := httptest.NewRecorder()
	handler.SaveConfig(rec, request)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	stored, err := manager.GetConfig(ctx, tuya_cloud.PluginType)
	require.NoError(t, err)
	var after map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(stored, &after))
	var schema, oauthClientID, oauthSecret, oauthIdentifier string
	require.NoError(t, json.Unmarshal(after["schema"], &schema))
	require.NoError(t, json.Unmarshal(after["oauth_client_id"], &oauthClientID))
	require.NoError(t, json.Unmarshal(after["oauth_secret"], &oauthSecret))
	require.NoError(t, json.Unmarshal(after["oauth_identifier"], &oauthIdentifier))
	require.Equal(t, "tuya-project-schema", schema)
	require.Equal(t, "app-authorization-id", oauthClientID)
	require.Equal(t, "app-authorization-secret", oauthSecret)
	require.Equal(t, "domus.test", oauthIdentifier)
	require.Contains(t, string(after["smart_life_oauth"]), "access-token-preserved")
	require.NotContains(t, rec.Body.String(), "replacement-secret")
}

func TestTuyaCloudDeleteConfigRemovesOnlyOpenAPICredentialsAndKeepsOAuth(t *testing.T) {
	_, manager, db, cipher := newManagedTuyaOAuthTestHandler(t)
	ctx := context.Background()
	seed := map[string]any{
		"client_id": "openapi-client-to-delete",
		"secret": "openapi-secret-to-delete",
		"base_url": "https://openapi.tuyaeu.com",
		"uid": "openapi-uid-to-delete",
		"schema": "openapi-schema-to-delete",
		"oauth_client_id": "app-authorization-id",
		"oauth_secret": "app-authorization-secret",
		"oauth_identifier": "domus.test",
		"smart_life_oauth": map[string]any{
			"admin-one": map[string]any{
				"uid": "oauth-uid", "access_token": "access-token-preserved",
				"refresh_token": "refresh-token-preserved", "expires_at": time.Now().Add(time.Hour).Unix(),
			},
		},
	}
	raw, err := json.Marshal(seed)
	require.NoError(t, err)
	_, err = manager.Create(ctx, tuya_cloud.PluginType, tuya_cloud.PluginType, "Tuya Cloud OpenAPI", raw, true)
	require.NoError(t, err)
	handler := NewTuyaCloudHandler(nil, manager).WithStorage(db, cipher)

	rec := httptest.NewRecorder()
	handler.DeleteConfig(rec, managedTuyaAdminRequest(http.MethodDelete, "/config", ""))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	stored, err := manager.GetConfig(ctx, tuya_cloud.PluginType)
	require.NoError(t, err)
	var after map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(stored, &after))
	for _, key := range []string{"client_id", "secret", "uid", "schema"} {
		require.Empty(t, after[key], "OpenAPI setting %q should be removed", key)
	}
	require.Contains(t, string(after["smart_life_oauth"]), "access-token-preserved")
	require.Contains(t, string(after["smart_life_oauth"]), "refresh-token-preserved")
	require.Equal(t, "app-authorization-id", strings.Trim(string(after["oauth_client_id"]), `"`))
	require.NotContains(t, rec.Body.String(), "app-authorization-secret")

	instance, config, err := handler.managedTuyaInstance(ctx)
	require.NoError(t, err)
	require.True(t, instance.Enabled)
	require.Contains(t, string(config["smart_life_oauth"]), "admin-one")
	provider := handler.getClient()
	require.NotNil(t, provider)
	require.Empty(t, provider.Config().ClientID)
	require.Empty(t, provider.Config().Secret)
}
