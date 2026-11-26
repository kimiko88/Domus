package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/domus-hub/domus/internal/application/plugin_runtime"
	"github.com/domus-hub/domus/internal/domain/integration"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/crypto"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

type pluginHandlerIntegrationRepo struct {
	mu sync.RWMutex
	items map[string]*integration.Integration
}
func newPluginHandlerIntegrationRepo() *pluginHandlerIntegrationRepo { return &pluginHandlerIntegrationRepo{items: make(map[string]*integration.Integration)} }
func (r *pluginHandlerIntegrationRepo) Create(ctx context.Context, in *integration.Integration) error {
	r.mu.Lock(); defer r.mu.Unlock(); if _, ok := r.items[in.ID]; ok { return integration.ErrIntegrationNotFound }; copy := *in; copy.ConfigEncrypted = append([]byte(nil), in.ConfigEncrypted...); r.items[in.ID] = &copy; return nil
}
func (r *pluginHandlerIntegrationRepo) GetByID(ctx context.Context, id string) (*integration.Integration, error) {
	r.mu.RLock(); defer r.mu.RUnlock(); in, ok := r.items[id]; if !ok { return nil, integration.ErrIntegrationNotFound }; copy := *in; copy.ConfigEncrypted = append([]byte(nil), in.ConfigEncrypted...); return &copy, nil
}
func (r *pluginHandlerIntegrationRepo) List(ctx context.Context) ([]*integration.Integration, error) {
	r.mu.RLock(); defer r.mu.RUnlock(); out := make([]*integration.Integration, 0, len(r.items)); for _, in := range r.items { copy := *in; copy.ConfigEncrypted = append([]byte(nil), in.ConfigEncrypted...); out = append(out, &copy) }; return out, nil
}
func (r *pluginHandlerIntegrationRepo) Update(ctx context.Context, in *integration.Integration) error {
	r.mu.Lock(); defer r.mu.Unlock(); if _, ok := r.items[in.ID]; !ok { return integration.ErrIntegrationNotFound }; copy := *in; copy.ConfigEncrypted = append([]byte(nil), in.ConfigEncrypted...); r.items[in.ID] = &copy; return nil
}
func (r *pluginHandlerIntegrationRepo) SetStatus(ctx context.Context, id, status, lastError string) error {
	r.mu.Lock(); defer r.mu.Unlock(); in, ok := r.items[id]; if !ok { return integration.ErrIntegrationNotFound }; in.Status, in.LastError = status, lastError; return nil
}
func (r *pluginHandlerIntegrationRepo) Delete(ctx context.Context, id string) error {
	r.mu.Lock(); defer r.mu.Unlock(); if _, ok := r.items[id]; !ok { return integration.ErrIntegrationNotFound }; delete(r.items, id); return nil
}

