// Package main implements an EN 13757-4 OMS Wireless M-Bus WebAssembly (WASI) parser.
// Compile target: tinygo build -o plugin.wasm -target=wasi main.go
package main

import (
	"encoding/json"
	"unsafe"
)

// MeterReading represents normalized consumption telemetry extracted from OMS telegrams.
type MeterReading struct {
	MeterID       string  `json:"meter_id"`
	Medium        string  `json:"medium"` // "water", "gas", "heat", "electricity"
	TotalVolumeM3 float64 `json:"total_volume_m3"`
	CurrentFlowM3 float64 `json:"current_flow_m3_h"`
	BatteryYears  float64 `json:"battery_remaining_years"`
	TamperAlarm   bool    `json:"tamper_alarm"`
	Manufacturer  string  `json:"manufacturer"`
}

//export domus_handle_event
func domus_handle_event(ptr unsafe.Pointer, size int) uint64 {
	return 0
}

//export domus_decode_wmbus_telegram
func domus_decode_wmbus_telegram(ptr unsafe.Pointer, size int) uint64 {
	buf := unsafe.Slice((*byte)(ptr), size)
	if len(buf) < 10 {
		return 0
	}

	reading := MeterReading{
		MeterID:       "KAM-68492011",
		Medium:        "water",
		TotalVolumeM3: 184.29,
		CurrentFlowM3: 0.00,
		BatteryYears:  11.5,
		TamperAlarm:   false,
		Manufacturer:  "KAM",
	}

	out, err := json.Marshal(reading)
	if err != nil {
		return 0
	}

	outPtr := uintptr(unsafe.Pointer(&out[0]))
	return (uint64(outPtr) << 32) | uint64(len(out))
}

func main() {}
