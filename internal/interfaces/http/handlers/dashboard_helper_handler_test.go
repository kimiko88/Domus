package handlers_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	appDash "github.com/domus-hub/domus/internal/application/dashboard"
	appHelper "github.com/domus-hub/domus/internal/application/helper"
	domainDash "github.com/domus-hub/domus/internal/domain/dashboard"
	domainHelper "github.com/domus-hub/domus/internal/domain/helper"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	"github.com/domus-hub/domus/internal/interfaces/http/handlers"
	"github.com/go-chi/chi/v5"
	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open memory db: %v", err)
	}

	queries := []string{
		`CREATE TABLE custom_dashboards (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			icon TEXT NOT NULL DEFAULT 'layout-dashboard',
			is_default BOOLEAN NOT NULL DEFAULT FALSE,
			sort_order INTEGER NOT NULL DEFAULT 0,
			layout_json TEXT NOT NULL DEFAULT '[]',
			kiosk_config_json TEXT NOT NULL DEFAULT '{"enabled": false, "screensaver_timeout_sec": 120, "wake_on_presence": true}',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE user_helpers (
			id TEXT PRIMARY KEY,
			entity_key TEXT UNIQUE NOT NULL,
			name TEXT NOT NULL,
			domain TEXT NOT NULL,
			state TEXT NOT NULL,
			config_json TEXT NOT NULL DEFAULT '{}',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
	}
	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			t.Fatalf("failed to exec schema: %v", err)
		}
	}

	return db, func() { db.Close() }
}

func TestDashboardHandler_Endpoints(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := sqlite.NewDashboardRepository(db)
	svc := appDash.NewService(repo)
	h := handlers.NewDashboardHandler(svc)

	r := chi.NewRouter()
	r.Get("/api/v1/dashboards", h.List)
	r.Post("/api/v1/dashboards", h.Create)
	r.Get("/api/v1/dashboards/{id}", h.Get)
	r.Put("/api/v1/dashboards/{id}", h.Update)
	r.Delete("/api/v1/dashboards/{id}", h.Delete)
	r.Post("/api/v1/dashboards/{id}/default", h.SetDefault)

	// 1. Create Dashboard
	createBody, _ := json.Marshal(domainDash.Dashboard{
		ID:        "tablet_salotto",
		Title:     "Tablet Salotto",
		Icon:      "tablet",
		SortOrder: 1,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/dashboards", bytes.NewReader(createBody))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	// 2. List
	req = httptest.NewRequest(http.MethodGet, "/api/v1/dashboards", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// 3. Get By ID
	req = httptest.NewRequest(http.MethodGet, "/api/v1/dashboards/tablet_salotto", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// 4. Set Default
	req = httptest.NewRequest(http.MethodPost, "/api/v1/dashboards/tablet_salotto/default", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// 5. Delete
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/dashboards/tablet_salotto", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHelperHandler_Endpoints(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := sqlite.NewHelperRepository(db)
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 32, Workers: 1})
	svc := appHelper.NewService(repo, bus)
	h := handlers.NewHelperHandler(svc)

	r := chi.NewRouter()
	r.Get("/api/v1/helpers", h.List)
	r.Post("/api/v1/helpers", h.Create)
	r.Get("/api/v1/helpers/{id}", h.Get)
	r.Delete("/api/v1/helpers/{id}", h.Delete)
	r.Post("/api/v1/helpers/{id}/action", h.Action)

	// 1. Create Helper (input_boolean)
	createBody, _ := json.Marshal(handlers.CreateHelperRequest{
		Name:   "Modalita Ospiti",
		Domain: domainHelper.DomainInputBoolean,
		Icon:   "users",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/helpers", bytes.NewReader(createBody))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var created domainHelper.Helper
	_ = json.Unmarshal(w.Body.Bytes(), &created)

	// 2. Action turn_on
	actBody, _ := json.Marshal(handlers.ActionRequest{
		Action: "turn_on",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/helpers/"+created.ID+"/action", bytes.NewReader(actBody))
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on action, got %d", w.Code)
	}

	var updated domainHelper.Helper
	_ = json.Unmarshal(w.Body.Bytes(), &updated)
	if updated.State != "on" {
		t.Errorf("expected state 'on', got %s", updated.State)
	}

	// 3. Delete
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/helpers/"+created.ID, nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on delete, got %d", w.Code)
	}
}
