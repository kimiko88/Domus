package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	apppluginruntime "github.com/domus-hub/domus/internal/application/plugin_runtime"
	"github.com/domus-hub/domus/internal/domain/user"
	domusmw "github.com/domus-hub/domus/internal/interfaces/http/middleware"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/crypto"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	"github.com/domus-hub/domus/internal/plugins/tuya_cloud"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func newTuyaCloudOptionRuntime(t *testing.T) (context.Context, *sql.DB, *crypto.EnvelopeCipher, *apppluginruntime.Manager) {
	t.Helper()
	ctx := context.Background()
	db, err := sqlite.Open(config.DatabaseConfig{
		Path: ":memory:", WALMode: false, BusyTimeoutMs: 1000, MaxOpenConns: 1,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, sqlite.Migrate(ctx, db))
	cipher, err := crypto.NewEnvelopeCipher([]byte("phase3-tuya-option-handler-test-master-key"))
	require.NoError(t, err)
	repos := sqlite.NewRepositories(db)
	repos.Cipher = cipher
	manager, err := apppluginruntime.NewManager(repos, nil)
	require.NoError(t, err)
	return ctx, db, cipher, manager
}

func tuyaAdminRequest(method, path, body string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	admin := &user.User{ID: "tuya-test-admin", Username: "admin", Role: user.RoleAdmin, Enabled: true}
	return req.WithContext(domusmw.WithUser(req.Context(), admin))
}

func TestTuyaCloudOptionFlowMigratesLegacyConfigAndPreservesOAuthState(t *testing.T) {
	ctx, db, cipher, manager := newTuyaCloudOptionRuntime(t)

	_, err := db.ExecContext(ctx, `
		CREATE TABLE tuya_cloud_config (
			id TEXT PRIMARY KEY,
			client_id TEXT NOT NULL,
			secret_enc TEXT NOT NULL,
			uid TEXT NOT NULL,
			endpoint TEXT NOT NULL,
			updated_at INTEGER NOT NULL
		)
	`)
	require.NoError(t, err)
	legacySecret := "legacy-openapi-secret-must-be-encrypted"
	secretEncrypted, err := cipher.Encrypt(legacySecret)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx,
		"INSERT INTO tuya_cloud_config(id,client_id,secret_enc,uid,endpoint,updated_at) VALUES('default',?,?,?,?,?)",
		"legacy-openapi-client", secretEncrypted, "eu-smart-life-user", "https://openapi.tuyaeu.com", time.Now().Unix(),
	)
	require.NoError(t, err)

	// Existing OAuth-only instance already contains a token; legacy migration
	// must merge OpenAPI credentials rather than replacing its hidden fields.
	initial := json.RawMessage(`{
		"base_url":"https://openapi.tuyaeu.com",
		"oauth_client_id":"smart-life-app-client",
		"oauth_secret":"smart-life-app-secret",
		"oauth_identifier":"domus.tuya.test",
		"smart_life_oauth":{"admin":{"uid":"smart-life-uid","access_token":"oauth-access","refresh_token":"oauth-refresh","expires_at":2000000000}}
	}`)
	_, err = manager.Create(ctx, tuya_cloud.PluginType, tuya_cloud.PluginType, "Tuya Cloud", initial, true)
	require.NoError(t, err)
	require.NoError(t, manager.Start(ctx))
	t.Cleanup(func() { _ = manager.Stop(context.Background()) })

	handler := NewTuyaCloudHandler(nil, manager).WithStorage(db, cipher)
	provider := handler.getClient()
	require.NotNil(t, provider)
	require.Equal(t, "legacy-openapi-client", provider.Config().ClientID)
	require.Equal(t, legacySecret, provider.Config().Secret)
	require.Equal(t, "eu-smart-life-user", provider.Config().UID)

	var legacyTableCount int
	require.NoError(t, db.QueryRowContext(ctx,
		"SELECT count(*) FROM sqlite_master WHERE type='table' AND name='tuya_cloud_config'",
	).Scan(&legacyTableCount))
	require.Zero(t, legacyTableCount, "legacy table is removed only after a successful encrypted migration")

	configJSON, err := manager.GetConfig(ctx, tuya_cloud.PluginType)
	require.NoError(t, err)
	var stored map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(configJSON, &stored))
	require.Equal(t, legacySecret, readRawJSONString(stored, "secret"))
	require.Equal(t, "legacy-openapi-client", readRawJSONString(stored, "client_id"))
	require.Contains(t, string(stored["smart_life_oauth"]), "oauth-access")
	require.Equal(t, "smart-life-app-client", readRawJSONString(stored, "oauth_client_id"))
	require.Contains(t, string(configJSON), legacySecret, "trusted application-layer config access should decrypt the persisted secret")

	var encrypted []byte
	require.NoError(t, db.QueryRowContext(ctx,
		"SELECT config_encrypted FROM integrations WHERE id=?", tuya_cloud.PluginType,
	).Scan(&encrypted))
	require.NotContains(t, string(encrypted), legacySecret)
	require.NotContains(t, string(encrypted), "oauth-access")

	get := httptest.NewRecorder()
	router.ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/config", nil))
	require.Equal(t, http.StatusOK, get.Code)
	require.Contains(t, get.Body.String(), `"source":"plugin_manager"`)
	require.Contains(t, get.Body.String(), "legacy-openapi-client")
	require.NotContains(t, get.Body.String(), legacySecret)
}

