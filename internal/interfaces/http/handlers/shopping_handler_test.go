package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	appShop "github.com/domus-hub/domus/internal/application/shopping"
	domainShop "github.com/domus-hub/domus/internal/domain/shopping"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/domus-hub/domus/internal/interfaces/http/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupShoppingRouter() (*chi.Mux, *handlers.ShoppingHandler) {
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 64, Workers: 1})
	repo := appShop.NewMemoryRepository()
	svc := appShop.NewService(repo, bus)
	h := handlers.NewShoppingHandler(svc)

	r := chi.NewRouter()
	r.Get("/shopping", h.List)
	r.Post("/shopping", h.Create)
	r.Post("/shopping/{id}/toggle", h.Toggle)
	r.Delete("/shopping/{id}", h.Delete)
	r.Post("/shopping/clear-completed", h.ClearCompleted)

	return r, h
}

func TestShoppingHandler_Endpoints(t *testing.T) {
	r, _ := setupShoppingRouter()

	// 1. Create item
	body := map[string]any{
		"name":     "Pasta di Gragnano",
		"quantity": "500g",
		"category": domainShop.CategoryGrocery,
		"added_by": "Giulia",
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/shopping", bytes.NewReader(raw))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusCreated, rr.Code)

	var created domainShop.Item
	err := json.Unmarshal(rr.Body.Bytes(), &created)
	require.NoError(t, err)
	assert.NotEmpty(t, created.ID)
	assert.Equal(t, "Pasta di Gragnano", created.Name)

	// 2. List items
	req = httptest.NewRequest(http.MethodGet, "/shopping", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	var list []domainShop.Item
	err = json.Unmarshal(rr.Body.Bytes(), &list)
	require.NoError(t, err)
	assert.Len(t, list, 1)

	// 3. Toggle item
	req = httptest.NewRequest(http.MethodPost, "/shopping/"+created.ID+"/toggle", bytes.NewReader([]byte(`{"completed_by":"Andrea"}`)))
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	var toggled domainShop.Item
	err = json.Unmarshal(rr.Body.Bytes(), &toggled)
	require.NoError(t, err)
	assert.True(t, toggled.Completed)
	assert.Equal(t, "Andrea", *toggled.CompletedBy)

	// 4. Clear completed
	req = httptest.NewRequest(http.MethodPost, "/shopping/clear-completed", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// 5. List items again -> 0
	req = httptest.NewRequest(http.MethodGet, "/shopping", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	err = json.Unmarshal(rr.Body.Bytes(), &list)
	require.NoError(t, err)
	assert.Len(t, list, 0)
}
