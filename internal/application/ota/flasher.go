package ota

import (
	"encoding/json"
	"fmt"
	"sync"
)

// FirmwarePartition describes a memory offset and binary component for ESP Web Tools.
type FirmwarePartition struct {
	Offset int    `json:"offset"` // e.g. 0x1000, 0x8000, 0x10000
	Name   string `json:"name"`   // "bootloader", "partitions", "app"
	Path   string `json:"path"`   // Relative download endpoint
}

// WebSerialFirmware represents a pre-compiled firmware image installable directly via browser WebSerial.
type WebSerialFirmware struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	ChipFamily  string              `json:"chip_family"` // "ESP32", "ESP32-C3", "ESP32-S3", "ESP8266"
	Version     string              `json:"version"`
	Description string              `json:"description"`
	BaudRate    int                 `json:"baud_rate"`
	Partitions  []FirmwarePartition `json:"partitions"`
	Features    []string            `json:"features"`
}

// ProvisionRequest holds Wi-Fi and Domus pairing parameters to flash into NVS/EEPROM.
type ProvisionRequest struct {
	FirmwareID string `json:"firmware_id"`
	WiFiSSID   string `json:"wifi_ssid"`
	WiFiPass   string `json:"wifi_pass"`
	HubURL     string `json:"hub_url"`
	DeviceName string `json:"device_name"`
}

// FlasherService manages firmware catalogs and WebSerial manifests for USB flashing.
type FlasherService struct {
	mu        sync.RWMutex
	firmwares map[string]*WebSerialFirmware
}

// NewFlasherService initializes the WebSerial flasher catalog.
func NewFlasherService() *FlasherService {
	f := &FlasherService{
		firmwares: make(map[string]*WebSerialFirmware),
	}

	// 1. ESP32 Domus Multi-Sensor Node (Temp, Hum, Radar mmWave, BLE Proxy)
	f.firmwares["domus_node_esp32"] = &WebSerialFirmware{
		ID:          "domus_node_esp32",
		Name:        "Domus Sensor & BLE Proxy Node (ESP32)",
		ChipFamily:  "ESP32",
		Version:     "2.4.0",
		Description: "Firmware ufficiale Domus: antenna BLE Proxy distribuita, radar di presenza mmWave LD2410 e sensori ambientali I2C.",
		BaudRate:    460800,
		Features:    []string{"BLE Proxy", "Radar mmWave", "BME280 / SHT30", "Zero-Conf mDNS"},
		Partitions: []FirmwarePartition{
			{Offset: 0x1000, Name: "bootloader", Path: "/api/v1/ota/firmware/domus_node_esp32/bootloader.bin"},
			{Offset: 0x8000, Name: "partitions", Path: "/api/v1/ota/firmware/domus_node_esp32/partitions.bin"},
			{Offset: 0xe000, Name: "boot_app0", Path: "/api/v1/ota/firmware/domus_node_esp32/boot_app0.bin"},
			{Offset: 0x10000, Name: "application", Path: "/api/v1/ota/firmware/domus_node_esp32/firmware.bin"},
		},
	}

	// 2. ESP32-C3 Ultra-Compact Mini Relay Node
	f.firmwares["domus_relay_c3"] = &WebSerialFirmware{
		ID:          "domus_relay_c3",
		Name:        "Domus Smart Switch Mini (ESP32-C3 RISC-V)",
		ChipFamily:  "ESP32-C3",
		Version:     "1.8.2",
		Description: "Firmware ultra-leggero per micro-moduli da incasso a 1 o 2 relè, interruttori a pulsante bistabile o invertitore.",
		BaudRate:    460800,
		Features:    []string{"Doppio Relè", "Ingresso Pulsante Fisico", "Misura Potenza HLW8012"},
		Partitions: []FirmwarePartition{
			{Offset: 0x0000, Name: "bootloader", Path: "/api/v1/ota/firmware/domus_relay_c3/bootloader.bin"},
			{Offset: 0x8000, Name: "partitions", Path: "/api/v1/ota/firmware/domus_relay_c3/partitions.bin"},
			{Offset: 0x10000, Name: "application", Path: "/api/v1/ota/firmware/domus_relay_c3/firmware.bin"},
		},
	}

	// 3. Tasmota Universale (Sonoff, Gosund, Athom, Shelly)
	f.firmwares["tasmota_universal"] = &WebSerialFirmware{
		ID:          "tasmota_universal",
		Name:        "Tasmota ESP8266/ESP32 Universale",
		ChipFamily:  "ESP8266",
		Version:     "14.3.0",
		Description: "Firmware open-source universale con autodiscovery per Domus e interfaccia Web locale autonoma.",
		BaudRate:    115200,
		Features:    []string{"Compatibilità Universale", "Web UI Integrata", "HTTP / MQTT"},
		Partitions: []FirmwarePartition{
			{Offset: 0x0000, Name: "application", Path: "/api/v1/ota/firmware/tasmota_universal/tasmota.bin"},
		},
	}

	// 4. WLED NeoPixel Controller
	f.firmwares["wled_controller"] = &WebSerialFirmware{
		ID:          "wled_controller",
		Name:        "WLED NeoPixel Controller (ESP32)",
		ChipFamily:  "ESP32",
		Version:     "0.14.4",
		Description: "Firmware per controllo di strisce LED digitali WS2812B / SK6812, con 100+ effetti dinamici integrati in Domus.",
		BaudRate:    460800,
		Features:    []string{"100+ Effetti RGB", "Sincronizzazione Audio E1.31", "Zero Latenza UDP"},
		Partitions: []FirmwarePartition{
			{Offset: 0x10000, Name: "application", Path: "/api/v1/ota/firmware/wled_controller/wled.bin"},
		},
	}

	return f
}

// ListFirmwares returns the available firmware catalog.
func (f *FlasherService) ListFirmwares() []*WebSerialFirmware {
	f.mu.RLock()
	defer f.mu.RUnlock()

	res := make([]*WebSerialFirmware, 0, len(f.firmwares))
	for _, fw := range f.firmwares {
		res = append(res, fw)
	}
	return res
}

// GetFirmware returns details and partition manifest for a specific firmware.
func (f *FlasherService) GetFirmware(id string) (*WebSerialFirmware, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	fw, ok := f.firmwares[id]
	if !ok {
		return nil, fmt.Errorf("firmware not found: %s", id)
	}
	return fw, nil
}

// GenerateProvisionConfig generates a JSON configuration payload to be written over Serial.
func (f *FlasherService) GenerateProvisionConfig(req ProvisionRequest) (string, error) {
	if req.WiFiSSID == "" {
		return "", fmt.Errorf("wifi ssid is required")
	}

	payload := map[string]string{
		"ssid":        req.WiFiSSID,
		"password":    req.WiFiPass,
		"hub_url":     req.HubURL,
		"device_name": req.DeviceName,
		"protocol":    "domus_native_v1",
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
