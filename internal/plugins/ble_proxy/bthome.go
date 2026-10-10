// Package ble_proxy implements Bluetooth Low Energy proxy ingestion and BTHome v2 / iBeacon decoding.
package ble_proxy

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

// BTHomeSensorData holds decoded telemetry values from a BTHome v2 broadcast.
type BTHomeSensorData struct {
	PacketID    *uint8   `json:"packet_id,omitempty"`
	Battery     *uint8   `json:"battery,omitempty"`
	Temperature *float64 `json:"temperature,omitempty"` // °C
	Humidity    *float64 `json:"humidity,omitempty"`    // %
	Illuminance *float64 `json:"illuminance,omitempty"` // lux
	Pressure    *float64 `json:"pressure,omitempty"`    // hPa
	WindowOpen  *bool    `json:"window_open,omitempty"`
	Motion      *bool    `json:"motion,omitempty"`
	ButtonEvent *uint8   `json:"button_event,omitempty"`
}

// ParseBTHomeV2 decodes a BTHome v2 advertisement payload.
// Specification: BTHome v2 standard by Home Assistant & Shelly BLU
func ParseBTHomeV2(data []byte) (*BTHomeSensorData, error) {
	if len(data) < 1 {
		return nil, errors.New("bthome: payload empty")
	}

	result := &BTHomeSensorData{}
	idx := 0

	// Optional BTHome header byte (e.g. 0x40 = BTHome v2 unencrypted)
	if data[0] == 0x40 {
		idx = 1
	}

	for idx < len(data) {
		objectID := data[idx]
		idx++

		switch objectID {
		case 0x00: // Packet ID (1 byte)
			if idx >= len(data) {
				break
			}
			val := data[idx]
			result.PacketID = &val
			idx += 1

		case 0x01: // Battery % (1 byte)
			if idx >= len(data) {
				break
			}
			val := data[idx]
			result.Battery = &val
			idx += 1

		case 0x02: // Temperature in 0.01 °C (int16)
			if idx+2 > len(data) {
				break
			}
			raw := int16(binary.LittleEndian.Uint16(data[idx : idx+2]))
			temp := float64(raw) * 0.01
			result.Temperature = &temp
			idx += 2

		case 0x03: // Humidity in 0.01 % (uint16)
			if idx+2 > len(data) {
				break
			}
			raw := binary.LittleEndian.Uint16(data[idx : idx+2])
			hum := float64(raw) * 0.01
			result.Humidity = &hum
			idx += 2

		case 0x05: // Illuminance in 0.01 lux (uint24)
			if idx+3 > len(data) {
				break
			}
			raw := uint32(data[idx]) | (uint32(data[idx+1]) << 8) | (uint32(data[idx+2]) << 16)
			lux := float64(raw) * 0.01
			result.Illuminance = &lux
			idx += 3

		case 0x1A: // Door / Window contact (1 byte boolean)
			if idx >= len(data) {
				break
			}
			open := data[idx] == 0x01
			result.WindowOpen = &open
			idx += 1

		case 0x21: // Motion / Presence (1 byte boolean)
			if idx >= len(data) {
				break
			}
			motion := data[idx] == 0x01
			result.Motion = &motion
			idx += 1

		default:
			// Unknown object ID, stop parsing remaining payload
			return result, nil
		}
	}

	return result, nil
}

// IBeaconData holds parsed Apple iBeacon advertisement payload.
type IBeaconData struct {
	UUID            string  `json:"uuid"`
	Major           uint16  `json:"major"`
	Minor           uint16  `json:"minor"`
	MeasuredPower   int8    `json:"measured_power"` // RSSI at 1 meter
	RSSI            int     `json:"rssi"`
	EstimatedDistM  float64 `json:"estimated_dist_m"`
}

// ParseIBeacon decodes an Apple iBeacon payload (25 bytes standard format).
func ParseIBeacon(data []byte, currentRSSI int) (*IBeaconData, error) {
	if len(data) < 25 {
		return nil, errors.New("ibeacon: payload too short")
	}

	// Apple Company ID (0x004C) and iBeacon prefix (0x02, 0x15)
	if data[0] != 0x4C || data[1] != 0x00 || data[2] != 0x02 || data[3] != 0x15 {
		return nil, errors.New("ibeacon: invalid Apple prefix")
	}

	uuidBytes := data[4:20]
	uuidStr := fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		binary.BigEndian.Uint32(uuidBytes[0:4]),
		binary.BigEndian.Uint16(uuidBytes[4:6]),
		binary.BigEndian.Uint16(uuidBytes[6:8]),
		binary.BigEndian.Uint16(uuidBytes[8:10]),
		uuidBytes[10:16])

	major := binary.BigEndian.Uint16(data[20:22])
	minor := binary.BigEndian.Uint16(data[22:24])
	txPower := int8(data[24])

	// Calculate estimated distance using log-distance path loss formula:
	// dist = 10 ^ ((MeasuredPower - RSSI) / (10 * N)), N = 2.0 (indoor propagation factor)
	dist := math.Pow(10.0, float64(int(txPower)-currentRSSI)/(10.0*2.0))

	return &IBeaconData{
		UUID:           uuidStr,
		Major:          major,
		Minor:          minor,
		MeasuredPower:  txPower,
		RSSI:           currentRSSI,
		EstimatedDistM: math.Round(dist*100) / 100,
	}, nil
}