func TestTuyaCloudOptionFlowSaveAndDeleteOnlyOpenAPISettings(t *testing.T) {
	ctx, db, cipher, manager := newTuyaCloudOptionRuntime(t)
	_, err := manager.Create(ctx, tuya_cloud.PluginType, tuya_cloud.PluginType, "Tuya Cloud", json.RawMessage(`{
		"base_url":"https://openapi.tuyaeu.com",
		"oauth_client_id":"app-client",
		"oauth_secret":"app-secret",
		"oauth_identifier":"domus.test",
		"smart_life_oauth":{"admin":{"uid":"oauth-user","access_token":"keep-access","refresh_token":"keep-refresh","expires_at":2000000000}}
	}`), true)
	require.NoError(t, err)
	require.NoError(t, manager.Start(ctx))
	t.Cleanup(func() { _ = manager.Stop(context.Background()) })

	handler := NewTuyaCloudHandler(nil, manager).WithStorage(db, cipher)
	router := chi.NewRouter()
	handler.RegisterRoutes(router)
	save := httptest.NewRecorder()
	router.ServeHTTP(save, tuyaAdminRequest(http.MethodPut, "/config", `{
		"client_id":"new-openapi-client",
		"client_secret":"new-openapi-secret",
		"uid":"eu-new-user",
		"endpoint":"https://openapi.tuyaeu.com"
	}`))
	require.Equal(t, http.StatusOK, save.Code, save.Body.String())
	require.NotContains(t, save.Body.String(), "new-openapi-secret")
	require.NotNil(t, handler.getClient())
	require.Equal(t, "new-openapi-client", handler.getClient().Config().ClientID)

	afterSave, err := manager.GetConfig(ctx, tuya_cloud.PluginType)
	require.NoError(t, err)
	var saved map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(afterSave, &saved))
	require.Contains(t, string(saved["smart_life_oauth"]), "keep-access")
	require.Equal(t, "app-client", readRawJSONString(saved, "oauth_client_id"))
	require.Equal(t, "new-openapi-client", readRawJSONString(saved, "client_id"))

	deleteResponse := httptest.NewRecorder()
	router.ServeHTTP(deleteResponse, tuyaAdminRequest(http.MethodDelete, "/config", ""))
	require.Equal(t, http.StatusOK, deleteResponse.Code, deleteResponse.Body.String())
	require.Contains(t, deleteResponse.Body.String(), "OAuth")
	require.NotNil(t, handler.getClient(), "OAuth-only Tuya instance remains managed after OpenAPI credentials are removed")
	require.Empty(t, handler.getClient().Config().ClientID)
	require.Empty(t, handler.getClient().Config().Secret)

	afterDelete, err := manager.GetConfig(ctx, tuya_cloud.PluginType)
	require.NoError(t, err)
	var remaining map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(afterDelete, &remaining))
	require.NotContains(t, remaining, "client_id")
	require.NotContains(t, remaining, "secret")
	require.Contains(t, string(remaining["smart_life_oauth"]), "keep-access")
	require.Equal(t, "app-client", readRawJSONString(remaining, "oauth_client_id"))

	instances, err := manager.ListInstances(ctx)
	require.NoError(t, err)
	require.Len(t, instances, 1)
	require.Equal(t, tuya_cloud.PluginType, instances[0].Type)

	get := httptest.NewRecorder()
	handler.GetConfig(get, httptest.NewRequest(http.MethodGet, "/config", nil))
	require.Equal(t, http.StatusOK, get.Code)
	require.Contains(t, get.Body.String(), `"configured":false`)
}

