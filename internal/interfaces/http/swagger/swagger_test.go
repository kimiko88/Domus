package swagger_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/domus-hub/domus/internal/interfaces/http/swagger"
)

func TestSwaggerHandler_ServeSpec(t *testing.T) {
	h := swagger.NewHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/openapi.yaml", nil)
	rr := httptest.NewRecorder()

	h.ServeSpec(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	ct := rr.Header().Get("Content-Type")
	if !strings.Contains(ct, "yaml") {
		t.Errorf("expected Content-Type containing yaml, got %q", ct)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "openapi: 3.1.0") {
		t.Errorf("expected body to contain 'openapi: 3.1.0', got %q", body[:min(len(body), 100)])
	}

	if !strings.Contains(body, "Domus API") {
		t.Errorf("expected body to contain 'Domus API'")
	}
}

func TestSwaggerHandler_ServeUI(t *testing.T) {
	h := swagger.NewHandler()

	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	rr := httptest.NewRecorder()

	h.ServeUI(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	ct := rr.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("expected Content-Type text/html, got %q", ct)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "SwaggerUIBundle") {
		t.Errorf("expected HTML to contain SwaggerUIBundle")
	}

	if !strings.Contains(body, "/api/openapi.yaml") {
		t.Errorf("expected HTML to point to /api/openapi.yaml")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
