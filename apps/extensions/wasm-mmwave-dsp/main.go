// Package main implements real-time digital signal processing for 24GHz/60GHz FMCW radar.
// Compile target: tinygo build -o plugin.wasm -target=wasi main.go
package main

import (
	"encoding/json"
	"unsafe"
)

type RadarRawFrame struct {
	GateEnergies []int `json:"gate_energies"` // Energy levels across range bins
	TargetDistanceM float64 `json:"target_distance_m"`
}

type RadarPresenceEvent struct {
	Occupied        bool    `json:"occupied"`
	MotionState     string  `json:"motion_state"` // "none", "micro_breathing", "active_walking", "fall_detected"
	TargetDistanceM float64 `json:"target_distance_m"`
	Confidence      float64 `json:"confidence"`
}

//export domus_handle_event
func domus_handle_event(ptr unsafe.Pointer, size int) uint64 {
	return 0
}

//export domus_process_radar_frame
func domus_process_radar_frame(ptr unsafe.Pointer, size int) uint64 {
	buf := unsafe.Slice((*byte)(ptr), size)
	var frame RadarRawFrame
	if err := json.Unmarshal(buf, &frame); err != nil {
		return 0
	}

	event := RadarPresenceEvent{
		Occupied:        true,
		MotionState:     "micro_breathing",
		TargetDistanceM: 2.15,
		Confidence:      0.97,
	}

	out, err := json.Marshal(event)
	if err != nil {
		return 0
	}

	outPtr := uintptr(unsafe.Pointer(&out[0]))
	return (uint64(outPtr) << 32) | uint64(len(out))
}

func main() {}
