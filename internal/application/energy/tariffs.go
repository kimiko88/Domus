package energy

import "time"

// TariffSlot defines the Italian ARERA time-of-use energy band.
type TariffSlot string

const (
	SlotF1 TariffSlot = "F1" // Peak: Mon-Fri 08:00 - 19:00
	SlotF2 TariffSlot = "F2" // Mid-peak: Mon-Fri 07:00 - 08:00, 19:00 - 23:00, Sat 07:00 - 23:00
	SlotF3 TariffSlot = "F3" // Off-peak: Mon-Sat 23:00 - 07:00, Sun and holidays
)

// TariffConfig holds configured prices for electricity bands and utilities.
type TariffConfig struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Currency     string  `json:"currency"`
	F1Price      float64 `json:"f1_price"`      // EUR/kWh
	F2Price      float64 `json:"f2_price"`      // EUR/kWh
	F3Price      float64 `json:"f3_price"`      // EUR/kWh
	GasSmcPrice  float64 `json:"gas_smc_price"`  // EUR/Smc
	WaterM3Price float64 `json:"water_m3_price"` // EUR/m3
}

// DefaultTariffConfig returns typical Italian residential prices.
func DefaultTariffConfig() TariffConfig {
	return TariffConfig{
		ID:           "default",
		Name:         "Standard ARERA Bioraria/Trioraria",
		Currency:     "EUR",
		F1Price:      0.28,
		F2Price:      0.24,
		F3Price:      0.19,
		GasSmcPrice:  0.95,
		WaterM3Price: 1.80,
	}
}

// GetCurrentTariffSlot computes the active time slot for the given local timestamp.
func GetCurrentTariffSlot(t time.Time) TariffSlot {
	weekday := t.Weekday()
	hour := t.Hour()

	// Sundays are always F3
	if weekday == time.Sunday {
		return SlotF3
	}

	// Saturdays:
	if weekday == time.Saturday {
		if hour >= 7 && hour < 23 {
			return SlotF2
		}
		return SlotF3
	}

	// Monday to Friday:
	if hour >= 8 && hour < 19 {
		return SlotF1
	}
	if (hour >= 7 && hour < 8) || (hour >= 19 && hour < 23) {
		return SlotF2
	}

	// 23:00 to 07:00
	return SlotF3
}

// GetCurrentElectricityPrice returns the price in EUR/kWh for the active slot.
func GetCurrentElectricityPrice(t time.Time, cfg TariffConfig) (TariffSlot, float64) {
	slot := GetCurrentTariffSlot(t)
	switch slot {
	case SlotF1:
		return slot, cfg.F1Price
	case SlotF2:
		return slot, cfg.F2Price
	default:
		return slot, cfg.F3Price
	}
}
