// Package climate provides the application service for multi-zone predictive climate control.
package climate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	appweather "github.com/domus-hub/domus/internal/application/weather"
	domclimate "github.com/domus-hub/domus/internal/domain/climate"
	"github.com/domus-hub/domus/internal/domain/entity"
	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/google/uuid"
)

// PressureSample holds a timestamped pressure measurement.
type PressureSample struct {
	Timestamp time.Time
	Pressure  float64
}

// PredictiveService coordinates predictive evaluations, sensor ingest, and TRV adjustments.
type PredictiveService struct {
	repo       domclimate.PredictiveRepository
	entityRepo entity.Repository
	weatherCli *appweather.Client
	bus        events.Bus
	logger     *slog.Logger

	mu              sync.RWMutex
	pressureHistory map[string][]PressureSample // key: entityKey -> samples
	lat             float64
	lon             float64
}

// ZoneDashboardStatus provides a real-time status summary for a predictive zone.
type ZoneDashboardStatus struct {
	Config           *domclimate.PredictiveZoneConfig `json:"config"`
	LatestEvaluation *domclimate.PredictiveEvaluation `json:"latest_evaluation,omitempty"`
}

// SystemPredictiveStatus aggregates status across all zones.
type SystemPredictiveStatus struct {
	TotalZones        int                   `json:"total_zones"`
	EnabledZones      int                   `json:"enabled_zones"`
	ColdFrontActive   bool                  `json:"cold_front_active"`
	SolarCutoffActive bool                  `json:"solar_cutoff_active"`
	Zones             []ZoneDashboardStatus `json:"zones"`
	EvaluatedAt       time.Time             `json:"evaluated_at"`
}

// NewPredictiveService constructs a new PredictiveService.
func NewPredictiveService(
	repo domclimate.PredictiveRepository,
	entityRepo entity.Repository,
	weatherCli *appweather.Client,
	bus events.Bus,
	logger *slog.Logger,
) *PredictiveService {
	if logger == nil {
		logger = slog.Default()
	}
	if weatherCli == nil {
		weatherCli = appweather.NewClient()
	}

	svc := &PredictiveService{
		repo:            repo,
		entityRepo:      entityRepo,
		weatherCli:      weatherCli,
		bus:             bus,
		logger:          logger,
		pressureHistory: make(map[string][]PressureSample),
		lat:             45.4642, // Default Milan, Italy coordinates
		lon:             9.1900,
	}

	// Subscribe to entity updates if event bus is available
	if bus != nil {
		bus.Subscribe(events.EntityStateChanged, svc.handleEntityStateChanged)
	}

	return svc
}

// SetCoordinates sets the geographic location for weather forecast queries.
func (s *PredictiveService) SetCoordinates(lat, lon float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lat = lat
	s.lon = lon
}

// SaveZone creates or updates a predictive zone configuration.
func (s *PredictiveService) SaveZone(ctx context.Context, cfg *domclimate.PredictiveZoneConfig) error {
	if cfg == nil {
		return domclimate.ErrInvalidZoneConfig
	}
	if strings.TrimSpace(cfg.Name) == "" || cfg.TRVEntityKey == "" || cfg.TempSensorKey == "" {
		return domclimate.ErrInvalidZoneConfig
	}
	if cfg.TargetTemperature <= 0 {
		cfg.TargetTemperature = 20.0
	}
	if cfg.WindowAreaSqm <= 0 {
		cfg.WindowAreaSqm = 2.0
	}
	if cfg.ThermalInertiaHours <= 0 {
		cfg.ThermalInertiaHours = 2.0
	}
	return s.repo.Save(ctx, cfg)
}

// GetZone retrieves a predictive zone by ID.
func (s *PredictiveService) GetZone(ctx context.Context, id string) (*domclimate.PredictiveZoneConfig, error) {
	return s.repo.GetByID(ctx, id)
}

// ListZones returns all predictive zones.
func (s *PredictiveService) ListZones(ctx context.Context) ([]*domclimate.PredictiveZoneConfig, error) {
	return s.repo.List(ctx)
}

