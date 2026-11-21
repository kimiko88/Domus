package climate_history

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	domclimate "github.com/domus-hub/domus/internal/domain/climate_history"
	"github.com/domus-hub/domus/internal/domain/entity"
	"github.com/stretchr/testify/require"
)

type climateRepoStub struct {
	settings domclimate.Settings
	samples  []domclimate.Sample
	points   []domclimate.Point
}
func (r *climateRepoStub) GetSettings(context.Context) (domclimate.Settings, error) { return r.settings, nil }
func (r *climateRepoStub) SaveSettings(_ context.Context, s domclimate.Settings) error { r.settings = s; return nil }
func (r *climateRepoStub) SaveSamples(_ context.Context, samples []domclimate.Sample) error { r.samples = append(r.samples, samples...); return nil }
func (r *climateRepoStub) GetHistory(context.Context, domclimate.Filter) ([]domclimate.Point, error) { return r.points, nil }
func (r *climateRepoStub) AggregateAndPrune(context.Context, time.Time, time.Time) (*domclimate.RetentionSummary, error) {
	return &domclimate.RetentionSummary{}, nil
}

type climateEntityRepoStub struct { entities []*entity.Entity }
func (r *climateEntityRepoStub) Create(context.Context, *entity.Entity) error { return errors.New("not implemented") }
func (r *climateEntityRepoStub) GetByID(_ context.Context, id string) (*entity.Entity, error) {
	for _, e := range r.entities { if e.ID == id { return e, nil } }
	return nil, entity.ErrEntityNotFound
}
func (r *climateEntityRepoStub) GetByKey(_ context.Context, key string) (*entity.Entity, error) {
	for _, e := range r.entities { if e.EntityKey == key { return e, nil } }
	return nil, entity.ErrEntityNotFound
}
func (r *climateEntityRepoStub) List(context.Context, entity.ListFilter) ([]*entity.Entity, error) { return r.entities, nil }
func (r *climateEntityRepoStub) Update(context.Context, *entity.Entity) error { return errors.New("not implemented") }
func (r *climateEntityRepoStub) UpdateState(context.Context, string, string, string, bool, time.Time) error { return errors.New("not implemented") }
func (r *climateEntityRepoStub) Delete(context.Context, string) error { return errors.New("not implemented") }

func testLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestCollectNowRecordsTemperatureHumidityAndThermostatSetpoint(t *testing.T) {
	settings := domclimate.DefaultSettings()
	repo := &climateRepoStub{settings: settings}
	entities := &climateEntityRepoStub{entities: []*entity.Entity{
		{ID:"temp-1", EntityKey:"sensor.living_temperature", Domain:"sensor", Name:"Living temperature", UnitOfMeasurement:"°C", State:"21.4", Attributes:"{\"device_class\":\"temperature\"}", Available:true, AreaID:"living"},
		{ID:"hum-1", EntityKey:"sensor.living_humidity", Domain:"sensor", Name:"Living humidity", UnitOfMeasurement:"%", State:"53", Attributes:"{\"device_class\":\"humidity\"}", Available:true, AreaID:"living"},
		{ID:"climate-1", EntityKey:"climate.living", Domain:"climate", Name:"Living thermostat", State:"heat", Attributes:"{\"current_temperature\":21.1,\"temperature\":22.0,\"hvac_action\":\"heating\"}", Available:true, AreaID:"living"},
	}}
	service := NewService(repo, entities, testLogger())
	count, err := service.CollectNow(context.Background())
	require.NoError(t, err)
	require.Equal(t, 3, count)
	require.Len(t, repo.samples, 3)

	byKey := map[string]domclimate.Sample{}
	for _, sample := range repo.samples { byKey[sample.EntityKey] = sample }
	require.InDelta(t, 21.4, *byKey["sensor.living_temperature"].Temperature, 0.001)
	require.InDelta(t, 53.0, *byKey["sensor.living_humidity"].Humidity, 0.001)
	require.InDelta(t, 21.1, *byKey["climate.living"].Temperature, 0.001)
	require.InDelta(t, 22.0, *byKey["climate.living"].TargetTemperature, 0.001)
	require.Equal(t, "heating", byKey["climate.living"].HVACAction)
	require.Equal(t, "auto", byKey["climate.living"].HVACMode)
}

func TestCollectNowHonorsDisabledAndSamplingInterval(t *testing.T) {
	settings := domclimate.DefaultSettings()
	settings.Enabled = false
	repo := &climateRepoStub{settings: settings}
	entities := &climateEntityRepoStub{entities: []*entity.Entity{
		{ID:"temp-1", EntityKey:"sensor.room_temperature", Domain:"sensor", Name:"Room temperature", UnitOfMeasurement:"°C", State:"20", Attributes:"{\"device_class\":\"temperature\"}", Available:true},
	}}
	service := NewService(repo, entities, testLogger())
	count, err := service.CollectNow(context.Background())
	require.NoError(t, err)
	require.Zero(t, count)
	require.Empty(t, repo.samples)

	settings.Enabled = true
	repo.settings = settings
	count, err = service.CollectNow(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, count)
	count, err = service.CollectNow(context.Background())
	require.NoError(t, err)
	require.Zero(t, count)
	require.Len(t, repo.samples, 1)
}

