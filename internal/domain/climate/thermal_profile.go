// Package climate defines domain models for thermal learning, building inertia estimation, and optimal start/stop control.
package climate

import (
	"math"
	"time"
)

// ThermalProfile stores the learned physical parameters for a specific heating zone.
type ThermalProfile struct {
	ZoneID           string    `json:"zone_id"`
	HeatRatePerHr    float64   `json:"heat_rate_per_hr"`   // Learned heating speed in °C/hr (e.g. 1.2 °C/h)
	CoolRatePerHr    float64   `json:"cool_rate_per_hr"`   // Learned cooling speed in °C/hr per 10°C delta (e.g. 0.25 °C/h)
	OutdoorLossCoeff float64   `json:"outdoor_loss_coeff"` // Rate reduction per degree of cold outside
	SolarBoostCoeff  float64   `json:"solar_boost_coeff"`  // °C gain boost per kW solar radiation
	SamplesCount     int       `json:"samples_count"`      // Number of cycles learned from
	LastTrainedAt    time.Time `json:"last_trained_at"`
}

// ThermalCycleObservation represents observed temperature changes during an active heating or passive cooling cycle.
type ThermalCycleObservation struct {
	ZoneID          string    `json:"zone_id"`
	CycleType       string    `json:"cycle_type"` // "heating" or "cooling"
	StartTemp       float64   `json:"start_temp"`
	EndTemp         float64   `json:"end_temp"`
	AvgOutdoorTemp  float64   `json:"avg_outdoor_temp"`
	SolarRadiation  float64   `json:"solar_radiation"` // W/m^2
	DurationMinutes int       `json:"duration_minutes"`
	ObservedAt      time.Time `json:"observed_at"`
}

// PreheatRecommendation provides the exact pre-heating calculation for reaching comfort by a target time.
type PreheatRecommendation struct {
	ZoneID            string    `json:"zone_id"`
	TargetTime        time.Time `json:"target_time"`
	TargetTemp        float64   `json:"target_temp"`
	CurrentTemp       float64   `json:"current_temp"`
	OutdoorTemp       float64   `json:"outdoor_temp"`
	PreheatMinutes    int       `json:"preheat_minutes"`
	StartTime         time.Time `json:"start_time"`
	EffectiveHeatRate float64   `json:"effective_heat_rate"`
	Reason            string    `json:"reason"`
}

// CoastDownRecommendation calculates how early heating can be turned off before departure or sleep.
type CoastDownRecommendation struct {
	ZoneID           string    `json:"zone_id"`
	DepartureTime    time.Time `json:"departure_time"`
	ComfortLimitTemp float64   `json:"comfort_limit_temp"` // e.g. target - 0.5°C
	CoastDownMinutes int       `json:"coast_down_minutes"`
	StopTime         time.Time `json:"stop_time"`
	EnergySavedKwh   float64   `json:"energy_saved_kwh"`
	Reason           string    `json:"reason"`
}

// DefaultThermalProfile generates initial baseline parameters according to the emitter type.
func DefaultThermalProfile(zoneID string, emitter EmitterType) ThermalProfile {
	heatRate := 1.2
	coolRate := 0.25
	switch emitter {
	case EmitterFloorRadiant:
		heatRate = 0.6 // radiant floor has slower response
		coolRate = 0.15 // retains heat much longer
	case EmitterFancoil:
		heatRate = 2.0 // fast air heating
		coolRate = 0.35 // faster loss without massive thermal mass
	}

	return ThermalProfile{
		ZoneID:           zoneID,
		HeatRatePerHr:    heatRate,
		CoolRatePerHr:    coolRate,
		OutdoorLossCoeff: 0.02,
		SolarBoostCoeff:  0.30,
		SamplesCount:     0,
		LastTrainedAt:    time.Now().UTC(),
	}
}

// CalculatePreheatMinutes computes the required advance start duration in minutes.
func (p *ThermalProfile) CalculatePreheatMinutes(
	currentTemp, targetTemp, outdoorTemp, solarKw float64,
) (int, float64) {
	if currentTemp >= targetTemp {
		return 0, p.HeatRatePerHr
	}
	deltaNeeded := targetTemp - currentTemp

	outdoorDelta := math.Max(0.0, targetTemp-outdoorTemp)
	effectiveRate := p.HeatRatePerHr - (outdoorDelta * p.OutdoorLossCoeff) + (solarKw * p.SolarBoostCoeff)
	if effectiveRate < 0.25 {
		effectiveRate = 0.25 // safety minimum to prevent division by zero or negative rate
	}

	hoursNeeded := deltaNeeded / effectiveRate
	minutes := int(math.Ceil(hoursNeeded * 60))

	// Clamp to realistic bounds (0 to 360 min / 6 hours)
	if minutes > 360 {
		minutes = 360
	}
	return minutes, math.Round(effectiveRate*100) / 100
}

// CalculateCoastDownMinutes determines how many minutes before departure/sleep heating can be shut off.
func (p *ThermalProfile) CalculateCoastDownMinutes(
	currentTemp, targetTemp, outdoorTemp float64,
) int {
	if currentTemp < targetTemp-0.5 {
		return 0 // do not shut off early if already below target
	}

	// Maximum allowable temp drop during coast-down (0.5 °C)
	allowableDrop := 0.5
	outdoorDelta := math.Max(1.0, currentTemp-outdoorTemp)

	// Cooling rate per hour scaled by temperature delta relative to nominal 10°C
	effectiveCoolRate := p.CoolRatePerHr * (outdoorDelta / 10.0)
	if effectiveCoolRate < 0.1 {
		effectiveCoolRate = 0.1
	}

	hoursCoast := allowableDrop / effectiveCoolRate
	minutes := int(math.Floor(hoursCoast * 60))

	// Clamp to 15m min and 120m max
	if minutes < 15 {
		minutes = 15
	}
	if minutes > 120 {
		minutes = 120
	}
	return minutes
}

// LearnFromCycle performs recursive exponential smoothing (EMA) on thermal parameters based on observed real-world behavior.
func (p *ThermalProfile) LearnFromCycle(obs ThermalCycleObservation) {
	if obs.DurationMinutes < 15 {
		return // ignore micro-cycles shorter than 15 mins
	}

	durationHours := float64(obs.DurationMinutes) / 60.0

	// Adaptive learning rate: higher for initial samples, smoothly decaying to 0.15 for long-term stability
	alpha := 0.35
	if p.SamplesCount > 10 {
		alpha = 0.15
	}

	if obs.CycleType == "heating" {
		tempDelta := obs.EndTemp - obs.StartTemp
		if tempDelta > 0 {
			observedRate := tempDelta / durationHours
			p.HeatRatePerHr = math.Round(((1-alpha)*p.HeatRatePerHr+alpha*observedRate)*100) / 100
		}
	} else if obs.CycleType == "cooling" {
		tempDrop := obs.StartTemp - obs.EndTemp
		if tempDrop > 0 {
			observedLoss := tempDrop / durationHours
			outdoorDelta := math.Max(1.0, obs.StartTemp-obs.AvgOutdoorTemp)
			normalized10cLoss := observedLoss * (10.0 / outdoorDelta)
			p.CoolRatePerHr = math.Round(((1-alpha)*p.CoolRatePerHr+alpha*normalized10cLoss)*100) / 100
		}
	}

	p.SamplesCount++
	p.LastTrainedAt = time.Now().UTC()
}
