// Package main implements a high-performance WebAssembly plugin for Domus OS.
// Compile target: WebAssembly (WASI) -> tinygo build -o plugin.wasm -target=wasi main.go
package main

import (
	"encoding/json"
	"unsafe"
)

// EnergyState represents incoming real-time telemetry from Domus Core.
type EnergyState struct {
	BatterySoC        float64 `json:"battery_soc"`        // In percentage 0-100%
	SolarPowerWatts   float64 `json:"solar_power_watts"`   // Current solar PV production
	CurrentTariffEuro float64 `json:"current_tariff_eur"`  // Real-time PUN / spot price
	HomeLoadWatts     float64 `json:"home_load_watts"`     // Domestic load absorption
}

// ArbitrageDecision represents the scheduled command dispatched to the inverter.
type ArbitrageDecision struct {
	Action      string  `json:"action"`       // "force_charge", "force_discharge", "self_consumption"
	TargetWatts float64 `json:"target_watts"` // Desired charge or discharge rate
	Reason      string  `json:"reason"`       // Human-readable diagnostic rationale
}

//export domus_evaluate_arbitrage
func domus_evaluate_arbitrage(ptr unsafe.Pointer, size int) uint64 {
	// 1. Read input buffer passed by Domus WASM host runtime
	buf := unsafe.Slice((*byte)(ptr), size)
	var state EnergyState
	if err := json.Unmarshal(buf, &state); err != nil {
		return 0
	}

	decision := ArbitrageDecision{
		Action:      "self_consumption",
		TargetWatts: 0,
		Reason:      "Tariff in normal band: prioritizing local solar self-consumption",
	}

	// 2. High-speed arbitrage optimization algorithm
	if state.CurrentTariffEuro <= 0.08 && state.BatterySoC < 95.0 {
		decision.Action = "force_charge"
		decision.TargetWatts = 3000.0
		decision.Reason = "Ultra-low/negative dynamic tariff detected: forcing grid battery charging"
	} else if state.CurrentTariffEuro >= 0.28 && state.BatterySoC > 20.0 {
		decision.Action = "force_discharge"
		decision.TargetWatts = state.HomeLoadWatts
		decision.Reason = "Peak evening tariff: zeroing grid import and powering house from BESS"
	}

	// 3. Serialize output JSON for Domus EventBus
	out, err := json.Marshal(decision)
	if err != nil {
		return 0
	}

	// Return pointer and length packed in 64-bit word
	outPtr := uintptr(unsafe.Pointer(&out[0]))
	return (uint64(outPtr) << 32) | uint64(len(out))
}

func main() {
	// Keep WASM module initialized
}