// DeleteZone removes a predictive zone by ID.
func (s *PredictiveService) DeleteZone(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// RecordPressureSample stores a pressure reading and trims samples older than 6 hours.
func (s *PredictiveService) RecordPressureSample(sensorKey string, pressure float64, at time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := at.Add(-6 * time.Hour)
	current := s.pressureHistory[sensorKey]
	filtered := make([]PressureSample, 0, len(current)+1)
	for _, sample := range current {
		if sample.Timestamp.After(cutoff) {
			filtered = append(filtered, sample)
		}
	}
	filtered = append(filtered, PressureSample{Timestamp: at, Pressure: pressure})
	s.pressureHistory[sensorKey] = filtered
}

// CalculatePressureTrend computes the rate of change of barometric pressure (hPa/hr) over the past 1-3 hours.
func (s *PredictiveService) CalculatePressureTrend(sensorKey string) float64 {
	s.mu.RLock()
	samples, exists := s.pressureHistory[sensorKey]
	s.mu.RUnlock()

	if !exists || len(samples) < 2 {
		return 0.0
	}

	latest := samples[len(samples)-1]
	// Look back ~2 hours for a reliable trend
	targetTime := latest.Timestamp.Add(-2 * time.Hour)
	var oldest PressureSample
	found := false

	for i := len(samples) - 2; i >= 0; i-- {
		oldest = samples[i]
		if oldest.Timestamp.Before(targetTime) {
			found = true
			break
		}
	}

	if !found {
		oldest = samples[0]
	}

	hours := latest.Timestamp.Sub(oldest.Timestamp).Hours()
	if hours < 0.25 { // require at least 15 minutes of history
		return 0.0
	}

	trend := (latest.Pressure - oldest.Pressure) / hours
	return math.Round(trend*10) / 10
}

// EvaluateZone performs a predictive evaluation cycle on a single zone.
func (s *PredictiveService) EvaluateZone(ctx context.Context, zoneID string) (*domclimate.PredictiveEvaluation, error) {
	cfg, err := s.repo.GetByID(ctx, zoneID)
	if err != nil {
		return nil, err
	}

	// 1. Fetch sensor states
	currentTemp := 20.0
	if s.entityRepo != nil && cfg.TempSensorKey != "" {
		if e, err := s.entityRepo.GetByKey(ctx, cfg.TempSensorKey); err == nil {
			if v, ok := parseNumericState(e.State); ok {
				currentTemp = v
			}
		}
	}

	currentHumidity := 50.0
	if s.entityRepo != nil && cfg.HumiditySensorKey != "" {
		if e, err := s.entityRepo.GetByKey(ctx, cfg.HumiditySensorKey); err == nil {
			if v, ok := parseNumericState(e.State); ok {
				currentHumidity = v
			}
		}
	}

	currentPressure := 1013.25
	if s.entityRepo != nil && cfg.PressureSensorKey != "" {
		if e, err := s.entityRepo.GetByKey(ctx, cfg.PressureSensorKey); err == nil {
			if v, ok := parseNumericState(e.State); ok {
				currentPressure = v
				s.RecordPressureSample(cfg.PressureSensorKey, currentPressure, time.Now().UTC())
			}
		}
	}

	windowOpen := false
	if s.entityRepo != nil && cfg.WindowSensorKey != "" {
		if e, err := s.entityRepo.GetByKey(ctx, cfg.WindowSensorKey); err == nil {
			windowOpen = parseBooleanState(e.State)
		}
	}

	// 2. Barometric trend
	pressureTrend := s.CalculatePressureTrend(cfg.PressureSensorKey)

	// 3. Weather forecast
	s.mu.RLock()
	lat, lon := s.lat, s.lon
	s.mu.RUnlock()

	var forecastPoints []domclimate.WeatherForecastPoint
	if s.weatherCli != nil {
		report, err := s.weatherCli.FetchForecast(ctx, lat, lon)
		if err == nil && report != nil {
			now := time.Now().UTC()
			for _, h := range report.Hourly {
				diff := h.Time.Sub(now).Hours()
				if diff >= 0 && diff <= 12 {
					forecastPoints = append(forecastPoints, domclimate.WeatherForecastPoint{
						HoursAhead:      diff,
						Time:            h.Time,
						OutdoorTemp:     h.Temperature,
						DirectRadiation: h.DirectRadiation,
					})
				}
			}
		}
	}

	// 4. Execute domain evaluation
	eval := domclimate.EvaluateZone(
		cfg,
		currentTemp,
		currentHumidity,
		currentPressure,
		pressureTrend,
		windowOpen,
		forecastPoints,
	)

	// 5. Persist evaluation
	if err := s.repo.SaveEvaluation(ctx, &eval); err != nil {
		s.logger.WarnContext(ctx, "failed to save predictive climate evaluation", "error", err, "zone_id", zoneID)
	}

	// 6. If enabled, dispatch command to TRV
	if cfg.Enabled && cfg.TRVEntityKey != "" && s.bus != nil {
		s.dispatchTRVSetpoint(ctx, cfg.TRVEntityKey, eval.CalculatedSetpoint, eval.Reason)
	}

	return &eval, nil
}

// EvaluateAll runs predictive evaluation for all registered zones.
func (s *PredictiveService) EvaluateAll(ctx context.Context) ([]*domclimate.PredictiveEvaluation, error) {
	zones, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	var results []*domclimate.PredictiveEvaluation
	for _, z := range zones {
		if !z.Enabled {
			continue
		}
		eval, err := s.EvaluateZone(ctx, z.ID)
		if err != nil {
			s.logger.WarnContext(ctx, "error evaluating predictive zone", "zone_id", z.ID, "error", err)
			continue
		}
		results = append(results, eval)
	}
	return results, nil
}

// GetDashboardStatus returns all zones paired with their latest evaluation.
func (s *PredictiveService) GetDashboardStatus(ctx context.Context) (*SystemPredictiveStatus, error) {
	zones, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	latestEvals, err := s.repo.GetLatestEvaluations(ctx)
	if err != nil {
		latestEvals = []*domclimate.PredictiveEvaluation{}
	}

	evalMap := make(map[string]*domclimate.PredictiveEvaluation, len(latestEvals))
	for _, ev := range latestEvals {
		evalMap[ev.ZoneID] = ev
	}

	status := &SystemPredictiveStatus{
		TotalZones:   len(zones),
		EnabledZones: 0,
		Zones:        make([]ZoneDashboardStatus, 0, len(zones)),
		EvaluatedAt:  time.Now().UTC(),
	}

	for _, z := range zones {
		if z.Enabled {
			status.EnabledZones++
		}
		ev := evalMap[z.ID]
		if ev != nil {
			if ev.ColdFrontDetected {
				status.ColdFrontActive = true
			}
			if ev.Mode == "solar_cutoff" {
				status.SolarCutoffActive = true
			}
		}
		status.Zones = append(status.Zones, ZoneDashboardStatus{
			Config:           z,
			LatestEvaluation: ev,
		})
	}

	return status, nil
}

// GetZoneEvaluations returns historical evaluation logs for a zone.
func (s *PredictiveService) GetZoneEvaluations(ctx context.Context, zoneID string, limit int) ([]*domclimate.PredictiveEvaluation, error) {
	return s.repo.GetZoneEvaluations(ctx, zoneID, limit)
}

// GetZoneProfile retrieves the learned thermal profile for a zone, initializing a default profile if none exists.
func (s *PredictiveService) GetZoneProfile(ctx context.Context, zoneID string) (*domclimate.ThermalProfile, error) {
	profile, err := s.repo.GetProfile(ctx, zoneID)
	if err == nil {
		return profile, nil
	}
	if !errors.Is(err, domclimate.ErrThermalProfileNotFound) {
		return nil, err
	}

	// Profile not found: check if zone exists to derive emitter-specific baseline
	zone, err := s.repo.GetByID(ctx, zoneID)
	if err != nil {
		return nil, err
	}

	defProfile := domclimate.DefaultThermalProfile(zone.ID, zone.EmitterType)
	if saveErr := s.repo.SaveProfile(ctx, &defProfile); saveErr != nil {
		s.logger.WarnContext(ctx, "failed to persist initial default thermal profile", "zone_id", zoneID, "error", saveErr)
	}
	return &defProfile, nil
}

// TrainFromCycle incorporates a real-world heating or cooling observation into the zone's thermal profile using EMA.
func (s *PredictiveService) TrainFromCycle(ctx context.Context, obs domclimate.ThermalCycleObservation) (*domclimate.ThermalProfile, error) {
	if obs.ZoneID == "" {
		return nil, errors.New("cannot train without zone_id")
	}
	profile, err := s.GetZoneProfile(ctx, obs.ZoneID)
	if err != nil {
		return nil, err
	}

	profile.LearnFromCycle(obs)
	if err := s.repo.SaveProfile(ctx, profile); err != nil {
		return nil, fmt.Errorf("failed to save trained thermal profile: %w", err)
	}
	s.logger.InfoContext(ctx, "thermal profile updated from cycle",
		"zone_id", obs.ZoneID,
		"cycle_type", obs.CycleType,
		"duration_min", obs.DurationMinutes,
		"heat_rate", profile.HeatRatePerHr,
		"cool_rate", profile.CoolRatePerHr,
		"samples", profile.SamplesCount,
	)
	return profile, nil
}

// CalculateOptimalStart computes the preheating advance duration and start time to reach target temperature on schedule.
func (s *PredictiveService) CalculateOptimalStart(
	ctx context.Context,
	zoneID string,
	targetTime time.Time,
	targetTemp float64,
) (*domclimate.PreheatRecommendation, error) {
	cfg, err := s.repo.GetByID(ctx, zoneID)
	if err != nil {
		return nil, err
	}

	if targetTemp <= 0 {
		targetTemp = cfg.TargetTemperature
	}
	if targetTemp <= 0 {
		targetTemp = 20.0
	}
	if targetTime.IsZero() {
		targetTime = time.Now().UTC().Add(2 * time.Hour)
	}

	profile, err := s.GetZoneProfile(ctx, zoneID)
	if err != nil {
		return nil, err
	}

	// Current indoor temperature
	currentTemp := 19.0
	if s.entityRepo != nil && cfg.TempSensorKey != "" {
		if e, err := s.entityRepo.GetByKey(ctx, cfg.TempSensorKey); err == nil {
			if v, ok := parseNumericState(e.State); ok {
				currentTemp = v
			}
		}
	}

	// Outdoor temperature & solar radiation forecast
	outdoorTemp := 10.0
	solarRadiation := 0.0
	s.mu.RLock()
	lat, lon := s.lat, s.lon
	s.mu.RUnlock()

	if s.weatherCli != nil {
		if report, err := s.weatherCli.FetchForecast(ctx, lat, lon); err == nil && report != nil {
			for _, h := range report.Hourly {
				if h.Time.After(targetTime.Add(-1*time.Hour)) && h.Time.Before(targetTime.Add(2*time.Hour)) {
					outdoorTemp = h.Temperature
					solarRadiation = h.DirectRadiation
					break
				}
			}
		}
	}

	// Solar kW boost calculation
	factor := domclimate.OrientationFactor(cfg.Orientation, 1.0)
	solarKw := (solarRadiation * cfg.WindowAreaSqm * factor * 0.7) / 1000.0

	minutes, effectiveRate := profile.CalculatePreheatMinutes(currentTemp, targetTemp, outdoorTemp, solarKw)
	startTime := targetTime.Add(-time.Duration(minutes) * time.Minute)

	var reason string
	if minutes == 0 {
		reason = fmt.Sprintf("Nessun anticipo necessario: temperatura attuale (%.1f°C) già al target comfort (%.1f°C)", currentTemp, targetTemp)
	} else {
		reason = fmt.Sprintf("Avvio anticipato di %d min alle %s per raggiungere %.1f°C alle %s (velocità stimata %.2f°C/h con %.1f°C esterni)",
			minutes,
			startTime.Format("15:04"),
			targetTemp,
			targetTime.Format("15:04"),
			effectiveRate,
			outdoorTemp,
		)
	}

	return &domclimate.PreheatRecommendation{
		ZoneID:            zoneID,
		TargetTime:        targetTime,
		TargetTemp:        targetTemp,
		CurrentTemp:       currentTemp,
		OutdoorTemp:       outdoorTemp,
		PreheatMinutes:    minutes,
		StartTime:         startTime,
		EffectiveHeatRate: effectiveRate,
		Reason:            reason,
	}, nil
}

// CalculateCoastDown calculates how early heating can be turned off before departure or sleep to exploit thermal inertia.
func (s *PredictiveService) CalculateCoastDown(
	ctx context.Context,
	zoneID string,
	departureTime time.Time,
) (*domclimate.CoastDownRecommendation, error) {
	cfg, err := s.repo.GetByID(ctx, zoneID)
	if err != nil {
		return nil, err
	}

	if departureTime.IsZero() {
		departureTime = time.Now().UTC().Add(2 * time.Hour)
	}

	profile, err := s.GetZoneProfile(ctx, zoneID)
	if err != nil {
		return nil, err
	}

	targetTemp := cfg.TargetTemperature
	if targetTemp <= 0 {
		targetTemp = 20.0
	}

	currentTemp := targetTemp
	if s.entityRepo != nil && cfg.TempSensorKey != "" {
		if e, err := s.entityRepo.GetByKey(ctx, cfg.TempSensorKey); err == nil {
			if v, ok := parseNumericState(e.State); ok {
				currentTemp = v
			}
		}
	}

	outdoorTemp := 10.0
	s.mu.RLock()
	lat, lon := s.lat, s.lon
	s.mu.RUnlock()

	if s.weatherCli != nil {
		if report, err := s.weatherCli.FetchForecast(ctx, lat, lon); err == nil && report != nil {
			for _, h := range report.Hourly {
				if h.Time.After(departureTime.Add(-1*time.Hour)) && h.Time.Before(departureTime.Add(2*time.Hour)) {
					outdoorTemp = h.Temperature
					break
				}
			}
		}
	}

	minutes := profile.CalculateCoastDownMinutes(currentTemp, targetTemp, outdoorTemp)
	stopTime := departureTime.Add(-time.Duration(minutes) * time.Minute)
	comfortLimit := targetTemp - 0.5

	// Approximate energy saved: assuming average radiator/emitter electrical/thermal draw of 1.5 kW
	energySaved := (float64(minutes) / 60.0) * 1.5

	var reason string
	if minutes == 0 {
		reason = fmt.Sprintf("Nessun arresto anticipato: temperatura attuale (%.1f°C) sotto il limite comfort (%.1f°C)", currentTemp, comfortLimit)
	} else {
		reason = fmt.Sprintf("Spegnimento anticipato di %d min alle %s (uscita alle %s): inerzia termica sfruttabile con risparmio stimato di %.2f kWh",
			minutes,
			stopTime.Format("15:04"),
			departureTime.Format("15:04"),
			energySaved,
		)
	}

	return &domclimate.CoastDownRecommendation{
		ZoneID:           zoneID,
		DepartureTime:    departureTime,
		ComfortLimitTemp: comfortLimit,
		CoastDownMinutes: minutes,
		StopTime:         stopTime,
		EnergySavedKwh:   math.Round(energySaved*100) / 100,
		Reason:           reason,
	}, nil
}

func (s *PredictiveService) dispatchTRVSetpoint(ctx context.Context, trvKey string, targetTemp float64, reason string) {
	if s.bus == nil || trvKey == "" {
		return
	}
	s.logger.InfoContext(ctx, "dispatching predictive TRV setpoint",
		"trv_key", trvKey,
		"setpoint", targetTemp,
		"reason", reason,
	)
	_ = s.bus.Publish(ctx, events.Event{
		ID:         uuid.NewString(),
		Type:       events.CommandRequested,
		OccurredAt: time.Now().UTC(),
		Source:     "climate:predictive_engine",
		Payload: map[string]any{
			"entity_key": trvKey,
			"service":    "set_temperature",
			"params": map[string]any{
				"temperature": targetTemp,
				"reason":      reason,
			},
		},
	})
}

func (s *PredictiveService) handleEntityStateChanged(ctx context.Context, evt events.Event) error {
	payload, ok := evt.Payload.(map[string]any)
	if !ok {
		return nil
	}
	entityKey, _ := payload["entity_id"].(string)
	newState, _ := payload["new_state"].(string)
	if entityKey == "" || newState == "" {
		return nil
	}

	// If it's a pressure sensor, update sample history
	if strings.Contains(entityKey, "pressure") || strings.Contains(entityKey, "pressione") {
		if p, ok := parseNumericState(newState); ok {
			s.RecordPressureSample(entityKey, p, time.Now().UTC())
		}
	}

	return nil
}

func parseNumericState(raw string) (float64, bool) {
	trimmed := strings.TrimSpace(raw)
	if v, err := strconv.ParseFloat(trimmed, 64); err == nil {
		return v, true
	}
	// Try parsing JSON payload if present e.g. {"value": 21.5}
	var data map[string]any
	if err := json.Unmarshal([]byte(trimmed), &data); err == nil {
		if val, exists := data["value"]; exists {
			switch num := val.(type) {
			case float64:
				return num, true
			case int:
				return float64(num), true
			case string:
				if parsed, err := strconv.ParseFloat(num, 64); err == nil {
					return parsed, true
				}
			}
		}
	}
	return 0.0, false
}

func parseBooleanState(raw string) bool {
	lower := strings.ToLower(strings.TrimSpace(raw))
	return lower == "on" || lower == "open" || lower == "true" || lower == "1"
}
