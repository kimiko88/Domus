package modbus_tcp_test

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/domus-hub/domus/internal/plugins/modbus_tcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMBAPHeader_EncodeDecode(t *testing.T) {
	orig := modbus_tcp.MBAPHeader{
		TransactionID: 1042,
		ProtocolID:    0,
		Length:        6,
		UnitID:        1,
	}

	encoded := orig.Encode()
	require.Len(t, encoded, 7)

	decoded, err := modbus_tcp.DecodeMBAP(encoded)
	require.NoError(t, err)
	assert.Equal(t, orig, decoded)
}

func TestBuildReadRequests(t *testing.T) {
	// 1. Read Holding Registers
	frame := modbus_tcp.BuildReadHoldingRegisters(10, 1, 40001, 2)
	require.Len(t, frame, 12) // 7 MBAP + 5 PDU
	assert.Equal(t, modbus_tcp.FuncReadHoldingRegisters, frame[7])
	assert.Equal(t, uint16(40001), binary.BigEndian.Uint16(frame[8:10]))
	assert.Equal(t, uint16(2), binary.BigEndian.Uint16(frame[10:12]))

	// 2. Read Coils
	coilFrame := modbus_tcp.BuildReadCoils(11, 2, 100, 1)
	require.Len(t, coilFrame, 12)
	assert.Equal(t, modbus_tcp.FuncReadCoils, coilFrame[7])

	// 3. Write Single Coil ON
	writeCoilOn := modbus_tcp.BuildWriteSingleCoil(12, 1, 5, true)
	require.Len(t, writeCoilOn, 12)
	assert.Equal(t, modbus_tcp.FuncWriteSingleCoil, writeCoilOn[7])
	assert.Equal(t, uint16(0xFF00), binary.BigEndian.Uint16(writeCoilOn[10:12]))

	// 4. Write Single Coil OFF
	writeCoilOff := modbus_tcp.BuildWriteSingleCoil(13, 1, 5, false)
	assert.Equal(t, uint16(0x0000), binary.BigEndian.Uint16(writeCoilOff[10:12]))

	// 5. Write Single Register
	writeReg := modbus_tcp.BuildWriteSingleRegister(14, 1, 40010, 1250)
	assert.Equal(t, modbus_tcp.FuncWriteSingleRegister, writeReg[7])
	assert.Equal(t, uint16(1250), binary.BigEndian.Uint16(writeReg[10:12]))
}

func TestParseReadResponse(t *testing.T) {
	// Normal response: Func 0x03, 4 bytes data
	payload := []byte{0x03, 0x04, 0x00, 0x64, 0x01, 0x2C}
	data, err := modbus_tcp.ParseReadResponse(modbus_tcp.FuncReadHoldingRegisters, payload)
	require.NoError(t, err)
	assert.Equal(t, []byte{0x00, 0x64, 0x01, 0x2C}, data)

	// Exception response: Func 0x83 (0x03 | 0x80), code 0x02 (Illegal Data Address)
	excPayload := []byte{0x83, 0x02}
	_, err = modbus_tcp.ParseReadResponse(modbus_tcp.FuncReadHoldingRegisters, excPayload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "modbus exception response code: 0x02")
}

func TestDecodeRegisterValue(t *testing.T) {
	// Bool / Coil
	val, err := modbus_tcp.DecodeRegisterValue([]byte{0x01}, modbus_tcp.TypeBool, modbus_tcp.BigEndian, 1.0)
	require.NoError(t, err)
	assert.Equal(t, true, val)

	val, err = modbus_tcp.DecodeRegisterValue([]byte{0x00}, modbus_tcp.TypeBool, modbus_tcp.BigEndian, 1.0)
	require.NoError(t, err)
	assert.Equal(t, false, val)

	// Int16 (negative temperature: -15.5 °C with scale 0.1)
	int16Bytes := make([]byte, 2)
	negTemp := int16(-155)
	binary.BigEndian.PutUint16(int16Bytes, uint16(negTemp))
	val, err = modbus_tcp.DecodeRegisterValue(int16Bytes, modbus_tcp.TypeInt16, modbus_tcp.BigEndian, 0.1)
	require.NoError(t, err)
	assert.InDelta(t, -15.5, val.(float64), 0.001)

	// Uint16 (Power: 3500 W)
	uint16Bytes := make([]byte, 2)
	binary.BigEndian.PutUint16(uint16Bytes, 3500)
	val, err = modbus_tcp.DecodeRegisterValue(uint16Bytes, modbus_tcp.TypeUint16, modbus_tcp.BigEndian, 1.0)
	require.NoError(t, err)
	assert.Equal(t, 3500.0, val)

	// Float32 (Inverter power: 4500.5 W)
	expectedFloat := float32(4500.5)
	bits := math.Float32bits(expectedFloat)
	fBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(fBytes, bits)

	val, err = modbus_tcp.DecodeRegisterValue(fBytes, modbus_tcp.TypeFloat32, modbus_tcp.BigEndian, 1.0)
	require.NoError(t, err)
	assert.InDelta(t, 4500.5, val.(float64), 0.01)

	// Float32 LittleEndian (Word Swap)
	swappedBytes := []byte{fBytes[2], fBytes[3], fBytes[0], fBytes[1]}
	val, err = modbus_tcp.DecodeRegisterValue(swappedBytes, modbus_tcp.TypeFloat32, modbus_tcp.LittleEndian, 1.0)
	require.NoError(t, err)
	assert.InDelta(t, 4500.5, val.(float64), 0.01)
}
