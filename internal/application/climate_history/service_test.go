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
		{ID:"climate-1", EntityKey:"climate.living", Domain:"climate", Name:"Living thermostat", State:"heat", Attributes:"{\"current_temperature\":21.1,\"temperature\":22.0,\"hvac_mode\":\"auto\",\"hvac_action\":\"heating\"}", Available:true, AreaID:"living"},
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

func TestCollectNowRecordsTuyaThermoHygrometerAttributes(t *testing.T) {
	settings := domclimate.DefaultSettings()
	settings.TrackedDeviceIDs = []string{"tuya_device_sensor_1"}
	repo := &climateRepoStub{settings: settings}
	entities := &climateEntityRepoStub{entities: []*entity.Entity{
		{
			ID: "tuya-sensor-1",
			EntityKey: "sensor.tuya_device_sensor_1", Domain: "sensor",
			Name: "Tuya Temperature and Humidity Sensor", State: "active",
			Attributes: "{\"current_temperature\":21.7,\"current_humidity\":48}",
			Available: true,
		},
	}}
	service := NewService(repo, entities, testLogger())

	// Attribute-only telemetry is sufficient to make the device selectable,
	// even when Tuya does not provide Home-Assistant-style device_class values.
	require.NoError(t, service.UpdateSettings(context.Background(), settings))
	count, err := service.CollectNow(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, count)
	require.Len(t, repo.samples, 1)
	require.Equal(t, "tuya_device_sensor_1", repo.samples[0].DeviceID)
	require.NotNil(t, repo.samples[0].Temperature)
	require.NotNil(t, repo.samples[0].Humidity)
	require.InDelta(t, 21.7, *repo.samples[0].Temperature, 0.001)
	require.InDelta(t, 48.0, *repo.samples[0].Humidity, 0.001)
}

func TestCollectNowHonorsSelectedIntegrationDevices(t *testing.T) {
	settings := domclimate.DefaultSettings()
	settings.TrackedDeviceIDs = []string{"tuya-device-1"}
	repo := &climateRepoStub{settings: settings}
	entities := &climateEntityRepoStub{entities: []*entity.Entity{
		{ID: "temperature-1", DeviceID: "tuya-device-1", EntityKey: "sensor.living_temperature", Domain: "sensor", Name: "Living temperature", UnitOfMeasurement: "°C", State: "21.5", Attributes: "{\"device_class\":\"temperature\"}", Available: true},
		{ID: "humidity-1", DeviceID: "tuya-device-1", EntityKey: "sensor.living_humidity", Domain: "sensor", Name: "Living humidity", UnitOfMeasurement: "%", State: "52", Attributes: "{\"device_class\":\"humidity\"}", Available: true},
		{ID: "temperature-2", DeviceID: "mqtt-device-2", EntityKey: "sensor.office_temperature", Domain: "sensor", Name: "Office temperature", UnitOfMeasurement: "°C", State: "19.5", Attributes: "{\"device_class\":\"temperature\"}", Available: true},
	}}
	service := NewService(repo, entities, testLogger())
	count, err := service.CollectNow(context.Background())
	require.NoError(t, err)
	require.Equal(t, 2, count)
	require.Len(t, repo.samples, 2)
	for _, sample := range repo.samples {
		require.Equal(t, "tuya-device-1", sample.DeviceID)
	}

	// An explicit empty selection is different from nil (legacy: collect all).
	repo.settings.TrackedDeviceIDs = []string{}
	count, err = service.CollectNow(context.Background())
	require.NoError(t, err)
	require.Zero(t, count)
}

