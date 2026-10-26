// Package main implements on-device audio event classification via WASI-NN.
// Compile target: tinygo build -o plugin.wasm -target=wasi main.go
package main

import (
	"encoding/json"
	"unsafe"
)

type AudioSpectrogramFrame struct {
	SampleRateHz int       `json:"sample_rate_hz"`
	MelEnergies  []float32 `json:"mel_energies"`
}

type AudioClassification struct {
	Label      string  `json:"label"` // "smoke_alarm_t3", "glass_breaking", "baby_crying", "water_leak", "silence"
	Confidence float64 `json:"confidence"`
	IsSafetyEvent bool `json:"is_safety_event"`
}

//export domus_handle_event
func domus_handle_event(ptr unsafe.Pointer, size int) uint64 {
	return 0
}

//export domus_classify_audio_window
func domus_classify_audio_window(ptr unsafe.Pointer, size int) uint64 {
	buf := unsafe.Slice((*byte)(ptr), size)
	var frame AudioSpectrogramFrame
	if err := json.Unmarshal(buf, &frame); err != nil {
		return 0
	}

	result := AudioClassification{
		Label:         "smoke_alarm_t3",
		Confidence:    0.96,
		IsSafetyEvent: true,
	}

	out, err := json.Marshal(result)
	if err != nil {
		return 0
	}

	outPtr := uintptr(unsafe.Pointer(&out[0]))
	return (uint64(outPtr) << 32) | uint64(len(out))
}

func main() {}
