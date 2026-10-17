package energy

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/google/uuid"
)

// InverterMode defines the operating command issued to the hybrid inverter/BESS.
type InverterMode string

const (
	InverterModeSelfConsumption InverterMode = "SELF_CONSUMPTION" // Normal solar self-consumption
	InverterModeForceChargeGrid InverterMode = "FORCE_CHARGE_GRID" // Force charging battery from grid (cheap/negative price)
	InverterModeForceDischarge  InverterMode = "FORCE_DISCHARGE"   // Force battery discharge (peak shaving during expensive hours)
	InverterModeHoldDischarge   InverterMode = "HOLD_DISCHARGE"    // Hold battery state of charge, do not discharge
)

// BESSConfig defines battery system parameters and operational constraints.
type BESSConfig struct {
	CapacityKWh           float64 `json:"capacity_kwh"`            // e.g. 10.0 kWh
	MaxChargePowerKW      float64 `json:"max_charge_power_kw"`     // e.g. 3.0 kW
	MaxDischargePowerKW   float64 `json:"max_discharge_power_kw"`  // e.g. 3.0 kW
	MinReserveSoC         float64 `json:"min_reserve_soc"`         // e.g. 10.0% minimum backup buffer
	MaxTargetSoC          float64 `json:"max_target_soc"`          // e.g. 100.0%
	RoundTripEfficiency   float64 `json:"round_trip_efficiency"`   // e.g. 0.92 (92%)
	NegativePriceArbitrage bool   `json:"negative_price_arbitrage"`// Automatically charge when price <= 0
	PeakShavingEnabled    bool    `json:"peak_shaving_enabled"`    // Discharge to shave expensive peaks
	TargetBatteryEntity   string  `json:"target_battery_entity"`   // e.g. "sensor.home_battery_soc"
	InverterControlEntity string  `json:"inverter_control_entity"` // e.g. "select.inverter_work_mode"
}

// BESSHourlySlot represents the optimized arbitrage decision for an hour.
type BESSHourlySlot struct {
	Hour               int          `json:"hour"`                 // 0 - 23
	PricePerKWh        float64      `json:"price_per_kwh"`        // Grid tariff EUR/kWh
	ForecastSolarKWh   float64      `json:"forecast_solar_kwh"`   // Expected solar yield
	ExpectedHomeLoadKW float64      `json:"expected_home_load_kw"`// Expected house consumption
	Mode               InverterMode `json:"mode"`                 // Planned inverter mode
	TargetPowerKW      float64      `json:"target_power_kw"`      // +Charge, -Discharge
	EstimatedSoCEnd    float64      `json:"estimated_soc_end"`    // Estimated SoC % at end of hour
	EstimatedSavingsEUR float64     `json:"estimated_savings_eur"`// Financial gain from arbitrage
}

// BESSSchedule contains the 24-hour predictive plan for the battery.
type BESSSchedule struct {
	Date                string           `json:"date"`
	CurrentSoC          float64          `json:"current_soc"`
	CurrentMode         InverterMode     `json:"current_mode"`
	CurrentPowerKW      float64          `json:"current_power_kw"`
	HourlyPlan          []BESSHourlySlot `json:"hourly_plan"`
	TotalArbitrageEUR   float64          `json:"total_arbitrage_eur"`
	GridChargedKWh      float64          `json:"grid_charged_kwh"`
	PeakDischargedKWh   float64          `json:"peak_discharged_kwh"`
	GeneratedAt         time.Time        `json:"generated_at"`
}

// BESSArbitrageOptimizer orchestrates predictive battery charging and peak shaving.
type BESSArbitrageOptimizer struct {
	mu         sync.RWMutex
	config     BESSConfig
	schedule   *BESSSchedule
	tariffOpt  *TariffOptimizer
	bus        events.Bus
	currentSoC float64
}

