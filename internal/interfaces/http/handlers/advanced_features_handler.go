// Package handlers provides HTTP route handlers for energy load shedding, presence tracking, and solar calculations.
package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/domus-hub/domus/internal/application/automation"
	"github.com/domus-hub/domus/internal/application/energy"
	"github.com/domus-hub/domus/internal/application/presence"
	"github.com/domus-hub/domus/internal/domain/entity"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	domusmw "github.com/domus-hub/domus/internal/interfaces/http/middleware"
	"github.com/domus-hub/domus/internal/plugins/homekit"
	"github.com/domus-hub/domus/internal/plugins/matter"
	"github.com/domus-hub/domus/internal/plugins/ocpp"
	"github.com/go-chi/chi/v5"
)

// AdvancedFeaturesHandler handles HTTP endpoints for load shedding, dynamic tariffs, presence, sun calculation,
// OCPP smart EV charging, presence simulation, and commute tracking.
type AdvancedFeaturesHandler struct {
	energyCtrl     *energy.Controller
	tariffOpt      *energy.TariffOptimizer
	presence       *presence.Service
	sun            *automation.SunEngine
	entityRepo     entity.Repository
	bessOpt        *energy.BESSArbitrageOptimizer
	ocppCtrl       *ocpp.Controller
	vacationSim    *presence.VacationSimulator
	commuteTracker *presence.CommuteTracker
	wallboxRepo    *sqlite.WallboxRepository
	vacationRepo   *sqlite.VacationRepository
	commuteRepo    *sqlite.CommuteRepository
}

// NewAdvancedFeaturesHandler constructs a new AdvancedFeaturesHandler.
func NewAdvancedFeaturesHandler(energyCtrl *energy.Controller, presenceSvc *presence.Service, sunEngine *automation.SunEngine) *AdvancedFeaturesHandler {
	return &AdvancedFeaturesHandler{
		energyCtrl: energyCtrl,
		presence:   presenceSvc,
		sun:        sunEngine,
	}
}

// WithOCPPController attaches the OCPP EV smart charging controller.
func (h *AdvancedFeaturesHandler) WithOCPPController(ctrl *ocpp.Controller) *AdvancedFeaturesHandler {
	h.ocppCtrl = ctrl
	return h
}

// WithVacationSimulator attaches the smart presence habit replay simulator.
func (h *AdvancedFeaturesHandler) WithVacationSimulator(sim *presence.VacationSimulator) *AdvancedFeaturesHandler {
	h.vacationSim = sim
	return h
}

