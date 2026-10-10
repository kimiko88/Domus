package handlers_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/domus-hub/domus/internal/application/mesh"
	"github.com/domus-hub/domus/internal/interfaces/http/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func setupBindingHandlerTest(t *testing.T) (*handlers.BindingHandler, *sql.DB) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	_, err = db.Exec(`
		CREATE TABLE zigbee_bindings (
			id TEXT PRIMARY KEY,
			source_device_id TEXT NOT NULL,
			source_device_name TEXT NOT NULL,
			target_device_id TEXT NOT NULL,
			target_device_name TEXT NOT NULL,
			clusters_json TEXT NOT NULL DEFAULT '["genOnOff"]',
			bound_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	require.NoError(t, err)

	svc := mesh.NewBindingService(db)
	return handlers.NewBindingHandler(svc), db
}

func TestBindingHandler_Routes(t *testing.T) {
	h, db := setupBindingHandlerTest(t)
	defer db.Close()

	r := chi.NewRouter()
	h.RegisterRoutes(r)

	// 1. Create binding
	body, _ := json.Marshal(map[string]any{
		"source_device_id":   "0x001",
		"source_device_name": "Switch 1",
		"target_device_id":   "0x002",
		"target_device_name": "Bulb 1",
		"clusters":           []string{"genOnOff"},
	})
	req := httptest.NewRequest(http.MethodPost, "/devices/bindings", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusCreated, rr.Code)

	var created mesh.ZigbeeBinding
	err := json.NewDecoder(rr.Body).Decode(&created)
	require.NoError(t, err)
	assert.NotEmpty(t, created.ID)

	// 2. List bindings
	req = httptest.NewRequest(http.MethodGet, "/devices/bindings", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	var list []mesh.ZigbeeBinding
	err = json.NewDecoder(rr.Body).Decode(&list)
	require.NoError(t, err)
	assert.Len(t, list, 1)

	// 3. Delete binding
	req = httptest.NewRequest(http.MethodDelete, "/devices/bindings/"+created.ID, nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusNoContent, rr.Code)
}
