package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/domus-hub/domus/internal/application/safety"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockSafetyRepo struct {
	valves    []safety.ValveRegistration
	incidents []map[string]any
}

func (m *mockSafetyRepo) SaveValve(ctx context.Context, v safety.ValveRegistration) error {
	m.valves = append(m.valves, v)
	return nil
}

func (m *mockSafetyRepo) ListValves(ctx context.Context) ([]safety.ValveRegistration, error) {
	return m.valves, nil
}

func (m *mockSafetyRepo) DeleteValve(ctx context.Context, id string) error {
	var remaining []safety.ValveRegistration
	for _, v := range m.valves {
		if v.ID != id {
			remaining = append(remaining, v)
		}
	}
	m.valves = remaining
	return nil
}

func (m *mockSafetyRepo) RecordIncident(ctx context.Context, alert safety.EmergencyAlert, result safety.EmergencyResult) error {
	m.incidents = append(m.incidents, map[string]any{
		"alert_id": alert.ID,
		"hazard":   string(alert.Hazard),
	})
	return nil
}

func (m *mockSafetyRepo) ListIncidents(ctx context.Context, limit int) ([]map[string]any, error) {
	return m.incidents, nil
}

func (m *mockSafetyRepo) ClearIncident(ctx context.Context, alertID, clearedBy, reason string) error {
	return nil
}

func TestSafetyHandler(t *testing.T) {
	repo := &mockSafetyRepo{
		valves: []safety.ValveRegistration{
			{ID: "valve-main", Type: safety.ValveWater, Name: "Main Water Valve", Location: "Basement"},
		},
		incidents: []map[string]any{
			{"alert_id": "inc-1", "hazard": "water_leak"},
		},
	}

	svc := safety.NewService(nil).WithRepository(repo)
	h := NewSafetyHandler(svc)

	r := chi.NewRouter()
	h.RegisterRoutes(r)

	t.Run("GetStatus", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/status", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var status safety.EmergencyStatus
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &status))
		assert.False(t, status.IsLatched)
		assert.Equal(t, 1, status.RegisteredCount)
	})

	t.Run("ListValves", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/valves", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var valves []safety.ValveRegistration
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &valves))
		assert.Len(t, valves, 1)
		assert.Equal(t, "valve-main", valves[0].ID)
	})

	t.Run("RegisterValve", func(t *testing.T) {
		body := bytes.NewReader([]byte(`{"id":"valve-gas-1","type":"gas","name":"Kitchen Gas","location":"Kitchen"}`))
		req := httptest.NewRequest(http.MethodPost, "/valves", body)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)
	})

	t.Run("DeleteValve", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/valves/valve-gas-1", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("TriggerSafeStop", func(t *testing.T) {
		body := bytes.NewReader([]byte(`{"hazard":"water_leak","source_sensor":"sensor.leak_1","location":"Basement","severity":"critical"}`))
		req := httptest.NewRequest(http.MethodPost, "/trigger", body)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var res safety.EmergencyResult
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &res))
		assert.True(t, res.Success)
	})

	t.Run("ResetEmergency", func(t *testing.T) {
		body := bytes.NewReader([]byte(`{"operator_id":"op-1","reason":"Leak verified and fixed"}`))
		req := httptest.NewRequest(http.MethodPost, "/reset", body)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("ListIncidents", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/incidents", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var incs []map[string]any
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &incs))
		assert.NotEmpty(t, incs)
	})
}
