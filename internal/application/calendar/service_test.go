package calendar_test

import (
	"context"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/application/calendar"
	"github.com/domus-hub/domus/internal/domain/schedule"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCalendarService_ScheduleEvaluation(t *testing.T) {
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 64, Workers: 1})
	repo := calendar.NewMemoryRepository()
	svc := calendar.NewService(repo, bus)
	ctx := context.Background()

	grid := schedule.WeeklyGrid{
		"monday": []schedule.TimeSlot{
			{Start: "06:30", End: "08:30", Value: 21.0},
			{Start: "18:00", End: "22:00", Value: 20.5},
		},
	}

	sched, err := svc.CreateSchedule(ctx, "Cronotermostato Salotto", "climate.living_room", grid)
	require.NoError(t, err)

	// Test Monday at 07:15 -> Active slot 21.0
	// 2026-10-12 is a Monday
	mondayMorning := time.Date(2026, 10, 12, 7, 15, 0, 0, time.UTC)
	slot, active, err := svc.EvaluateSchedule(ctx, sched.ID, mondayMorning)
	require.NoError(t, err)
	assert.True(t, active)
	require.NotNil(t, slot)
	assert.Equal(t, 21.0, slot.Value)

	// Test Monday at 12:00 -> No active slot
	mondayNoon := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	slot, active, err = svc.EvaluateSchedule(ctx, sched.ID, mondayNoon)
	require.NoError(t, err)
	assert.False(t, active)
	assert.Nil(t, slot)
}

func TestCalendarService_ICalImport(t *testing.T) {
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 64, Workers: 1})
	repo := calendar.NewMemoryRepository()
	svc := calendar.NewService(repo, bus)
	ctx := context.Background()

	icsData := `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
SUMMARY:Lavoro da Remoto
DESCRIPTION:Giornata di smart working
DTSTART:20261015T090000Z
DTEND:20261015T180000Z
LOCATION:Ufficio Casa
END:VEVENT
END:VCALENDAR`

	imported, err := svc.ImportICal(ctx, icsData)
	require.NoError(t, err)
	assert.Equal(t, 1, imported)

	events, err := svc.ListEvents(ctx, time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.Len(t, events, 1)
	assert.Equal(t, "Lavoro da Remoto", events[0].Summary)
	assert.Equal(t, "Ufficio Casa", events[0].Location)
}
