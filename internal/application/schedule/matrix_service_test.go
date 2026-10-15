package schedule_test

import (
	"context"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	appschedule "github.com/domus-hub/domus/internal/application/schedule"
	domainschedule "github.com/domus-hub/domus/internal/domain/schedule"
	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatrixService_EvaluateAndTransition(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_matrix_service.db")

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
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 100, Workers: 2})

	var lastTargetTemp atomic.Value
	bus.Subscribe(events.CommandRequested, func(ctx context.Context, evt events.Event) error {
		if p, ok := evt.Payload.(map[string]any); ok {
			if params, ok := p["params"].(map[string]any); ok {
				if temp, ok := params["temperature"].(float64); ok {
					lastTargetTemp.Store(temp)
				}
			}
		}
		return nil
	})

	svc := appschedule.NewMatrixService(repo, bus, nil)

	grid := domainschedule.WeeklyGrid{
		"monday": []domainschedule.TimeSlot{
			{Start: "06:30", End: "08:30", Value: 21.5, State: "comfort"},
			{Start: "17:00", End: "22:00", Value: 20.0, State: "evening"},
		},
	}

	ws := &domainschedule.WeeklySchedule{
		ID:               "ws-test-1",
		Name:             "Termostato Camera",
		TargetEntityID:   "climate.bedroom_trv",
		TargetService:    "set_temperature",
		Grid:             grid,
		CurrentSlotState: "off",
	}

	require.NoError(t, svc.Create(ctx, ws))

	// Test 1: Monday at 07:15 (inside morning comfort slot: 21.5)
	// 2026-10-19 was a Monday
	mondayMorning := time.Date(2026, 10, 19, 7, 15, 0, 0, time.UTC)
	require.NoError(t, svc.EvaluateAll(ctx, mondayMorning))

	assert.Eventually(t, func() bool {
		val := lastTargetTemp.Load()
		if val == nil {
			return false
		}
		return val.(float64) == 21.5
	}, 1*time.Second, 20*time.Millisecond)

	updated, err := svc.Get(ctx, "ws-test-1")
	require.NoError(t, err)
	assert.Equal(t, "comfort", updated.CurrentSlotState)

	// Test 2: Monday at 12:00 (outside any slot -> off)
	mondayNoon := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	require.NoError(t, svc.EvaluateAll(ctx, mondayNoon))

	updatedNoon, err := svc.Get(ctx, "ws-test-1")
	require.NoError(t, err)
	assert.Equal(t, "off", updatedNoon.CurrentSlotState)
}
