package energy

import (
	"context"
	"testing"
)

func TestTariffOptimizer_ScheduleGeneration(t *testing.T) {
	bus := &mockBus{}
	opt := NewTariffOptimizer(MarketPUN, bus)

	schedule := opt.GetSchedule()
	if schedule == nil {
		t.Fatalf("expected non-nil schedule")
	}

	if len(schedule.HourlyPrices) != 24 {
		t.Errorf("expected 24 hourly prices, got %d", len(schedule.HourlyPrices))
	}

	if schedule.MinPrice >= schedule.MaxPrice {
		t.Errorf("expected min price (%f) < max price (%f)", schedule.MinPrice, schedule.MaxPrice)
	}

	if len(schedule.CheapestHours) != 4 {
		t.Errorf("expected 4 cheapest hours, got %d", len(schedule.CheapestHours))
	}
}

func TestTariffOptimizer_FlexibleLoadsCalculation(t *testing.T) {
	bus := &mockBus{}
	opt := NewTariffOptimizer(MarketPUN, bus)

	loads := opt.GetFlexibleLoads()
	if len(loads) < 2 {
		t.Fatalf("expected at least 2 default flexible loads, got %d", len(loads))
	}

	for _, l := range loads {
		if len(l.ScheduledHours) != l.RequiredHours {
			t.Errorf("load %s: expected %d scheduled hours, got %d", l.Name, l.RequiredHours, len(l.ScheduledHours))
		}
		if l.SavingsPercent <= 0 {
			t.Errorf("load %s: expected positive savings percentage, got %f", l.Name, l.SavingsPercent)
		}
	}
}

func TestTariffOptimizer_Evaluation(t *testing.T) {
	bus := &mockBus{}
	opt := NewTariffOptimizer(MarketPUN, bus)

	decisions := opt.EvaluateCurrentHour(context.Background())
	if len(decisions) == 0 {
		t.Errorf("expected load decisions for current hour")
	}
}
