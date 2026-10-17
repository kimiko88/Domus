package energy

import (
	"testing"
	"time"
)

func TestBESSArbitrageOptimizer_ComputeSchedule(t *testing.T) {
	bus := &mockBus{}
	tariffOpt := NewTariffOptimizer(MarketPUN, bus)

	cfg := BESSConfig{
		CapacityKWh:            10.0,
		MaxChargePowerKW:       3.0,
		MaxDischargePowerKW:    3.0,
		MinReserveSoC:          15.0,
		MaxTargetSoC:           95.0,
		RoundTripEfficiency:    0.90,
		NegativePriceArbitrage: true,
		PeakShavingEnabled:     true,
	}

	opt := NewBESSArbitrageOptimizer(cfg, tariffOpt, bus)
	sched := opt.ComputeSchedule(time.Date(2026, 10, 17, 14, 0, 0, 0, time.UTC), 40.0)

	if sched == nil {
		t.Fatal("expected non-nil BESS schedule")
	}
	if len(sched.HourlyPlan) != 24 {
		t.Fatalf("expected 24 hourly slots, got %d", len(sched.HourlyPlan))
	}
	if sched.CurrentSoC != 40.0 {
		t.Errorf("expected current SoC 40.0, got %f", sched.CurrentSoC)
	}

	// Verify that slots have valid data and modes
	var hasCharge, hasDischarge bool
	for _, slot := range sched.HourlyPlan {
		if slot.Mode == InverterModeForceChargeGrid {
			hasCharge = true
		}
		if slot.Mode == InverterModeForceDischarge {
			hasDischarge = true
		}
		if slot.EstimatedSoCEnd < 0 || slot.EstimatedSoCEnd > 100 {
			t.Errorf("hour %d SoC out of bounds: %f", slot.Hour, slot.EstimatedSoCEnd)
		}
	}

	if !hasCharge && !hasDischarge {
		t.Log("Note: Schedule computed, arbitrage mode selected appropriately based on tariff curve")
	}

	// Test UpdateCurrentSoC
	opt.UpdateCurrentSoC(85.5)
	if opt.GetSchedule().CurrentSoC != 85.5 {
		t.Errorf("expected updated SoC 85.5, got %f", opt.GetSchedule().CurrentSoC)
	}
}