func TestSmartSamplingRecordsSignificantChanges(t *testing.T) {
	settings := domclimate.DefaultSettings()
	settings.SmartSampling = true
	repo := &climateRepoStub{settings: settings}
	room := &entity.Entity{ID:"temp-1", EntityKey:"sensor.room_temperature", Domain:"sensor", Name:"Room temperature", UnitOfMeasurement:"°C", State:"20", Attributes:"{\"device_class\":\"temperature\"}", Available:true}
	service := NewService(repo, &climateEntityRepoStub{entities: []*entity.Entity{room}}, testLogger())
	_, err := service.CollectNow(context.Background())
	require.NoError(t, err)
	room.State = "20.5"
	count, err := service.CollectNow(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, count)
	require.Len(t, repo.samples, 2)
	require.InDelta(t, 20.5, *repo.samples[1].Temperature, 0.001)
}

func TestSummarizeAppliesAlertsAndThermalAnalysis(t *testing.T) {
	settings := domclimate.DefaultSettings()
	low, highHumidity := 15.0, 75.0
	repo := &climateRepoStub{
		settings: settings,
		points: []domclimate.Point{
			{EntityID:"temp-1", EntityKey:"sensor.room_temperature", EntityName:"Room temperature", AreaID:"bedroom", RecordedAt:time.Now().Add(-time.Hour), TemperatureAverage:floatPtr(18), HumidityAverage:floatPtr(55), Period:"raw"},
			{EntityID:"temp-1", EntityKey:"sensor.room_temperature", EntityName:"Room temperature", AreaID:"bedroom", RecordedAt:time.Now(), TemperatureAverage:&low, HumidityAverage:&highHumidity, Period:"raw"},
		},
	}
	service := NewService(repo, &climateEntityRepoStub{}, testLogger())
	summary, err := service.Summarize(context.Background(), domclimate.Filter{From:time.Now().Add(-2*time.Hour), To:time.Now()})
	require.NoError(t, err)
	require.Len(t, summary.Rooms, 1)
	require.NotNil(t, summary.Rooms[0].TemperatureChangeRate)
	require.NotNil(t, summary.Rooms[0].ForecastTemperature)
	foundLow, foundHumidity := false, false
	for _, insight := range summary.Insights {
		foundLow = foundLow || insight.Code == "temperature_low"
		foundHumidity = foundHumidity || insight.Code == "humidity_high"
	}
	require.True(t, foundLow)
	require.True(t, foundHumidity)

	settings.TemperatureAlertsEnabled = false
	settings.HumidityAlertsEnabled = false
	repo.settings = settings
	summary, err = service.Summarize(context.Background(), domclimate.Filter{})
	require.NoError(t, err)
	for _, insight := range summary.Insights {
		require.NotEqual(t, "temperature_low", insight.Code)
		require.NotEqual(t, "humidity_high", insight.Code)
	}
}

func TestForecastCanRunWithoutThermalRateAnalysis(t *testing.T) {
	settings := domclimate.DefaultSettings()
	settings.ThermalAnalysisEnabled = false
	settings.ForecastEnabled = true
	first, second := time.Now().Add(-time.Hour), time.Now()
	repo := &climateRepoStub{settings: settings, points: []domclimate.Point{
		{EntityID: "temp-1", AreaID: "bedroom", RecordedAt: first, TemperatureAverage: floatPtr(19), Period: "raw"},
		{EntityID: "temp-1", AreaID: "bedroom", RecordedAt: second, TemperatureAverage: floatPtr(20), Period: "raw"},
	}}
	service := NewService(repo, &climateEntityRepoStub{}, testLogger())
	summary, err := service.Summarize(context.Background(), domclimate.Filter{})
	require.NoError(t, err)
	require.Len(t, summary.Rooms, 1)
	require.Nil(t, summary.Rooms[0].TemperatureChangeRate)
	require.NotNil(t, summary.Rooms[0].ForecastTemperature)
	require.InDelta(t, 21.0, *summary.Rooms[0].ForecastTemperature, 0.01)
}

func TestSettingsValidation(t *testing.T) {
	settings := domclimate.DefaultSettings()
	require.NoError(t, settings.Validate())
	settings.IntervalMinutes = 3
	require.ErrorIs(t, settings.Validate(), domclimate.ErrInvalidSettings)
}
