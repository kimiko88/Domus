// Package handlers provides HTTP route handlers for energy load shedding, presence tracking, and solar calculations.
package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/domus-hub/domus/internal/application/automation"
	"github.com/domus-hub/domus/internal/application/energy"
	"github.com/domus-hub/domus/internal/application/presence"
	"github.com/domus-hub/domus/internal/domain/entity"
	domusmw "github.com/domus-hub/domus/internal/interfaces/http/middleware"
	"github.com/domus-hub/domus/internal/plugins/homekit"
	"github.com/domus-hub/domus/internal/plugins/matter"
)

// AdvancedFeaturesHandler handles HTTP endpoints for load shedding, dynamic tariffs, presence, and sun calculation.
type AdvancedFeaturesHandler struct {
	energyCtrl *energy.Controller
	tariffOpt  *energy.TariffOptimizer
	presence   *presence.Service
	sun        *automation.SunEngine
	entityRepo entity.Repository
	bessOpt    *energy.BESSArbitrageOptimizer
}

// NewAdvancedFeaturesHandler constructs a new AdvancedFeaturesHandler.
func NewAdvancedFeaturesHandler(energyCtrl *energy.Controller, presenceSvc *presence.Service, sunEngine *automation.SunEngine) *AdvancedFeaturesHandler {
	return &AdvancedFeaturesHandler{
		energyCtrl: energyCtrl,
		presence:   presenceSvc,
		sun:        sunEngine,
	}
}

// WithBESSOptimizer attaches the battery energy storage optimizer.
func (h *AdvancedFeaturesHandler) WithBESSOptimizer(opt *energy.BESSArbitrageOptimizer) *AdvancedFeaturesHandler {
	h.bessOpt = opt
	return h
}

// WithTariffOptimizer attaches the dynamic energy tariff optimizer.
func (h *AdvancedFeaturesHandler) WithTariffOptimizer(opt *energy.TariffOptimizer) *AdvancedFeaturesHandler {
	h.tariffOpt = opt
	return h
}

// WithEntityRepository attaches the entity repository to calculate real power and energy flow.
func (h *AdvancedFeaturesHandler) WithEntityRepository(repo entity.Repository) *AdvancedFeaturesHandler {
	h.entityRepo = repo
	return h
}

// GetLoadSheddingStatus handles GET /api/v1/energy/load-shedding.
func (h *AdvancedFeaturesHandler) GetLoadSheddingStatus(w http.ResponseWriter, r *http.Request) {
	if h.energyCtrl == nil {
		writeError(w, http.StatusNotFound, "NOT_CONFIGURED", "load shedding controller is not initialized")
		return
	}
	writeJSON(w, http.StatusOK, h.energyCtrl.GetStatus())
}

// UpdateLoadShedding handles PUT /api/v1/energy/load-shedding.
func (h *AdvancedFeaturesHandler) UpdateLoadShedding(w http.ResponseWriter, r *http.Request) {
	if h.energyCtrl == nil {
		writeError(w, http.StatusNotFound, "NOT_CONFIGURED", "load shedding controller is not initialized")
		return
	}

	var cfg energy.LoadSheddingConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "malformed request payload")
		return
	}

	h.energyCtrl.UpdateConfig(cfg)
	writeJSON(w, http.StatusOK, h.energyCtrl.GetStatus())
}

type gpsPayload struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// UpdateGPS handles POST /api/v1/presence/gps.
func (h *AdvancedFeaturesHandler) UpdateGPS(w http.ResponseWriter, r *http.Request) {
	if h.presence == nil {
		writeError(w, http.StatusNotFound, "NOT_CONFIGURED", "presence service is not initialized")
		return
	}

	u := domusmw.GetUser(r.Context())
	if u == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	var req gpsPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "malformed request payload")
		return
	}

	state := h.presence.UpdateGPS(r.Context(), u.ID, req.Latitude, req.Longitude, time.Now().UTC())
	writeJSON(w, http.StatusOK, map[string]any{
		"user_id": u.ID,
		"state":   string(state),
	})
}