func TestUpdateSettingsRejectsUnknownClimateSourceDevice(t *testing.T) {
	settings := domclimate.DefaultSettings()
	repo := &climateRepoStub{settings: settings}
	entities := &climateEntityRepoStub{entities: []*entity.Entity{
		{ID: "temp-1", DeviceID: "known-device", EntityKey: "sensor.room_temperature", Domain: "sensor", Name: "Room temperature", UnitOfMeasurement: "°C", State: "20", Attributes: "{\"device_class\":\"temperature\"}", Available: true},
	}}
	service := NewService(repo, entities, testLogger())

	settings.TrackedDeviceIDs = []string{"known-device"}
	require.NoError(t, service.UpdateSettings(context.Background(), settings))
	require.Equal(t, []string{"known-device"}, repo.settings.TrackedDeviceIDs)

	settings.TrackedDeviceIDs = []string{"not-a-climate-device"}
	require.ErrorIs(t, service.UpdateSettings(context.Background(), settings), domclimate.ErrInvalidSettings)
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

func TestSummarizeReportsUnavailableClimateSensors(t *testing.T) {
	settings := domclimate.DefaultSettings()
	offline := &entity.Entity{
		ID: "offline-1", EntityKey: "sensor.office_temperature", Domain: "sensor",
		Name: "Office temperature", UnitOfMeasurement: "°C", State: "unavailable",
		Attributes: "{\"device_class\":\"temperature\"}", Available: true, AreaID: "office",
	}
	repo := &climateRepoStub{settings: settings}
	service := NewService(repo, &climateEntityRepoStub{entities: []*entity.Entity{offline}}, testLogger())
	summary, err := service.Summarize(context.Background(), domclimate.Filter{})
	require.NoError(t, err)
	require.Len(t, summary.Insights, 1)
	require.Equal(t, "sensor_offline", summary.Insights[0].Code)
	require.Equal(t, "office", summary.Insights[0].AreaID)
	require.Equal(t, "Office temperature", summary.Insights[0].EntityName)

	settings.AnomalyDetectionEnabled = false
	repo.settings = settings
	summary, err = service.Summarize(context.Background(), domclimate.Filter{})
	require.NoError(t, err)
	require.Empty(t, summary.Insights)
}

func TestThermostatFahrenheitNormalizesMeasuredTemperatureAndSetpoint(t *testing.T) {
	settings := domclimate.DefaultSettings()
	repo := &climateRepoStub{settings: settings}
	thermostat := &entity.Entity{
		ID: "climate-fahrenheit", EntityKey: "climate.office", Domain: "climate",
		Name: "Office thermostat", State: "heat",
		Attributes: "{\"current_temperature\":68,\"temperature\":71.6,\"unit_of_measurement\":\"°F\"}",
		Available: true, AreaID: "office",
	}
	service := NewService(repo, &climateEntityRepoStub{entities: []*entity.Entity{thermostat}}, testLogger())
	count, err := service.CollectNow(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, count)
	require.InDelta(t, 20.0, *repo.samples[0].Temperature, 0.001)
	require.InDelta(t, 22.0, *repo.samples[0].TargetTemperature, 0.001)
}

func TestTemperatureSensorUsesAttributeUnitAndSmartSamplingAccumulatesChanges(t *testing.T) {
	settings := domclimate.DefaultSettings()
	settings.SmartSampling = true
	settings.TemperatureChangeThreshold = 0.2
	repo := &climateRepoStub{settings: settings}
	room := &entity.Entity{
		ID: "fahrenheit-sensor", EntityKey: "sensor.room_temp", Domain: "sensor",
		Name: "Room sensor", State: "68", Attributes: "{\"unit_of_measurement\":\"°F\"}",
		Available: true,
	}
	service := NewService(repo, &climateEntityRepoStub{entities: []*entity.Entity{room}}, testLogger())
	count, err := service.CollectNow(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, count)
	require.InDelta(t, 20.0, *repo.samples[0].Temperature, 0.001)

	room.State = "68.18"
	count, err = service.CollectNow(context.Background())
	require.NoError(t, err)
	require.Zero(t, count, "a change below threshold should not be persisted")
	room.State = "68.4"
	count, err = service.CollectNow(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, count, "small changes should accumulate against the last persisted sample")
	require.Len(t, repo.samples, 2)
	require.InDelta(t, (68.4-32)*5/9, *repo.samples[1].Temperature, 0.001)
}

func TestCollectNowSkipsUnavailableStateEvenWhenAvailabilityFlagIsTrue(t *testing.T) {
	settings := domclimate.DefaultSettings()
	repo := &climateRepoStub{settings: settings}
	unavailable := &entity.Entity{
		ID: "sensor-unavailable", EntityKey: "sensor.office_temperature", Domain: "sensor",
		Name: "Office temperature", State: "unavailable",
		Attributes: "{\"device_class\":\"temperature\"}", Available: true,
	}
	service := NewService(repo, &climateEntityRepoStub{entities: []*entity.Entity{unavailable}}, testLogger())
	count, err := service.CollectNow(context.Background())
	require.NoError(t, err)
	require.Zero(t, count)
	require.Empty(t, repo.samples)
}

func TestSettingsValidation(t *testing.T) {
	settings := domclimate.DefaultSettings()
	require.NoError(t, settings.Validate())
	settings.IntervalMinutes = 3
	require.ErrorIs(t, settings.Validate(), domclimate.ErrInvalidSettings)
}
