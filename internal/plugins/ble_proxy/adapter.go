package ble_proxy

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/google/uuid"
)

// BLEAdvertisement represents a raw advertisement packet forwarded by an ESPHome BLE proxy gateway.
type BLEAdvertisement struct {
	GatewayMAC string `json:"gateway_mac"` // ESP32 node MAC
	GatewayName string `json:"gateway_name"`
	DeviceMAC  string `json:"device_mac"`  // Target beacon/sensor MAC
	RSSI       int    `json:"rssi"`
	RawPayload string `json:"raw_payload"` // Hex string
}

// Adapter processes forwarded BLE packets and publishes Domus sensor telemetry and presence updates.
type Adapter struct {
	id     string
	bus    events.Bus
	mu     sync.RWMutex
	active bool
}

// NewAdapter constructs a new BLE proxy adapter.
func NewAdapter(id string, bus events.Bus) *Adapter {
	if id == "" {
		id = "ble_proxy"
	}
	return &Adapter{
		id:  id,
		bus: bus,
	}
}

func (a *Adapter) ID() string {
	return a.id
}

func (a *Adapter) Type() string {
	return "ble_proxy"
}

func (a *Adapter) Start(ctx context.Context) error {
	a.mu.Lock()
	a.active = true
	a.mu.Unlock()

	slog.Info("ble_proxy: adapter started", "id", a.id)
	return nil
}

func (a *Adapter) Stop(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.active = false
	slog.Info("ble_proxy: adapter stopped")
	return nil
}

// IngestAdvertisement processes an incoming BLE advertisement forwarded from an ESPHome proxy.
func (a *Adapter) IngestAdvertisement(ctx context.Context, adv BLEAdvertisement) error {
	rawBytes, err := hex.DecodeString(strings.ReplaceAll(adv.RawPayload, ":", ""))
	if err != nil {
		return fmt.Errorf("ble_proxy: invalid hex payload: %w", err)
	}

	cleanMAC := strings.ToLower(strings.ReplaceAll(adv.DeviceMAC, ":", ""))
	now := time.Now().UTC()

	// 1. Try decoding as BTHome v2
	if sensorData, err := ParseBTHomeV2(rawBytes); err == nil && (sensorData.Temperature != nil || sensorData.Humidity != nil || sensorData.Motion != nil || sensorData.WindowOpen != nil) {
		a.publishBTHomeEntities(ctx, cleanMAC, adv.GatewayName, sensorData, now)
		return nil
	}

	// 2. Try decoding as Apple iBeacon (for micro-location / room presence)
	if beaconData, err := ParseIBeacon(rawBytes, adv.RSSI); err == nil {
		a.publishIBeaconPresence(ctx, cleanMAC, adv.GatewayName, beaconData, now)
		return nil
	}

	return nil
}

func (a *Adapter) publishBTHomeEntities(ctx context.Context, mac, gateway string, data *BTHomeSensorData, now time.Time) {
	if a.bus == nil {
		return
	}

	attrs, _ := json.Marshal(map[string]any{
		"gateway": gateway,
		"mac":     mac,
	})

	if data.Temperature != nil {
		_ = a.bus.Publish(ctx, events.Event{
			ID:         uuid.NewString(),
			Type:       events.EntityStateChanged,
			OccurredAt: now,
			Source:     "ble_proxy",
			Payload: map[string]any{
				"entity_key": fmt.Sprintf("sensor.ble_%s_temperature", mac),
				"state":      fmt.Sprintf("%.2f", *data.Temperature),
				"attributes": string(attrs),
				"available":  true,
			},
		})
	}

	if data.Humidity != nil {
		_ = a.bus.Publish(ctx, events.Event{
			ID:         uuid.NewString(),
			Type:       events.EntityStateChanged,
			OccurredAt: now,
			Source:     "ble_proxy",
			Payload: map[string]any{
				"entity_key": fmt.Sprintf("sensor.ble_%s_humidity", mac),
				"state":      fmt.Sprintf("%.1f", *data.Humidity),
				"attributes": string(attrs),
				"available":  true,
			},
		})
	}

	if data.Motion != nil {
		state := "off"
		if *data.Motion {
			state = "on"
		}
		_ = a.bus.Publish(ctx, events.Event{
			ID:         uuid.NewString(),
			Type:       events.EntityStateChanged,
			OccurredAt: now,
			Source:     "ble_proxy",
			Payload: map[string]any{
				"entity_key": fmt.Sprintf("binary_sensor.ble_%s_motion", mac),
				"state":      state,
				"attributes": string(attrs),
				"available":  true,
			},
		})
	}
}

func (a *Adapter) publishIBeaconPresence(ctx context.Context, mac, gateway string, data *IBeaconData, now time.Time) {
	if a.bus == nil {
		return
	}

	_ = a.bus.Publish(ctx, events.Event{
		ID:         uuid.NewString(),
		Type:       events.EntityStateChanged,
		OccurredAt: now,
		Source:     "ble_proxy",
		Payload: map[string]any{
			"entity_key": fmt.Sprintf("sensor.ibeacon_%s_distance", mac),
			"state":      fmt.Sprintf("%.2f", data.EstimatedDistM),
			"attributes": fmt.Sprintf(`{"gateway":"%s","uuid":"%s","major":%d,"minor":%d,"rssi":%d}`, gateway, data.UUID, data.Major, data.Minor, data.RSSI),
			"available":  true,
		},
	})
}
