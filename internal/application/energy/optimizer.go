package energy

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/google/uuid"
)

// MarketType defines the energy wholesale market source.
type MarketType string

const (
	MarketPUN          MarketType = "PUN_ITALY"     // Prezzo Unico Nazionale (GME Italia)
	MarketNordPool     MarketType = "NORD_POOL"     // Nord Pool Day-Ahead (Europa)
	MarketOctopusAgile MarketType = "OCTOPUS_AGILE" // Tariffe dinamiche Octopus Energy
	MarketCustomSlots  MarketType = "CUSTOM_SLOTS"  // Fasce orarie configurate manualmente
)

// HourlyTariff represents the electricity price for a specific hour of the day.
type HourlyTariff struct {
	Hour        int        `json:"hour"`          // 0 - 23
	PricePerKWh float64    `json:"price_per_kwh"` // EUR/kWh
	Slot        TariffSlot `json:"slot"`          // F1, F2, F3
	IsCheapest  bool       `json:"is_cheapest"`   // True if in the cheapest quartile
	IsExpensive bool       `json:"is_expensive"`  // True if in the peak quartile
	IsNegative  bool       `json:"is_negative"`   // True if price <= 0 (e.g. overproduzione solare/eolica)
}

// DailyTariffSchedule holds the 24-hour price curve for a specific day.
type DailyTariffSchedule struct {
	Date          string         `json:"date"`
	Market        MarketType     `json:"market"`
	Currency      string         `json:"currency"`
	Source        string         `json:"source"`
	IsEstimate    bool           `json:"is_estimate"`
	HourlyPrices  []HourlyTariff `json:"hourly_prices"`
	AveragePrice  float64        `json:"average_price"`
	MinPrice      float64        `json:"min_price"`
	MaxPrice      float64        `json:"max_price"`
	CheapestHours []int          `json:"cheapest_hours"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

// FlexibleLoadRule defines an appliance that can be automatically scheduled in cheap hours.
type FlexibleLoadRule struct {
	EntityKey        string  `json:"entity_key"`      // e.g. "switch.ev_charger", "switch.water_heater"
	Name             string  `json:"name"`            // Friendly name
	RequiredHours    int     `json:"required_hours"`  // Number of hours needed per day (e.g. 3)
	MaxPriceKWh      float64 `json:"max_price_kwh"`   // Hard limit above which load is not turned on
	PowerKW          float64 `json:"power_kw,omitempty"` // Configured average load power for cost estimation
	ActiveNow        bool    `json:"active_now"`      // Whether load is currently turned on by optimizer
	AutoSchedule     bool    `json:"auto_schedule"`   // True if Domus should toggle automatically
	ScheduledHours   []int   `json:"scheduled_hours"` // Best hours chosen for today
	EstimatedCostEUR float64 `json:"estimated_cost_eur"`
	SavingsPercent   float64 `json:"savings_percent"`
}

// TariffOptimizer manages dynamic hourly price curves and optimizes heavy load execution.
type TariffOptimizer struct {
	mu       sync.RWMutex
	market   MarketType
	schedule *DailyTariffSchedule
	loads    map[string]*FlexibleLoadRule
	bus      events.Bus
	stopCh   chan struct{}
}

// NewTariffOptimizer creates a new dynamic tariff and load optimizer.
func NewTariffOptimizer(market MarketType, bus events.Bus) *TariffOptimizer {
	if market == "" {
		market = MarketPUN
	}

	opt := &TariffOptimizer{
		market: market,
		loads:  make(map[string]*FlexibleLoadRule),
		bus:    bus,
		stopCh: make(chan struct{}),
	}

	// Generate baseline schedule for today
	opt.schedule = opt.GenerateDailySchedule(time.Now().UTC())

	// Default smart loads
	opt.loads["switch.ev_charger"] = &FlexibleLoadRule{
		EntityKey:     "switch.ev_charger",
		Name:          "Ricarica Auto Elettrica (Wallbox)",
		RequiredHours: 4,
		MaxPriceKWh:   0.20,
		AutoSchedule:  true,
	}
	opt.loads["switch.water_heater"] = &FlexibleLoadRule{
		EntityKey:     "switch.water_heater",
		Name:          "Scaldabagno / Pompa di Calore ACS",
		RequiredHours: 3,
		MaxPriceKWh:   0.18,
		AutoSchedule:  true,
	}

	opt.RecalculateLoadPlans()
	return opt
}

// GenerateDailySchedule calculates or estimates the 24h curve based on market profile.
func (o *TariffOptimizer) GenerateDailySchedule(t time.Time) *DailyTariffSchedule {
	dateStr := t.Format("2006-01-02")
	hours := make([]HourlyTariff, 24)

	// Typical European daily wholesale price curve (solar valley around 12:00-15:00, evening peak 19:00-21:00, night dip 01:00-05:00)
	var sum float64
	minP := 999.0
	maxP := -999.0

	for h := 0; h < 24; h++ {
		var basePrice float64
		switch {
		case h >= 1 && h <= 5: // Deep night dip
			basePrice = 0.115 + (float64(h) * 0.005)
		case h >= 6 && h <= 8: // Morning ramp
			basePrice = 0.190 + (float64(h-6) * 0.02)
		case h >= 9 && h <= 11: // Morning working peak
			basePrice = 0.225
		case h >= 12 && h <= 15: // Solar duck curve depression (abundant PV generation)
			basePrice = 0.130 - (float64(h-12) * 0.005)
		case h >= 16 && h <= 18: // Afternoon ramp
			basePrice = 0.210
		case h >= 19 && h <= 21: // Evening absolute peak
			basePrice = 0.275 + (float64(h-19) * 0.01)
		default: // Night descent 22-23
			basePrice = 0.160
		}

		price := math.Round(basePrice*1000) / 1000
		sum += price
		if price < minP {
			minP = price
		}
		if price > maxP {
			maxP = price
		}

		slot := GetCurrentTariffSlot(time.Date(t.Year(), t.Month(), t.Day(), h, 30, 0, 0, t.Location()))
		hours[h] = HourlyTariff{
			Hour:        h,
			PricePerKWh: price,
			Slot:        slot,
			IsNegative:  price <= 0,
		}
	}

	avg := math.Round((sum/24.0)*1000) / 1000

	// Sort hours by price to find cheapest and expensive quartiles
	type indexedPrice struct {
		hour  int
		price float64
	}
	sorted := make([]indexedPrice, 24)
	for i, h := range hours {
		sorted[i] = indexedPrice{hour: h.Hour, price: h.PricePerKWh}
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].price < sorted[j].price
	})

	cheapestHours := make([]int, 4)
	for i := 0; i < 4; i++ {
		cheapestHours[i] = sorted[i].hour
	}
	sort.Ints(cheapestHours)

	// Mark flags
	for i := 0; i < 6; i++ {
		hours[sorted[i].hour].IsCheapest = true
	}
	for i := 18; i < 24; i++ {
		hours[sorted[i].hour].IsExpensive = true
	}

	return &DailyTariffSchedule{
		Date:          dateStr,
		Market:        o.market,
		Currency:      "EUR",
		HourlyPrices:  hours,
		AveragePrice:  avg,
		MinPrice:      minP,
		MaxPrice:      maxP,
		CheapestHours: cheapestHours,
		Source: "synthetic_estimate",
		IsEstimate: true,
		UpdatedAt: time.Now().UTC(),
	}
}

// SetTariffSchedule accepts a complete, externally sourced daily tariff.
// Synthetic or incomplete data is never sufficient to control physical loads.
func (o *TariffOptimizer) SetTariffSchedule(schedule *DailyTariffSchedule) error {
	if schedule == nil || schedule.IsEstimate || strings.TrimSpace(schedule.Source) == "" {
		return errors.New("energy: a non-estimated tariff schedule with a source is required")
	}
	if _, err := time.ParseInLocation("2006-01-02", schedule.Date, time.Local); err != nil {
		return errors.New("energy: tariff date must use YYYY-MM-DD")
	}
	if len(schedule.HourlyPrices) != 24 {
		return errors.New("energy: tariff schedule must contain exactly 24 hourly prices")
	}
	seen := make(map[int]bool, 24)
	for _, hour := range schedule.HourlyPrices {
		if hour.Hour < 0 || hour.Hour > 23 || seen[hour.Hour] ||
			math.IsNaN(hour.PricePerKWh) || math.IsInf(hour.PricePerKWh, 0) {
			return errors.New("energy: tariff schedule contains an invalid hour or price")
		}
		seen[hour.Hour] = true
	}
	copySchedule := *schedule
	copySchedule.HourlyPrices = append([]HourlyTariff(nil), schedule.HourlyPrices...)
	sort.Slice(copySchedule.HourlyPrices, func(i, j int) bool {
		return copySchedule.HourlyPrices[i].Hour < copySchedule.HourlyPrices[j].Hour
	})
	copySchedule.CheapestHours = nil
	type pricePoint struct { hour int; price float64 }
	points := make([]pricePoint, 0, 24)
	var sum float64
	copySchedule.MinPrice, copySchedule.MaxPrice = math.Inf(1), math.Inf(-1)
	for i := range copySchedule.HourlyPrices {
		hour := &copySchedule.HourlyPrices[i]
		sum += hour.PricePerKWh
		if hour.PricePerKWh < copySchedule.MinPrice { copySchedule.MinPrice = hour.PricePerKWh }
		if hour.PricePerKWh > copySchedule.MaxPrice { copySchedule.MaxPrice = hour.PricePerKWh }
		hour.IsNegative = hour.PricePerKWh <= 0
		hour.IsCheapest, hour.IsExpensive = false, false
		points = append(points, pricePoint{hour: hour.Hour, price: hour.PricePerKWh})
	}
	copySchedule.AveragePrice = sum / 24
	sort.Slice(points, func(i, j int) bool { return points[i].price < points[j].price })
	for i := 0; i < 4; i++ { copySchedule.CheapestHours = append(copySchedule.CheapestHours, points[i].hour) }
	sort.Ints(copySchedule.CheapestHours)
	for i := 0; i < 6; i++ { copySchedule.HourlyPrices[points[i].hour].IsCheapest = true }
	for i := 18; i < 24; i++ { copySchedule.HourlyPrices[points[i].hour].IsExpensive = true }
	if copySchedule.UpdatedAt.IsZero() { copySchedule.UpdatedAt = time.Now().UTC() }
	o.mu.Lock()
	o.schedule = &copySchedule
	o.recalculateLoadPlansLocked()
	o.mu.Unlock()
	return nil
}

// RecalculateLoadPlans determines optimal hours for each configured flexible load.
func (o *TariffOptimizer) RecalculateLoadPlans() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.recalculateLoadPlansLocked()
}

func (o *TariffOptimizer) recalculateLoadPlansLocked() {
	if o.schedule == nil || len(o.schedule.HourlyPrices) != 24 {
		return
	}

	type hourPrice struct {
		hour  int
		price float64
	}
	sorted := make([]hourPrice, 24)
	for i, h := range o.schedule.HourlyPrices {
		sorted[i] = hourPrice{hour: h.Hour, price: h.PricePerKWh}
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].price < sorted[j].price
	})

	peakPrice := o.schedule.MaxPrice

	for _, rule := range o.loads {
		reqH := rule.RequiredHours
		if reqH <= 0 {
			reqH = 1
		}
		if reqH > 24 {
			reqH = 24
		}

		chosen := make([]int, 0, reqH)
		var costSum float64
		for i := 0; i < len(sorted) && len(chosen) < reqH; i++ {
			if rule.MaxPriceKWh > 0 && sorted[i].price > rule.MaxPriceKWh { continue }
			chosen = append(chosen, sorted[i].hour)
			costSum += sorted[i].price
		}
		sort.Ints(chosen)

		rule.ScheduledHours = chosen
		// EUR/kWh multiplied by a duration in hours is not a cost unless
		// the device's average power is known. Never assume an arbitrary 1 kW.
		rule.EstimatedCostEUR = 0
		rule.SavingsPercent = 0
		if rule.PowerKW > 0 && !math.IsNaN(rule.PowerKW) && !math.IsInf(rule.PowerKW, 0) {
			estimatedCost := costSum * rule.PowerKW
			rule.EstimatedCostEUR = math.Round(estimatedCost*100) / 100
			worstCost := peakPrice * float64(len(chosen)) * rule.PowerKW
			if worstCost > 0 && len(chosen) > 0 {
				savings := ((worstCost - estimatedCost) / worstCost) * 100.0
				rule.SavingsPercent = math.Round(savings*10) / 10
			}
		}
	}
}

// GetCurrentHourTariff returns the active price for the current hour.
func (o *TariffOptimizer) GetCurrentHourTariff() HourlyTariff {
	o.mu.RLock()
	defer o.mu.RUnlock()

	nowH := time.Now().Hour()
	if o.schedule != nil && nowH < len(o.schedule.HourlyPrices) {
		return o.schedule.HourlyPrices[nowH]
	}
	return HourlyTariff{Hour: nowH, PricePerKWh: 0.22, Slot: SlotF2}
}

// GetSchedule returns the complete daily schedule.
func (o *TariffOptimizer) GetSchedule() *DailyTariffSchedule {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.schedule
}

// GetFlexibleLoads returns all configured flexible loads with their optimized plans.
func (o *TariffOptimizer) GetFlexibleLoads() []*FlexibleLoadRule {
	o.mu.RLock()
	defer o.mu.RUnlock()

	res := make([]*FlexibleLoadRule, 0, len(o.loads))
	for _, l := range o.loads {
		res = append(res, l)
	}
	return res
}

// SetFlexibleLoad configures or updates an appliance load rule.
func (o *TariffOptimizer) SetFlexibleLoad(rule FlexibleLoadRule) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.loads[rule.EntityKey] = &rule
	o.recalculateLoadPlansLocked()
}

// EvaluateCurrentHour checks if scheduled loads should be activated or turned off.
func (o *TariffOptimizer) EvaluateCurrentHour(ctx context.Context) map[string]bool {
	o.mu.Lock()
	defer o.mu.Unlock()

	now := time.Now()
	nowH := now.Hour()
	decisions := make(map[string]bool)
	scheduleTrusted := o.schedule != nil && !o.schedule.IsEstimate &&
		o.schedule.Date == now.Format("2006-01-02") && len(o.schedule.HourlyPrices) == 24

	for _, rule := range o.loads {
		if !rule.AutoSchedule { continue }
		shouldBeActive := false
		if scheduleTrusted {
			for _, h := range rule.ScheduledHours {
				if h == nowH { shouldBeActive = true; break }
			}
			currentPrice := o.schedule.HourlyPrices[nowH].PricePerKWh
			if rule.MaxPriceKWh > 0 && currentPrice > rule.MaxPriceKWh { shouldBeActive = false }
		}
		previouslyActive := rule.ActiveNow
		if previouslyActive != shouldBeActive {
			// Estimated/stale schedules never turn loads on. When a trusted
			// schedule expires, try to turn off loads previously activated by
			// this optimizer. Failed dispatches retain the old state for retry.
			mayDispatch := o.bus != nil && (scheduleTrusted || (previouslyActive && !shouldBeActive))
			if !mayDispatch {
				decisions[rule.EntityKey] = previouslyActive
				continue
			}
			service := "turn_off"
			if shouldBeActive { service = "turn_on" }
			err := o.bus.Publish(ctx, events.Event{
				ID: uuid.NewString(),
				Type: events.CommandRequested,
				OccurredAt: time.Now().UTC(),
				Source: "energy.tariff_optimizer",
				Payload: map[string]any{
					"entity_key": rule.EntityKey,
					"service": service,
					"params": map[string]any{"reason": fmt.Sprintf("dynamic_tariff_hour_%d", nowH)},
				},
			})
			if err != nil {
				decisions[rule.EntityKey] = previouslyActive
				continue
			}
		}
		rule.ActiveNow = shouldBeActive
		decisions[rule.EntityKey] = shouldBeActive
	}

	return decisions
}
