package weather_test

import (
	"context"
	"testing"

	"github.com/domus-hub/domus/internal/application/weather"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWeatherClient_FallbackAndLive(t *testing.T) {
	client := weather.NewClient()
	ctx := context.Background()

	report, err := client.FetchForecast(ctx, 45.4642, 9.1900) // Milan coordinates
	require.NoError(t, err)
	assert.NotEmpty(t, report.Hourly)
	assert.True(t, report.CurrentTemp > -40 && report.CurrentTemp < 60)
}

func TestSolarForecast_Computation(t *testing.T) {
	client := weather.NewClient()
	ctx := context.Background()

	report, err := client.FetchForecast(ctx, 45.4642, 9.1900)
	require.NoError(t, err)

	solar := weather.ComputeSolarForecast(report, 6.0, 30.0, 0.85)
	assert.Equal(t, 6.0, solar.InstalledKwPeak)
	assert.Equal(t, 30.0, solar.TiltDegrees)
	assert.True(t, solar.TodayTotalKwh >= 0)
	assert.True(t, solar.TomorrowTotalKwh >= 0)
	assert.NotEmpty(t, solar.Hourly)
}
