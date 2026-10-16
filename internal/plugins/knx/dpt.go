// Package knx provides a KNXnet/IP bridge for wired European home and building automation systems.
package knx

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Supported KNX Data Point Types (DPT)
const (
	DPT1_Switch      = "1.001"  // 1-bit boolean: off/on
	DPT5_Scaling     = "5.001"  // 8-bit unsigned: 0..100% (0..255)
	DPT9_Temperature = "9.001"  // 2-octet float: temperature in °C
	DPT14_Power      = "14.056" // 4-byte IEEE 754 float: electrical power in W
)

// ParseGroupAddress converts a 3-level KNX group address (e.g., "1/2/3") into a 16-bit unsigned integer.
// Format: Main (5 bits) / Middle (3 bits) / Sub (8 bits)
func ParseGroupAddress(addrStr string) (uint16, error) {
	parts := strings.Split(addrStr, "/")
	if len(parts) != 3 {
		return 0, fmt.Errorf("knx: invalid group address format %q, expected 'main/middle/sub'", addrStr)
	}

	main, err := strconv.ParseUint(parts[0], 10, 8)
	if err != nil || main > 31 {
		return 0, fmt.Errorf("knx: invalid main group %q (must be 0..31)", parts[0])
	}

	middle, err := strconv.ParseUint(parts[1], 10, 8)
	if err != nil || middle > 7 {
		return 0, fmt.Errorf("knx: invalid middle group %q (must be 0..7)", parts[1])
	}

	sub, err := strconv.ParseUint(parts[2], 10, 8)
	if err != nil || sub > 255 {
		return 0, fmt.Errorf("knx: invalid sub group %q (must be 0..255)", parts[2])
	}

	raw := (uint16(main) << 11) | (uint16(middle) << 8) | uint16(sub)
	return raw, nil
}

// FormatGroupAddress converts a 16-bit KNX group address into 3-level string format "main/middle/sub".
func FormatGroupAddress(raw uint16) string {
	main := (raw >> 11) & 0x1F
	middle := (raw >> 8) & 0x07
	sub := raw & 0xFF
	return fmt.Sprintf("%d/%d/%d", main, middle, sub)
}

// ParseIndividualAddress converts a physical individual address (e.g. "1.1.250") into uint16.
// Format: Area (4 bits) . Line (4 bits) . Device (8 bits)
func ParseIndividualAddress(addrStr string) (uint16, error) {
	parts := strings.Split(addrStr, ".")
	if len(parts) != 3 {
		return 0, fmt.Errorf("knx: invalid individual address format %q, expected 'area.line.device'", addrStr)
	}

	area, err := strconv.ParseUint(parts[0], 10, 8)
	if err != nil || area > 15 {
		return 0, fmt.Errorf("knx: invalid area %q (0..15)", parts[0])
	}

	line, err := strconv.ParseUint(parts[1], 10, 8)
	if err != nil || line > 15 {
		return 0, fmt.Errorf("knx: invalid line %q (0..15)", parts[1])
	}

	device, err := strconv.ParseUint(parts[2], 10, 8)
	if err != nil || device > 255 {
		return 0, fmt.Errorf("knx: invalid device %q (0..255)", parts[2])
	}

	raw := (uint16(area) << 12) | (uint16(line) << 8) | uint16(device)
	return raw, nil
}

// FormatIndividualAddress formats an individual address into "area.line.device".
func FormatIndividualAddress(raw uint16) string {
	area := (raw >> 12) & 0x0F
	line := (raw >> 8) & 0x0F
	device := raw & 0xFF
	return fmt.Sprintf("%d.%d.%d", area, line, device)
}

// EncodeDPT1 encodes a boolean value into a 1-bit KNX representation (0 or 1).
func EncodeDPT1(val bool) byte {
	if val {
		return 0x01
	}
	return 0x00
}

// DecodeDPT1 decodes a byte into a boolean.
func DecodeDPT1(b byte) bool {
	return (b & 0x01) == 0x01
}

// EncodeDPT5 encodes 0..100 percentage into an 8-bit unsigned value (0..255).
func EncodeDPT5(percent float64) byte {
	if percent <= 0 {
		return 0
	}
	if percent >= 100 {
		return 255
	}
	return byte(math.Round((percent * 255.0) / 100.0))
}

// DecodeDPT5 decodes an 8-bit unsigned value into 0..100 percentage.
func DecodeDPT5(b byte) float64 {
	return math.Round((float64(b)*100.0/255.0)*10) / 10
}

// EncodeDPT9 encodes a float32 into 2-octet KNX floating point representation.
// Formula: Value = (0.01 * M) * 2^E
// Bit 15: Sign, Bits 14..11: Exponent (0..15), Bits 10..0: Mantissa (-2048..2047)
func EncodeDPT9(val float64) ([]byte, error) {
	if val < -671088.64 || val > 670760.96 {
		return nil, errors.New("knx: float value out of DPT9 range")
	}

	sign := uint16(0)
	if val < 0 {
		sign = 0x8000
	}

	// Find exponent E and mantissa M such that val = 0.01 * M * 2^E
	var exp uint16
	mant := val * 100.0

	for mant < -2048.0 || mant > 2047.0 {
		mant /= 2.0
		exp++
		if exp > 15 {
			return nil, errors.New("knx: DPT9 overflow")
		}
	}

	mInt := int16(math.Round(mant))
	encoded := sign | ((exp & 0x0F) << 11) | (uint16(mInt) & 0x07FF)

	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, encoded)
	return buf, nil
}

// DecodeDPT9 decodes 2 bytes of DPT9 into a float64.
func DecodeDPT9(data []byte) (float64, error) {
	if len(data) < 2 {
		return 0, errors.New("knx: need at least 2 bytes for DPT9")
	}

	raw := binary.BigEndian.Uint16(data[0:2])
	sign := (raw & 0x8000) != 0
	exp := (raw >> 11) & 0x0F
	mant := raw & 0x07FF

	var mVal float64
	if sign {
		// 11-bit two's complement negative
		mVal = float64(int16(mant | 0xF800))
	} else {
		mVal = float64(mant)
	}

	val := 0.01 * mVal * math.Pow(2, float64(exp))
	return math.Round(val*100) / 100, nil
}

// EncodeDPT14 encodes a float32 into 4-byte IEEE 754 float.
func EncodeDPT14(val float64) []byte {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, math.Float32bits(float32(val)))
	return buf
}

// DecodeDPT14 decodes 4 bytes into a float64.
func DecodeDPT14(data []byte) (float64, error) {
	if len(data) < 4 {
		return 0, errors.New("knx: need at least 4 bytes for DPT14")
	}
	f := math.Float32frombits(binary.BigEndian.Uint32(data[0:4]))
	return float64(f), nil
}
