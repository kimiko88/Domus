// Package modbus_tcp implements the Modbus TCP protocol adapter for Domus.
package modbus_tcp

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

// Modbus function codes
const (
	FuncReadCoils              byte = 0x01
	FuncReadDiscreteInputs     byte = 0x02
	FuncReadHoldingRegisters   byte = 0x03
	FuncReadInputRegisters     byte = 0x04
	FuncWriteSingleCoil        byte = 0x05
	FuncWriteSingleRegister    byte = 0x06
	FuncWriteMultipleRegisters byte = 0x10
)

// WordOrder defines endianness for 32-bit registers.
type WordOrder string

const (
	BigEndian    WordOrder = "BIG_ENDIAN"
	LittleEndian WordOrder = "LITTLE_ENDIAN"
)

// DataType indicates the format of the register value.
type DataType string

const (
	TypeBool    DataType = "BOOL"
	TypeInt16   DataType = "INT16"
	TypeUint16  DataType = "UINT16"
	TypeInt32   DataType = "INT32"
	TypeUint32  DataType = "UINT32"
	TypeFloat32 DataType = "FLOAT32"
)

// MBAPHeader represents the Modbus Application Protocol header (7 bytes).
type MBAPHeader struct {
	TransactionID uint16
	ProtocolID    uint16 // Always 0 for Modbus TCP
	Length        uint16 // Number of following bytes (UnitID + PDU)
	UnitID        uint8  // Slave / device address
}

// EncodeMBAP serializes the MBAP header to 7 bytes.
func (h MBAPHeader) Encode() []byte {
	b := make([]byte, 7)
	binary.BigEndian.PutUint16(b[0:2], h.TransactionID)
	binary.BigEndian.PutUint16(b[2:4], h.ProtocolID)
	binary.BigEndian.PutUint16(b[4:6], h.Length)
	b[6] = h.UnitID
	return b
}

// DecodeMBAP parses an MBAP header from 7 bytes.
func DecodeMBAP(b []byte) (MBAPHeader, error) {
	if len(b) < 7 {
		return MBAPHeader{}, errors.New("modbus: header too short, need 7 bytes")
	}
	return MBAPHeader{
		TransactionID: binary.BigEndian.Uint16(b[0:2]),
		ProtocolID:    binary.BigEndian.Uint16(b[2:4]),
		Length:        binary.BigEndian.Uint16(b[4:6]),
		UnitID:        b[6],
	}, nil
}

// BuildReadHoldingRegisters creates a Modbus TCP frame to read holding registers.
func BuildReadHoldingRegisters(txID uint16, unitID uint8, address uint16, quantity uint16) []byte {
	return buildReadRequest(txID, unitID, FuncReadHoldingRegisters, address, quantity)
}

// BuildReadInputRegisters creates a Modbus TCP frame to read input registers.
func BuildReadInputRegisters(txID uint16, unitID uint8, address uint16, quantity uint16) []byte {
	return buildReadRequest(txID, unitID, FuncReadInputRegisters, address, quantity)
}

// BuildReadCoils creates a Modbus TCP frame to read coils.
func BuildReadCoils(txID uint16, unitID uint8, address uint16, quantity uint16) []byte {
	return buildReadRequest(txID, unitID, FuncReadCoils, address, quantity)
}

// BuildWriteSingleCoil creates a Modbus TCP frame to write a single coil (0xFF00 for ON, 0x0000 for OFF).
func BuildWriteSingleCoil(txID uint16, unitID uint8, address uint16, on bool) []byte {
	var val uint16
	if on {
		val = 0xFF00
	}
	pdu := make([]byte, 5)
	pdu[0] = FuncWriteSingleCoil
	binary.BigEndian.PutUint16(pdu[1:3], address)
	binary.BigEndian.PutUint16(pdu[3:5], val)

	hdr := MBAPHeader{
		TransactionID: txID,
		ProtocolID:    0,
		Length:        uint16(1 + len(pdu)),
		UnitID:        unitID,
	}
	return append(hdr.Encode(), pdu...)
}

// BuildWriteSingleRegister creates a Modbus TCP frame to write a 16-bit register.
func BuildWriteSingleRegister(txID uint16, unitID uint8, address uint16, value uint16) []byte {
	pdu := make([]byte, 5)
	pdu[0] = FuncWriteSingleRegister
	binary.BigEndian.PutUint16(pdu[1:3], address)
	binary.BigEndian.PutUint16(pdu[3:5], value)

	hdr := MBAPHeader{
		TransactionID: txID,
		ProtocolID:    0,
		Length:        uint16(1 + len(pdu)),
		UnitID:        unitID,
	}
	return append(hdr.Encode(), pdu...)
}

