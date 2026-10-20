package anomaly

import (
	"fmt"
	"time"
)

// ApplianceType categorizes monitored electrical appliances.
type ApplianceType string

const (
	ApplianceRefrigerator ApplianceType = "refrigerator"
	ApplianceFreezer      ApplianceType = "freezer"
	AppliancePump         ApplianceType = "water_pump"
	ApplianceHVAC         ApplianceType = "hvac_heatpump"
	ApplianceGeneral      ApplianceType = "general"
)

// ApplianceHealthAlert details a detected malfunction, door ajar condition, or power anomaly.
type ApplianceHealthAlert struct {
	ApplianceID   string        `json:"appliance_id"`
	Type          ApplianceType `json:"type"`
	Severity      string        `json:"severity"` // "warning", "critical"
	Reason        string        `json:"reason"`
	ObservedValue float64       `json:"observed_value"`
	ExpectedRange string        `json:"expected_range"`
	DetectedAt    time.Time     `json:"detected_at"`
}

// ApplianceHealthMonitor monitors power telemetry and internal compartment temperatures.
type ApplianceHealthMonitor struct {
	applianceID  string
	kind         ApplianceType
	nominalWatts float64

	// Temperature state
	tempOverThresholdStart time.Time
	maxSafeTempC           float64
	durationAlarm          time.Duration

	// Continuous high-power state
	highPowerStart time.Time
}

// NewApplianceHealthMonitor creates a monitor for an appliance.
func NewApplianceHealthMonitor(id string, kind ApplianceType, nominalWatts float64) *ApplianceHealthMonitor {
	maxTemp := 8.0 // default for fridge
	if kind == ApplianceFreezer {
		maxTemp = -10.0
	}

	return &ApplianceHealthMonitor{
		applianceID:   id,
		kind:          kind,
		nominalWatts:  nominalWatts,
		maxSafeTempC:  maxTemp,
		durationAlarm: 40 * time.Minute,
	}
}

// IngestTemperature evaluates whether a fridge/freezer door has been left ajar or cooling failed.
func (m *ApplianceHealthMonitor) IngestTemperature(tempC float64, now time.Time) (bool, *ApplianceHealthAlert) {
	if tempC > m.maxSafeTempC {
		if m.tempOverThresholdStart.IsZero() {
			m.tempOverThresholdStart = now
			return false, nil
		}

		duration := now.Sub(m.tempOverThresholdStart)
		if duration >= m.durationAlarm {
			alert := &ApplianceHealthAlert{
				ApplianceID:   m.applianceID,
				Type:          m.kind,
				Severity:      "critical",
				Reason:        fmt.Sprintf("%s door left ajar or compressor failed! Temperature reached %.1f°C for %v", m.kind, tempC, duration.Round(time.Minute)),
				ObservedValue: tempC,
				ExpectedRange: fmt.Sprintf("< %.1f°C", m.maxSafeTempC),
				DetectedAt:    now,
			}
			return true, alert
		}
	} else {
		// Temperature returned to normal zone
		m.tempOverThresholdStart = time.Time{}
	}

	return false, nil
}

// IngestPowerWatts monitors electrical telemetry for pump cavitation, rotor lock, or continuous overload.
func (m *ApplianceHealthMonitor) IngestPowerWatts(watts float64, now time.Time) (bool, *ApplianceHealthAlert) {
	if m.nominalWatts <= 0 {
		return false, nil
	}

	// 1. Rotor lock or excessive friction (power > 135% of nominal)
	if watts > m.nominalWatts*1.35 {
		if m.highPowerStart.IsZero() {
			m.highPowerStart = now
			return false, nil
		}
		if now.Sub(m.highPowerStart) >= 15*time.Second {
			alert := &ApplianceHealthAlert{
				ApplianceID:   m.applianceID,
				Type:          m.kind,
				Severity:      "critical",
				Reason:        fmt.Sprintf("Appliance overload or locked rotor detected! Drawing %.0fW (nominal %.0fW)", watts, m.nominalWatts),
				ObservedValue: watts,
				ExpectedRange: fmt.Sprintf("<= %.0fW", m.nominalWatts*1.2),
				DetectedAt:    now,
			}
			return true, alert
		}
	} else {
		m.highPowerStart = time.Time{}
	}

	// 2. Pump dry-run / cavitation: pump running but drawing < 30% nominal
	if m.kind == AppliancePump && watts > 10.0 && watts < m.nominalWatts*0.30 {
		alert := &ApplianceHealthAlert{
			ApplianceID:   m.applianceID,
			Type:          m.kind,
			Severity:      "warning",
			Reason:        fmt.Sprintf("Pump dry-running or air cavitation detected! Under-load power %.0fW (nominal %.0fW)", watts, m.nominalWatts),
			ObservedValue: watts,
			ExpectedRange: fmt.Sprintf(">= %.0fW", m.nominalWatts*0.6),
			DetectedAt:    now,
		}
		return true, alert
	}

	return false, nil
}
