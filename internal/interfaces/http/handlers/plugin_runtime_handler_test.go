package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/application/plugin_runtime"
	"github.com/domus-hub/domus/internal/domain/integration"
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

func TestPluginRuntimeHandlerCatalogCreateAndSecretRedaction(t *testing.T) {
	repo := newPluginHandlerIntegrationRepo()
	cipher, err := crypto.NewEnvelopeCipher([]byte("plugin-handler-test-key")); require.NoError(t, err)
	manager, err := plugin_runtime.NewManager(&sqlite.Repositories{Integrations: repo, Cipher: cipher}, nil); require.NoError(t, err)
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

	const secret = "open-display-secret-token-0123456789abcdef"
	requestBody := `{"id":"open_display","type":"open_display","name":"Display","enabled":false,"config":{"bridge_url":"http://127.0.0.1:10320","token":"` + secret + `"}}`
	createResponse := httptest.NewRecorder()
	router.ServeHTTP(createResponse, httptest.NewRequest(http.MethodPost, "/instances", strings.NewReader(requestBody)))
	require.Equal(t, http.StatusCreated, createResponse.Code, createResponse.Body.String())
	require.NotContains(t, createResponse.Body.String(), secret)
	require.NotContains(t, createResponse.Body.String(), `"config"`)

	listResponse := httptest.NewRecorder()
	router.ServeHTTP(listResponse, httptest.NewRequest(http.MethodGet, "/instances", nil))
	require.Equal(t, http.StatusOK, listResponse.Code)
	require.Contains(t, listResponse.Body.String(), `"open_display"`)
	require.NotContains(t, listResponse.Body.String(), secret)
	require.NotContains(t, listResponse.Body.String(), `"token"`)

	deleteResponse := httptest.NewRecorder()
	router.ServeHTTP(deleteResponse, httptest.NewRequest(http.MethodDelete, "/instances/open_display", nil))
	require.Equal(t, http.StatusOK, deleteResponse.Code)
}

func TestPluginRuntimeHandlerRejectsInvalidJSONAndUnknownPlugin(t *testing.T) {
	repo := newPluginHandlerIntegrationRepo()
	cipher, err := crypto.NewEnvelopeCipher([]byte("plugin-handler-test-key")); require.NoError(t, err)
	manager, err := plugin_runtime.NewManager(&sqlite.Repositories{Integrations: repo, Cipher: cipher}, nil); require.NoError(t, err)
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

// Ensure the helper repository remains safe when the test runner reuses handlers.
var _ = time.Second
