package ota

import (
	"testing"
)

func TestFlasherService_ListAndGet(t *testing.T) {
	flasher := NewFlasherService()

	list := flasher.ListFirmwares()
	if len(list) < 4 {
		t.Fatalf("expected at least 4 flasher firmwares, got %d", len(list))
	}

	fw, err := flasher.GetFirmware("domus_node_esp32")
	if err != nil {
		t.Fatalf("GetFirmware failed: %v", err)
	}
	if fw.ChipFamily != "ESP32" {
		t.Errorf("expected chip family ESP32, got %s", fw.ChipFamily)
	}
	if len(fw.Partitions) < 3 {
		t.Errorf("expected at least 3 partitions, got %d", len(fw.Partitions))
	}
}

func TestFlasherService_GenerateProvisionConfig(t *testing.T) {
	flasher := NewFlasherService()

	cfg, err := flasher.GenerateProvisionConfig(ProvisionRequest{
		FirmwareID: "domus_node_esp32",
		WiFiSSID:   "Casa_WiFi_5G",
		WiFiPass:   "SuperSecretPass123",
		HubURL:     "http://192.168.1.100:8080",
		DeviceName: "Sensore Soggiorno",
	})
	if err != nil {
		t.Fatalf("GenerateProvisionConfig failed: %v", err)
	}

	if cfg == "" {
		t.Errorf("expected non-empty config string")
	}
}
