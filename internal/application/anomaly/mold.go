// Package anomaly implements predictive maintenance, statistical baseline telemetry analysis,
// micro-leak detection, appliance failure diagnostics, and dynamic dew-point mold risk calculations.
package anomaly

import (
	"math"
	"time"
)

// MoldRiskLevel categorizes the probability of fungal proliferation.
type MoldRiskLevel string

const (
	MoldRiskNone     MoldRiskLevel = "none"
	MoldRiskLow      MoldRiskLevel = "low"
	MoldRiskModerate MoldRiskLevel = "moderate"
	MoldRiskHigh     MoldRiskLevel = "high"
	MoldRiskCritical MoldRiskLevel = "critical"
)

// MoldAssessment provides physical dew point and mold risk diagnostics.
type MoldAssessment struct {
	AmbientTempC   float64       `json:"ambient_temp_c"`
	RelativeHumPct float64       `json:"relative_humidity_pct"`
	SurfaceTempC   float64       `json:"surface_temp_c"`
	DewPointC      float64       `json:"dew_point_c"`
	DeltaToDewC    float64       `json:"delta_to_dew_point_c"` // surfaceTemp - dewPoint
	SurfaceRHPct   float64       `json:"surface_rh_pct"`       // Estimated RH at the wall surface
	RiskLevel      MoldRiskLevel `json:"risk_level"`
	Warning        string        `json:"warning,omitempty"`
	AssessedAt     time.Time     `json:"assessed_at"`
}

// CalculateDewPoint computes the dew point temperature in Celsius using the Magnus-Tetens formula.
// Valid for ambient temperatures between -40°C and +60°C and RH 1% - 100%.
func CalculateDewPoint(tempC, rhPct float64) float64 {
	if rhPct <= 0 {
		rhPct = 0.01
	}
	if rhPct > 100 {
		rhPct = 100
	}

	const (
		a = 17.27
		b = 237.7
	)

	alpha := ((a * tempC) / (b + tempC)) + math.Log(rhPct/100.0)
	dewPoint := (b * alpha) / (a - alpha)
	return math.Round(dewPoint*100) / 100
}

// AssessMoldRisk calculates the condensation margin and fungal risk at a cold wall or window surface.
func AssessMoldRisk(ambientTempC, rhPct, surfaceTempC float64, now time.Time) MoldAssessment {
	dewPoint := CalculateDewPoint(ambientTempC, rhPct)
	delta := surfaceTempC - dewPoint

	// Estimate relative humidity directly at the cold surface boundary layer
	// Saturated vapor pressure (hPa)
	pSatAmbient := 6.112 * math.Exp((17.67*ambientTempC)/(ambientTempC+243.5))
	actualVaporPressure := (rhPct / 100.0) * pSatAmbient
	pSatSurface := 6.112 * math.Exp((17.67*surfaceTempC)/(surfaceTempC+243.5))

	surfaceRH := 100.0
	if pSatSurface > 0 {
		surfaceRH = math.Min(100.0, (actualVaporPressure/pSatSurface)*100.0)
	}

	risk := MoldRiskNone
	var warning string

	if delta <= 0 || surfaceRH >= 95.0 {
		risk = MoldRiskCritical
		warning = "Immediate active condensation detected on surface! Extreme risk of mold growth."
	} else if delta <= 2.0 || surfaceRH >= 80.0 {
		risk = MoldRiskHigh
		warning = "Surface temperature within 2°C of dew point. Spore germination likely if sustained >24h."
	} else if delta <= 3.5 || surfaceRH >= 70.0 {
		risk = MoldRiskModerate
		warning = "Elevated surface moisture. Recommend ventilation or lowering indoor humidity."
	} else if surfaceRH >= 60.0 {
		risk = MoldRiskLow
	}

	return MoldAssessment{
		AmbientTempC:   ambientTempC,
		RelativeHumPct: rhPct,
		SurfaceTempC:   surfaceTempC,
		DewPointC:      dewPoint,
		DeltaToDewC:    math.Round(delta*100) / 100,
		SurfaceRHPct:   math.Round(surfaceRH*10) / 10,
		RiskLevel:      risk,
		Warning:        warning,
		AssessedAt:     now,
	}
}
