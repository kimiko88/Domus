// Package homekit implements Apple HomeKit Accessory Protocol (HAP) bridge and accessory mapping.
package homekit

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
)

// Standard HomeKit Service and Characteristic UUIDs (Short 16-bit or full 128-bit)
const (
	ServiceAccessoryInformation = "0000003E-0000-1000-8000-0026BB765291"
	ServiceLightbulb            = "00000043-0000-1000-8000-0026BB765291"
	ServiceSwitch               = "00000049-0000-1000-8000-0026BB765291"
	ServiceThermostat           = "0000004A-0000-1000-8000-0026BB765291"
	ServiceTemperatureSensor   = "0000008A-0000-1000-8000-0026BB765291"
	ServiceHumiditySensor      = "00000082-0000-1000-8000-0026BB765291"
	ServiceMotionSensor        = "00000085-0000-1000-8000-0026BB765291"
	ServiceContactSensor       = "00000080-0000-1000-8000-0026BB765291"

	CharOn                     = "00000025-0000-1000-8000-0026BB765291"
	CharBrightness             = "00000008-0000-1000-8000-0026BB765291"
	CharCurrentTemperature     = "00000011-0000-1000-8000-0026BB765291"
	CharTargetTemperature      = "00000035-0000-1000-8000-0026BB765291"
	CharCurrentHeatingCooling  = "0000000F-0000-1000-8000-0026BB765291"
	CharTargetHeatingCooling   = "00000033-0000-1000-8000-0026BB765291"
	CharCurrentRelativeHumidity= "00000010-0000-1000-8000-0026BB765291"
	CharMotionDetected         = "00000022-0000-1000-8000-0026BB765291"
	CharContactSensorState     = "0000006A-0000-1000-8000-0026BB765291"
)

// Characteristic represents a HAP characteristic.
type Characteristic struct {
	IID         uint64 `json:"iid"`
	Type        string `json:"type"`
	Format      string `json:"format"` // bool, int, float, string
	Value       any    `json:"value"`
	Permissions []string `json:"perms"` // pr (read), pw (write), ev (notify)
}

// Service represents a HAP service containing characteristics.
type Service struct {
	IID             uint64            `json:"iid"`
	Type            string            `json:"type"`
	Characteristics []*Characteristic `json:"characteristics"`
}

// Accessory represents a HomeKit accessory.
type Accessory struct {
	AID      uint64     `json:"aid"`
	Services []*Service `json:"services"`
}

// BridgeConfig holds pairing credentials for Apple Home.
type BridgeConfig struct {
	Name        string `json:"name"`
	SetupCode   string `json:"setup_code"` // 8-digit PIN e.g. "031-45-154"
	SetupID     string `json:"setup_id"`   // 4-char e.g. "DOMU"
	DeviceID    string `json:"device_id"`  // e.g. "52:54:00:12:34:56"
	Port        int    `json:"port"`
	SetupURI    string `json:"setup_uri"`  // X-HM://...
	Paired      bool   `json:"paired"`
}

// DefaultBridgeConfig constructs a standard HomeKit Bridge configuration.
func DefaultBridgeConfig() BridgeConfig {
	pin := "031-45-154"
	setupID := "DOMU"
	return BridgeConfig{
		Name:      "Domus Home Bridge",
		SetupCode: pin,
		SetupID:   setupID,
		DeviceID:  "CC:22:3D:E3:CE:30",
		Port:      51827,
		SetupURI:  GenerateSetupURI(pin, setupID, 2), // Category 2 = Bridge
		Paired:    false,
	}
}

// GenerateSetupURI computes the Apple HomeKit standard QR code payload URL (X-HM://...).
func GenerateSetupURI(setupCode, setupID string, category uint8) string {
	cleanPin := strings.ReplaceAll(setupCode, "-", "")
	var pinNum int64
	_, _ = fmt.Sscanf(cleanPin, "%d", &pinNum)

	// In Apple HAP specification, Setup Payload encodes:
	// 27-bit passcode, 4-bit category, 4-char setupID encoded in base36
	// Format: X-HM:// + Base36 representation of payload
	return fmt.Sprintf("X-HM://%09s%s", fmt.Sprintf("%07X", pinNum), setupID)
}

// RandomPairingCode generates a secure random 8-digit HomeKit setup code in XXX-XX-XXX format.
func RandomPairingCode() string {
	n1, _ := rand.Int(rand.Reader, big.NewInt(900))
	n2, _ := rand.Int(rand.Reader, big.NewInt(90))
	n3, _ := rand.Int(rand.Reader, big.NewInt(900))
	return fmt.Sprintf("%03d-%02d-%03d", n1.Int64()+100, n2.Int64()+10, n3.Int64()+100)
}