// NewBESSArbitrageOptimizer constructs a new battery storage optimizer.
func NewBESSArbitrageOptimizer(cfg BESSConfig, tariffOpt *TariffOptimizer, bus events.Bus) *BESSArbitrageOptimizer {
	if cfg.CapacityKWh <= 0 {
		cfg.CapacityKWh = 10.0
	}
	if cfg.MaxChargePowerKW <= 0 {
		cfg.MaxChargePowerKW = 3.3
	}
	if cfg.MaxDischargePowerKW <= 0 {
		cfg.MaxDischargePowerKW = 3.3
	}
	if cfg.MinReserveSoC <= 0 {
		cfg.MinReserveSoC = 10.0
	}
	if cfg.MaxTargetSoC <= 0 {
		cfg.MaxTargetSoC = 100.0
	}
	if cfg.RoundTripEfficiency <= 0 {
		cfg.RoundTripEfficiency = 0.92
	}
	if cfg.TargetBatteryEntity == "" {
		cfg.TargetBatteryEntity = "sensor.battery_soc"
	}
	if cfg.InverterControlEntity == "" {
		cfg.InverterControlEntity = "select.inverter_mode"
	}

	opt := &BESSArbitrageOptimizer{
		config:     cfg,
		tariffOpt:  tariffOpt,
		bus:        bus,
		currentSoC: 50.0, // Default baseline if not yet read from hardware
	}

	opt.ComputeSchedule(time.Now().UTC(), 50.0)
	return opt
}

// UpdateConfig updates operational settings and recalculates plan.
func (b *BESSArbitrageOptimizer) UpdateConfig(cfg BESSConfig) *BESSSchedule {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.config = cfg
	return b.computeScheduleLocked(time.Now().UTC(), b.currentSoC)
}

// GetConfig returns the current BESS configuration.
func (b *BESSArbitrageOptimizer) GetConfig() BESSConfig {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.config
}

// UpdateCurrentSoC updates live battery state of charge from sensors.
func (b *BESSArbitrageOptimizer) UpdateCurrentSoC(soc float64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if soc < 0 {
		soc = 0
	} else if soc > 100 {
		soc = 100
	}
	b.currentSoC = soc
	if b.schedule != nil {
		b.schedule.CurrentSoC = soc
	}
}

// GetSchedule returns the active 24h arbitrage schedule.
func (b *BESSArbitrageOptimizer) GetSchedule() *BESSSchedule {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.schedule
}

// ComputeSchedule calculates the optimal 24h battery charge/discharge schedule.
func (b *BESSArbitrageOptimizer) ComputeSchedule(now time.Time, currentSoC float64) *BESSSchedule {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.computeScheduleLocked(now, currentSoC)
}

