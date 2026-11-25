package handlers_test

import (
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"

    "github.com/domus-hub/domus/internal/application/marketplace"
    "github.com/domus-hub/domus/internal/interfaces/http/handlers"
    "github.com/go-chi/chi/v5"
)

func TestMarketplaceHandlerAddsOnlyHTTPSRepositories(t *testing.T) {
    service, err := marketplace.NewService(marketplace.NewMemoryRepository(), marketplace.Config{})
    if err != nil { t.Fatal(err) }
    h := handlers.NewMarketplaceHandler(service)
    router := chi.NewRouter()
    router.Post("/repositories", h.AddRepository)

    request := httptest.NewRequest(http.MethodPost, "/repositories", strings.NewReader(`{"url":"http://example.invalid/plugins"}`))
    response := httptest.NewRecorder()
    router.ServeHTTP(response, request)
    if response.Code != http.StatusBadRequest {
        t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
    }
}

func TestMarketplaceHandlerListsEmptyRepositories(t *testing.T) {
    service, err := marketplace.NewService(marketplace.NewMemoryRepository(), marketplace.Config{})
    if err != nil { t.Fatal(err) }
    h := handlers.NewMarketplaceHandler(service)
    router := chi.NewRouter()
    router.Get("/repositories", h.ListRepositories)

    response := httptest.NewRecorder()
    router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/repositories", nil))
    if response.Code != http.StatusOK { t.Fatalf("status=%d body=%s", response.Code, response.Body.String()) }
}
