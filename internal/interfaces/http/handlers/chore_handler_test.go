package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appchore "github.com/domus-hub/domus/internal/application/chore"
	domainchore "github.com/domus-hub/domus/internal/domain/chore"
	"github.com/domus-hub/domus/internal/interfaces/http/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChoreHandler_Lifecycle(t *testing.T) {
	repos, bus := setupTestEnvironment(t)
	choreService := appchore.NewService(repos.Chores, bus)
	handler := handlers.NewChoreHandler(repos.Chores, choreService)

	r := chi.NewRouter()
	r.Get("/chores", handler.List)
	r.Post("/chores", handler.Create)
	r.Get("/chores/{id}", handler.GetByID)
	r.Put("/chores/{id}", handler.Update)
	r.Delete("/chores/{id}", handler.Delete)
	r.Post("/chores/{id}/complete", handler.Complete)

	// 1. Create a Chore with daily recurrence
	dueStr := time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)
	createBody, _ := json.Marshal(map[string]any{
		"title":           "Pulizia filtri condizionatore",
		"description":     "Lavare e asciugare i filtri del salotto",
		"priority":        "high",
		"due_date":        dueStr,
		"recurrence_type": "daily",
		"recurrence_rule": "1",
	})
	req := httptest.NewRequest(http.MethodPost, "/chores", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	var created domainchore.Chore
	require.NoError(t, json.NewDecoder(w.Body).Decode(&created))
	assert.NotEmpty(t, created.ID)
	assert.Equal(t, "Pulizia filtri condizionatore", created.Title)
	assert.Equal(t, domainchore.PriorityHigh, created.Priority)
	assert.Equal(t, domainchore.StatusPending, created.Status)

	// 2. Get Chore by ID
	req = httptest.NewRequest(http.MethodGet, "/chores/"+created.ID, nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var fetched domainchore.Chore
	require.NoError(t, json.NewDecoder(w.Body).Decode(&fetched))
	assert.Equal(t, created.ID, fetched.ID)

	// 3. Update Chore
	updateBody, _ := json.Marshal(map[string]any{
		"title":       "Pulizia filtri condizionatore salotto e camera",
		"description": "Lavare filtri aria",
		"priority":    "medium",
	})
	req = httptest.NewRequest(http.MethodPut, "/chores/"+created.ID, bytes.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var updated domainchore.Chore
	require.NoError(t, json.NewDecoder(w.Body).Decode(&updated))
	assert.Equal(t, "Pulizia filtri condizionatore salotto e camera", updated.Title)
	assert.Equal(t, domainchore.PriorityMedium, updated.Priority)

	// 4. Complete Chore
	req = httptest.NewRequest(http.MethodPost, "/chores/"+created.ID+"/complete", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// 5. Verify chore was marked as completed
	req = httptest.NewRequest(http.MethodGet, "/chores/"+created.ID, nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var afterComplete domainchore.Chore
	require.NoError(t, json.NewDecoder(w.Body).Decode(&afterComplete))
	assert.Equal(t, domainchore.StatusCompleted, afterComplete.Status)
	assert.NotNil(t, afterComplete.LastCompletedAt)

	// 6. List pending chores to verify the next occurrence was provisioned
	req = httptest.NewRequest(http.MethodGet, "/chores?status=pending", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var pendingChores []*domainchore.Chore
	require.NoError(t, json.NewDecoder(w.Body).Decode(&pendingChores))
	assert.Len(t, pendingChores, 1)
	assert.Equal(t, "Pulizia filtri condizionatore salotto e camera", pendingChores[0].Title)

	// 7. Delete Chore
	req = httptest.NewRequest(http.MethodDelete, "/chores/"+created.ID, nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}