func (b *BESSArbitrageOptimizer) computeScheduleLocked(now time.Time, currentSoC float64) *BESSSchedule {
	if b.tariffOpt == nil {
		b.tariffOpt = NewTariffOptimizer(MarketPUN, b.bus)
	}

	dailyTariffs := b.tariffOpt.GetSchedule()
	if dailyTariffs == nil || len(dailyTariffs.HourlyPrices) != 24 {
		dailyTariffs = b.tariffOpt.GenerateDailySchedule(now)
	}

	hourlyPlan := make([]BESSHourlySlot, 24)
	var totalArbitrageEUR, totalGridCharged, totalPeakDischarged float64
	simSoC := currentSoC

	// Baseline average price
	avgPrice := dailyTariffs.AveragePrice
	if avgPrice <= 0 {
		avgPrice = 0.15
	}

	// Typical solar bell curve baseline if solar forecast not present
	typicalSolarProfile := []float64{
		0, 0, 0, 0, 0, 0,
		0.1, 0.4, 0.9, 1.5, 2.2, 2.8,
		3.0, 2.7, 2.1, 1.3, 0.6, 0.2,
		0, 0, 0, 0, 0, 0,
	}

	// Average household load baseline (kW)
	typicalHomeLoad := []float64{
		0.3, 0.25, 0.25, 0.25, 0.3, 0.4,
		0.7, 1.1, 0.8, 0.6, 0.7, 0.9,
		1.0, 0.7, 0.6, 0.7, 0.9, 1.4,
		1.8, 2.0, 1.6, 1.1, 0.7, 0.4,
	}

	for h := 0; h < 24; h++ {
		t := dailyTariffs.HourlyPrices[h]
		solar := typicalSolarProfile[h]
		load := typicalHomeLoad[h]

		mode := InverterModeSelfConsumption
		targetPower := 0.0
		savings := 0.0

		// Strategy 1: Negative Price or Ultra-cheap quartile -> Force Charge from Grid
		if (t.IsNegative && b.config.NegativePriceArbitrage) || (t.IsCheapest && t.PricePerKWh < avgPrice*0.65) {
			if simSoC < b.config.MaxTargetSoC {
				mode = InverterModeForceChargeGrid
				energyNeededKWh := ((b.config.MaxTargetSoC - simSoC) / 100.0) * b.config.CapacityKWh
				powerToCharge := math.Min(b.config.MaxChargePowerKW, energyNeededKWh)
				targetPower = powerToCharge

				// Update simulated SoC
				gainedSoC := (powerToCharge * b.config.RoundTripEfficiency / b.config.CapacityKWh) * 100.0
				simSoC = math.Min(b.config.MaxTargetSoC, simSoC+gainedSoC)
				totalGridCharged += powerToCharge

				// If negative price, we get paid to charge; else we save against average price
				if t.PricePerKWh < 0 {
					savings = math.Abs(t.PricePerKWh) * powerToCharge
				} else {
					savings = (avgPrice - t.PricePerKWh) * powerToCharge
				}
			}
		} else if t.IsExpensive && b.config.PeakShavingEnabled && simSoC > b.config.MinReserveSoC {
			// Strategy 2: Peak expensive hours -> Force Discharge to cover household and peak shave
			mode = InverterModeForceDischarge
			availableEnergyKWh := ((simSoC - b.config.MinReserveSoC) / 100.0) * b.config.CapacityKWh
			dischargePower := math.Min(b.config.MaxDischargePowerKW, math.Min(load, availableEnergyKWh))
			targetPower = -dischargePower

			lostSoC := (dischargePower / b.config.CapacityKWh) * 100.0
			simSoC = math.Max(b.config.MinReserveSoC, simSoC-lostSoC)
			totalPeakDischarged += dischargePower

			// Savings compared to buying from grid at peak price
			savings = dischargePower * (t.PricePerKWh - (avgPrice * (1.0 / b.config.RoundTripEfficiency)))
			if savings < 0 {
				savings = dischargePower * (t.PricePerKWh * 0.5)
			}
		} else if solar > load && simSoC < b.config.MaxTargetSoC {
			// Strategy 3: Daytime solar surplus self-consumption
			surplus := solar - load
			chargePower := math.Min(b.config.MaxChargePowerKW, surplus)
			energyKWh := math.Min(chargePower, ((b.config.MaxTargetSoC-simSoC)/100.0)*b.config.CapacityKWh)
			gainedSoC := (energyKWh * b.config.RoundTripEfficiency / b.config.CapacityKWh) * 100.0
			simSoC = math.Min(b.config.MaxTargetSoC, simSoC+gainedSoC)
			targetPower = energyKWh
			mode = InverterModeSelfConsumption
			savings = energyKWh * t.PricePerKWh
		}

		if savings > 0 {
			totalArbitrageEUR += savings
		}

		hourlyPlan[h] = BESSHourlySlot{
			Hour:               h,
			PricePerKWh:        t.PricePerKWh,
			ForecastSolarKWh:   solar,
			ExpectedHomeLoadKW: load,
			Mode:               mode,
			TargetPowerKW:      targetPower,
			EstimatedSoCEnd:    math.Round(simSoC*10) / 10,
			EstimatedSavingsEUR: math.Round(savings*100) / 100,
		}
	}

	currentHour := now.Hour()
	activeSlot := hourlyPlan[currentHour]

	sched := &BESSSchedule{
		Date:              dailyTariffs.Date,
		CurrentSoC:        currentSoC,
		CurrentMode:       activeSlot.Mode,
		CurrentPowerKW:    activeSlot.TargetPowerKW,
		HourlyPlan:        hourlyPlan,
		TotalArbitrageEUR: math.Round(totalArbitrageEUR*100) / 100,
		GridChargedKWh:    math.Round(totalGridCharged*10) / 10,
		PeakDischargedKWh: math.Round(totalPeakDischarged*10) / 10,
		GeneratedAt:       now,
	}

	b.schedule = sched

	// Dispatch domain event if bus is present
	if b.bus != nil {
		_ = b.bus.Publish(context.Background(), events.Event{
			ID:         uuid.New().String(),
			Type:       events.EventType("energy.bess.schedule_updated"),
			OccurredAt: now,
			Payload: map[string]any{
				"current_mode":        activeSlot.Mode,
				"current_target_kw":   activeSlot.TargetPowerKW,
				"current_soc":         currentSoC,
				"total_arbitrage_eur": sched.TotalArbitrageEUR,
			},
		})
	}

	return sched
}