func newPluginRuntimeHandlerManager(t *testing.T, repo *pluginHandlerIntegrationRepo, cipher *crypto.EnvelopeCipher) *plugin_runtime.Manager {
	t.Helper()
	db, err := sqlite.Open(config.DatabaseConfig{Path: ":memory:", WALMode: false, BusyTimeoutMs: 1000, MaxOpenConns: 1})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, sqlite.Migrate(context.Background(), db))
	repos := sqlite.NewRepositories(db)
	repos.Integrations = repo
	repos.Cipher = cipher
	manager, err := plugin_runtime.NewManager(repos, nil)
	require.NoError(t, err)
	return manager
}
func TestPluginRuntimeHandlerCatalogCreateAndSecretRedaction(t *testing.T) {
	repo := newPluginHandlerIntegrationRepo()
	cipher, err := crypto.NewEnvelopeCipher([]byte("plugin-handler-test-key")); require.NoError(t, err)
	manager := newPluginRuntimeHandlerManager(t, repo, cipher)
	handler := NewPluginRuntimeHandler(manager)
	router := chi.NewRouter()
	router.Get("/types", handler.ListTypes)
	router.Get("/instances", handler.ListInstances)
	router.Post("/instances", handler.Create)
	router.Delete("/instances/{id}", handler.Delete)

	typesResponse := httptest.NewRecorder()
	router.ServeHTTP(typesResponse, httptest.NewRequest(http.MethodGet, "/types", nil))
	require.Equal(t, http.StatusOK, typesResponse.Code)
	require.Contains(t, typesResponse.Body.String(), `"mqtt_generic"`)
	require.Contains(t, typesResponse.Body.String(), `"multi_instance":true`)
	require.Contains(t, typesResponse.Body.String(), `"imou_life"`)
	require.Contains(t, typesResponse.Body.String(), `"vesync_cloud"`)
	require.NotContains(t, typesResponse.Body.String(), `"session"`)
	require.NotContains(t, typesResponse.Body.String(), `"vesync_transport"`)
	require.NotContains(t, typesResponse.Body.String(), `"xiaomi_miio_transport"`)
	require.Contains(t, typesResponse.Body.String(), `"xiaomi_home_cloud"`)
	require.NotContains(t, typesResponse.Body.String(), `"access_token"`)
	require.NotContains(t, typesResponse.Body.String(), `"callback_secret"`)
	require.NotContains(t, typesResponse.Body.String(), `"xiaomi_open_platform_transport"`)
	require.NotContains(t, typesResponse.Body.String(), `"admin_access_token"`)
	require.NotContains(t, typesResponse.Body.String(), `"refresh_token"`)
	require.NotContains(t, typesResponse.Body.String(), `"midea_a1_protocol"`)

	const secret = "open-display-secret-token-0123456789abcdef"
	requestBody := `{"id":"open_display","type":"open_display","name":"Display","enabled":false,"config":{"bridge_url":"http://127.0.0.1:10320","token":"` + secret + `"}}`
	createResponse := httptest.NewRecorder()
	router.ServeHTTP(createResponse, httptest.NewRequest(http.MethodPost, "/instances", strings.NewReader(requestBody)))
	require.Equal(t, http.StatusCreated, createResponse.Code, createResponse.Body.String())

	const imouSecret = "imou-project-secret-not-for-api"
	imouBody := `{"id":"integration_imou_life","type":"imou_life","name":"Imou Life","enabled":false,"config":{"access_key":"imou-project-key","secret_key":"` + imouSecret + `","data_center":"eu","authorization_url":"https://auth.example/imou","public_base_url":"https://domus.example"}}`
	imouResponse := httptest.NewRecorder()
	router.ServeHTTP(imouResponse, httptest.NewRequest(http.MethodPost, "/instances", strings.NewReader(imouBody)))
	require.Equal(t, http.StatusCreated, imouResponse.Code, imouResponse.Body.String())
	require.NotContains(t, imouResponse.Body.String(), imouSecret)

	vesyncBody := `{"id":"integration_vesync_cloud","type":"vesync_cloud","name":"VeSync Cloud","enabled":false,"config":{}}`
	vesyncResponse := httptest.NewRecorder()
	router.ServeHTTP(vesyncResponse, httptest.NewRequest(http.MethodPost, "/instances", strings.NewReader(vesyncBody)))
	require.Equal(t, http.StatusCreated, vesyncResponse.Code, vesyncResponse.Body.String())
	require.NotContains(t, vesyncResponse.Body.String(), `"config"`)
	require.NotContains(t, vesyncResponse.Body.String(), `"session"`)

	internalResponse := httptest.NewRecorder()
	internalBody := `{"id":"xiaomi_miio_transport","type":"xiaomi_miio_transport","name":"hidden transport","config":{}}`
	router.ServeHTTP(internalResponse, httptest.NewRequest(http.MethodPost, "/instances", strings.NewReader(internalBody)))
	require.Equal(t, http.StatusBadRequest, internalResponse.Code)
	require.Contains(t, internalResponse.Body.String(), "PLUGIN_CREATE_FAILED")
	require.NotContains(t, createResponse.Body.String(), secret)
	require.NotContains(t, createResponse.Body.String(), `"config"`)

	listResponse := httptest.NewRecorder()
	router.ServeHTTP(listResponse, httptest.NewRequest(http.MethodGet, "/instances", nil))
	require.Equal(t, http.StatusOK, listResponse.Code)
	require.Contains(t, listResponse.Body.String(), `"open_display"`)
	require.Contains(t, listResponse.Body.String(), `"imou_life"`)
	require.Contains(t, listResponse.Body.String(), `"vesync_cloud"`)
	require.NotContains(t, listResponse.Body.String(), imouSecret)
	require.NotContains(t, listResponse.Body.String(), secret)
	require.NotContains(t, listResponse.Body.String(), `"token"`)

	deleteResponse := httptest.NewRecorder()
	router.ServeHTTP(deleteResponse, httptest.NewRequest(http.MethodDelete, "/instances/open_display", nil))
	require.Equal(t, http.StatusOK, deleteResponse.Code)

	imouDelete := httptest.NewRecorder()
	router.ServeHTTP(imouDelete, httptest.NewRequest(http.MethodDelete, "/instances/integration_imou_life", nil))
	require.Equal(t, http.StatusOK, imouDelete.Code)

	vesyncDelete := httptest.NewRecorder()
	router.ServeHTTP(vesyncDelete, httptest.NewRequest(http.MethodDelete, "/instances/integration_vesync_cloud", nil))
	require.Equal(t, http.StatusOK, vesyncDelete.Code)
}

func TestPluginRuntimeHandlerRejectsInvalidJSONAndUnknownPlugin(t *testing.T) {
	repo := newPluginHandlerIntegrationRepo()
	cipher, err := crypto.NewEnvelopeCipher([]byte("plugin-handler-test-key")); require.NoError(t, err)
	manager := newPluginRuntimeHandlerManager(t, repo, cipher)
	handler := NewPluginRuntimeHandler(manager)
	router := chi.NewRouter(); router.Post("/instances", handler.Create)
	invalid := httptest.NewRecorder()
	router.ServeHTTP(invalid, httptest.NewRequest(http.MethodPost, "/instances", strings.NewReader(`{"id":"x","type":"matter","name":"bad","config":{}} {}`)))
	require.Equal(t, http.StatusBadRequest, invalid.Code)
	unknown := httptest.NewRecorder()
	router.ServeHTTP(unknown, httptest.NewRequest(http.MethodPost, "/instances", strings.NewReader(`{"id":"unknown","type":"not_registered","name":"bad","config":{}}`)))
	require.Equal(t, http.StatusNotFound, unknown.Code)
}

func TestPluginRuntimeHandlerWithoutManagerReturnsUnavailable(t *testing.T) {
	handler := NewPluginRuntimeHandler(nil)
	response := httptest.NewRecorder()
	handler.ListTypes(response, httptest.NewRequest(http.MethodGet, "/types", nil))
	require.Equal(t, http.StatusServiceUnavailable, response.Code)
}