func buildReadRequest(txID uint16, unitID uint8, funcCode byte, address uint16, quantity uint16) []byte {
	pdu := make([]byte, 5)
	pdu[0] = funcCode
	binary.BigEndian.PutUint16(pdu[1:3], address)
	binary.BigEndian.PutUint16(pdu[3:5], quantity)

	hdr := MBAPHeader{
		TransactionID: txID,
		ProtocolID:    0,
		Length:        uint16(1 + len(pdu)),
		UnitID:        unitID,
	}
	return append(hdr.Encode(), pdu...)
}

// ParseReadResponse parses the returned bytes from a read coils or registers request.
func ParseReadResponse(expectedFunc byte, payload []byte) ([]byte, error) {
	if len(payload) < 2 {
		return nil, errors.New("modbus: response too short")
	}
	funcCode := payload[0]
	// Modbus error response has high bit set (funcCode | 0x80)
	if funcCode == (expectedFunc | 0x80) {
		errCode := payload[1]
		return nil, fmt.Errorf("modbus exception response code: 0x%02X", errCode)
	}
	if funcCode != expectedFunc {
		return nil, fmt.Errorf("modbus unexpected function code: got 0x%02X, want 0x%02X", funcCode, expectedFunc)
	}
	byteCount := int(payload[1])
	if len(payload) < 2+byteCount {
		return nil, fmt.Errorf("modbus payload truncated: got %d bytes, need %d", len(payload)-2, byteCount)
	}
	return payload[2 : 2+byteCount], nil
}

// DecodeRegisterValue decodes raw register bytes into a typed float64 or bool representation.
func DecodeRegisterValue(raw []byte, dataType DataType, order WordOrder, scale float64) (any, error) {
	if scale == 0 {
		scale = 1.0
	}

	switch dataType {
	case TypeBool:
		if len(raw) < 1 {
			return false, errors.New("modbus: missing coil data")
		}
		return (raw[0] & 0x01) != 0, nil

	case TypeInt16:
		if len(raw) < 2 {
			return 0.0, errors.New("modbus: need 2 bytes for int16")
		}
		val := int16(binary.BigEndian.Uint16(raw[0:2]))
		return float64(val) * scale, nil

	case TypeUint16:
		if len(raw) < 2 {
			return 0.0, errors.New("modbus: need 2 bytes for uint16")
		}
		val := binary.BigEndian.Uint16(raw[0:2])
		return float64(val) * scale, nil

	case TypeInt32:
		if len(raw) < 4 {
			return 0.0, errors.New("modbus: need 4 bytes for int32")
		}
		var raw32 uint32
		if order == LittleEndian {
			// Word-swapped: lower word first
			raw32 = (uint32(binary.BigEndian.Uint16(raw[2:4])) << 16) | uint32(binary.BigEndian.Uint16(raw[0:2]))
		} else {
			raw32 = binary.BigEndian.Uint32(raw[0:4])
		}
		return float64(int32(raw32)) * scale, nil

	case TypeUint32:
		if len(raw) < 4 {
			return 0.0, errors.New("modbus: need 4 bytes for uint32")
		}
		var raw32 uint32
		if order == LittleEndian {
			raw32 = (uint32(binary.BigEndian.Uint16(raw[2:4])) << 16) | uint32(binary.BigEndian.Uint16(raw[0:2]))
		} else {
			raw32 = binary.BigEndian.Uint32(raw[0:4])
		}
		return float64(raw32) * scale, nil

	case TypeFloat32:
		if len(raw) < 4 {
			return 0.0, errors.New("modbus: need 4 bytes for float32")
		}
		var raw32 uint32
		if order == LittleEndian {
			raw32 = (uint32(binary.BigEndian.Uint16(raw[2:4])) << 16) | uint32(binary.BigEndian.Uint16(raw[0:2]))
		} else {
			raw32 = binary.BigEndian.Uint32(raw[0:4])
		}
		f := math.Float32frombits(raw32)
		return float64(f) * scale, nil

	default:
		return 0.0, fmt.Errorf("modbus: unsupported data type %s", dataType)
	}
}
