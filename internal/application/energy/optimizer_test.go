package energy

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTariffOptimizer_ScheduleGeneration(t *testing.T) {
	bus := &mockBus{}
	opt := NewTariffOptimizer(MarketPUN, bus)
	schedule := opt.GetSchedule()
	require.NotNil(t, schedule)
	require.Len(t, schedule.HourlyPrices, 24)
	require.Less(t, schedule.MinPrice, schedule.MaxPrice)
	require.Len(t, schedule.CheapestHours, 4)
	require.True(t, schedule.IsEstimate)
	require.Equal(t, "synthetic_estimate", schedule.Source)
}

func TestTariffOptimizer_FlexibleLoadsCalculation(t *testing.T) {
	opt := NewTariffOptimizer(MarketPUN, &mockBus{})
	loads := opt.GetFlexibleLoads()
	require.GreaterOrEqual(t, len(loads), 2)
	for _, load := range loads {
		require.Len(t, load.ScheduledHours, load.RequiredHours)
		require.Zero(t, load.PowerKW)
		require.Zero(t, load.EstimatedCostEUR, "cost is unknown without configured power")
		require.Zero(t, load.SavingsPercent, "savings is unknown without configured power")
	}
}

func TestTariffOptimizer_EstimatedScheduleNeverControlsLoads(t *testing.T) {
	bus := &mockBus{}
	opt := NewTariffOptimizer(MarketPUN, bus)
	decisions := opt.EvaluateCurrentHour(context.Background())
	require.NotEmpty(t, decisions)
	for key, active := range decisions {
		require.False(t, active, "estimated tariffs must not activate %s", key)
	}
	require.Empty(t, bus.events)
}

func TestTariffOptimizer_SetScheduleAndEnforceMaximumPrice(t *testing.T) {
	bus := &mockBus{}
	opt := NewTariffOptimizer(MarketCustomSlots, bus)
	now := time.Now()
	makeSchedule := func(currentPrice float64) *DailyTariffSchedule {
		prices := make([]HourlyTariff, 24)
		for hour := range prices {
			price := 0.10
			if hour == now.Hour() { price = currentPrice }
			prices[hour] = HourlyTariff{Hour: hour, PricePerKWh: price}
		}
		return &DailyTariffSchedule{
			Date: now.Format("2006-01-02"), Market: MarketCustomSlots,
			Currency: "EUR", Source: "test-provider", HourlyPrices: prices,
		}
	}
	rule := FlexibleLoadRule{
		EntityKey: "switch.phase0_test_load", Name: "Phase 0 test load",
		RequiredHours: 24, MaxPriceKWh: 0.20, AutoSchedule: true,
	}
	opt.SetFlexibleLoad(rule)
	require.NoError(t, opt.SetTariffSchedule(makeSchedule(0.25)))
	decisions := opt.EvaluateCurrentHour(context.Background())
	require.False(t, decisions[rule.EntityKey])
	require.NoError(t, opt.SetTariffSchedule(makeSchedule(0.15)))
	decisions = opt.EvaluateCurrentHour(context.Background())
	require.True(t, decisions[rule.EntityKey])
	for _, load := range opt.GetFlexibleLoads() {
		if load.EntityKey == rule.EntityKey {
			require.True(t, load.ActiveNow)
		}
	}
	// An expired trusted schedule must shut down a load that the optimizer
	// previously enabled, rather than leaving it on indefinitely.
	stale := makeSchedule(0.15)
	stale.Date = now.AddDate(0, 0, -1).Format("2006-01-02")
	require.NoError(t, opt.SetTariffSchedule(stale))
	decisions = opt.EvaluateCurrentHour(context.Background())
	require.False(t, decisions[rule.EntityKey])
	require.Len(t, bus.events, 2)
	last := bus.events[len(bus.events)-1].Payload.(map[string]any)
	require.Equal(t, "turn_off", last["service"])
}

func TestTariffOptimizer_RejectsEstimatedOrIncompleteSchedules(t *testing.T) {
	opt := NewTariffOptimizer(MarketPUN, nil)
	require.Error(t, opt.SetTariffSchedule(opt.GenerateDailySchedule(time.Now())))
	invalid := &DailyTariffSchedule{
		Date: time.Now().Format("2006-01-02"), Source: "test-provider",
		Currency: "EUR", HourlyPrices: []HourlyTariff{{Hour: 0, PricePerKWh: 0.1}},
	}
	require.Error(t, opt.SetTariffSchedule(invalid))
}
