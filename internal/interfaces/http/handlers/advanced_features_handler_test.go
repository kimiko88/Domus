package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/domus-hub/domus/internal/application/automation"
	"github.com/domus-hub/domus/internal/application/energy"
	"github.com/domus-hub/domus/internal/application/presence"
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
