package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/application/automation"
	"github.com/domus-hub/domus/internal/application/energy"
	"github.com/domus-hub/domus/internal/application/presence"
	"github.com/domus-hub/domus/internal/domain/entity"
	"github.com/domus-hub/domus/internal/domain/user"
	domusmw "github.com/domus-hub/domus/internal/interfaces/http/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdvancedFeaturesHandler_LoadShedding(t *testing.T) {
	cfg := energy.LoadSheddingConfig{
		Enabled:          true,
		MeterEntityKey:   "sensor.power",
		MaxPowerWatts:    3300.0,
		OverloadDuration: 60,
		RecoveryWatts:    2500.0,
		RecoveryDelaySec: 120,
		PriorityLoads:    []string{"switch.ac", "switch.water_heater"},
	}
	energyCtrl := energy.NewController(cfg, nil)

	t.Run("nil controller returns 404", func(t *testing.T) {
		h := NewAdvancedFeaturesHandler(nil, nil, nil)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/energy/load-shedding", nil)
		rr := httptest.NewRecorder()
		h.GetLoadSheddingStatus(rr, req)
		assert.Equal(t, http.StatusNotFound, rr.Code)

		req = httptest.NewRequest(http.MethodPut, "/api/v1/energy/load-shedding", bytes.NewReader([]byte("{}")))
		rr = httptest.NewRecorder()
		h.UpdateLoadShedding(rr, req)
		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("valid controller returns status and updates config", func(t *testing.T) {
		h := NewAdvancedFeaturesHandler(energyCtrl, nil, nil)

		// Get status
		req := httptest.NewRequest(http.MethodGet, "/api/v1/energy/load-shedding", nil)
		rr := httptest.NewRecorder()
		h.GetLoadSheddingStatus(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)

		var st map[string]any
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &st))
		assert.True(t, st["enabled"].(bool))
		assert.Equal(t, 3300.0, st["max_power_watts"])

		// Update config
		newCfgBody, _ := json.Marshal(map[string]any{
			"enabled":         false,
			"max_power_watts": 4500.0,
		})
		req = httptest.NewRequest(http.MethodPut, "/api/v1/energy/load-shedding", bytes.NewReader(newCfgBody))
		rr = httptest.NewRecorder()
		h.UpdateLoadShedding(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	})
}

func TestAdvancedFeaturesHandler_Presence(t *testing.T) {
	geofence := presence.HomeGeofence{
		Latitude:  45.4642,
		Longitude: 9.1900,
		RadiusM:   150.0,
	}
	presenceSvc := presence.NewService(geofence, nil)

	t.Run("nil presence returns 404", func(t *testing.T) {
		h := NewAdvancedFeaturesHandler(nil, nil, nil)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/presence", nil)
		rr := httptest.NewRecorder()
		h.GetPresenceStatus(rr, req)
		assert.Equal(t, http.StatusNotFound, rr.Code)

		req = httptest.NewRequest(http.MethodPost, "/api/v1/presence/gps", bytes.NewReader([]byte("{}")))
		rr = httptest.NewRecorder()
		h.UpdateGPS(rr, req)
		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("valid presence GPS update requires user auth", func(t *testing.T) {
		h := NewAdvancedFeaturesHandler(nil, presenceSvc, nil)

		// Unauthenticated returns 401
		gpsBody, _ := json.Marshal(map[string]any{
			"latitude":  45.4642,
			"longitude": 9.1900,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/presence/gps", bytes.NewReader(gpsBody))
		rr := httptest.NewRecorder()
		h.UpdateGPS(rr, req)
		assert.Equal(t, http.StatusUnauthorized, rr.Code)

		// Authenticated returns 200
		testUser := &user.User{ID: "user-mario", Username: "mario", Role: user.RoleOperator}
		reqWithAuth := req.WithContext(domusmw.WithUser(req.Context(), testUser))
		reqWithAuth.Body = io.NopCloser(bytes.NewReader(gpsBody))
		rr = httptest.NewRecorder()
		h.UpdateGPS(rr, reqWithAuth)
		assert.Equal(t, http.StatusOK, rr.Code)

		var resp map[string]any
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
		assert.Equal(t, "user-mario", resp["user_id"])
		assert.NotEmpty(t, resp["state"])

		// Get Presence status
		req = httptest.NewRequest(http.MethodGet, "/api/v1/presence/status", nil)
		rr = httptest.NewRecorder()
		h.GetPresenceStatus(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	})
}

func TestAdvancedFeaturesHandler_SunPosition(t *testing.T) {
	t.Run("nil sun engine defaults to fallback coordinates and returns 200", func(t *testing.T) {
		h := NewAdvancedFeaturesHandler(nil, nil, nil)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/system/sun", nil)
		rr := httptest.NewRecorder()
		h.GetSunPosition(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)

		var pos automation.SunPosition
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &pos))
		assert.NotZero(t, pos.Sunrise)
	})

	t.Run("explicit sun engine returns calculated coordinates", func(t *testing.T) {
		sunEngine := automation.NewSunEngine(45.4642, 9.1900)
		h := NewAdvancedFeaturesHandler(nil, nil, sunEngine)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/system/sun", nil)
		rr := httptest.NewRecorder()
		h.GetSunPosition(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)

		var pos automation.SunPosition
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &pos))
		assert.NotZero(t, pos.Sunrise)
	})
}

type mockEntityRepo struct {
	entities []*entity.Entity
}

func (m *mockEntityRepo) Create(ctx context.Context, e *entity.Entity) error { return nil }
func (m *mockEntityRepo) GetByID(ctx context.Context, id string) (*entity.Entity, error) {
	return nil, nil
}
func (m *mockEntityRepo) GetByKey(ctx context.Context, key string) (*entity.Entity, error) {
	return nil, nil
}
func (m *mockEntityRepo) List(ctx context.Context, filter entity.ListFilter) ([]*entity.Entity, error) {
	return m.entities, nil
}
func (m *mockEntityRepo) Update(ctx context.Context, e *entity.Entity) error { return nil }
func (m *mockEntityRepo) UpdateState(ctx context.Context, id string, stateJSON, attributesJSON string, available bool, at time.Time) error {
	return nil
}
func (m *mockEntityRepo) Delete(ctx context.Context, id string) error { return nil }

func TestAdvancedFeaturesHandler_EnergyFlow(t *testing.T) {
	t.Run("nil entity repo returns zeroed flow", func(t *testing.T) {
		h := NewAdvancedFeaturesHandler(nil, nil, nil)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/energy/flow", nil)
		rr := httptest.NewRecorder()
		h.GetEnergyFlow(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)

		var flow energy.SankeyFlow
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &flow))
		assert.Equal(t, 0.0, flow.SolarWatts)
		assert.Equal(t, 0.0, flow.GridWatts)
	})

	t.Run("computes flow from registered database entities", func(t *testing.T) {
		repo := &mockEntityRepo{
			entities: []*entity.Entity{
				{
					ID:        "ent_solar",
					EntityKey: "sensor.pv_solar_power",
					State:     "3200",
				},
				{
					ID:        "ent_grid",
					EntityKey: "sensor.grid_power",
					State:     "-500",
				},
				{
					ID:        "ent_battery_pwr",
					EntityKey: "sensor.battery_power",
					State:     "-1200",
				},
				{
					ID:        "ent_battery_soc",
					EntityKey: "sensor.battery_soc",
					State:     "85",
				},
			},
		}

		h := NewAdvancedFeaturesHandler(nil, nil, nil).WithEntityRepository(repo)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/energy/flow", nil)
		rr := httptest.NewRecorder()
		h.GetEnergyFlow(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)

		var flow energy.SankeyFlow
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &flow))
		assert.Equal(t, 3200.0, flow.SolarWatts)
		assert.Equal(t, -500.0, flow.GridWatts)
		assert.Equal(t, -1200.0, flow.BatteryWatts)
		assert.Equal(t, 85.0, flow.BatterySoC)
		assert.Equal(t, 1500.0, flow.HouseConsumptionWatts)
		assert.Equal(t, 500.0, flow.SolarToGridWatts)
	})
}