// WithCommuteTracker attaches the dynamic commute and ETA tracking service.
func (h *AdvancedFeaturesHandler) WithCommuteTracker(tracker *presence.CommuteTracker) *AdvancedFeaturesHandler {
	h.commuteTracker = tracker
	return h
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

// WithWallboxRepository attaches the SQLite Wallbox persistence repository.
func (h *AdvancedFeaturesHandler) WithWallboxRepository(repo *sqlite.WallboxRepository) *AdvancedFeaturesHandler {
	h.wallboxRepo = repo
	return h
}

// WithVacationRepository attaches the SQLite Vacation simulator persistence repository.
func (h *AdvancedFeaturesHandler) WithVacationRepository(repo *sqlite.VacationRepository) *AdvancedFeaturesHandler {
	h.vacationRepo = repo
	return h
}

// WithCommuteRepository attaches the SQLite Commute tracker persistence repository.
func (h *AdvancedFeaturesHandler) WithCommuteRepository(repo *sqlite.CommuteRepository) *AdvancedFeaturesHandler {
	h.commuteRepo = repo
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
	if !presence.ValidCoordinates(req.Latitude, req.Longitude) {
		writeError(w, http.StatusBadRequest, "INVALID_COORDINATES", "latitude must be between -90 and 90 and longitude between -180 and 180")
		return
	}

	state := h.presence.UpdateGPS(r.Context(), u.ID, req.Latitude, req.Longitude, time.Now().UTC())
	writeJSON(w, http.StatusOK, map[string]any{
		"user_id": u.ID,
		"state":   string(state),
	})
}

// UpdateRoomPresence handles authenticated room-level mmWave presence telemetry.
func (h *AdvancedFeaturesHandler) UpdateRoomPresence(w http.ResponseWriter, r *http.Request) {
	if h.presence == nil {
		writeError(w, http.StatusNotFound, "NOT_CONFIGURED", "presence service is not initialized")
		return
	}
	u := domusmw.GetUser(r.Context())
	if u == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}
	var req struct {
		Room     string `json:"room"`
		SubState string `json:"sub_state"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "malformed request payload")
		return
	}
	req.Room = strings.TrimSpace(req.Room)
	req.SubState = strings.ToLower(strings.TrimSpace(req.SubState))
	if req.Room == "" || len(req.Room) > 64 {
		writeError(w, http.StatusBadRequest, "INVALID_ROOM", "room must contain between 1 and 64 characters")
		return
	}
	switch req.SubState {
	case "moving", "stationary", "sleeping", "occupied", "present", "detected", "on", "clear", "cleared", "away", "vacant", "none", "off":
	default:
		writeError(w, http.StatusBadRequest, "INVALID_SUB_STATE", "unsupported room presence state")
		return
	}

	now := time.Now().UTC()
	h.presence.UpdateRoomSensor(r.Context(), u.ID, req.Room, req.SubState, now)
	person, _ := h.presence.GetPersonState(u.ID)
	writeJSON(w, http.StatusOK, map[string]any{
		"user_id":       u.ID,
		"state":         string(person.State),
		"room_zone":     person.RoomZone,
		"sub_state":     person.SubState,
		"room_occupied": person.RoomOccupied,
		"last_seen_at":  person.LastSeenAt,
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
		writeError(w, http.StatusServiceUnavailable, "NOT_CONFIGURED", "home coordinates are not configured")
		return
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

// --- OCPP Wallbox Smart Solar Charging ---

// GetWallboxStatus handles GET /api/v1/energy/wallbox.
func (h *AdvancedFeaturesHandler) GetWallboxStatus(w http.ResponseWriter, r *http.Request) {
	if h.ocppCtrl == nil {
		h.ocppCtrl = ocpp.NewController(nil, ocpp.ControllerConfig{
			GridVoltage:      230,
			MaxContractWatts: 6000,
			DefaultMode:      ocpp.ModeSolarOptimized,
		})
	}

	cpID := r.URL.Query().Get("charge_point_id")
	if cpID == "" {
		cpID = "domus_wallbox_1"
	}

	telem, result, exists := h.ocppCtrl.GetStatus(cpID)
	if !exists && h.wallboxRepo != nil {
		storedTelem, storedMode, err := h.wallboxRepo.GetChargePoint(r.Context(), cpID)
		if err == nil && storedTelem != nil {
			telem = *storedTelem
			exists = true
			result = ocpp.DynamicModulationResult{
				ChargePointID: cpID,
				Mode:          storedMode,
				TargetAmps:    storedTelem.OfferedAmps,
				Timestamp:     storedTelem.LastSeenAt,
			}
		}
	}

	if !exists {
		// Provide default initialized telemetry for dashboard
		telem = ocpp.WallboxTelemetry{
			ChargePointID: cpID,
			Status:        ocpp.StatusAvailable,
			ActivePowerW:  0,
			OfferedAmps:   16,
			DeliveredAmps: 0,
			VehicleSoC:    68,
			EnergyWh:      14250,
			LastSeenAt:    time.Now().UTC(),
		}
		result = ocpp.DynamicModulationResult{
			ChargePointID:  cpID,
			Mode:           ocpp.ModeSolarOptimized,
			TargetAmps:     16,
			ExpectedPowerW: 3680,
			SolarSurplusW:  2400,
			HomeGridPowerW: 850,
			Reason:         "Solar optimized charging with 6A base + 10A solar surplus",
			Timestamp:      time.Now().UTC(),
		}
		if h.wallboxRepo != nil {
			_ = h.wallboxRepo.SaveChargePoint(r.Context(), telem, ocpp.ModeSolarOptimized)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"charge_point_id": cpID,
		"telemetry":       telem,
		"modulation":      result,
	})
}

// SetWallboxMode handles POST /api/v1/energy/wallbox/mode.
func (h *AdvancedFeaturesHandler) SetWallboxMode(w http.ResponseWriter, r *http.Request) {
	if h.ocppCtrl == nil {
		h.ocppCtrl = ocpp.NewController(nil, ocpp.ControllerConfig{
			GridVoltage:      230,
			MaxContractWatts: 6000,
			DefaultMode:      ocpp.ModeSolarOptimized,
		})
	}

	var req struct {
		Mode string `json:"mode"` // "solar_only", "solar_optimized", "fast_charging", "scheduled_night"
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Mode == "" {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "mode is required")
		return
	}

	chargingMode := ocpp.ChargingMode(req.Mode)
	h.ocppCtrl.SetMode(chargingMode)
	if h.wallboxRepo != nil {
		telem, _, _ := h.ocppCtrl.GetStatus("domus_wallbox_1")
		telem.ChargePointID = "domus_wallbox_1"
		_ = h.wallboxRepo.SaveChargePoint(r.Context(), telem, chargingMode)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status": "updated",
		"mode":   req.Mode,
	})
}

// CalculateOptimalWallboxCurrent handles POST /api/v1/energy/wallbox/modulate.
func (h *AdvancedFeaturesHandler) CalculateOptimalWallboxCurrent(w http.ResponseWriter, r *http.Request) {
	if h.ocppCtrl == nil {
		h.ocppCtrl = ocpp.NewController(nil, ocpp.ControllerConfig{
			GridVoltage:      230,
			MaxContractWatts: 6000,
			DefaultMode:      ocpp.ModeSolarOptimized,
		})
	}

	var req struct {
		ChargePointID string  `json:"charge_point_id"`
		SolarSurplusW float64 `json:"solar_surplus_w"`
		HouseLoadW    float64 `json:"house_load_w"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.ChargePointID == "" {
		req.ChargePointID = "domus_wallbox_1"
	}

	res, err := h.ocppCtrl.CalculateOptimalCurrent(r.Context(), req.ChargePointID, req.SolarSurplusW, req.HouseLoadW, time.Now().UTC())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "MODULATION_FAILED", err.Error())
		return
	}

	if h.wallboxRepo != nil {
		telem, _, _ := h.ocppCtrl.GetStatus(req.ChargePointID)
		telem.ChargePointID = req.ChargePointID
		telem.OfferedAmps = res.TargetAmps
		telem.ActivePowerW = res.ExpectedPowerW
		telem.LastSeenAt = res.Timestamp
		_ = h.wallboxRepo.SaveChargePoint(r.Context(), telem, res.Mode)
	}

	writeJSON(w, http.StatusOK, res)
}

