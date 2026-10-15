package sqlite_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/domain/climate"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClimateCompensatorRepository(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_climate.db")

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

	comp1 := &climate.ClimateCompensation{
		ID:                  "comp-1",
		TRVEntityKey:        "climate.living_room_trv",
		RoomTempSensorKey:   "sensor.living_room_temperature",
		WindowSensorKey:     "binary_sensor.living_room_window",
		TargetTemperature:   21.0,
		DeltaFactor:         1.5,
		CompensatedSetpoint: 22.5,
		WindowOpen:          false,
		Enabled:             true,
	}

	// 1. Save new
	require.NoError(t, repo.Save(ctx, comp1))

	// 2. GetByID
	fetched, err := repo.GetByID(ctx, "comp-1")
	require.NoError(t, err)
	assert.Equal(t, "climate.living_room_trv", fetched.TRVEntityKey)
	assert.Equal(t, 21.0, fetched.TargetTemperature)
	assert.Equal(t, 1.5, fetched.DeltaFactor)
	assert.False(t, fetched.WindowOpen)

	// 3. GetByTRV
	byTRV, err := repo.GetByTRV(ctx, "climate.living_room_trv")
	require.NoError(t, err)
	assert.Equal(t, "comp-1", byTRV.ID)

	// 4. Upsert (update on conflict)
	comp1.TargetTemperature = 22.0
	comp1.WindowOpen = true
	comp1.CompensatedSetpoint = 5.0
	require.NoError(t, repo.Save(ctx, comp1))

	updated, err := repo.GetByID(ctx, "comp-1")
	require.NoError(t, err)
	assert.Equal(t, 22.0, updated.TargetTemperature)
	assert.True(t, updated.WindowOpen)
	assert.Equal(t, 5.0, updated.CompensatedSetpoint)

	// 5. List
	all, err := repo.List(ctx)
	require.NoError(t, err)
	assert.Len(t, all, 1)

	// 6. Delete
	require.NoError(t, repo.Delete(ctx, "comp-1"))
	_, err = repo.GetByID(ctx, "comp-1")
	assert.ErrorIs(t, err, climate.ErrCompensationNotFound)
}
