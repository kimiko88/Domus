package sqlite_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/domain/schedule"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWeeklyScheduleRepository(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_weekly_schedule.db")

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

	repo := sqlite.NewWeeklyScheduleRepository(db)

	grid := schedule.WeeklyGrid{
		"monday": []schedule.TimeSlot{
			{Start: "06:30", End: "08:30", Value: 21.0, State: "comfort"},
			{Start: "17:00", End: "22:30", Value: 20.5, State: "comfort"},
		},
		"tuesday": []schedule.TimeSlot{
			{Start: "06:30", End: "08:30", Value: 21.0, State: "comfort"},
		},
	}

	ws := &schedule.WeeklySchedule{
		ID:               "sched-cronotermostato-1",
		Name:             "Riscaldamento Zona Giorno",
		Icon:             "radiator",
		TargetEntityID:   "climate.living_room_thermostat",
		TargetService:    "set_temperature",
		Grid:             grid,
		CurrentSlotState: "eco",
	}

	// 1. Save
	require.NoError(t, repo.Save(ctx, ws))

	// 2. GetByID
	fetched, err := repo.GetByID(ctx, "sched-cronotermostato-1")
	require.NoError(t, err)
	assert.Equal(t, "Riscaldamento Zona Giorno", fetched.Name)
	assert.Equal(t, "climate.living_room_thermostat", fetched.TargetEntityID)
	assert.Len(t, fetched.Grid["monday"], 2)

	// 3. Update
	ws.CurrentSlotState = "comfort"
	require.NoError(t, repo.Save(ctx, ws))

	updated, err := repo.GetByID(ctx, "sched-cronotermostato-1")
	require.NoError(t, err)
	assert.Equal(t, "comfort", updated.CurrentSlotState)

	// 4. List
	all, err := repo.List(ctx)
	require.NoError(t, err)
	assert.Len(t, all, 1)

	// 5. Delete
	require.NoError(t, repo.Delete(ctx, "sched-cronotermostato-1"))
	_, err = repo.GetByID(ctx, "sched-cronotermostato-1")
	assert.ErrorIs(t, err, schedule.ErrScheduleNotFound)
}