func TestAdvancedFeaturesHandler_DynamicTariffsAndFlexibleLoads(t *testing.T) {
	h := NewAdvancedFeaturesHandler(nil, nil, nil)

	// 1. GetTariffs
	req := httptest.NewRequest(http.MethodGet, "/api/v1/energy/tariffs", nil)
	rr := httptest.NewRecorder()
	h.GetTariffs(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	var tariffsResp map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &tariffsResp))
	assert.NotEmpty(t, tariffsResp["current_slot"])
	assert.NotNil(t, tariffsResp["tariffs"])

	// 2. GetDynamicTariffs
	req = httptest.NewRequest(http.MethodGet, "/api/v1/energy/dynamic-tariffs", nil)
	rr = httptest.NewRecorder()
	h.GetDynamicTariffs(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	var dynResp map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &dynResp))
	assert.NotNil(t, dynResp["schedule"])
	assert.NotNil(t, dynResp["current_tariff"])
	assert.NotNil(t, dynResp["flexible_loads"])

	// 3. UpdateFlexibleLoad
	newLoad := energy.FlexibleLoadRule{
		EntityKey:     "switch.ev_charger",
		Name:          "Wallbox EV",
		RequiredHours: 3,
		MaxPriceKWh:   0.14,
		AutoSchedule:  true,
	}
	rawLoad, _ := json.Marshal(newLoad)
	req = httptest.NewRequest(http.MethodPut, "/api/v1/energy/flexible-loads", bytes.NewReader(rawLoad))
	rr = httptest.NewRecorder()
	h.UpdateFlexibleLoad(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// 4. EvaluateTariffs
	req = httptest.NewRequest(http.MethodPost, "/api/v1/energy/evaluate-tariffs", nil)
	rr = httptest.NewRecorder()
	h.EvaluateTariffs(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestAdvancedFeaturesHandler_HomeKitAndMatter(t *testing.T) {
	h := NewAdvancedFeaturesHandler(nil, nil, nil)

	// 1. HomeKit pairing info
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hardware/homekit", nil)
	rr := httptest.NewRecorder()
	h.GetHomeKitPairing(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	var hkResp map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &hkResp))
	assert.Equal(t, "Domus Bridge", hkResp["bridge_name"])
	assert.NotEmpty(t, hkResp["setup_uri"])

	// 2. Matter commissioning with missing code -> 400
	req = httptest.NewRequest(http.MethodPost, "/api/v1/hardware/matter/commission", bytes.NewReader([]byte(`{"code":""}`)))
	rr = httptest.NewRecorder()
	h.CommissionMatter(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)

	// 3. Matter commissioning with valid manual code -> 200
	// 11-digit manual pairing code (e.g., standard Matter test payload)
	manualCode := "34970112332"
	body, _ := json.Marshal(map[string]string{"code": manualCode})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/hardware/matter/commission", bytes.NewReader(body))
	rr = httptest.NewRecorder()
	h.CommissionMatter(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	var matterResp map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &matterResp))
	assert.Equal(t, "commissioning_started", matterResp["status"])
}

