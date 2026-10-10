package ble_proxy

import (
	"context"
	"testing"

	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBTHomeV2_Decoder(t *testing.T) {
	// Sample BTHome v2 packet:
	// 0x40 (header) + 0x02 (temp id) + 0xE8 0x08 (22.80 °C = 2280) + 0x03 (hum id) + 0x90 0x19 (65.44 % = 6544)
	payload := []byte{0x40, 0x02, 0xE8, 0x08, 0x03, 0x90, 0x19}
	data, err := ParseBTHomeV2(payload)
	require.NoError(t, err)

	require.NotNil(t, data.Temperature)
	assert.InDelta(t, 22.80, *data.Temperature, 0.01)

	require.NotNil(t, data.Humidity)
	assert.InDelta(t, 65.44, *data.Humidity, 0.01)
}

func TestIBeacon_Decoder(t *testing.T) {
	// Standard Apple iBeacon payload (25 bytes)
	// Apple header: 0x4C 0x00 0x02 0x15
	// Proximity UUID: 16 bytes
	// Major: 0x00 0x0A (10)
	// Minor: 0x00 0x14 (20)
	// Measured Power: 0xC5 (-59 dBm)
	payload := []byte{
		0x4C, 0x00, 0x02, 0x15,
		0xE2, 0xC5, 0x6D, 0xB5, 0xDF, 0xFB, 0x48, 0xD2, 0xB0, 0x60, 0xD0, 0xF5, 0xA7, 0x10, 0x96, 0xE0,
		0x00, 0x0A,
		0x00, 0x14,
		0xC5,
	}

	beacon, err := ParseIBeacon(payload, -65)
	require.NoError(t, err)

	assert.Equal(t, uint16(10), beacon.Major)
	assert.Equal(t, uint16(20), beacon.Minor)
	assert.Equal(t, int8(-59), beacon.MeasuredPower)
	assert.Greater(t, beacon.EstimatedDistM, 0.5)
}

func TestBLEProxyAdapter_Ingestion(t *testing.T) {
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 64, Workers: 1})
	defer bus.Close()

	adapter := NewAdapter("ble_proxy", bus)
	ctx := context.Background()
	require.NoError(t, adapter.Start(ctx))
	defer adapter.Stop(ctx)

	// Ingest hex BTHome packet
	adv := BLEAdvertisement{
		GatewayMAC:  "A4:CF:12:88:99:AA",
		GatewayName: "esp32_hallway",
		DeviceMAC:   "A4:C1:38:12:34:56",
		RSSI:        -58,
		RawPayload:  "4002E8080390192101", // Temp 22.8°C, Hum 65.44%, Motion: true
	}

	err := adapter.IngestAdvertisement(ctx, adv)
	require.NoError(t, err)
}
