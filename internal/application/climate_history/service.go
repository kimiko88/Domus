// Package climate_history collects and analyses climate telemetry independently of the UI.
package climate_history

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	domclimate "github.com/domus-hub/domus/internal/domain/climate_history"
	"github.com/domus-hub/domus/internal/domain/entity"
)

type Service struct {
	repo       domclimate.Repository
	entities   entity.Repository
	logger     *slog.Logger
	mu         sync.Mutex
	last       map[string]domclimate.Sample
	lastSaved  map[string]time.Time
}

func NewService(repo domclimate.Repository, entities entity.Repository, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{repo: repo, entities: entities, logger: logger, last: map[string]domclimate.Sample{}, lastSaved: map[string]time.Time{}}
}

func (s *Service) GetSettings(ctx context.Context) (domclimate.Settings, error) {
	return s.repo.GetSettings(ctx)
}

func (s *Service) UpdateSettings(ctx context.Context, settings domclimate.Settings) error {
	if err := settings.Validate(); err != nil {
		return err
	}
	return s.repo.SaveSettings(ctx, settings)
}

func (s *Service) GetHistory(ctx context.Context, filter domclimate.Filter) ([]domclimate.Point, error) {
	if filter.Limit <= 0 || filter.Limit > 10000 {
		filter.Limit = 2000
	}
	return s.repo.GetHistory(ctx, filter)
}

// CollectNow reads all available temperature/humidity sensors and climate entities.
func (s *Service) CollectNow(ctx context.Context) (int, error) {
	settings, err := s.repo.GetSettings(ctx)
	if err != nil {
		return 0, fmt.Errorf("load climate history settings: %w", err)
	}
	if !settings.Enabled {
		return 0, nil
	}
	if s.entities == nil {
		return 0, fmt.Errorf("climate history requires an entity repository")
	}
	entities, err := s.entities.List(ctx, entity.ListFilter{})
	if err != nil {
		return 0, fmt.Errorf("list climate entities: %w", err)
	}
	now := time.Now().UTC()
	interval := time.Duration(settings.IntervalMinutes) * time.Minute
	batch := make([]domclimate.Sample, 0, len(entities))

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, ent := range entities {
		if ent == nil || !ent.Available {
			continue
		}
		sample, ok := sampleEntity(ent, settings, now)
		if !ok {
			continue
		}
		previous, hasPrevious := s.last[ent.ID]
		lastSaved := s.lastSaved[ent.ID]
		due := lastSaved.IsZero() || now.Sub(lastSaved) >= interval
		changed := settings.SmartSampling && hasPrevious && significantChange(previous, sample, settings)
		if !due && !changed {
			s.last[ent.ID] = sample
			continue
		}
		batch = append(batch, sample)
	}
	if len(batch) == 0 {
		return 0, nil
	}
	if err := s.repo.SaveSamples(ctx, batch); err != nil {
		return 0, fmt.Errorf("persist climate samples: %w", err)
	}
	for _, sample := range batch {
		s.last[sample.EntityID] = sample
		s.lastSaved[sample.EntityID] = now
	}
	return len(batch), nil
}

func significantChange(previous, current domclimate.Sample, cfg domclimate.Settings) bool {
	if previous.Temperature != nil && current.Temperature != nil &&
		math.Abs(*previous.Temperature-*current.Temperature) >= cfg.TemperatureChangeThreshold {
		return true
	}
	if previous.Humidity != nil && current.Humidity != nil &&
		math.Abs(*previous.Humidity-*current.Humidity) >= cfg.HumidityChangeThreshold {
		return true
	}
	return false
}

func sampleEntity(ent *entity.Entity, cfg domclimate.Settings, at time.Time) (domclimate.Sample, bool) {
	attrs := map[string]any{}
	_ = json.Unmarshal([]byte(ent.Attributes), &attrs)
	s := domclimate.Sample{
		EntityID: ent.ID, EntityKey: ent.EntityKey, EntityName: ent.Name,
		DeviceID: ent.DeviceID, AreaID: ent.AreaID, RecordedAt: at,
	}
	switch ent.Domain {
	case "climate":
		if !cfg.RecordThermostat {
			return s, false
		}
		if cfg.RecordTemperature {
			if v, ok := attributeNumber(attrs, "current_temperature", "current_temp", "measured_temperature"); ok {
				v = normalizeTemperature(v, ent.UnitOfMeasurement, attrs)
				if plausibleTemperature(v) { s.Temperature = &v }
			}
		}
		if cfg.RecordHumidity {
			if v, ok := attributeNumber(attrs, "current_humidity", "humidity"); ok && plausibleHumidity(v) {
				s.Humidity = &v
			}
		}
		if v, ok := attributeNumber(attrs, "target_temperature", "temperature_target", "target_temp", "temperature"); ok {
			if plausibleTemperature(v) { s.TargetTemperature = &v }
		}
		s.HVACMode = strings.ToLower(strings.TrimSpace(ent.State))
		s.HVACAction = stringAttribute(attrs, "hvac_action", "action")
	case "sensor":
		if cfg.RecordTemperature && isTemperatureSensor(ent, attrs) {
			if v, err := strconv.ParseFloat(strings.TrimSpace(ent.State), 64); err == nil {
				v = normalizeTemperature(v, ent.UnitOfMeasurement, attrs)
				if plausibleTemperature(v) { s.Temperature = &v }
			}
		}
		if cfg.RecordHumidity && isHumiditySensor(ent, attrs) {
			if v, err := strconv.ParseFloat(strings.TrimSpace(ent.State), 64); err == nil && plausibleHumidity(v) {
				s.Humidity = &v
			}
		}
	default:
		return s, false
	}
	return s, s.Temperature != nil || s.Humidity != nil || s.TargetTemperature != nil
}

