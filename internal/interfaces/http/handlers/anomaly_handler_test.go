package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/application/anomaly"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockAnomalyRepo struct {
	alerts     []*anomaly.AnomalyAlert
	appliances []map[string]any
}

func (m *mockAnomalyRepo) SaveAlert(ctx context.Context, a *anomaly.AnomalyAlert) error {
	m.alerts = append(m.alerts, a)
	return nil
}

func (m *mockAnomalyRepo) ListAlerts(ctx context.Context, ack *bool, limit int) ([]*anomaly.AnomalyAlert, error) {
	var res []*anomaly.AnomalyAlert
	for _, a := range m.alerts {
		if ack != nil && a.Acknowledged != *ack {
			continue
		}
		res = append(res, a)
	}
	return res, nil
}

func (m *mockAnomalyRepo) AcknowledgeAlert(ctx context.Context, id string) error {
	for _, a := range m.alerts {
		if a.ID == id {
			a.Acknowledged = true
		}
	}
	return nil
}

func (m *mockAnomalyRepo) SaveWaterMeter(ctx context.Context, meterID string, flowLMin float64, lastSeen time.Time) error {
	return nil
}

func (m *mockAnomalyRepo) GetWaterMeter(ctx context.Context, meterID string) (float64, time.Time, error) {
	return 0.0, time.Now().UTC(), nil
}

func (m *mockAnomalyRepo) SaveAppliance(ctx context.Context, id, kind string, nominalWatts, lastWatts, lastTemp float64, lastSeen time.Time) error {
	return nil
}

func (m *mockAnomalyRepo) ListAppliances(ctx context.Context) ([]map[string]any, error) {
	return m.appliances, nil
}

func TestAnomalyHandler(t *testing.T) {
	repo := &mockAnomalyRepo{
		alerts: []*anomaly.AnomalyAlert{
			{
				ID:           "alert-1",
				Category:     "water_leak",
				Severity:     "critical",
				EntityID:     "sensor.water_meter",
				Title:        "Water Leak",
				Acknowledged: false,
			},
		},
		appliances: []map[string]any{
			{"id": "fridge-1", "kind": "refrigerator", "status": "healthy"},
		},
	}

	svc := anomaly.NewService(nil).WithRepository(repo)
	h := NewAnomalyHandler(svc)

	r := chi.NewRouter()
	h.RegisterRoutes(r)

	t.Run("ListAlerts", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/?acknowledged=false", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var alerts []*anomaly.AnomalyAlert
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &alerts))
		assert.Len(t, alerts, 1)
		assert.Equal(t, "alert-1", alerts[0].ID)
	})

	t.Run("AcknowledgeAlert", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/alert-1/acknowledge", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.True(t, repo.alerts[0].Acknowledged)
	})

	t.Run("GetWaterStatus", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/water", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("IngestWaterFlow", func(t *testing.T) {
		body := bytes.NewReader([]byte(`{"meter_id":"meter-main","flow_l_min":0.0}`))
		req := httptest.NewRequest(http.MethodPost, "/water/ingest", body)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("AssessMold", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/mold?ambient_c=20.0&rh_pct=65.0&surface_c=14.0", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var res anomaly.MoldAssessment
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &res))
		assert.NotEmpty(t, res.RiskLevel)
	})

	t.Run("ListAppliances", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/appliances", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var apps []map[string]any
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &apps))
		assert.Len(t, apps, 1)
	})
}
