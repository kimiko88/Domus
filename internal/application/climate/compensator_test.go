package climate_test

import (
	"context"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	appclimate "github.com/domus-hub/domus/internal/application/climate"
	domainclimate "github.com/domus-hub/domus/internal/domain/climate"
	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClimateCompensatorService_Workflow(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_climate_service.db")

	db, err := sqlite.Open(config.DatabaseConfig{
		Path:          dbPath,
		WALMode:       true,
		BusyTimeoutMs: 5000,
		MaxOpenConns:  1,
	})
	require.NoError(t, err)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	require.NoError(t, sqlite.Migrate(ctx, db))

	repo := sqlite.NewClimateCompensatorRepository(db)
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 100, Workers: 2})

	var lastCommandTemp atomic.Value
	bus.Subscribe(events.CommandRequested, func(ctx context.Context, evt events.Event) error {
		if p, ok := evt.Payload.(map[string]any); ok {
			if params, ok := p["params"].(map[string]any); ok {
				if temp, ok := params["temperature"].(float64); ok {
					lastCommandTemp.Store(temp)
				}
			}
		}
		return nil
	})

	svc, err := appclimate.NewService(repo, bus, nil)
	require.NoError(t, err)

	comp := &domainclimate.ClimateCompensation{
		ID:                  "comp-bt-1",
		TRVEntityKey:        "climate.living_room_valve",
		RoomTempSensorKey:   "sensor.living_room_temp",
		WindowSensorKey:     "binary_sensor.living_room_window",
		TargetTemperature:   21.0,
		DeltaFactor:         1.5,
		CompensatedSetpoint: 21.0,
		WindowOpen:          false,
		Enabled:             true,
	}

	require.NoError(t, svc.Save(ctx, comp))

	// 1. External room sensor drops to 19.0°C (Delta = 21.0 - 19.0 = +2.0; Compensated = 21.0 + 2.0*1.5 = 24.0)
	err = bus.Publish(ctx, events.Event{
		ID:         "evt-1",
		Type:       events.EntityStateChanged,
		OccurredAt: time.Now().UTC(),
		Payload: map[string]any{
			"entity_id": "sensor.living_room_temp",
			"new_state": "19.0",
		},
	})
	require.NoError(t, err)

	// Wait for bus dispatch
	assert.Eventually(t, func() bool {
		val := lastCommandTemp.Load()
		if val == nil {
			return false
		}
		return val.(float64) == 24.0
	}, 1*time.Second, 20*time.Millisecond)

	// Verify repo was updated with new compensated setpoint
	updated, err := svc.GetByID(ctx, "comp-bt-1")
	require.NoError(t, err)
	assert.Equal(t, 24.0, updated.CompensatedSetpoint)
	assert.False(t, updated.WindowOpen)

	// 2. Window opens -> Immediate shutoff (5.0°C frost protection)
	err = bus.Publish(ctx, events.Event{
		ID:         "evt-2",
		Type:       events.EntityStateChanged,
		OccurredAt: time.Now().UTC(),
		Payload: map[string]any{
			"entity_id": "binary_sensor.living_room_window",
			"new_state": "on",
		},
	})
	require.NoError(t, err)

	assert.Eventually(t, func() bool {
		val := lastCommandTemp.Load()
		if val == nil {
			return false
		}
		return val.(float64) == 5.0
	}, 1*time.Second, 20*time.Millisecond)

	updatedWindow, err := svc.GetByID(ctx, "comp-bt-1")
	require.NoError(t, err)
	assert.True(t, updatedWindow.WindowOpen)
	assert.Equal(t, 5.0, updatedWindow.CompensatedSetpoint)

	// 3. Window closes -> Restores heating
	err = bus.Publish(ctx, events.Event{
		ID:         "evt-3",
		Type:       events.EntityStateChanged,
		OccurredAt: time.Now().UTC(),
		Payload: map[string]any{
			"entity_id": "binary_sensor.living_room_window",
			"new_state": "off",
		},
	})
	require.NoError(t, err)

	assert.Eventually(t, func() bool {
		val := lastCommandTemp.Load()
		if val == nil {
			return false
		}
		return val.(float64) == 21.0
	}, 1*time.Second, 20*time.Millisecond)
}
