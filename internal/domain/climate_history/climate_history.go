// Package climate_history defines climate telemetry history and retention contracts.
package climate_history

import (
	"context"
	"errors"
	"time"
)

var ErrInvalidSettings = errors.New("climate history: invalid settings")

type Settings struct {
	Enabled                     bool    `json:"enabled"`
	IntervalMinutes             int     `json:"interval_minutes"`
	RecordTemperature           bool    `json:"record_temperature"`
	RecordHumidity              bool    `json:"record_humidity"`
	RecordThermostat             bool    `json:"record_thermostat"`
	SmartSampling               bool    `json:"smart_sampling"`
	TemperatureChangeThreshold  float64 `json:"temperature_change_threshold"`
	HumidityChangeThreshold     float64 `json:"humidity_change_threshold"`
	RawRetentionDays            int     `json:"raw_retention_days"`
	HourlyRetentionDays         int     `json:"hourly_retention_days"`
	TemperatureAlertsEnabled    bool    `json:"temperature_alerts_enabled"`
	HumidityAlertsEnabled       bool    `json:"humidity_alerts_enabled"`
	ComfortAnalysisEnabled      bool    `json:"comfort_analysis_enabled"`
	RoomComparisonEnabled       bool    `json:"room_comparison_enabled"`
	ThermalAnalysisEnabled      bool    `json:"thermal_analysis_enabled"`
	AnomalyDetectionEnabled     bool    `json:"anomaly_detection_enabled"`
	ForecastEnabled             bool    `json:"forecast_enabled"`
	RoomMapEnabled              bool    `json:"room_map_enabled"`
	LowTemperatureLimit         float64 `json:"low_temperature_limit"`
	HighTemperatureLimit        float64 `json:"high_temperature_limit"`
	LowHumidityLimit            float64 `json:"low_humidity_limit"`
	HighHumidityLimit           float64 `json:"high_humidity_limit"`
}

func DefaultSettings() Settings {
	return Settings{
		Enabled: true, IntervalMinutes: 10,
		RecordTemperature: true, RecordHumidity: true, RecordThermostat: true,
		SmartSampling: false, TemperatureChangeThreshold: 0.2, HumidityChangeThreshold: 1,
		RawRetentionDays: 7, HourlyRetentionDays: 90,
		TemperatureAlertsEnabled: true, HumidityAlertsEnabled: true,
		ComfortAnalysisEnabled: true, RoomComparisonEnabled: true,
		ThermalAnalysisEnabled: true, AnomalyDetectionEnabled: true,
		ForecastEnabled: true, RoomMapEnabled: true,
		LowTemperatureLimit: 16, HighTemperatureLimit: 28,
		LowHumidityLimit: 30, HighHumidityLimit: 70,
	}
}

func (s Settings) Validate() error {
	switch s.IntervalMinutes {
	case 1, 2, 5, 10, 15, 30, 60:
	default:
		return ErrInvalidSettings
	}
	if s.RawRetentionDays < 1 || s.RawRetentionDays > 30 ||
		s.HourlyRetentionDays < 30 || s.HourlyRetentionDays > 365 ||
		s.TemperatureChangeThreshold <= 0 || s.TemperatureChangeThreshold > 10 ||
		s.HumidityChangeThreshold <= 0 || s.HumidityChangeThreshold > 30 ||
		s.LowTemperatureLimit < -10 || s.LowTemperatureLimit >= s.HighTemperatureLimit ||
		s.HighTemperatureLimit > 50 ||
		s.LowHumidityLimit < 0 || s.LowHumidityLimit >= s.HighHumidityLimit ||
		s.HighHumidityLimit > 100 {
		return ErrInvalidSettings
	}
	return nil
}

type Sample struct {
	EntityID          string
	EntityKey         string
	EntityName        string
	DeviceID          string
	AreaID            string
	RecordedAt        time.Time
	Temperature       *float64
	Humidity          *float64
	TargetTemperature *float64
	HVACMode          string
	HVACAction        string
}

type Point struct {
	EntityID             string    `json:"entity_id"`
	EntityKey            string    `json:"entity_key"`
	EntityName           string    `json:"entity_name"`
	DeviceID             string    `json:"device_id,omitempty"`
	AreaID               string    `json:"area_id,omitempty"`
	RecordedAt           time.Time `json:"recorded_at"`
	TemperatureAverage   *float64  `json:"temperature_average,omitempty"`
	TemperatureMinimum   *float64  `json:"temperature_minimum,omitempty"`
	TemperatureMaximum   *float64  `json:"temperature_maximum,omitempty"`
	HumidityAverage      *float64  `json:"humidity_average,omitempty"`
	HumidityMinimum      *float64  `json:"humidity_minimum,omitempty"`
	HumidityMaximum      *float64  `json:"humidity_maximum,omitempty"`
	TargetTemperatureAvg *float64  `json:"target_temperature_average,omitempty"`
	HVACMode             string    `json:"hvac_mode,omitempty"`
	HVACAction           string    `json:"hvac_action,omitempty"`
	SampleCount          int       `json:"sample_count"`
	TemperatureCount      int       `json:"temperature_sample_count,omitempty"`
	HumidityCount         int       `json:"humidity_sample_count,omitempty"`
	HeatingSamples       int       `json:"heating_samples,omitempty"`
	Period               string    `json:"period"`
}

type Filter struct {
	EntityID  string
	EntityIDs []string
	AreaIDs   []string
	AreaID    string
	From     time.Time
	To       time.Time
	Limit    int
}

type RoomSummary struct {
	AreaID             string   `json:"area_id"`
	SampleCount        int      `json:"sample_count"`
	CurrentTemperature *float64 `json:"current_temperature,omitempty"`
	AverageTemperature *float64 `json:"average_temperature,omitempty"`
	MinimumTemperature *float64 `json:"minimum_temperature,omitempty"`
	MaximumTemperature *float64 `json:"maximum_temperature,omitempty"`
	CurrentHumidity    *float64 `json:"current_humidity,omitempty"`
	AverageHumidity    *float64 `json:"average_humidity,omitempty"`
	TemperatureChangeRate *float64 `json:"temperature_change_rate_c_per_hour,omitempty"`
	ForecastTemperature *float64 `json:"forecast_temperature_1h,omitempty"`
	LastUpdatedAt      time.Time `json:"last_updated_at"`
}

type Insight struct {
	Code       string  `json:"code"`
	AreaID     string  `json:"area_id,omitempty"`
	EntityName string  `json:"entity_name,omitempty"`
	Metric     string  `json:"metric,omitempty"`
	Value      float64 `json:"value,omitempty"`
	Limit      float64 `json:"limit,omitempty"`
}

type DashboardSummary struct {
	Rooms       []RoomSummary `json:"rooms"`
	Insights    []Insight     `json:"insights"`
	SampleCount int           `json:"sample_count"`
	From        time.Time     `json:"from"`
	To          time.Time     `json:"to"`
}

type RetentionSummary struct {
	HourlyBuckets int64 `json:"hourly_buckets"`
	DailyBuckets  int64 `json:"daily_buckets"`
	RawDeleted    int64 `json:"raw_deleted"`
	HourlyDeleted int64 `json:"hourly_deleted"`
}

type Repository interface {
	GetSettings(ctx context.Context) (Settings, error)
	SaveSettings(ctx context.Context, settings Settings) error
	SaveSamples(ctx context.Context, samples []Sample) error
	GetHistory(ctx context.Context, filter Filter) ([]Point, error)
	AggregateAndPrune(ctx context.Context, rawCutoff, hourlyCutoff time.Time) (*RetentionSummary, error)
}
