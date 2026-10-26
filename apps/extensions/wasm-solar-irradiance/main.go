// Package main implements a local-first Clear-Sky solar irradiance prediction engine.
// Compile target: tinygo build -o plugin.wasm -target=wasi main.go
package main

import (
	"encoding/json"
	"math"
	"unsafe"
)

type SolarForecastRequest struct {
	LatitudeDeg  float64 `json:"latitude_deg"`
	LongitudeDeg float64 `json:"longitude_deg"`
	NominalKwp   float64 `json:"nominal_kwp"`
	DayOfYear    int     `json:"day_of_year"`
	HourOfDay    float64 `json:"hour_of_day"`
}

type SolarForecastResult struct {
	EstimatedPowerWatts float64 `json:"estimated_power_watts"`
	SolarElevationDeg   float64 `json:"solar_elevation_deg"`
	ClearSkyIndex       float64 `json:"clear_sky_index"`
	ExpectedEnergyKwh   float64 `json:"expected_day_energy_kwh"`
}

//export domus_handle_event
func domus_handle_event(ptr unsafe.Pointer, size int) uint64 {
	return 0
}

//export domus_forecast_pv_power
func domus_forecast_pv_power(ptr unsafe.Pointer, size int) uint64 {
	buf := unsafe.Slice((*byte)(ptr), size)
	var req SolarForecastRequest
	if err := json.Unmarshal(buf, &req); err != nil {
		return 0
	}

	// Approximate local astronomical elevation
	elevation := math.Sin((req.HourOfDay - 6.0) * math.Pi / 12.0)
	var estPower float64
	if elevation > 0 {
		estPower = req.NominalKwp * 1000.0 * elevation * 0.85
	}

	result := SolarForecastResult{
		EstimatedPowerWatts: math.Max(0, estPower),
		SolarElevationDeg:   math.Max(0, elevation*90.0),
		ClearSkyIndex:       0.92,
		ExpectedEnergyKwh:   req.NominalKwp * 4.8,
	}

	out, err := json.Marshal(result)
	if err != nil {
		return 0
	}

	outPtr := uintptr(unsafe.Pointer(&out[0]))
	return (uint64(outPtr) << 32) | uint64(len(out))
}

func main() {}
