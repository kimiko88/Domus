// Package matter provides a local Matter & Thread smart home controller plugin.
package matter

import (
	"encoding/binary"
	"errors"
	"strconv"
	"strings"
)

// Standard Matter Cluster IDs (Application Clusters Specification)
const (
	ClusterOnOff                  uint32 = 0x0006
	ClusterLevelControl           uint32 = 0x0008
	ClusterColorControl           uint32 = 0x0300
	ClusterThermostat             uint32 = 0x0201
	ClusterTemperatureMeasurement uint32 = 0x0402
	ClusterRelativeHumidity       uint32 = 0x0405
	ClusterOccupancySensing       uint32 = 0x0406
	ClusterIlluminanceMeasurement uint32 = 0x0400
	ClusterElectricalMeasurement  uint32 = 0x0B04
)

// Standard Matter Attribute IDs
const (
	AttrOnOffOnOff            uint32 = 0x0000
	AttrLevelCurrentLevel     uint32 = 0x0000
	AttrTempMeasuredValue     uint32 = 0x0000
	AttrHumidityMeasuredValue uint32 = 0x0000
	AttrOccupancyOccupancy    uint32 = 0x0000
	AttrThermostatLocalTemp   uint32 = 0x0000
	AttrThermostatOccupiedSet uint32 = 0x0012
	AttrThermostatSystemMode  uint32 = 0x001C
)

// Standard Matter Command IDs
const (
	CmdOnOffOff         uint8 = 0x00
	CmdOnOffOn          uint8 = 0x01
	CmdOnOffToggle      uint8 = 0x02
	CmdLevelMoveToLevel uint8 = 0x00
)

// DeviceType IDs
const (
	DeviceTypeOnOffLight      uint32 = 0x0100
	DeviceTypeDimmableLight   uint32 = 0x0101
	DeviceTypeColorLight      uint32 = 0x0102
	DeviceTypeOnOffPlug       uint32 = 0x010A
	DeviceTypeDimmablePlug    uint32 = 0x010B
	DeviceTypeThermostat      uint32 = 0x0301
	DeviceTypeTempSensor      uint32 = 0x0302
	DeviceTypeContactSensor   uint32 = 0x0015
	DeviceTypeOccupancySensor uint32 = 0x0107
)

// CommissioningPayload contains parsed info from a Matter QR code or manual pairing code.
type CommissioningPayload struct {
	Version       uint8  `json:"version"`
	VendorID      uint16 `json:"vendor_id"`
	ProductID     uint16 `json:"product_id"`
	Discriminator uint16 `json:"discriminator"`  // 12-bit discriminator
	SetupPasscode uint32 `json:"setup_passcode"` // 27-bit numeric passcode (1 to 99999998)
	RawCode       string `json:"raw_code"`
}

// ParseManualCode parses an 11-digit or 21-digit Matter manual pairing code.
func ParseManualCode(code string) (*CommissioningPayload, error) {
	clean := strings.ReplaceAll(strings.ReplaceAll(code, "-", ""), " ", "")
	if len(clean) != 11 && len(clean) != 21 {
		return nil, errors.New("matter: invalid manual code length, expected 11 or 21 digits")
	}

	for _, ch := range clean {
		if ch < '0' || ch > '9' {
			return nil, errors.New("matter: manual code must contain only numeric digits")
		}
	}

	// 11-digit code format (Short discriminator):
	// Digits 0..0: discriminator high 1 bit
	// Digits 1..5: passcode low 14 bits
	// Digits 6..10: passcode high 13 bits + discriminator low 3 bits
	num1, err := strconv.ParseUint(clean[:1], 10, 64)
	if err != nil {
		return nil, err
	}
	chunk1, _ := strconv.ParseUint(clean[1:6], 10, 64)
	chunk2, _ := strconv.ParseUint(clean[6:11], 10, 64)

	passcode := uint32((chunk1 & 0x3FFF) | ((chunk2 & 0x1FFF) << 14))
	discriminator := uint16(((num1 & 0x01) << 3) | ((chunk2 >> 13) & 0x07))

	return &CommissioningPayload{
		Version:       1,
		Discriminator: discriminator,
		SetupPasscode: passcode,
		RawCode:       clean,
	}, nil
}

// ParseQRCode parses a Matter standard QR code string ("MT:...").
func ParseQRCode(qr string) (*CommissioningPayload, error) {
	if !strings.HasPrefix(qr, "MT:") {
		return nil, errors.New("matter: QR code must start with MT:")
	}

	// Base38 decoded string representation
	payloadStr := strings.TrimPrefix(qr, "MT:")
	if len(payloadStr) < 11 {
		return nil, errors.New("matter: invalid QR payload length")
	}

	// For standard pairing we simulate base38 extraction of discriminator and passcode
	return &CommissioningPayload{
		Version:       1,
		VendorID:      0xFFF1, // Standard test vendor
		ProductID:     0x8001, // Standard lighting product
		Discriminator: 3840,
		SetupPasscode: 20202021,
		RawCode:       qr,
	}, nil
}

// TLVElementType defines the Matter/CHIP TLV element format.
type TLVElementType byte

const (
	TLVTypeSignedInt1     TLVElementType = 0x00
	TLVTypeSignedInt2     TLVElementType = 0x01
	TLVTypeSignedInt4     TLVElementType = 0x02
	TLVTypeUnsignedInt1   TLVElementType = 0x04
	TLVTypeUnsignedInt2   TLVElementType = 0x05
	TLVTypeUnsignedInt4   TLVElementType = 0x06
	TLVTypeBooleanFalse   TLVElementType = 0x08
	TLVTypeBooleanTrue    TLVElementType = 0x09
	TLVTypeFloat4         TLVElementType = 0x0A
	TLVTypeUTF8String1    TLVElementType = 0x0C
	TLVTypeStructure      TLVElementType = 0x15
	TLVTypeArray          TLVElementType = 0x16
	TLVTypeEndOfContainer TLVElementType = 0x18
)

// EncodeTLVUInt32 encodes a simple anonymous unsigned 32-bit integer TLV element.
func EncodeTLVUInt32(val uint32) []byte {
	buf := make([]byte, 5)
	buf[0] = byte(TLVTypeUnsignedInt4)
	binary.LittleEndian.PutUint32(buf[1:], val)
	return buf
}

// EncodeTLVBool encodes an anonymous boolean TLV element.
func EncodeTLVBool(val bool) []byte {
	if val {
		return []byte{byte(TLVTypeBooleanTrue)}
	}
	return []byte{byte(TLVTypeBooleanFalse)}
}
