package climate_test

import (
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/domain/climate"
	"github.com/stretchr/testify/assert"
)

func TestThermalProfile_DefaultProfiles(t *testing.T) {
	radiatorProfile := climate.DefaultThermalProfile("zone_rad", climate.EmitterRadiator)
	assert.Equal(t, "zone_rad", radiatorProfile.ZoneID)
	assert.Equal(t, 1.2, radiatorProfile.HeatRatePerHr)
	assert.Equal(t, 0.25, radiatorProfile.CoolRatePerHr)
	assert.Equal(t, 0, radiatorProfile.SamplesCount)

	floorProfile := climate.DefaultThermalProfile("zone_floor", climate.EmitterFloorRadiant)
	assert.Equal(t, 0.6, floorProfile.HeatRatePerHr)
	assert.Equal(t, 0.15, floorProfile.CoolRatePerHr)

	fancoilProfile := climate.DefaultThermalProfile("zone_fan", climate.EmitterFancoil)
	assert.Equal(t, 2.0, fancoilProfile.HeatRatePerHr)
	assert.Equal(t, 0.35, fancoilProfile.CoolRatePerHr)
}

func TestThermalProfile_CalculatePreheatMinutes(t *testing.T) {
	profile := climate.DefaultThermalProfile("zone_test", climate.EmitterRadiator)

	// Case 1: Room already at or above target temperature
	minutes, rate := profile.CalculatePreheatMinutes(21.0, 20.0, 5.0, 0.0)
	assert.Equal(t, 0, minutes)
	assert.Equal(t, 1.2, rate)

	// Case 2: Room needs 2°C heating (from 18 to 20°C) with 10°C outside, no sun
	// deltaNeeded = 2.0
	// outdoorDelta = 20 - 10 = 10
	// effectiveRate = 1.2 - (10 * 0.02) = 1.0 °C/h
	// hours = 2.0 / 1.0 = 2 hours -> 120 minutes
	minutes, rate = profile.CalculatePreheatMinutes(18.0, 20.0, 10.0, 0.0)
	assert.Equal(t, 120, minutes)
	assert.Equal(t, 1.0, rate)

	// Case 3: Same but with strong solar radiation (0.5 kW boost)
	// effectiveRate = 1.0 + (0.5 * 0.30) = 1.15 °C/h
	// hours = 2.0 / 1.15 = 1.739 hours -> 105 minutes
	minutesWithSun, rateWithSun := profile.CalculatePreheatMinutes(18.0, 20.0, 10.0, 0.5)
	assert.Less(t, minutesWithSun, minutes)
	assert.Greater(t, rateWithSun, rate)

	// Case 4: Extreme cold outdoor limit clamp
	minutesExtreme, _ := profile.CalculatePreheatMinutes(10.0, 22.0, -15.0, 0.0)
	assert.LessOrEqual(t, minutesExtreme, 360) // Clamped to 360m maximum
}

func TestThermalProfile_CalculateCoastDownMinutes(t *testing.T) {
	profile := climate.DefaultThermalProfile("zone_test", climate.EmitterFloorRadiant)

	// Case 1: Room is currently cooler than target - 0.5°C
	minutes := profile.CalculateCoastDownMinutes(19.0, 20.0, 5.0)
	assert.Equal(t, 0, minutes, "Should not coast down if room is cold")

	// Case 2: Room is warm (21°C, target 20.5°C), cold outside (0°C)
	// outdoorDelta = 21 - 0 = 21
	// effectiveCoolRate = 0.15 * (21 / 10) = 0.315 °C/h
	// allowableDrop = 0.5°C -> hours = 0.5 / 0.315 = 1.587 hours -> ~95 minutes
	minutes = profile.CalculateCoastDownMinutes(21.0, 20.5, 0.0)
	assert.GreaterOrEqual(t, minutes, 15)
	assert.LessOrEqual(t, minutes, 120)

	// Case 3: Very mild outside (18°C), very slow cooling -> clamped to 120 min max
	minutesMild := profile.CalculateCoastDownMinutes(21.0, 21.0, 20.0)
	assert.Equal(t, 120, minutesMild)
}

func TestThermalProfile_LearnFromCycle(t *testing.T) {
	profile := climate.DefaultThermalProfile("zone_test", climate.EmitterRadiator)
	initialHeatRate := profile.HeatRatePerHr
	initialCoolRate := profile.CoolRatePerHr

	// 1. Short cycle (< 15 mins) should be discarded
	profile.LearnFromCycle(climate.ThermalCycleObservation{
		ZoneID:          "zone_test",
		CycleType:       "heating",
		StartTemp:       19.0,
		EndTemp:         19.5,
		DurationMinutes: 10,
	})
	assert.Equal(t, initialHeatRate, profile.HeatRatePerHr)
	assert.Equal(t, 0, profile.SamplesCount)

	// 2. Active heating cycle: 60 mins, temp rose from 19.0 to 20.5 (+1.5 °C/h observed)
	// Initial rate: 1.2
	// Observed rate: 1.5
	// alpha: 0.35
	// expected: (0.65 * 1.2) + (0.35 * 1.5) = 0.78 + 0.525 = 1.305 -> rounded to 1.31
	profile.LearnFromCycle(climate.ThermalCycleObservation{
		ZoneID:          "zone_test",
		CycleType:       "heating",
		StartTemp:       19.0,
		EndTemp:         20.5,
		DurationMinutes: 60,
		ObservedAt:      time.Now().UTC(),
	})
	assert.Equal(t, 1.31, profile.HeatRatePerHr)
	assert.Equal(t, 1, profile.SamplesCount)

	// 3. Passive cooling cycle: 120 mins (2h), temp dropped from 20.5 to 20.0 (-0.25 °C/h)
	// StartTemp: 20.5, AvgOutdoorTemp: 10.5 -> delta: 10.0°C (normalized factor = 10/10 = 1.0)
	// Observed loss: 0.5 / 2 = 0.25 °C/h
	profile.LearnFromCycle(climate.ThermalCycleObservation{
		ZoneID:          "zone_test",
		CycleType:       "cooling",
		StartTemp:       20.5,
		EndTemp:         20.0,
		AvgOutdoorTemp:  10.5,
		DurationMinutes: 120,
		ObservedAt:      time.Now().UTC(),
	})
	assert.Equal(t, 2, profile.SamplesCount)
	assert.Equal(t, initialCoolRate, profile.CoolRatePerHr) // 0.25 matched nominal
}