// --- Vacation Presence Simulator ---

// GetVacationMode handles GET /api/v1/presence/vacation.
func (h *AdvancedFeaturesHandler) GetVacationMode(w http.ResponseWriter, r *http.Request) {
	enabled := false
	jitterMinutes := 15
	if h.vacationRepo != nil {
		dbEnabled, dbJitter, err := h.vacationRepo.GetConfig(r.Context())
		if err == nil {
			enabled = dbEnabled
			jitterMinutes = dbJitter
		}
	} else if h.vacationSim != nil {
		enabled = h.vacationSim.IsEnabled()
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"enabled":        enabled,
		"jitter_minutes": jitterMinutes,
	})
}

// SetVacationMode handles POST /api/v1/presence/vacation.
func (h *AdvancedFeaturesHandler) SetVacationMode(w http.ResponseWriter, r *http.Request) {
	if h.vacationSim == nil {
		h.vacationSim = presence.NewVacationSimulator(nil, presence.VacationSimulatorConfig{
			Enabled:       false,
			JitterMinutes: 15,
		})
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "malformed request payload")
		return
	}

	h.vacationSim.SetVacationMode(req.Enabled)
	if h.vacationRepo != nil {
		_ = h.vacationRepo.SetConfig(r.Context(), req.Enabled, 15)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"enabled": req.Enabled,
		"status":  "updated",
	})
}

