package modbus_tcp_test

import (
	"testing"

	"github.com/domus-hub/domus/internal/plugins/modbus_tcp"
)

// FuzzDecodeMBAP tests that MBAP header decoding never panics on arbitrary binary input.
func FuzzDecodeMBAP(f *testing.F) {
	// Seed valid and edge-case frames
	f.Add([]byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x06, 0x01}) // Standard valid 7-byte header
	f.Add([]byte{})                                           // Empty
	f.Add([]byte{0x00, 0x01, 0x00})                           // Truncated (3 bytes)
	f.Add([]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}) // Max values
	f.Add(make([]byte, 256))                                 // Oversized buffer

	f.Fuzz(func(t *testing.T, data []byte) {
		hdr, err := modbus_tcp.DecodeMBAP(data)
		if err == nil {
			// If valid, length must be at least 7 bytes
			if len(data) < 7 {
				t.Fatalf("expected error for data shorter than 7 bytes, got header: %+v", hdr)
			}
		}
	})
}

// FuzzParseReadResponse tests that ParseReadResponse never panics on arbitrary payloads.
func FuzzParseReadResponse(f *testing.F) {
	f.Add(byte(modbus_tcp.FuncReadHoldingRegisters), []byte{0x03, 0x04, 0x00, 0x64, 0x00, 0xC8})
	f.Add(byte(modbus_tcp.FuncReadCoils), []byte{0x01, 0x01, 0x01})
	f.Add(byte(modbus_tcp.FuncReadHoldingRegisters), []byte{0x83, 0x02}) // Modbus exception code 0x02
	f.Add(byte(modbus_tcp.FuncReadInputRegisters), []byte{})
	f.Add(byte(0x00), []byte{0xFF, 0xFF, 0xFF, 0xFF})

	f.Fuzz(func(t *testing.T, expectedFunc byte, payload []byte) {
		// Calling with fuzz-mutated inputs must never panic or crash
		_, _ = modbus_tcp.ParseReadResponse(expectedFunc, payload)
	})
}

// FuzzDecodeRegisterValue tests decoding raw register bytes into typed values.
func FuzzDecodeRegisterValue(f *testing.F) {
	f.Add([]byte{0x01}, string(modbus_tcp.TypeBool), string(modbus_tcp.BigEndian), 1.0)
	f.Add([]byte{0x00}, string(modbus_tcp.TypeBool), string(modbus_tcp.BigEndian), 1.0)
	f.Add([]byte{0x04, 0xD2}, string(modbus_tcp.TypeInt16), string(modbus_tcp.BigEndian), 0.1)
	f.Add([]byte{0x04, 0xD2}, string(modbus_tcp.TypeUint16), string(modbus_tcp.BigEndian), 1.0)
	f.Add([]byte{0x00, 0x01, 0x86, 0xA0}, string(modbus_tcp.TypeInt32), string(modbus_tcp.BigEndian), 1.0)
	f.Add([]byte{0x00, 0x01, 0x86, 0xA0}, string(modbus_tcp.TypeUint32), string(modbus_tcp.BigEndian), 1.0)
	f.Add([]byte{0x41, 0xC8, 0x00, 0x00}, string(modbus_tcp.TypeFloat32), string(modbus_tcp.BigEndian), 1.0)
	f.Add([]byte{0x00, 0x00, 0x41, 0xC8}, string(modbus_tcp.TypeFloat32), string(modbus_tcp.LittleEndian), 1.0)
	f.Add([]byte{}, "UNKNOWN", "UNKNOWN", 0.0)

	f.Fuzz(func(t *testing.T, raw []byte, dataTypeStr string, orderStr string, scale float64) {
		dataType := modbus_tcp.DataType(dataTypeStr)
		order := modbus_tcp.WordOrder(orderStr)

		// Must handle any slice length, unknown type, or invalid order safely without panicking
		_, _ = modbus_tcp.DecodeRegisterValue(raw, dataType, order, scale)
	})
}
