// Package main implements a Meshtastic LoRa radio bridge compiled to WASI.
// Compile target: tinygo build -o plugin.wasm -target=wasi main.go
package main

import (
	"encoding/json"
	"fmt"
	"unsafe"
)

type MeshPacket struct {
	FromNode string `json:"from_node"`
	ToNode   string `json:"to_node"`
	Payload  string `json:"payload"`
	SNR      float64`json:"snr_db"`
	RSSI     int    `json:"rssi_dbm"`
}

type EmergencyBroadcast struct {
	IncidentType string `json:"incident_type"` // "intrusion", "fire", "flood", "power_outage"
	Location     string `json:"location"`
	Message      string `json:"message"`
}

//export domus_handle_event
func domus_handle_event(ptr unsafe.Pointer, size int) uint64 {
	return 0
}

//export domus_encode_mesh_alert
func domus_encode_mesh_alert(ptr unsafe.Pointer, size int) uint64 {
	buf := unsafe.Slice((*byte)(ptr), size)
	var alert EmergencyBroadcast
	if err := json.Unmarshal(buf, &alert); err != nil {
		return 0
	}

	packet := MeshPacket{
		FromNode: "!domus-hub",
		ToNode:   "^all",
		Payload:  fmt.Sprintf("[DOMUS ALERT] %s at %s: %s", alert.IncidentType, alert.Location, alert.Message),
		SNR:      8.5,
		RSSI:     -74,
	}

	out, err := json.Marshal(packet)
	if err != nil {
		return 0
	}

	outPtr := uintptr(unsafe.Pointer(&out[0]))
	return (uint64(outPtr) << 32) | uint64(len(out))
}

func main() {}
