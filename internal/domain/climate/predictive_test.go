package climate_test

import (
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/domain/climate"
)

func TestCalculateApparentTemp(t *testing.T) {
	// Standard indoor temperature with moderate humidity
	t1 := climate.CalculateApparentTemp(21.0, 50.0, 1013.25)
	if t1 < 19.0 || t1 > 23.0 {
		t.Errorf("expected apparent temp around 21, got %f", t1)
	}

	// High humidity should feel warmer
	tHigh := climate.CalculateApparentTemp(22.0, 80.0, 1013.25)
	tLow := climate.CalculateApparentTemp(22.0, 30.0, 1013.25)
	if tHigh <= tLow {
		t.Errorf("high humidity apparent temp (%f) should be higher than low humidity (%f)", tHigh, tLow)
	}
}

func TestEvaluateZone_WindowOpen(t *testing.T) {
	cfg := &climate.PredictiveZoneConfig{
		ID:                "zone_1",
		Name:              "Salone",
		TargetTemperature: 21.0,
		Orientation:       climate.OrientationSouth,
		EmitterType:       climate.EmitterRadiator,
	}

	eval := climate.EvaluateZone(cfg, 19.5, 45.0, 1012.0, 0.0, true, nil)
	if eval.CalculatedSetpoint != 5.0 {
		t.Errorf("expected 5.0 when window is open, got %f", eval.CalculatedSetpoint)
	}
	if eval.Mode != "frost_protection" {
		t.Errorf("expected mode frost_protection, got %s", eval.Mode)
	}
}

func TestEvaluateZone_ColdFrontDetection(t *testing.T) {
	cfg := &climate.PredictiveZoneConfig{
		ID:                   "zone_2",
		Name:                 "Camera Nord",
		TargetTemperature:    20.0,
		Orientation:          climate.OrientationNorth,
		EmitterType:          climate.EmitterRadiator,
		PressureBoostEnabled: true,
	}

	forecast := []climate.WeatherForecastPoint{
		{HoursAhead: 1.0, OutdoorTemp: 2.0, DirectRadiation: 0.0},
		{HoursAhead: 2.0, OutdoorTemp: 1.0, DirectRadiation: 0.0},
	}

	// Strong pressure drop of -2.5 hPa/h
	eval := climate.EvaluateZone(cfg, 19.8, 50.0, 1005.0, -2.5, false, forecast)
	if !eval.ColdFrontDetected {
		t.Errorf("expected cold front detected")
	}
	if eval.Mode != "cold_front_boost" {
		t.Errorf("expected mode cold_front_boost, got %s", eval.Mode)
	}
	if eval.CalculatedSetpoint <= 20.0 {
		t.Errorf("expected boosted setpoint > 20.0, got %f", eval.CalculatedSetpoint)
	}
}

func TestEvaluateZone_SolarCutoff(t *testing.T) {
	cfg := &climate.PredictiveZoneConfig{
		ID:                 "zone_3",
		Name:               "Veranda Sud",
		TargetTemperature:  20.0,
		Orientation:        climate.OrientationSouth,
		WindowAreaSqm:      8.0,
		EmitterType:        climate.EmitterFloorRadiant,
		SolarCutoffEnabled: true,
	}

	// High solar radiation coming in next 2-4 hours
	forecast := []climate.WeatherForecastPoint{
		{HoursAhead: 1.0, Time: time.Now().Add(1 * time.Hour), OutdoorTemp: 12.0, DirectRadiation: 650.0},
		{HoursAhead: 2.0, Time: time.Now().Add(2 * time.Hour), OutdoorTemp: 14.0, DirectRadiation: 750.0},
		{HoursAhead: 3.0, Time: time.Now().Add(3 * time.Hour), OutdoorTemp: 15.0, DirectRadiation: 800.0},
	}

	eval := climate.EvaluateZone(cfg, 20.2, 40.0, 1020.0, 0.2, false, forecast)
	if eval.Mode != "solar_cutoff" {
		t.Errorf("expected mode solar_cutoff, got %s", eval.Mode)
	}
	if eval.EstimatedSolarGainKw <= 0.5 {
		t.Errorf("expected estimated solar gain > 0.5 kW, got %f", eval.EstimatedSolarGainKw)
	}
	if eval.CalculatedSetpoint > 20.0 {
		t.Errorf("expected reduced setpoint <= 20.0 due to solar cutoff, got %f", eval.CalculatedSetpoint)
	}
}
