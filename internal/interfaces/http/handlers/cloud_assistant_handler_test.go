package handlers_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appCloud "github.com/domus-hub/domus/internal/application/cloud_assistant"
	"github.com/domus-hub/domus/internal/domain/entity"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/domus-hub/domus/internal/interfaces/http/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockEntityRepoForCloud struct {
	items map[string]*entity.Entity
}

func (m *mockEntityRepoForCloud) Create(ctx context.Context, e *entity.Entity) error { return nil }
func (m *mockEntityRepoForCloud) GetByID(ctx context.Context, id string) (*entity.Entity, error) {
	if it, ok := m.items[id]; ok {
		return it, nil
	}
	return nil, entity.ErrEntityNotFound
}
func (m *mockEntityRepoForCloud) GetByKey(ctx context.Context, key string) (*entity.Entity, error) {
	return nil, entity.ErrEntityNotFound
}
func (m *mockEntityRepoForCloud) List(ctx context.Context, filter entity.ListFilter) ([]*entity.Entity, error) {
	res := make([]*entity.Entity, 0, len(m.items))
	for _, it := range m.items {
		res = append(res, it)
	}
	return res, nil
}
func (m *mockEntityRepoForCloud) Update(ctx context.Context, e *entity.Entity) error { return nil }
func (m *mockEntityRepoForCloud) UpdateState(ctx context.Context, id string, stateJSON, attributesJSON string, available bool, at time.Time) error {
	return nil
}
func (m *mockEntityRepoForCloud) Delete(ctx context.Context, id string) error { return nil }

func setupCloudTestRouter(token string, enabled bool) chi.Router {
	repo := &mockEntityRepoForCloud{
		items: map[string]*entity.Entity{
			"light.living": {
				ID:         "light.living",
				Name:       "Living Light",
				Domain:     "light",
				State:      `"on"`,
				Attributes: `{"brightness": 128}`,
			},
		},
	}
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 16, Workers: 1})
	svc := appCloud.NewService(repo, bus, token, enabled)
	h := handlers.NewCloudAssistantHandler(svc)

	r := chi.NewRouter()
	h.RegisterRoutes(r)
	return r
}

func TestCloudAssistantHandler_Auth(t *testing.T) {
	r := setupCloudTestRouter("secret-token", true)

	// 1. Missing auth
	req := httptest.NewRequest(http.MethodPost, "/cloud/alexa", bytes.NewBufferString("{}"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// 2. Invalid auth
	req = httptest.NewRequest(http.MethodPost, "/cloud/alexa", bytes.NewBufferString("{}"))
	req.Header.Set("Authorization", "Bearer invalid-token")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCloudAssistantHandler_AlexaAndGoogle_Success(t *testing.T) {
	r := setupCloudTestRouter("secret-token", true)

	// Alexa Discovery
	alexaBody := `{
		"directive": {
			"header": {
				"namespace": "Alexa.Discovery",
				"name": "Discover",
				"payloadVersion": "3",
				"messageId": "msg-1",
				"correlationToken": "token-1"
			},
			"payload": {}
		}
	}`
	req := httptest.NewRequest(http.MethodPost, "/cloud/alexa", bytes.NewBufferString(alexaBody))
	req.Header.Set("Authorization", "Bearer secret-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Discover.Response")

	// Google SYNC
	googleBody := `{
		"requestId": "req-1",
		"inputs": [{"intent": "action.devices.SYNC"}]
	}`
	req = httptest.NewRequest(http.MethodPost, "/cloud/google", bytes.NewBufferString(googleBody))
	req.Header.Set("Authorization", "Bearer secret-token")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "action.devices.types.LIGHT")
}

func TestCloudAssistantHandler_Disabled(t *testing.T) {
	r := setupCloudTestRouter("secret-token", false)

	req := httptest.NewRequest(http.MethodPost, "/cloud/alexa", bytes.NewBufferString("{}"))
	req.Header.Set("Authorization", "Bearer secret-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}