// GetPresenceStatus handles GET /api/v1/presence/status.
func (h *AdvancedFeaturesHandler) GetPresenceStatus(w http.ResponseWriter, r *http.Request) {
	if h.presence == nil {
		writeError(w, http.StatusNotFound, "NOT_CONFIGURED", "presence service is not initialized")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"anyone_home": h.presence.IsAnyoneHome(),
	})
}

// GetSunPosition handles GET /api/v1/system/sun.
func (h *AdvancedFeaturesHandler) GetSunPosition(w http.ResponseWriter, r *http.Request) {
	engine := h.sun
	if engine == nil {
		// Default to Rome coords if unconfigured
		engine = automation.NewSunEngine(41.9028, 12.4964)
	}

	targetTime := time.Now().UTC()
	if qTime := r.URL.Query().Get("time"); qTime != "" {
		if parsed, err := time.Parse(time.RFC3339, qTime); err == nil {
			targetTime = parsed
		}
	}

	pos := engine.Calculate(targetTime)
	writeJSON(w, http.StatusOK, pos)
}

// GetEnergyFlow handles GET /api/v1/energy/flow.
func (h *AdvancedFeaturesHandler) GetEnergyFlow(w http.ResponseWriter, r *http.Request) {
	var solar, grid, battery, soc float64

	if h.energyCtrl != nil {
		st := h.energyCtrl.GetStatus()
		if p, ok := st["current_power_watts"].(float64); ok {
			grid = p
		}
	}

	if h.entityRepo != nil {
		entities, err := h.entityRepo.List(r.Context(), entity.ListFilter{})
		if err == nil {
			for _, ent := range entities {
				k := strings.ToLower(ent.EntityKey)
				val, err := strconv.ParseFloat(ent.State, 64)
				if err != nil {
					continue
				}
				if strings.Contains(k, "solar") || strings.Contains(k, "pv_power") || strings.Contains(k, "photovoltaic") {
					solar += val
				} else if strings.Contains(k, "grid_power") || strings.Contains(k, "main_power") {
					grid = val
				} else if strings.Contains(k, "battery_power") {
					battery = val
				} else if strings.Contains(k, "battery_soc") || strings.Contains(k, "battery_level") {
					soc = val
				}
			}
		}
	}

	flow := energy.ComputeSankeyFlow(solar, grid, battery, soc)
	writeJSON(w, http.StatusOK, flow)
}

// GetTariffs handles GET /api/v1/energy/tariffs.
func (h *AdvancedFeaturesHandler) GetTariffs(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()
	cfg := energy.DefaultTariffConfig()
	slot, price := energy.GetCurrentElectricityPrice(now, cfg)

	writeJSON(w, http.StatusOK, map[string]any{
		"current_slot":  slot,
		"current_price": price,
		"currency":      cfg.Currency,
		"tariffs":       cfg,
		"timestamp":     now,
	})
}

// GetHomeKitPairing handles GET /api/v1/hardware/homekit.
func (h *AdvancedFeaturesHandler) GetHomeKitPairing(w http.ResponseWriter, r *http.Request) {
	passcode := "031-45-154"
	setupID := "DOMU"
	uri := homekit.GenerateSetupURI(passcode, setupID, 2)

	writeJSON(w, http.StatusOK, map[string]any{
		"bridge_name": "Domus Bridge",
		"setup_code":  passcode,
		"setup_id":    setupID,
		"setup_uri":   uri,
		"paired":      false,
	})
}

// CommissionMatterRequest defines body for Matter commissioning.
type CommissionMatterRequest struct {
	Code string `json:"code"` // QR string ("MT:...") or manual pin (11/21 digits)
}

