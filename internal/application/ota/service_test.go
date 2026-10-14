package ota_test

import (
	"context"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/application/ota"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOTAService_ListAndUpgrade(t *testing.T) {
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 64, Workers: 1})
	repo := ota.NewMemoryRepository()
	ctx := context.Background()

	testDev := &ota.DeviceOTA{
		ID:             "ota_test_1",
		Name:           "Philips Hue Bulb Test",
		DeviceKey:      "0x0017880108a12345",
		Platform:       "zigbee",
		CurrentVersion: "1.93.7",
		LatestVersion:  "1.104.2",
		HasUpdate:      true,
		UpgradeState:   ota.StateIdle,
		CheckedAt:      time.Now().UTC(),
	}
	require.NoError(t, repo.SaveDevice(ctx, testDev))

	svc := ota.NewService(repo, bus)

	devices, err := svc.ListDevices(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, devices)

	// Trigger 1-click upgrade
	updated, err := svc.TriggerUpgrade(ctx, testDev.ID)
	require.NoError(t, err)
	assert.False(t, updated.HasUpdate)
	assert.Equal(t, testDev.LatestVersion, updated.CurrentVersion)
	assert.Equal(t, 100, updated.Progress)
}
