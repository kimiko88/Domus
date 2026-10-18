// Package climate defines models and algorithms for multi-zone predictive climate control.
package climate

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"
)

var (
	ErrPredictiveZoneNotFound = errors.New("predictive climate zone not found")
	ErrInvalidZoneConfig      = errors.New("invalid predictive climate zone configuration")
)

// RoomOrientation defines the cardinal direction the room primarily faces.
type RoomOrientation string

const (
	OrientationNorth RoomOrientation = "north"
	OrientationSouth RoomOrientation = "south"
	OrientationEast  RoomOrientation = "east"
	OrientationWest  RoomOrientation = "west"
)

// EmitterType describes the physical heating/cooling emission mechanism.
type EmitterType string

const (
	EmitterRadiator     EmitterType = "radiator"
	EmitterFloorRadiant EmitterType = "floor_radiant"
	EmitterFancoil      EmitterType = "fancoil"
)

// PredictiveZoneConfig contains configuration for an intelligent multi-zone room.
type PredictiveZoneConfig struct {
	ID                   string          `json:"id"`
	Name                 string          `json:"name"`
	AreaID               string          `json:"area_id,omitempty"`
	TRVEntityKey         string          `json:"trv_entity_key"`
	TempSensorKey        string          `json:"temp_sensor_key"`
	PressureSensorKey    string          `json:"pressure_sensor_key,omitempty"`
	HumiditySensorKey    string          `json:"humidity_sensor_key,omitempty"`
	WindowSensorKey      string          `json:"window_sensor_key,omitempty"`
	TargetTemperature    float64         `json:"target_temperature"`
	Orientation          RoomOrientation `json:"orientation"`
	WindowAreaSqm        float64         `json:"window_area_sqm"`
	EmitterType          EmitterType     `json:"emitter_type"`
	ThermalInertiaHours  float64         `json:"thermal_inertia_hours"`
	PreheatEnabled       bool            `json:"preheat_enabled"`
	SolarCutoffEnabled   bool            `json:"solar_cutoff_enabled"`
	PressureBoostEnabled bool            `json:"pressure_boost_enabled"`
	Enabled              bool            `json:"enabled"`
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`
}

// WeatherForecastPoint encapsulates an hourly forecast point for prediction.
type WeatherForecastPoint struct {
	HoursAhead      float64   `json:"hours_ahead"`
	Time            time.Time `json:"time"`
	OutdoorTemp     float64   `json:"outdoor_temp"`
	DirectRadiation float64   `json:"direct_radiation"` // W/m^2
	WindSpeedKmh    float64   `json:"wind_speed_kmh"`
}

// PredictiveEvaluation represents the computed result of an MPC evaluation cycle for a zone.
type PredictiveEvaluation struct {
	ID                      string    `json:"id"`
	ZoneID                  string    `json:"zone_id"`
	ZoneName                string    `json:"zone_name"`
	CurrentTemp             float64   `json:"current_temp"`
	TargetTemp              float64   `json:"target_temp"`
	CurrentPressure         float64   `json:"current_pressure"` // hPa
	PressureTrendHpaHr      float64   `json:"pressure_trend_hpa_hr"`
	Humidity                float64   `json:"humidity"` // %
	ApparentTemp            float64   `json:"apparent_temp"`
	OutdoorTempForecast     float64   `json:"outdoor_temp_forecast"`
	DirectRadiationForecast float64   `json:"direct_radiation_forecast"`
	EstimatedSolarGainKw    float64   `json:"estimated_solar_gain_kw"`
	ColdFrontDetected       bool      `json:"cold_front_detected"`
	WindowOpen              bool      `json:"window_open"`
	CalculatedSetpoint      float64   `json:"calculated_setpoint"`
	Mode                    string    `json:"mode"`
	Reason                  string    `json:"reason"`
	EvaluatedAt             time.Time `json:"evaluated_at"`
}

// PredictiveRepository defines persistence operations for predictive climate zones and evaluations.
type PredictiveRepository interface {
	Save(ctx context.Context, cfg *PredictiveZoneConfig) error
	GetByID(ctx context.Context, id string) (*PredictiveZoneConfig, error)
	List(ctx context.Context) ([]*PredictiveZoneConfig, error)
	Delete(ctx context.Context, id string) error
	SaveEvaluation(ctx context.Context, eval *PredictiveEvaluation) error
	GetLatestEvaluations(ctx context.Context) ([]*PredictiveEvaluation, error)
	GetZoneEvaluations(ctx context.Context, zoneID string, limit int) ([]*PredictiveEvaluation, error)
}

// CalculateApparentTemp computes the perceived indoor temperature taking humidity and pressure into account.
func CalculateApparentTemp(temp, humidity, pressure float64) float64 {
	if humidity <= 0 {
		return temp
	}
	if pressure <= 0 {
		pressure = 1013.25 // Standard sea level pressure
	}
	// Water vapor pressure in hPa using Magnus formula
	e := (humidity / 100.0) * 6.105 * math.Exp((17.27*temp)/(237.7+temp))
	// Indoor apparent temperature approximation with gentle indoor ventilation (0.15 m/s)
	// T_app = T + 0.33*e - 0.70*v - 4.00 (adjusted for indoor comfort)
	apparent := temp + (5.0/9.0)*(e-10.0)
	if apparent < temp-3.0 {
		apparent = temp - 3.0
	}
	if apparent > temp+4.0 {
		apparent = temp + 4.0
	}
	return math.Round(apparent*10) / 10
}

// OrientationFactor returns the solar exposure multiplier based on cardinal orientation and sun time.
func OrientationFactor(orient RoomOrientation, hoursAhead float64) float64 {
	switch orient {
	case OrientationSouth:
		return 0.75
	case OrientationEast:
		if hoursAhead <= 3.0 {
			return 0.65 // strong morning solar gain
		}
		return 0.30
	case OrientationWest:
		if hoursAhead > 2.0 {
			return 0.60 // strong afternoon solar gain
		}
		return 0.25
	case OrientationNorth:
		return 0.10
	default:
		return 0.40
	}
}

// EvaluateZone executes the predictive feedforward algorithm for a given zone configuration.
func EvaluateZone(
	cfg *PredictiveZoneConfig,
	currentTemp float64,
	humidity float64,
	currentPressure float64,
	pressureTrendHpaHr float64,
	windowOpen bool,
	forecast []WeatherForecastPoint,
) PredictiveEvaluation {
	now := time.Now().UTC()
	target := cfg.TargetTemperature
	if target <= 0 {
		target = 20.0
	}

	apparent := CalculateApparentTemp(currentTemp, humidity, currentPressure)

	eval := PredictiveEvaluation{
		ZoneID:             cfg.ID,
		ZoneName:           cfg.Name,
		CurrentTemp:        currentTemp,
		TargetTemp:         target,
		CurrentPressure:    currentPressure,
		PressureTrendHpaHr: pressureTrendHpaHr,
		Humidity:           humidity,
		ApparentTemp:       apparent,
		WindowOpen:         windowOpen,
		EvaluatedAt:        now,
	}

	// 1. Window open check: frost protection shutoff
	if windowOpen {
		eval.CalculatedSetpoint = 5.0
		eval.Mode = "frost_protection"
		eval.Reason = "Finestra aperta: riscaldamento spento per prevenire sprechi"
		return eval
	}

	// 2. Determine prediction horizon based on thermal inertia and emitter
	horizonHours := 2.0
	switch cfg.EmitterType {
	case EmitterFloorRadiant:
		horizonHours = 4.0
	case EmitterFancoil:
		horizonHours = 1.0
	}
	if cfg.ThermalInertiaHours > 0 {
		horizonHours = math.Max(1.0, math.Min(6.0, cfg.ThermalInertiaHours))
	}

	// 3. Aggregate forecast metrics over horizon
	solarGainSum := 0.0
	outdoorDeltaSum := 0.0
	radSum := 0.0
	count := 0.0

	for _, pt := range forecast {
		if pt.HoursAhead <= horizonHours {
			factor := OrientationFactor(cfg.Orientation, pt.HoursAhead)
			// Solar gain: Radiation * Window Area * Factor * SHGC (0.7 glass transmissivity) / 1000 kW
			gain := pt.DirectRadiation * cfg.WindowAreaSqm * factor * 0.7 / 1000.0
			solarGainSum += gain
			radSum += pt.DirectRadiation
			outdoorDeltaSum += (target - pt.OutdoorTemp)
			count++
		}
	}

	avgOutdoorTemp := target
	avgRadiation := 0.0
	avgSolarGain := 0.0
	if count > 0 {
		avgRadiation = radSum / count
		avgSolarGain = solarGainSum / count
		avgOutdoorTemp = target - (outdoorDeltaSum / count)
	}

	eval.OutdoorTempForecast = math.Round(avgOutdoorTemp*10) / 10
	eval.DirectRadiationForecast = math.Round(avgRadiation*10) / 10
	eval.EstimatedSolarGainKw = math.Round(avgSolarGain*100) / 100

	// 4. Barometric Nowcasting: Cold front detection
	coldFrontDetected := false
	coldFrontBoost := 0.0
	if cfg.PressureBoostEnabled && pressureTrendHpaHr < -1.2 {
		coldFrontDetected = true
		eval.ColdFrontDetected = true
		// Compensate with 0.3°C to 1.0°C boost depending on rate of drop
		coldFrontBoost = math.Min(1.0, math.Abs(pressureTrendHpaHr)*0.35)
	}

	// 5. Solar Gain Cutoff: prevent room overheating if upcoming sun provides free heat
	solarCutoff := 0.0
	if cfg.SolarCutoffEnabled && avgSolarGain > 0.25 {
		// Cut setpoint if current temp is already close to or above target
		if currentTemp >= target-1.0 {
			solarCutoff = math.Min(1.5, avgSolarGain*1.2)
		}
	}

	// 6. Pre-heating adjustment: if outdoor temperature is dropping sharply in next hours
	preheatBoost := 0.0
	if cfg.PreheatEnabled && avgOutdoorTemp < 5.0 && currentTemp < target {
		preheatBoost = math.Min(0.8, (5.0-avgOutdoorTemp)*0.1)
	}

	// 7. Compute effective target and final compensated setpoint
	effectiveTarget := target + coldFrontBoost + preheatBoost - solarCutoff

	// Calculate proportional compensation relative to current or apparent temperature
	controlTemp := currentTemp
	if humidity > 60.0 && apparent > currentTemp {
		// Feel warmer due to high humidity, use apparent temperature
		controlTemp = apparent
	}

	delta := effectiveTarget - controlTemp
	compensated := effectiveTarget + (delta * 1.2)

	// Clamp within safe and realistic residential bounds
	if compensated < 15.0 {
		compensated = 15.0
	}
	if compensated > 28.0 {
		compensated = 28.0
	}
	// Round to nearest 0.5 °C step
	finalSetpoint := math.Round(compensated*2.0) / 2.0
	eval.CalculatedSetpoint = finalSetpoint

	// 8. Determine dominant mode and explanation
	if solarCutoff > 0.3 {
		eval.Mode = "solar_cutoff"
		eval.Reason = "Taglio solare preventivo: forte irraggiamento previsto in arrivo (" +
			formatFloat(avgRadiation) + " W/m²)"
	} else if coldFrontDetected {
		eval.Mode = "cold_front_boost"
		eval.Reason = "Anticipo termico: rilevato crollo barometrico rapido (" +
			formatFloat(pressureTrendHpaHr) + " hPa/h, fronte freddo in avvicinamento)"
	} else if preheatBoost > 0.3 {
		eval.Mode = "preheat"
		eval.Reason = "Pre-riscaldamento attivo: temperatura esterna in picchiata nelle prossime ore"
	} else if delta <= -0.5 {
		eval.Mode = "eco"
		eval.Reason = "Temperatura a regime: modulazione a basso consumo"
	} else {
		eval.Mode = "comfort"
		eval.Reason = "Riscaldamento a regime di comfort standard"
	}

	return eval
}

func formatFloat(f float64) string {
	return fmt.Sprintf("%.1f", f)
}