func attributeNumber(attrs map[string]any, keys ...string) (float64, bool) {
	for _, key := range keys {
		if raw, ok := attrs[key]; ok {
			switch v := raw.(type) {
			case float64:
				return v, true
			case json.Number:
				f, err := v.Float64(); return f, err == nil
			case string:
				f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); return f, err == nil
			}
		}
	}
	return 0, false
}

func stringAttribute(attrs map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := attrs[key].(string); ok && value != "" { return strings.ToLower(strings.TrimSpace(value)) }
	}
	return ""
}

func isTemperatureSensor(ent *entity.Entity, attrs map[string]any) bool {
	class := strings.ToLower(fmt.Sprint(attrs["device_class"]))
	unit := strings.ToLower(ent.UnitOfMeasurement)
	key := strings.ToLower(ent.EntityKey + " " + ent.Name)
	return class == "temperature" || strings.Contains(unit, "°c") || strings.Contains(unit, "°f") ||
		strings.Contains(unit, "celsius") || strings.Contains(key, "temperature") || strings.Contains(key, "temperatura")
}

func isHumiditySensor(ent *entity.Entity, attrs map[string]any) bool {
	class := strings.ToLower(fmt.Sprint(attrs["device_class"]))
	unit := strings.TrimSpace(ent.UnitOfMeasurement)
	key := strings.ToLower(ent.EntityKey + " " + ent.Name)
	return class == "humidity" || strings.Contains(class, "humidity") || strings.Contains(key, "humidity") || strings.Contains(key, "umid")
}

func normalizeTemperature(value float64, unit string, attrs map[string]any) float64 {
	u := strings.ToLower(unit + " " + fmt.Sprint(attrs["unit_of_measurement"]))
	if strings.Contains(u, "°f") || strings.Contains(u, "fahrenheit") { return (value - 32) * 5 / 9 }
	return value
}
func plausibleTemperature(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) && v >= -50 && v <= 70 }
func plausibleHumidity(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) && v >= 0 && v <= 100 }

// Start polls once per minute so intervals and smart-change thresholds take effect without restart.
func (s *Service) Start(ctx context.Context) {
	go func() {
		_, err := s.CollectNow(ctx)
		if err != nil { s.logger.Warn("initial climate history collection failed", "error", err) }
		collectTicker := time.NewTicker(time.Minute)
		maintenanceTicker := time.NewTicker(6 * time.Hour)
		defer collectTicker.Stop()
		defer maintenanceTicker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-collectTicker.C:
				if _, err := s.CollectNow(ctx); err != nil && ctx.Err() == nil {
					s.logger.Warn("climate history collection failed", "error", err)
				}
			case now := <-maintenanceTicker.C:
				cfg, err := s.repo.GetSettings(ctx)
				if err != nil { s.logger.Warn("climate history settings unavailable for rollup", "error", err); continue }
				result, err := s.repo.AggregateAndPrune(ctx,
					now.UTC().Add(-time.Duration(cfg.RawRetentionDays)*24*time.Hour),
					now.UTC().Add(-time.Duration(cfg.HourlyRetentionDays)*24*time.Hour))
				if err != nil { s.logger.Warn("climate history rollup failed", "error", err); continue }
				s.logger.Info("climate history compacted", "hourly_buckets", result.HourlyBuckets,
					"daily_buckets", result.DailyBuckets, "raw_deleted", result.RawDeleted, "hourly_deleted", result.HourlyDeleted)
			}
		}
	}()
}

