package ota

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/google/uuid"
)

type UpgradeState string

const (
	StateIdle        UpgradeState = "idle"
	StateDownloading UpgradeState = "downloading"
	StateFlashing    UpgradeState = "flashing"
	StateSuccess     UpgradeState = "success"
	StateFailed      UpgradeState = "failed"
)

type DeviceOTA struct {
	ID             string       `json:"id"`
	Name           string       `json:"name"`
	DeviceKey      string       `json:"device_key"`
	Platform       string       `json:"platform"` // "zigbee", "esphome", "matter", "zwave"
	CurrentVersion string       `json:"current_version"`
	LatestVersion  string       `json:"latest_version"`
	HasUpdate      bool         `json:"has_update"`
	UpgradeState   UpgradeState `json:"upgrade_state"`
	Progress       int          `json:"progress"` // 0 to 100%
	ReleaseNotes   string       `json:"release_notes,omitempty"`
	CheckedAt      time.Time    `json:"checked_at"`
}

type Service struct {
	mu      sync.RWMutex
	devices map[string]*DeviceOTA
	bus     events.Bus
}

func NewService(bus events.Bus) *Service {
	s := &Service{
		devices: make(map[string]*DeviceOTA),
		bus:     bus,
	}

	// Pre-seed sample devices with pending OTA firmware
	now := time.Now().UTC()
	s.devices["ota_1"] = &DeviceOTA{
		ID:             "ota_1",
		Name:           "Philips Hue Bulb E27",
		DeviceKey:      "0x0017880108a12345",
		Platform:       "zigbee",
		CurrentVersion: "1.93.7",
		LatestVersion:  "1.104.2",
		HasUpdate:      true,
		UpgradeState:   StateIdle,
		ReleaseNotes:   "Migliorata stabilità del mesh e transizioni colore ultra-fluide",
		CheckedAt:      now,
	}
	s.devices["ota_2"] = &DeviceOTA{
		ID:             "ota_2",
		Name:           "Sensore Clima Salotto (ESPHome)",
		DeviceKey:      "esphome_living_climate",
		Platform:       "esphome",
		CurrentVersion: "2024.9.1",
		LatestVersion:  "2024.10.0",
		HasUpdate:      true,
		UpgradeState:   StateIdle,
		ReleaseNotes:   "Supporto BTHome v2 avanzato e BLE Proxy v3",
		CheckedAt:      now,
	}
	s.devices["ota_3"] = &DeviceOTA{
		ID:             "ota_3",
		Name:           "Shelly Wave 1PM (Z-Wave)",
		DeviceKey:      "zwave_node_14",
		Platform:       "zwave",
		CurrentVersion: "12.01",
		LatestVersion:  "12.01",
		HasUpdate:      false,
		UpgradeState:   StateIdle,
		ReleaseNotes:   "Dispositivo aggiornato all'ultima versione",
		CheckedAt:      now,
	}

	return s
}

func (s *Service) ListDevices(ctx context.Context) ([]*DeviceOTA, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]*DeviceOTA, 0, len(s.devices))
	for _, d := range s.devices {
		list = append(list, d)
	}
	return list, nil
}

func (s *Service) TriggerUpgrade(ctx context.Context, id string) (*DeviceOTA, error) {
	s.mu.Lock()
	dev, ok := s.devices[id]
	if !ok {
		s.mu.Unlock()
		return nil, fmt.Errorf("device not found: %s", id)
	}
	if !dev.HasUpdate {
		s.mu.Unlock()
		return nil, fmt.Errorf("no update available for device %s", dev.Name)
	}

	dev.UpgradeState = StateFlashing
	dev.Progress = 100
	dev.CurrentVersion = dev.LatestVersion
	dev.HasUpdate = false
	s.mu.Unlock()

	if s.bus != nil {
		_ = s.bus.Publish(ctx, events.Event{
			ID:         uuid.NewString(),
			Type:       events.OTAStateChanged,
			Payload:    dev,
			OccurredAt: time.Now().UTC(),
		})
	}

	return dev, nil
}
