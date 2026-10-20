package anomaly

import (
	"fmt"
	"time"
)

// MicroLeakAlert details a detected continuous trickle flow.
type MicroLeakAlert struct {
	MeterEntityID      string        `json:"meter_entity_id"`
	ContinuousFlowLMin float64       `json:"continuous_flow_l_min"`
	Duration           time.Duration `json:"duration"`
	TotalLeakedLiters  float64       `json:"total_leaked_liters"`
	IsNightTime        bool          `json:"is_night_time"`
	Severity           string        `json:"severity"` // "warning", "critical"
	Reason             string        `json:"reason"`
	FirstObservedAt    time.Time     `json:"first_observed_at"`
	LastObservedAt     time.Time     `json:"last_observed_at"`
}

// WaterLeakDetector tracks water meter flow rates over time to catch micro-leaks early.
type WaterLeakDetector struct {
	MinFlowThresholdLMin   float64       // Default 0.05 L/min
	DaytimeDurationAlarm   time.Duration // Default 75 minutes of continuous flow
	NighttimeDurationAlarm time.Duration // Default 30 minutes of continuous flow (01:00 - 06:00)

	meterID       string
	flowStartTime time.Time
	lastSampleTime time.Time
	totalVolumeL   float64
	inLeakState   bool
}

// NewWaterLeakDetector creates a water flow anomaly detector.
func NewWaterLeakDetector(meterID string) *WaterLeakDetector {
	return &WaterLeakDetector{
		MinFlowThresholdLMin:   0.05,
		DaytimeDurationAlarm:   75 * time.Minute,
		NighttimeDurationAlarm: 30 * time.Minute,
		meterID:                meterID,
	}
}

// IngestFlowRate processes a new flow reading (liters per minute) from a water pulse/modbus meter.
func (d *WaterLeakDetector) IngestFlowRate(flowLMin float64, now time.Time) (bool, *MicroLeakAlert) {
	if flowLMin < d.MinFlowThresholdLMin {
		// Flow has stopped or dropped below trickle threshold -> reset tracker
		d.flowStartTime = time.Time{}
		d.totalVolumeL = 0
		d.inLeakState = false
		d.lastSampleTime = now
		return false, nil
	}

	// Active continuous flow
	if d.flowStartTime.IsZero() {
		d.flowStartTime = now
		d.lastSampleTime = now
		return false, nil
	}

	deltaMinutes := now.Sub(d.lastSampleTime).Minutes()
	if deltaMinutes > 0 {
		d.totalVolumeL += flowLMin * deltaMinutes
	}
	d.lastSampleTime = now

	continuousDuration := now.Sub(d.flowStartTime)

	// Check if reading occurred during night hours (01:00 - 06:00)
	hour := now.Hour()
	isNight := hour >= 1 && hour < 6

	alarmThreshold := d.DaytimeDurationAlarm
	if isNight {
		alarmThreshold = d.NighttimeDurationAlarm
	}

	if continuousDuration >= alarmThreshold {
		severity := "warning"
		if continuousDuration >= 2*alarmThreshold || d.totalVolumeL > 50.0 {
			severity = "critical"
		}

		alert := &MicroLeakAlert{
			MeterEntityID:      d.meterID,
			ContinuousFlowLMin: flowLMin,
			Duration:           continuousDuration,
			TotalLeakedLiters:  d.totalVolumeL,
			IsNightTime:        isNight,
			Severity:           severity,
			Reason: fmt.Sprintf("Continuous uninterrupted water flow (%.2f L/min) detected for %v (accumulated %.1f L)",
				flowLMin, continuousDuration.Round(time.Minute), d.totalVolumeL),
			FirstObservedAt: d.flowStartTime,
			LastObservedAt:  now,
		}
		d.inLeakState = true
		return true, alert
	}

	return false, nil
}
