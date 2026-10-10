package energy_test

import (
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/application/energy"
)

func TestComputeSankeyFlow(t *testing.T) {
	// Scenario 1: Solar generates 4500W, House uses 1500W, Battery charges at 2000W (-2000W), Excess to Grid (1000W export = -1000W)
	flow := energy.ComputeSankeyFlow(4500, -1000, -2000, 65.0)

	if flow.HouseConsumptionWatts != 1500 {
		t.Errorf("expected house consumption 1500W, got %.1f", flow.HouseConsumptionWatts)
	}
	if flow.SolarToHouseWatts != 1500 {
		t.Errorf("expected solar to house 1500W, got %.1f", flow.SolarToHouseWatts)
	}
	if flow.SolarToBatteryWatts != 2000 {
		t.Errorf("expected solar to battery 2000W, got %.1f", flow.SolarToBatteryWatts)
	}
	if flow.SolarToGridWatts != 1000 {
		t.Errorf("expected solar to grid 1000W, got %.1f", flow.SolarToGridWatts)
	}
	if flow.BatterySoC != 65.0 {
		t.Errorf("expected battery SoC 65%%, got %.1f", flow.BatterySoC)
	}

	// Scenario 2: Night time. Solar 0W, Battery discharges 1200W, Grid imports 800W, House uses 2000W
	nightFlow := energy.ComputeSankeyFlow(0, 800, 1200, 42.0)
	if nightFlow.HouseConsumptionWatts != 2000 {
		t.Errorf("expected night house consumption 2000W, got %.1f", nightFlow.HouseConsumptionWatts)
	}
	if nightFlow.BatteryToHouseWatts != 1200 {
		t.Errorf("expected battery to house 1200W, got %.1f", nightFlow.BatteryToHouseWatts)
	}
	if nightFlow.GridToHouseWatts != 800 {
		t.Errorf("expected grid to house 800W, got %.1f", nightFlow.GridToHouseWatts)
	}
}

func TestTariffSlots(t *testing.T) {
	cfg := energy.DefaultTariffConfig()

	// Monday 10:00 -> F1 (Peak)
	mon10 := time.Date(2026, 10, 12, 10, 0, 0, 0, time.UTC) // 2026-10-12 is Monday
	slot1, price1 := energy.GetCurrentElectricityPrice(mon10, cfg)
	if slot1 != energy.SlotF1 || price1 != 0.28 {
		t.Errorf("expected F1 at 0.28, got %s at %.2f", slot1, price1)
	}

	// Saturday 14:00 -> F2 (Mid-peak)
	sat14 := time.Date(2026, 10, 17, 14, 0, 0, 0, time.UTC) // 2026-10-17 is Saturday
	slot2, price2 := energy.GetCurrentElectricityPrice(sat14, cfg)
	if slot2 != energy.SlotF2 || price2 != 0.24 {
		t.Errorf("expected F2 at 0.24, got %s at %.2f", slot2, price2)
	}

	// Sunday 12:00 -> F3 (Off-peak)
	sun12 := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC) // 2026-10-18 is Sunday
	slot3, price3 := energy.GetCurrentElectricityPrice(sun12, cfg)
	if slot3 != energy.SlotF3 || price3 != 0.19 {
		t.Errorf("expected F3 at 0.19, got %s at %.2f", slot3, price3)
	}
}

func TestPulseMeter(t *testing.T) {
	// 1000 pulses per m3 (1 pulse = 1 liter)
	meter := energy.NewPulseMeter("water_main", energy.UtilityWater, 1000, 10.5)

	vol := meter.RecordPulses(500) // 500 liters = 0.5 m3
	if vol != 11.0 {
		t.Errorf("expected 11.0 m3, got %.2f", vol)
	}

	cost := meter.EstimateCost(1.80)
	if cost != 11.0*1.80 {
		t.Errorf("expected cost %.2f, got %.2f", 11.0*1.80, cost)
	}
}
