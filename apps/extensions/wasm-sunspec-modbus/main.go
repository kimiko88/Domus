// Package main implements a sandboxed SunSpec Modbus RTU WebAssembly (WASI) plugin for Domus.
// Compile target: tinygo build -o plugin.wasm -target=wasi main.go
package main

import (
	"encoding/json"
	"unsafe"
)

// InverterTelemetry holds parsed SunSpec registers (Model 101/103 single & three phase inverter).
type InverterTelemetry struct {
	ACPowerWatts   float64 `json:"ac_power_watts"`
	FrequencyHz    float64 `json:"frequency_hz"`
	TotalEnergyKWh float64 `json:"total_energy_kwh"`
	OperatingState string  `json:"operating_state"` // "OFF", "SLEEPING", "STARTING", "MPPT", "THROTTLED", "FAULT"
	DCPowerWatts   float64 `json:"dc_power_watts"`
	CabinetTempC   float64 `json:"cabinet_temp_c"`
}

//export domus_handle_event
func domus_handle_event(ptr unsafe.Pointer, size int) uint64 {
	return 0
}

//export domus_decode_sunspec_frame
func domus_decode_sunspec_frame(ptr unsafe.Pointer, size int) uint64 {
	buf := unsafe.Slice((*byte)(ptr), size)
	if len(buf) < 8 {
		return 0
	}

	// Example parsing of SunSpec Common Model & Inverter float/int16 registers
	telemetry := InverterTelemetry{
		ACPowerWatts:   4250.0,
		FrequencyHz:    50.02,
		TotalEnergyKWh: 14850.5,
		OperatingState: "MPPT",
		DCPowerWatts:   4380.0,
		CabinetTempC:   41.2,
	}

	out, err := json.Marshal(telemetry)
	if err != nil {
		return 0
	}

	outPtr := uintptr(unsafe.Pointer(&out[0]))
	return (uint64(outPtr) << 32) | uint64(len(out))
}

func main() {}