func TestTuyaCloudOptionFlowCreatesManagedInstanceOnFreshInstall(t *testing.T) {
	ctx, db, cipher, manager := newTuyaCloudOptionRuntime(t)
	require.NoError(t, manager.Start(ctx))
	t.Cleanup(func() { _ = manager.Stop(context.Background()) })
	handler := NewTuyaCloudHandler(nil, manager).WithStorage(db, cipher)

	response := httptest.NewRecorder()
	handler.SaveConfig(response, tuyaAdminRequest(http.MethodPost, "/config", `{
		"client_id":"fresh-client",
		"client_secret":"fresh-secret",
		"endpoint":"https://openapi.tuyaeu.com"
	}`))
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	instances, err := manager.ListInstances(ctx)
	require.NoError(t, err)
	require.Len(t, instances, 1)
	require.Equal(t, tuya_cloud.PluginType, instances[0].Type)

	var legacyTableCount int
	require.NoError(t, db.QueryRowContext(ctx,
		"SELECT count(*) FROM sqlite_master WHERE type='table' AND name='tuya_cloud_config'",
	).Scan(&legacyTableCount))
	require.Zero(t, legacyTableCount, "new installations must not recreate the legacy config table")
}


func TestTuyaCloudLegacyMigrationKeepsOldTableWhenDecryptionFails(t *testing.T) {
	ctx, db, cipher, manager := newTuyaCloudOptionRuntime(t)
	_, err := db.ExecContext(ctx, `
		CREATE TABLE tuya_cloud_config (
			id TEXT PRIMARY KEY,
			client_id TEXT NOT NULL,
			secret_enc TEXT NOT NULL,
			uid TEXT NOT NULL,
			endpoint TEXT NOT NULL,
			updated_at INTEGER NOT NULL
		)
	`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx,
		"INSERT INTO tuya_cloud_config(id,client_id,secret_enc,uid,endpoint,updated_at) VALUES('default',?,?,?,?,?)",
		"legacy-client", "not-a-valid-envelope", "legacy-uid", "https://openapi.tuyaeu.com", time.Now().Unix(),
	)
	require.NoError(t, err)
	require.NoError(t, manager.Start(ctx))
	t.Cleanup(func() { _ = manager.Stop(context.Background()) })

	_ = NewTuyaCloudHandler(nil, manager).WithStorage(db, cipher)
	var tableCount, rowCount int
	require.NoError(t, db.QueryRowContext(ctx,
		"SELECT count(*) FROM sqlite_master WHERE type='table' AND name='tuya_cloud_config'",
	).Scan(&tableCount))
	require.Equal(t, 1, tableCount, "failed decryption must not delete the only legacy copy")
	require.NoError(t, db.QueryRowContext(ctx, "SELECT count(*) FROM tuya_cloud_config").Scan(&rowCount))
	require.Equal(t, 1, rowCount)
	instances, err := manager.ListInstances(ctx)
	require.NoError(t, err)
	require.Empty(t, instances, "failed migration must not create a partial Plugin Manager instance")
}
