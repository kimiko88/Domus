package handlers_test

import (
    "encoding/base64"
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"

    "github.com/domus-hub/domus/internal/interfaces/http/handlers"
    "github.com/go-chi/chi/v5"
)

func TestOpenDisplayRoutesReturnServiceUnavailableWhenDisabled(t *testing.T) {
    h := handlers.NewOpenDisplayHandler(nil)
    r := chi.NewRouter()
    r.Get("/devices", h.ListDevices)
    r.Post("/devices/{address}/image", h.UploadImage)
    r.Post("/devices/{address}/actions/{action}", h.Action)

    get := httptest.NewRecorder()
    r.ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/devices", nil))
    if get.Code != http.StatusServiceUnavailable {
        t.Fatalf("GET status=%d, want 503", get.Code)
    }

    post := httptest.NewRecorder()
    body := `{"image_base64":"` + base64.StdEncoding.EncodeToString([]byte("not-an-image")) + `"}`
    r.ServeHTTP(post, httptest.NewRequest(http.MethodPost, "/devices/AA:BB:CC:DD:EE:FF/image", strings.NewReader(body)))
    if post.Code != http.StatusServiceUnavailable {
        t.Fatalf("POST status=%d, want 503", post.Code)
    }

    action := httptest.NewRecorder()
    r.ServeHTTP(action, httptest.NewRequest(http.MethodPost, "/devices/AA:BB:CC:DD:EE:FF/actions/led", strings.NewReader(`{"brightness":8}`)))
    if action.Code != http.StatusServiceUnavailable {
        t.Fatalf("Action status=%d, want 503", action.Code)
    }
}
