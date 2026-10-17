package weather

import (
	"math"
	"time"
)

type HourlySolarProduction struct {
	Time           time.Time `json:"time"`
	RadiationWm2   float64   `json:"radiation_wm2"`
	EstimatedKwh   float64   `json:"estimated_kwh"`
	EstimatedWatts float64   `json:"estimated_watts"`
}

type SolarForecast struct {
	InstalledKwPeak  float64                 `json:"installed_kw_peak"`
	TiltDegrees      float64                 `json:"tilt_degrees"`
	EfficiencyRatio  float64                 `json:"efficiency_ratio"`
	TodayTotalKwh    float64                 `json:"today_total_kwh"`
	TomorrowTotalKwh float64                 `json:"tomorrow_total_kwh"`
	Hourly           []HourlySolarProduction `json:"hourly"`
}

// ComputeSolarForecast calculates photovoltaic output from radiation data, panel capacity (kWp), tilt angle, and PR.
func ComputeSolarForecast(report *WeatherReport, kwPeak, tiltDegrees, efficiency float64) *SolarForecast {
	if kwPeak <= 0 {
		kwPeak = 6.0 // Default 6 kWp system
	}
	if efficiency <= 0 {
		efficiency = 0.85 // Default 85% system performance ratio
	}
	if tiltDegrees <= 0 {
		tiltDegrees = 30.0 // Default 30 deg optimal tilt in Europe
	}

	// Geometric tilt factor (approximation for mid-latitudes ~35-45 deg)
	tiltFactor := 1.0 + 0.15*math.Sin(tiltDegrees*math.Pi/180.0)

	forecast := &SolarForecast{
		InstalledKwPeak: kwPeak,
		TiltDegrees:     tiltDegrees,
		EfficiencyRatio: efficiency,
		Hourly:          make([]HourlySolarProduction, 0, len(report.Hourly)),
	}

	now := time.Now().UTC()
	todayDay := now.Day()

	for _, h := range report.Hourly {
		// Effective incident radiation W/m^2 adjusted by tilt factor
		effectiveRad := h.DirectRadiation * tiltFactor

		// Estimated generation Watts = (effectiveRad / 1000 W/m^2 standard test conditions) * (kwPeak * 1000) * efficiency
		watts := (effectiveRad / 1000.0) * (kwPeak * 1000.0) * efficiency
		if watts < 0 {
			watts = 0
		}
		// Energy in kWh for this 1-hour interval
		kwh := watts / 1000.0

		forecast.Hourly = append(forecast.Hourly, HourlySolarProduction{
			Time:           h.Time,
			RadiationWm2:   h.DirectRadiation,
			EstimatedWatts: math.Round(watts*10) / 10,
			EstimatedKwh:   math.Round(kwh*100) / 100,
		})

		if h.Time.Day() == todayDay {
			forecast.TodayTotalKwh += kwh
		} else {
			forecast.TomorrowTotalKwh += kwh
		}
	}

	forecast.TodayTotalKwh = math.Round(forecast.TodayTotalKwh*10) / 10
	forecast.TomorrowTotalKwh = math.Round(forecast.TomorrowTotalKwh*10) / 10

	return forecast
}
