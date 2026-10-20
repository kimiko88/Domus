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
	"github.com/domus-hub/domus/internal/plugins/ocpp"
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

func TestAdvancedFeaturesHandler_BESSArbitrage(t *testing.T) {
	tariffOpt := energy.NewTariffOptimizer(energy.MarketPUN, nil)
	bessOpt := energy.NewBESSArbitrageOptimizer(energy.BESSConfig{
		CapacityKWh:            12.0,
		MaxChargePowerKW:       3.5,
		MaxDischargePowerKW:    3.5,
		NegativePriceArbitrage: true,
		PeakShavingEnabled:     true,
	}, tariffOpt, nil)

	h := NewAdvancedFeaturesHandler(nil, nil, nil).WithBESSOptimizer(bessOpt).WithTariffOptimizer(tariffOpt)

	// 1. Get Schedule
	req := httptest.NewRequest(http.MethodGet, "/api/v1/energy/bess/schedule", nil)
	rr := httptest.NewRecorder()
	h.GetBESSSchedule(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	var sched energy.BESSSchedule
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &sched))
	assert.Equal(t, 24, len(sched.HourlyPlan))

	// 2. Optimize BESS
	body, _ := json.Marshal(map[string]float64{"current_soc": 72.5})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/energy/bess/optimize", bytes.NewReader(body))
	rr = httptest.NewRecorder()
	h.OptimizeBESS(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	var optSched energy.BESSSchedule
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &optSched))
	assert.Equal(t, 72.5, optSched.CurrentSoC)

	// 3. Update BESS Config
	cfgBody, _ := json.Marshal(energy.BESSConfig{
		CapacityKWh:         15.0,
		MaxChargePowerKW:    4.0,
		MaxDischargePowerKW: 4.0,
		MinReserveSoC:       20.0,
		MaxTargetSoC:        90.0,
	})
	req = httptest.NewRequest(http.MethodPut, "/api/v1/energy/bess/config", bytes.NewReader(cfgBody))
	rr = httptest.NewRecorder()
	h.UpdateBESSConfig(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	var updateResp map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &updateResp))
	assert.Equal(t, "updated", updateResp["status"])
}

func TestAdvancedFeaturesHandler_WallboxOCPP(t *testing.T) {
	ocppCtrl := ocpp.NewController(nil, ocpp.ControllerConfig{
		GridVoltage:      230,
		MaxContractWatts: 6000,
		DefaultMode:      ocpp.ModeSolarOptimized,
	})

	h := NewAdvancedFeaturesHandler(nil, nil, nil).WithOCPPController(ocppCtrl)

	// 1. Get Status
	req := httptest.NewRequest(http.MethodGet, "/api/v1/energy/wallbox?charge_point_id=domus_wb_test", nil)
	rr := httptest.NewRecorder()
	h.GetWallboxStatus(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	var statusResp map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &statusResp))
	assert.Equal(t, "domus_wb_test", statusResp["charge_point_id"])

	// 2. Set Charging Mode
	modeBody, _ := json.Marshal(map[string]string{"mode": "solar_only"})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/energy/wallbox/mode", bytes.NewReader(modeBody))
	rr = httptest.NewRecorder()
	h.SetWallboxMode(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// 3. Modulate Current dynamically
	modBody, _ := json.Marshal(map[string]any{
		"charge_point_id": "domus_wb_test",
		"solar_surplus_w": 2800.0,
		"house_load_w":    650.0,
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/energy/wallbox/modulate", bytes.NewReader(modBody))
	rr = httptest.NewRecorder()
	h.CalculateOptimalWallboxCurrent(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	var modResp ocpp.DynamicModulationResult
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &modResp))
	assert.Equal(t, ocpp.ModeSolarOnly, modResp.Mode)
	assert.Greater(t, modResp.TargetAmps, 0.0)
}

func TestAdvancedFeaturesHandler_VacationPresenceSimulator(t *testing.T) {
	vacationSim := presence.NewVacationSimulator(nil, presence.VacationSimulatorConfig{
		Enabled:       false,
		JitterMinutes: 15,
	})

	h := NewAdvancedFeaturesHandler(nil, nil, nil).WithVacationSimulator(vacationSim)

	// 1. Get Vacation Mode -> disabled
	req := httptest.NewRequest(http.MethodGet, "/api/v1/presence/vacation", nil)
	rr := httptest.NewRecorder()
	h.GetVacationMode(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	var vacResp map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &vacResp))
	assert.Equal(t, false, vacResp["enabled"])

	// 2. Enable Vacation Mode -> enabled
	enableBody, _ := json.Marshal(map[string]bool{"enabled": true})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/presence/vacation", bytes.NewReader(enableBody))
	rr = httptest.NewRecorder()
	h.SetVacationMode(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// 3. Register Simulated Action
	actBody, _ := json.Marshal(presence.SimulatedAction{
		EntityID:     "light.living_room",
		Command:      "turn_on",
		TimeOfDaySec: 73800, // 20:30
		JitterMin:    15,
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/presence/vacation/actions", bytes.NewReader(actBody))
	rr = httptest.NewRecorder()
	h.RegisterVacationAction(rr, req)
	assert.Equal(t, http.StatusCreated, rr.Code)
}

func TestAdvancedFeaturesHandler_CommuteTracker(t *testing.T) {
	tracker := presence.NewCommuteTracker(nil, presence.CommuteConfig{
		HomeLatitude:  41.9028,
		HomeLongitude: 12.4964,
	})

	h := NewAdvancedFeaturesHandler(nil, nil, nil).WithCommuteTracker(tracker)

	// Update Commute Location
	locBody, _ := json.Marshal(map[string]any{
		"user_id":   "user_alice",
		"latitude":  41.9250,
		"longitude": 12.5150,
		"speed_kmh": 50.0,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/presence/commute", bytes.NewReader(locBody))
	rr := httptest.NewRecorder()
	h.UpdateCommuteLocation(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	var commuteResp presence.CommuteStatus
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &commuteResp))
	assert.Equal(t, "user_alice", commuteResp.UserID)
	assert.Greater(t, commuteResp.DistanceMeters, 0.0)
	assert.Greater(t, commuteResp.MinutesToHome, 0.0)
}

