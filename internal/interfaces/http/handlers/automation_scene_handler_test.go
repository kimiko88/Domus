// Package handlers_test provides unit tests for Automation and Scene HTTP handlers.
package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appauto "github.com/domus-hub/domus/internal/application/automation"
	appscene "github.com/domus-hub/domus/internal/application/scene"
	"github.com/domus-hub/domus/internal/domain/automation"
	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/domain/scene"
	"github.com/domus-hub/domus/internal/interfaces/http/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAutomationHandler_Lifecycle(t *testing.T) {
	repos, bus := setupTestEnvironment(t)
	engine := appauto.NewEngine(repos.Automations, repos.Entities, bus)
	handler := handlers.NewAutomationHandler(repos.Automations, engine)

	r := chi.NewRouter()
	r.Get("/automations", handler.List)
	r.Post("/automations", handler.Create)
	r.Get("/automations/{id}", handler.Get)
	r.Put("/automations/{id}", handler.Update)
	r.Delete("/automations/{id}", handler.Delete)
	r.Post("/automations/{id}/trigger", handler.Trigger)
	r.Get("/automations/{id}/runs", handler.ListRuns)

	// 1. Create Automation
	body, _ := json.Marshal(map[string]any{
		"name": "Turn off hallway light after 5m",
		"triggers": []map[string]any{
			{"platform": "state", "entity_key": "binary_sensor.motion", "to": "OFF"},
		},
		"actions": []map[string]any{
			{"type": "service", "service": "turn_off", "target_entity_key": "light.hallway"},
		},
		"cooldown_seconds": 10,
	})
	req := httptest.NewRequest(http.MethodPost, "/automations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	var created automation.Automation
	require.NoError(t, json.NewDecoder(w.Body).Decode(&created))
	assert.NotEmpty(t, created.ID)
	assert.Equal(t, "Turn off hallway light after 5m", created.Name)

	// 2. Get Automation
	req = httptest.NewRequest(http.MethodGet, "/automations/"+created.ID, nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 3. List Automations
	req = httptest.NewRequest(http.MethodGet, "/automations", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var listResp struct {
		Automations []*automation.Automation `json:"automations"`
		Total       int                      `json:"total"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&listResp))
	assert.Equal(t, 1, listResp.Total)

	// 4. Update Automation
	body, _ = json.Marshal(map[string]any{
		"name":             "Renamed Automation",
		"enabled":          false,
		"cooldown_seconds": 15,
	})
	req = httptest.NewRequest(http.MethodPut, "/automations/"+created.ID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 5. Trigger Automation
	req = httptest.NewRequest(http.MethodPost, "/automations/"+created.ID+"/trigger", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusAccepted, w.Code)

	// Allow engine to process manual trigger
	time.Sleep(100 * time.Millisecond)

	// 6. List Runs
	req = httptest.NewRequest(http.MethodGet, "/automations/"+created.ID+"/runs", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var runsResp struct {
		Runs []*automation.AutomationRun `json:"runs"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&runsResp))
	assert.NotEmpty(t, runsResp.Runs)

	// 7. Delete Automation
	req = httptest.NewRequest(http.MethodDelete, "/automations/"+created.ID, nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestSceneHandler_Lifecycle(t *testing.T) {
	repos, bus := setupTestEnvironment(t)
	sceneSvc := appscene.NewService(repos.Scenes, bus)
	handler := handlers.NewSceneHandler(repos.Scenes, sceneSvc)

	r := chi.NewRouter()
	r.Get("/scenes", handler.List)
	r.Post("/scenes", handler.Create)
	r.Get("/scenes/{id}", handler.Get)
	r.Put("/scenes/{id}", handler.Update)
	r.Delete("/scenes/{id}", handler.Delete)
	r.Post("/scenes/{id}/apply", handler.Apply)

	// 1. Create Scene
	body, _ := json.Marshal(map[string]any{
		"name": "Romantic Dinner",
		"icon": "heart",
		"actions": []map[string]any{
			{
				"entity_key": "light.dining",
				"service":    "turn_on",
				"params":     map[string]any{"brightness": 50},
			},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/scenes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	var created scene.Scene
	require.NoError(t, json.NewDecoder(w.Body).Decode(&created))
	assert.NotEmpty(t, created.ID)
	assert.Equal(t, "Romantic Dinner", created.Name)

	// 2. List Scenes
	req = httptest.NewRequest(http.MethodGet, "/scenes", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 3. Apply Scene
	actChan := make(chan events.Event, 5)
	unsub := bus.Subscribe(events.SceneActivated, func(c context.Context, e events.Event) error {
		actChan <- e
		return nil
	})
	defer unsub()

	req = httptest.NewRequest(http.MethodPost, "/scenes/"+created.ID+"/apply", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	select {
	case actEvt := <-actChan:
		payload := actEvt.Payload.(map[string]any)
		assert.Equal(t, created.ID, payload["scene_id"])
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for SceneActivated")
	}

	// 4. Delete Scene
	req = httptest.NewRequest(http.MethodDelete, "/scenes/"+created.ID, nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestAutomationHandler_BlueprintsAndDryRun(t *testing.T) {
	repos, bus := setupTestEnvironment(t)
	engine := appauto.NewEngine(repos.Automations, repos.Entities, bus)
	handler := handlers.NewAutomationHandler(repos.Automations, engine).WithEntities(repos.Entities)

	r := chi.NewRouter()
	r.Get("/automations/blueprints", handler.ListBlueprints)
	r.Post("/automations/blueprints/{id}/instantiate", handler.InstantiateBlueprint)
	r.Post("/automations/{id}/dry-run", handler.DryRun)

	// 1. List Blueprints
	req := httptest.NewRequest(http.MethodGet, "/automations/blueprints", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var blueprints []map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&blueprints))
	assert.NotEmpty(t, blueprints)

	// 2. Instantiate Blueprint (window_climate_saver)
	paramsBody, _ := json.Marshal(map[string]string{
		"window_sensor":  "binary_sensor.window_office",
		"climate_entity": "climate.office_ac",
	})
	req = httptest.NewRequest(http.MethodPost, "/automations/blueprints/window_climate_saver/instantiate", bytes.NewReader(paramsBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	var instantiated automation.Automation
	require.NoError(t, json.NewDecoder(w.Body).Decode(&instantiated))
	assert.NotEmpty(t, instantiated.ID)

	// 3. Dry-Run evaluation
	req = httptest.NewRequest(http.MethodPost, "/automations/"+instantiated.ID+"/dry-run", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var dryRunRes map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&dryRunRes))
	assert.Contains(t, dryRunRes, "passed")
	assert.Contains(t, dryRunRes, "traces")
}

