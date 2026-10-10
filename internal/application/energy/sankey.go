package energy

import "math"

// SankeyFlow represents the computed power distribution vectors across Solar, Battery, Grid, and House.
type SankeyFlow struct {
	SolarWatts            float64 `json:"solar_watts"`
	GridWatts             float64 `json:"grid_watts"`    // Positive = Import, Negative = Export
	BatteryWatts          float64 `json:"battery_watts"` // Positive = Discharging, Negative = Charging
	BatterySoC            float64 `json:"battery_soc"`   // 0 - 100%
	HouseConsumptionWatts float64 `json:"house_consumption_watts"`

	// Flow Vectors (Watts)
	SolarToHouseWatts   float64 `json:"solar_to_house_watts"`
	SolarToBatteryWatts float64 `json:"solar_to_battery_watts"`
	SolarToGridWatts    float64 `json:"solar_to_grid_watts"`
	BatteryToHouseWatts float64 `json:"battery_to_house_watts"`
	GridToHouseWatts    float64 `json:"grid_to_house_watts"`
	GridToBatteryWatts  float64 `json:"grid_to_battery_watts"`
}

// ComputeSankeyFlow calculates directed flow vectors balancing generation, storage, grid, and load.
func ComputeSankeyFlow(solarWatts, gridWatts, batteryWatts, batterySoC float64) SankeyFlow {
	flow := SankeyFlow{
		SolarWatts:   math.Max(0, solarWatts),
		GridWatts:    gridWatts,
		BatteryWatts: batteryWatts,
		BatterySoC:   math.Min(100, math.Max(0, batterySoC)),
	}

	// Calculate home consumption:
	// House = Solar + GridImport (if >0) + BatteryDischarge (if >0) - BatteryCharge (if <0) - GridExport (if <0)
	// Or dynamically: House = Solar + Grid - Battery (where battery is + for discharge, - for charge)
	houseWatts := flow.SolarWatts + gridWatts + batteryWatts
	if houseWatts < 0 {
		houseWatts = 0
	}
	flow.HouseConsumptionWatts = math.Round(houseWatts*100) / 100

	remSolar := flow.SolarWatts

	// 1. Solar to House
	if remSolar > 0 && houseWatts > 0 {
		toHouse := math.Min(remSolar, houseWatts)
		flow.SolarToHouseWatts = math.Round(toHouse*100) / 100
		remSolar -= toHouse
		houseWatts -= toHouse
	}

	// 2. Solar to Battery (if charging, batteryWatts < 0)
	if remSolar > 0 && batteryWatts < 0 {
		batteryChargeNeeded := math.Abs(batteryWatts)
		toBattery := math.Min(remSolar, batteryChargeNeeded)
		flow.SolarToBatteryWatts = math.Round(toBattery*100) / 100
		remSolar -= toBattery
	}

	// 3. Excess Solar to Grid
	if remSolar > 0 {
		flow.SolarToGridWatts = math.Round(remSolar*100) / 100
	}

	// 4. Battery to House (if discharging, batteryWatts > 0)
	if batteryWatts > 0 && houseWatts > 0 {
		fromBattery := math.Min(batteryWatts, houseWatts)
		flow.BatteryToHouseWatts = math.Round(fromBattery*100) / 100
		houseWatts -= fromBattery
	}

	// 5. Grid to House (if importing, gridWatts > 0)
	if gridWatts > 0 && houseWatts > 0 {
		fromGrid := math.Min(gridWatts, houseWatts)
		flow.GridToHouseWatts = math.Round(fromGrid*100) / 100
	}

	// 6. Grid to Battery (rare: night battery top-up during low tariff)
	if gridWatts > 0 && batteryWatts < 0 {
		gridToBat := math.Min(gridWatts, math.Abs(batteryWatts)-flow.SolarToBatteryWatts)
		if gridToBat > 0 {
			flow.GridToBatteryWatts = math.Round(gridToBat*100) / 100
		}
	}

	return flow
}
