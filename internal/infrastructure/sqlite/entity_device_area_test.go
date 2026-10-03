// Package sqlite_test provides TDD integration tests for Area, Integration, Device, and Entity repositories.
package sqlite_test

import (
	"context"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/domain/area"
	"github.com/domus-hub/domus/internal/domain/device"
	"github.com/domus-hub/domus/internal/domain/entity"
	"github.com/domus-hub/domus/internal/domain/integration"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAreaRepository(t *testing.T) {
	repos := setupTestDB(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	a := &area.Area{
		ID:        "salon",
		Name:      "Living Room",
		Icon:      "mdi:sofa",
		SortOrder: 1,
		CreatedAt: now,
		UpdatedAt: now,
	}

	require.NoError(t, repos.Areas.Create(ctx, a))

	fetched, err := repos.Areas.GetByID(ctx, "salon")
	require.NoError(t, err)
	assert.Equal(t, "Living Room", fetched.Name)
	assert.Equal(t, "mdi:sofa", fetched.Icon)

	// Update area
	fetched.Name = "Grand Salon"
	require.NoError(t, repos.Areas.Update(ctx, fetched))

	updated, err := repos.Areas.GetByID(ctx, "salon")
	require.NoError(t, err)
	assert.Equal(t, "Grand Salon", updated.Name)

	// List
	list, err := repos.Areas.List(ctx)
	require.NoError(t, err)
	assert.Len(t, list, 1)

	// Delete
	require.NoError(t, repos.Areas.Delete(ctx, "salon"))
	_, err = repos.Areas.GetByID(ctx, "salon")
	assert.ErrorIs(t, err, area.ErrAreaNotFound)
}

func TestIntegrationAndDeviceRepository(t *testing.T) {
	repos := setupTestDB(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)

	// 1. Integration
	in := &integration.Integration{
		ID:        "mqtt_default",
		Type:      "mqtt_generic",
		Name:      "Local Mosquitto",
		Enabled:   true,
		Status:    "connected",
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, repos.Integrations.Create(ctx, in))

	inFetched, err := repos.Integrations.GetByID(ctx, "mqtt_default")
	require.NoError(t, err)
	assert.Equal(t, "connected", inFetched.Status)

	require.NoError(t, repos.Integrations.SetStatus(ctx, "mqtt_default", "error", "connection refused"))
	inFetched, err = repos.Integrations.GetByID(ctx, "mqtt_default")
	require.NoError(t, err)
	assert.Equal(t, "error", inFetched.Status)
	assert.Equal(t, "connection refused", inFetched.LastError)

	// 2. Device
	dev := &device.Device{
		ID:            uuid.NewString(),
		IntegrationID: in.ID,
		ExternalID:    "sonoff_basic_01",
		Name:          "Living Room Main Light Switch",
		Manufacturer:  "Sonoff",
		Model:         "Basic R2",
		Online:        true,
		Metadata:      `{"ip":"192.168.1.120"}`,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	require.NoError(t, repos.Devices.Create(ctx, dev))

	devFetched, err := repos.Devices.GetByID(ctx, dev.ID)
	require.NoError(t, err)
	assert.Equal(t, dev.Name, devFetched.Name)
	assert.True(t, devFetched.Online)

	byExt, err := repos.Devices.GetByExternalID(ctx, in.ID, "sonoff_basic_01")
	require.NoError(t, err)
	assert.Equal(t, dev.ID, byExt.ID)

	// Set online status
	require.NoError(t, repos.Devices.SetOnline(ctx, dev.ID, false))
	devFetched, _ = repos.Devices.GetByID(ctx, dev.ID)
	assert.False(t, devFetched.Online)

	// List devices with filter
	onlineFalse := false
	filtered, err := repos.Devices.List(ctx, device.ListFilter{Online: &onlineFalse})
	require.NoError(t, err)
	assert.Len(t, filtered, 1)
}

func TestEntityRepository(t *testing.T) {
	repos := setupTestDB(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)

	// Setup integration and device
	in := &integration.Integration{
		ID:        "zigbee_bridge",
		Type:      "zigbee2mqtt",
		Name:      "Zigbee Bridge",
		Enabled:   true,
		Status:    "connected",
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, repos.Integrations.Create(ctx, in))

	dev := &device.Device{
		ID:            uuid.NewString(),
		IntegrationID: in.ID,
		ExternalID:    "zigbee_lamp_1",
		Name:          "Desk Lamp",
		Online:        true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	require.NoError(t, repos.Devices.Create(ctx, dev))

	// Create Entity
	ent := &entity.Entity{
		ID:            uuid.NewString(),
		DeviceID:      dev.ID,
		EntityKey:     "light.desk_lamp",
		Domain:        "light",
		Name:          "Desk Lamp",
		Icon:          "mdi:desk-lamp",
		Writable:      true,
		State:         `{"state":"OFF"}`,
		Attributes:    `{"supported_color_modes":["brightness"]}`,
		Available:     true,
		LastChangedAt: now,
		LastUpdatedAt: now,
	}
	require.NoError(t, repos.Entities.Create(ctx, ent))

	// Get by Key
	fetched, err := repos.Entities.GetByKey(ctx, "light.desk_lamp")
	require.NoError(t, err)
	assert.Equal(t, ent.ID, fetched.ID)
	assert.Equal(t, `{"state":"OFF"}`, fetched.State)
	assert.True(t, fetched.Writable)

	// Update State
	newStateTime := now.Add(5 * time.Minute)
	err = repos.Entities.UpdateState(ctx, ent.ID, `{"state":"ON","brightness":200}`, `{"brightness":200}`, true, newStateTime)
	require.NoError(t, err)

	updated, err := repos.Entities.GetByID(ctx, ent.ID)
	require.NoError(t, err)
	assert.Equal(t, `{"state":"ON","brightness":200}`, updated.State)
	assert.Equal(t, `{"brightness":200}`, updated.Attributes)
	assert.Equal(t, newStateTime.Unix(), updated.LastChangedAt.Unix())

	// List filter
	isWritable := true
	list, err := repos.Entities.List(ctx, entity.ListFilter{
		Domain:   "light",
		Writable: &isWritable,
	})
	require.NoError(t, err)
	assert.Len(t, list, 1)
}
