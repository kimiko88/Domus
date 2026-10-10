package ota_test

import (
	"context"
	"testing"

	"github.com/domus-hub/domus/internal/application/ota"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOTAService_ListAndUpgrade(t *testing.T) {
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 64, Workers: 1})
	svc := ota.NewService(bus)
	ctx := context.Background()

	devices, err := svc.ListDevices(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, devices)

	// Find device with update
	var target *ota.DeviceOTA
	for _, d := range devices {
		if d.HasUpdate {
			target = d
			break
		}
	}
	require.NotNil(t, target)

	// Trigger 1-click upgrade
	updated, err := svc.TriggerUpgrade(ctx, target.ID)
	require.NoError(t, err)
	assert.False(t, updated.HasUpdate)
	assert.Equal(t, target.LatestVersion, updated.CurrentVersion)
	assert.Equal(t, 100, updated.Progress)
}
