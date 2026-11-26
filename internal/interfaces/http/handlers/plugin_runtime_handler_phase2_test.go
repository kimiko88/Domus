package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/domus-hub/domus/internal/infrastructure/crypto"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func TestPluginRuntimeHandlerUpdatesEncryptedConfigurationWithoutExposingSecrets(t *testing.T) {
	repo := newPluginHandlerIntegrationRepo()
	cipher, err := newHandlerTestCipher()
	require.NoError(t, err)
	manager := newPluginRuntimeHandlerManager(t, repo, cipher)
	handler := NewPluginRuntimeHandler(manager)
	router := chi.NewRouter()
	router.Post("/instances", handler.Create)
	router.Put("/instances/{id}/config", handler.UpdateConfig)

	const oldSecret = "old-open-display-secret-0123456789abcdef"
	createBody := `{"id":"display_reconfigure","type":"open_display","name":"Display","enabled":false,"config":{"bridge_url":"http://127.0.0.1:10320","token":"` + oldSecret + `"}}`
	create := httptest.NewRecorder()
	router.ServeHTTP(create, httptest.NewRequest(http.MethodPost, "/instances", strings.NewReader(createBody)))
	require.Equal(t, http.StatusCreated, create.Code, create.Body.String())

	const newSecret = "new-open-display-secret-0123456789abcdef"
	updateBody := `{"config":{"bridge_url":"http://127.0.0.1:10321","token":"` + newSecret + `"}}`
	update := httptest.NewRecorder()
	router.ServeHTTP(update, httptest.NewRequest(http.MethodPut, "/instances/display_reconfigure/config", strings.NewReader(updateBody)))
	require.Equal(t, http.StatusOK, update.Code, update.Body.String())
	require.Contains(t, update.Body.String(), "display_reconfigure")
	require.NotContains(t, update.Body.String(), oldSecret)
	require.NotContains(t, update.Body.String(), newSecret)
	require.NotContains(t, update.Body.String(), `"config"`)
	require.NotContains(t, update.Body.String(), `"token"`)

	stored, err := repo.GetByID(context.Background(), "display_reconfigure")
	require.NoError(t, err)
	plain, err := cipher.Decrypt(string(stored.ConfigEncrypted))
	require.NoError(t, err)
	var saved map[string]string
	require.NoError(t, json.Unmarshal([]byte(plain), &saved))
	require.Equal(t, "http://127.0.0.1:10321", saved["bridge_url"])
	require.Equal(t, newSecret, saved["token"])

	invalid := httptest.NewRecorder()
	router.ServeHTTP(invalid, httptest.NewRequest(http.MethodPut, "/instances/display_reconfigure/config", strings.NewReader(`{"config":{"unknown":"field"}}`)))
	require.Equal(t, http.StatusBadRequest, invalid.Code)
	stored, err = repo.GetByID(context.Background(), "display_reconfigure")
	require.NoError(t, err)
	plain, err = cipher.Decrypt(string(stored.ConfigEncrypted))
	require.NoError(t, err)
	require.Contains(t, plain, "http://127.0.0.1:10321")
	require.Contains(t, plain, newSecret)
}

func TestPluginRuntimeHandlerRejectsMissingConfigOnUpdate(t *testing.T) {
	repo := newPluginHandlerIntegrationRepo()
	cipher, err := newHandlerTestCipher()
	require.NoError(t, err)
	manager := newPluginRuntimeHandlerManager(t, repo, cipher)
	handler := NewPluginRuntimeHandler(manager)
	router := chi.NewRouter()
	router.Put("/instances/{id}/config", handler.UpdateConfig)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPut, "/instances/missing/config", strings.NewReader(`{}`)))
	require.Equal(t, http.StatusBadRequest, response.Code)
	require.Contains(t, response.Body.String(), "INVALID_PLUGIN_CONFIG")
}

func newHandlerTestCipher() (*crypto.EnvelopeCipher, error) {
	return crypto.NewEnvelopeCipher([]byte("phase-two-plugin-handler-key"))
}