func (s *Service) Summarize(ctx context.Context, filter domclimate.Filter) (*domclimate.DashboardSummary, error) {
	settings, err := s.repo.GetSettings(ctx)
	if err != nil { return nil, err }
	points, err := s.GetHistory(ctx, filter)
	if err != nil { return nil, err }
	out := &domclimate.DashboardSummary{Rooms: []domclimate.RoomSummary{}, Insights: []domclimate.Insight{}, SampleCount: len(points), From: filter.From, To: filter.To}
	type bucket struct { points []domclimate.Point }
	rooms := map[string]*bucket{}
	entities := map[string][]domclimate.Point{}
	for _, point := range points {
		area := point.AreaID
		if area == "" { area = "unassigned" }
		if rooms[area] == nil { rooms[area] = &bucket{} }
		rooms[area].points = append(rooms[area].points, point)
		entities[point.EntityID] = append(entities[point.EntityID], point)
	}
	for area, room := range rooms {
		summary := domclimate.RoomSummary{AreaID: area, SampleCount: len(room.points)}
		var tempSum, humiditySum float64
		var tempCount, humidityCount int
		for _, point := range room.points {
			if point.TemperatureAverage != nil {
				v := *point.TemperatureAverage; tempSum += v; tempCount++
				if summary.MinimumTemperature == nil || v < *summary.MinimumTemperature { summary.MinimumTemperature = floatPtr(v) }
				if summary.MaximumTemperature == nil || v > *summary.MaximumTemperature { summary.MaximumTemperature = floatPtr(v) }
				summary.CurrentTemperature = floatPtr(v); summary.LastUpdatedAt = point.RecordedAt
			}
			if point.HumidityAverage != nil {
				v := *point.HumidityAverage; humiditySum += v; humidityCount++; summary.CurrentHumidity = floatPtr(v)
			}
		}
		if tempCount > 0 { summary.AverageTemperature = floatPtr(tempSum/float64(tempCount)) }
		if humidityCount > 0 { summary.AverageHumidity = floatPtr(humiditySum/float64(humidityCount)) }
		out.Rooms = append(out.Rooms, summary)
	}
	for _, series := range entities {
		if len(series) == 0 { continue }
		first, last := series[0], series[len(series)-1]
		name, area := last.EntityName, last.AreaID
		if last.TemperatureAverage != nil {
			v := *last.TemperatureAverage
			if settings.TemperatureAlertsEnabled {
				if v < settings.LowTemperatureLimit { out.Insights = append(out.Insights, domclimate.Insight{Code:"temperature_low", AreaID:area, EntityName:name, Metric:"temperature", Value:v, Limit:settings.LowTemperatureLimit}) }
				if v > settings.HighTemperatureLimit { out.Insights = append(out.Insights, domclimate.Insight{Code:"temperature_high", AreaID:area, EntityName:name, Metric:"temperature", Value:v, Limit:settings.HighTemperatureLimit}) }
			}
			if settings.ComfortAnalysisEnabled && (v < 18 || v > 24) {
				out.Insights = append(out.Insights, domclimate.Insight{Code:"temperature_comfort", AreaID:area, EntityName:name, Metric:"temperature", Value:v})
			}
			if settings.AnomalyDetectionEnabled && len(series) >= 3 {
				var sum float64; var count int
				for _, p := range series { if p.TemperatureAverage != nil { sum += *p.TemperatureAverage; count++ } }
				if count > 0 && math.Abs(v-sum/float64(count)) >= 4 {
					out.Insights = append(out.Insights, domclimate.Insight{Code:"temperature_anomaly", AreaID:area, EntityName:name, Metric:"temperature", Value:v})
				}
			}
			if settings.ThermalAnalysisEnabled && len(series) >= 2 {
				hours := last.RecordedAt.Sub(first.RecordedAt).Hours()
				if hours >= 0.25 && first.TemperatureAverage != nil {
					rate := (v-*first.TemperatureAverage)/hours
					// Attach the measured rate and a one-hour linear estimate to the room containing this entity.
					for i := range out.Rooms { if out.Rooms[i].AreaID == area { out.Rooms[i].HeatingRate = floatPtr(rate); if settings.ForecastEnabled { out.Rooms[i].ForecastTemperature = floatPtr(v+rate) }; break } }
				}
			}
		}
		if last.HumidityAverage != nil {
			v := *last.HumidityAverage
			if settings.HumidityAlertsEnabled {
				if v < settings.LowHumidityLimit { out.Insights = append(out.Insights, domclimate.Insight{Code:"humidity_low", AreaID:area, EntityName:name, Metric:"humidity", Value:v, Limit:settings.LowHumidityLimit}) }
				if v > settings.HighHumidityLimit { out.Insights = append(out.Insights, domclimate.Insight{Code:"humidity_high", AreaID:area, EntityName:name, Metric:"humidity", Value:v, Limit:settings.HighHumidityLimit}) }
			}
			if settings.ComfortAnalysisEnabled && (v < 40 || v > 60) {
				out.Insights = append(out.Insights, domclimate.Insight{Code:"humidity_comfort", AreaID:area, EntityName:name, Metric:"humidity", Value:v})
			}
		}
	}
	if !settings.RoomComparisonEnabled && !settings.RoomMapEnabled { out.Rooms = out.Rooms[:0] }
	return out, nil
}

func floatPtr(v float64) *float64 { return &v }
