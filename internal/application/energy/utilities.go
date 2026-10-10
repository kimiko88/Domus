package energy

import (
	"sync"
	"time"
)

// UtilityType indicates the monitored resource.
type UtilityType string

const (
	UtilityWater UtilityType = "water" // m³ or Liters
	UtilityGas   UtilityType = "gas"   // Standard Cubic Meters (Smc)
)

// PulseMeter tracks reed switch or optical sensor impulse counts and converts to physical units.
type PulseMeter struct {
	mu           sync.RWMutex
	ID           string      `json:"id"`
	Type         UtilityType `json:"type"`
	Unit         string      `json:"unit"`          // "m3", "L", "Smc"
	PulsesPerUnit float64    `json:"pulses_per_unit"` // e.g. 1 pulse = 1 liter -> 1000 pulses/m3, or 100 pulses/Smc
	TotalPulses  int64       `json:"total_pulses"`
	TotalVolume  float64     `json:"total_volume"`
	LastPulseAt  time.Time   `json:"last_pulse_at"`
}

// NewPulseMeter constructs a new utility impulse meter.
func NewPulseMeter(id string, uType UtilityType, pulsesPerUnit float64, initialVolume float64) *PulseMeter {
	if pulsesPerUnit <= 0 {
		pulsesPerUnit = 1000 // default 1000 pulses per m³ (1 pulse = 1L)
	}
	unit := "m3"
	if uType == UtilityGas {
		unit = "Smc"
	}
	return &PulseMeter{
		ID:            id,
		Type:          uType,
		Unit:          unit,
		PulsesPerUnit: pulsesPerUnit,
		TotalVolume:   initialVolume,
		TotalPulses:   int64(initialVolume * pulsesPerUnit),
		LastPulseAt:   time.Now().UTC(),
	}
}

// RecordPulses records incoming impulses and returns the updated total volume.
func (m *PulseMeter) RecordPulses(count int64) float64 {
	m.mu.Lock()
	defer m.mu.Unlock()

	if count <= 0 {
		return m.TotalVolume
	}

	m.TotalPulses += count
	m.TotalVolume += float64(count) / m.PulsesPerUnit
	m.LastPulseAt = time.Now().UTC()
	return m.TotalVolume
}

// EstimateCost calculates the total cost for the consumed volume.
func (m *PulseMeter) EstimateCost(unitPrice float64) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.TotalVolume * unitPrice
}
