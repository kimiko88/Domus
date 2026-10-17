package energy

import (
	"context"
	"fmt"
	"math"
	"sort"
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
	Date          string         `json:"date"` // YYYY-MM-DD
	Market        MarketType     `json:"market"`
	Currency      string         `json:"currency"`
	HourlyPrices  []HourlyTariff `json:"hourly_prices"`
	AveragePrice  float64        `json:"average_price"`
	MinPrice      float64        `json:"min_price"`
	MaxPrice      float64        `json:"max_price"`
	CheapestHours []int          `json:"cheapest_hours"` // Top 4 cheapest hours
	UpdatedAt     time.Time      `json:"updated_at"`
}

// FlexibleLoadRule defines an appliance that can be automatically scheduled in cheap hours.
type FlexibleLoadRule struct {
	EntityKey        string  `json:"entity_key"`      // e.g. "switch.ev_charger", "switch.water_heater"
	Name             string  `json:"name"`            // Friendly name
	RequiredHours    int     `json:"required_hours"`  // Number of hours needed per day (e.g. 3)
	MaxPriceKWh      float64 `json:"max_price_kwh"`   // Hard limit above which load is not turned on
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
		UpdatedAt:     time.Now().UTC(),
	}
}

// RecalculateLoadPlans determines optimal hours for each configured flexible load.
func (o *TariffOptimizer) RecalculateLoadPlans() {
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
		for i := 0; i < reqH && i < len(sorted); i++ {
			chosen = append(chosen, sorted[i].hour)
			costSum += sorted[i].price
		}
		sort.Ints(chosen)

		rule.ScheduledHours = chosen
		rule.EstimatedCostEUR = math.Round(costSum*100) / 100

		worstCost := peakPrice * float64(reqH)
		if worstCost > 0 {
			savings := ((worstCost - costSum) / worstCost) * 100.0
			rule.SavingsPercent = math.Round(savings*10) / 10
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
	o.RecalculateLoadPlans()
}

// EvaluateCurrentHour checks if scheduled loads should be activated or turned off.
func (o *TariffOptimizer) EvaluateCurrentHour(ctx context.Context) map[string]bool {
	o.mu.Lock()
	defer o.mu.Unlock()

	nowH := time.Now().Hour()
	decisions := make(map[string]bool)

	for _, rule := range o.loads {
		if !rule.AutoSchedule {
			continue
		}

		shouldBeActive := false
		for _, h := range rule.ScheduledHours {
			if h == nowH {
				shouldBeActive = true
				break
			}
		}

		rule.ActiveNow = shouldBeActive
		decisions[rule.EntityKey] = shouldBeActive

		// Emit event for automation engine or actuators
		if o.bus != nil {
			service := "turn_off"
			if shouldBeActive {
				service = "turn_on"
			}
			_ = o.bus.Publish(ctx, events.Event{
				ID:         uuid.NewString(),
				Type:       events.CommandRequested,
				OccurredAt: time.Now().UTC(),
				Source:     "energy.tariff_optimizer",
				Payload: map[string]any{
					"entity_key": rule.EntityKey,
					"service":    service,
					"params": map[string]any{
						"reason": fmt.Sprintf("dynamic_tariff_hour_%d", nowH),
					},
				},
			})
		}
	}

	return decisions
}
