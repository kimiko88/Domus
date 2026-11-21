package sqlite_test

import (
	"context"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/domain/climate_history"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	"github.com/stretchr/testify/require"
)

func TestClimateHistoryRepositorySettingsSamplesAndTieredAggregation(t *testing.T) {
	ctx := context.Background()
	db, err := sqlite.Open(config.DatabaseConfig{Path: ":memory:", WALMode: false, BusyTimeoutMs: 1000})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, sqlite.Migrate(ctx, db))

	repo := sqlite.NewClimateHistoryRepository(db)
	settings, err := repo.GetSettings(ctx)
	require.NoError(t, err)
	require.Equal(t, 10, settings.IntervalMinutes)
	settings.IntervalMinutes = 5
	require.NoError(t, repo.SaveSettings(ctx, settings))
	settings, err = repo.GetSettings(ctx)
	require.NoError(t, err)
	require.Equal(t, 5, settings.IntervalMinutes)

	now := time.Now().UTC().Truncate(time.Second)
	tempA, tempB := 19.5, 21.5
	humA, humB := 48.0, 56.0
	tempC, humC := 23.0, 60.0
	target := 20.0
	samples := []climate_history.Sample{
		{EntityID:"sensor-1", EntityKey:"sensor.bedroom_temperature", EntityName:"Bedroom temperature", AreaID:"bedroom", RecordedAt:now.Truncate(time.Hour).Add(-8*24*time.Hour + 10*time.Minute), Temperature:&tempA, Humidity:&humA},
		{EntityID:"sensor-1", EntityKey:"sensor.bedroom_temperature", EntityName:"Bedroom temperature", AreaID:"bedroom", RecordedAt:now.Truncate(time.Hour).Add(-8*24*time.Hour + 20*time.Minute), Temperature:&tempB, Humidity:&humB, TargetTemperature:&target, HVACMode:"heat", HVACAction:"heating"},
		{EntityID:"sensor-1", EntityKey:"sensor.bedroom_temperature", EntityName:"Bedroom temperature", AreaID:"living", RecordedAt:now.Truncate(time.Hour).Add(-8*24*time.Hour + 15*time.Minute), Temperature:&tempC, Humidity:&humC},
	}
	require.NoError(t, repo.SaveSamples(ctx, samples))

	points, err := repo.GetHistory(ctx, climate_history.Filter{AreaID:"bedroom", Limit:100})
	require.NoError(t, err)
	require.Len(t, points, 2)
	require.InDelta(t, 19.5, *points[0].TemperatureAverage, 0.001)

	first, err := repo.AggregateAndPrune(ctx, now.Add(-7*24*time.Hour), now.Add(-90*24*time.Hour))
	require.NoError(t, err)
	require.EqualValues(t, 2, first.HourlyBuckets)
	require.EqualValues(t, 3, first.RawDeleted)
	points, err = repo.GetHistory(ctx, climate_history.Filter{Limit:100})
	require.NoError(t, err)
	require.Len(t, points, 2)
	require.Equal(t, "aggregate", points[0].Period)
	require.Equal(t, "bedroom", points[0].AreaID)
	require.Equal(t, "living", points[1].AreaID)
	require.InDelta(t, 20.5, *points[0].TemperatureAverage, 0.001)
	require.InDelta(t, 23.0, *points[1].TemperatureAverage, 0.001)
	require.InDelta(t, 19.5, *points[0].TemperatureMinimum, 0.001)
	require.InDelta(t, 21.5, *points[0].TemperatureMaximum, 0.001)
	require.InDelta(t, 52.0, *points[0].HumidityAverage, 0.001)
			require.Equal(t, 2, points[0].TemperatureCount)
			require.Equal(t, 2, points[0].HumidityCount)

	second, err := repo.AggregateAndPrune(ctx, now.Add(-7*24*time.Hour), now.Add(-7*24*time.Hour))
	require.NoError(t, err)
	require.EqualValues(t, 2, second.DailyBuckets)
	require.EqualValues(t, 2, second.HourlyDeleted)
	points, err = repo.GetHistory(ctx, climate_history.Filter{Limit:100})
	require.NoError(t, err)
	require.Len(t, points, 2)
	require.Equal(t, "aggregate", points[0].Period)
	require.Equal(t, "bedroom", points[0].AreaID)
	require.Equal(t, "living", points[1].AreaID)
	require.InDelta(t, 20.5, *points[0].TemperatureAverage, 0.001)
	require.InDelta(t, 23.0, *points[1].TemperatureAverage, 0.001)
}