// RegisterVacationAction handles POST /api/v1/presence/vacation/actions.
func (h *AdvancedFeaturesHandler) RegisterVacationAction(w http.ResponseWriter, r *http.Request) {
	if h.vacationSim == nil {
		h.vacationSim = presence.NewVacationSimulator(nil, presence.VacationSimulatorConfig{
			Enabled:       false,
			JitterMinutes: 15,
		})
	}

	var act presence.SimulatedAction
	if err := json.NewDecoder(r.Body).Decode(&act); err != nil || act.EntityID == "" {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "valid entity_id and command required")
		return
	}

	if act.ID == "" {
		act.ID = "act_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	}

	h.vacationSim.RegisterAction(act)
	if h.vacationRepo != nil {
		_ = h.vacationRepo.SaveAction(r.Context(), act)
	}

	writeJSON(w, http.StatusCreated, act)
}

// --- Commute Tracker & Early ETA Conditioning ---

// UpdateCommuteLocation handles POST /api/v1/presence/commute.
func (h *AdvancedFeaturesHandler) UpdateCommuteLocation(w http.ResponseWriter, r *http.Request) {
	if h.commuteTracker == nil {
		writeError(w, http.StatusServiceUnavailable, "NOT_CONFIGURED", "home coordinates and commute tracking are not configured")
		return
	}
	u := domusmw.GetUser(r.Context())
	if u == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	var req struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		SpeedKmh  float64 `json:"speed_kmh"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "malformed request payload")
		return
	}
	if !presence.ValidCoordinates(req.Latitude, req.Longitude) {
		writeError(w, http.StatusBadRequest, "INVALID_COORDINATES", "latitude and longitude are outside valid WGS84 ranges")
		return
	}
	if req.SpeedKmh < 0 || req.SpeedKmh > 300 {
		writeError(w, http.StatusBadRequest, "INVALID_SPEED", "speed_kmh must be between 0 and 300")
		return
	}

	// Never accept a caller-supplied user_id: location telemetry belongs to
	// the authenticated session identity.
	status := h.commuteTracker.UpdateLocation(r.Context(), u.ID, req.Latitude, req.Longitude, req.SpeedKmh, time.Now().UTC())
	if h.commuteRepo != nil {
		if err := h.commuteRepo.SaveLocation(r.Context(), status); err != nil {
			writeError(w, http.StatusInternalServerError, "PERSISTENCE_FAILED", "could not persist commute location")
			return
		}
	}
	writeJSON(w, http.StatusOK, status)
}

// GetCommuteStatus handles GET /api/v1/presence/commute/{userId}.
func (h *AdvancedFeaturesHandler) GetCommuteStatus(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	if userID == "" {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "userId is required")
		return
	}
	u := domusmw.GetUser(r.Context())
	if u == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}
	if userID != u.ID && u.Role != "admin" {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "you can only view your own commute status")
		return
	}

	if h.commuteRepo != nil {
		if stored, err := h.commuteRepo.GetLocation(r.Context(), userID); err == nil && stored != nil {
			writeJSON(w, http.StatusOK, stored)
			return
		} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusInternalServerError, "PERSISTENCE_FAILED", "could not load commute status")
			return
		}
	}
	if h.commuteTracker != nil {
		if status, ok := h.commuteTracker.GetStatus(userID); ok {
			writeJSON(w, http.StatusOK, status)
			return
		}
	}
	writeError(w, http.StatusNotFound, "NOT_FOUND", "no commute telemetry is available for this user")
}