// CommissionMatter handles POST /api/v1/hardware/matter/commission.
func (h *AdvancedFeaturesHandler) CommissionMatter(w http.ResponseWriter, r *http.Request) {
	var req CommissionMatterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "malformed request payload")
		return
	}

	if req.Code == "" {
		writeError(w, http.StatusBadRequest, "MISSING_CODE", "qr or manual pairing code is required")
		return
	}

	var payload *matter.CommissioningPayload
	var err error
	if strings.HasPrefix(req.Code, "MT:") {
		payload, err = matter.ParseQRCode(req.Code)
	} else {
		payload, err = matter.ParseManualCode(req.Code)
	}

	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_CODE", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":        "commissioning_started",
		"discriminator": payload.Discriminator,
		"passcode":      payload.SetupPasscode,
		"vendor_id":     payload.VendorID,
		"product_id":    payload.ProductID,
	})
}

// GetDynamicTariffs handles GET /api/v1/energy/dynamic-tariffs.
func (h *AdvancedFeaturesHandler) GetDynamicTariffs(w http.ResponseWriter, r *http.Request) {
	if h.tariffOpt == nil {
		h.tariffOpt = energy.NewTariffOptimizer(energy.MarketPUN, nil)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"schedule":       h.tariffOpt.GetSchedule(),
		"current_tariff": h.tariffOpt.GetCurrentHourTariff(),
		"flexible_loads": h.tariffOpt.GetFlexibleLoads(),
	})
}

// UpdateFlexibleLoad handles PUT /api/v1/energy/flexible-loads.
func (h *AdvancedFeaturesHandler) UpdateFlexibleLoad(w http.ResponseWriter, r *http.Request) {
	if h.tariffOpt == nil {
		h.tariffOpt = energy.NewTariffOptimizer(energy.MarketPUN, nil)
	}

	var rule energy.FlexibleLoadRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "malformed request payload")
		return
	}

	h.tariffOpt.SetFlexibleLoad(rule)
	writeJSON(w, http.StatusOK, map[string]any{
		"status":         "updated",
		"flexible_loads": h.tariffOpt.GetFlexibleLoads(),
	})
}

// EvaluateTariffs handles POST /api/v1/energy/evaluate-tariffs.
func (h *AdvancedFeaturesHandler) EvaluateTariffs(w http.ResponseWriter, r *http.Request) {
	if h.tariffOpt == nil {
		h.tariffOpt = energy.NewTariffOptimizer(energy.MarketPUN, nil)
	}

	decisions := h.tariffOpt.EvaluateCurrentHour(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{
		"status":    "evaluated",
		"decisions": decisions,
	})
}

// GetBESSSchedule handles GET /api/v1/energy/bess/schedule.
func (h *AdvancedFeaturesHandler) GetBESSSchedule(w http.ResponseWriter, r *http.Request) {
	if h.bessOpt == nil {
		h.bessOpt = energy.NewBESSArbitrageOptimizer(energy.BESSConfig{}, h.tariffOpt, nil)
	}
	writeJSON(w, http.StatusOK, h.bessOpt.GetSchedule())
}

// OptimizeBESS handles POST /api/v1/energy/bess/optimize.
func (h *AdvancedFeaturesHandler) OptimizeBESS(w http.ResponseWriter, r *http.Request) {
	if h.bessOpt == nil {
		h.bessOpt = energy.NewBESSArbitrageOptimizer(energy.BESSConfig{}, h.tariffOpt, nil)
	}

	var req struct {
		CurrentSoC float64 `json:"current_soc"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	soc := req.CurrentSoC
	if soc <= 0 {
		soc = 50.0
	}

	sched := h.bessOpt.ComputeSchedule(time.Now().UTC(), soc)
	writeJSON(w, http.StatusOK, sched)
}

// UpdateBESSConfig handles PUT /api/v1/energy/bess/config.
func (h *AdvancedFeaturesHandler) UpdateBESSConfig(w http.ResponseWriter, r *http.Request) {
	if h.bessOpt == nil {
		h.bessOpt = energy.NewBESSArbitrageOptimizer(energy.BESSConfig{}, h.tariffOpt, nil)
	}

	var cfg energy.BESSConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "malformed request payload")
		return
	}

	sched := h.bessOpt.UpdateConfig(cfg)
	writeJSON(w, http.StatusOK, map[string]any{
		"status":   "updated",
		"config":   h.bessOpt.GetConfig(),
		"schedule": sched,
	})
}


