// Package sqlite provides persistent storage for climate history and rollups.
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	domclimate "github.com/domus-hub/domus/internal/domain/climate_history"
)

type ClimateHistoryRepository struct {
	db     *sql.DB
	readDB *sql.DB
}

func NewClimateHistoryRepository(db *sql.DB, readDB ...*sql.DB) *ClimateHistoryRepository {
	r := &ClimateHistoryRepository{db: db, readDB: db}
	if len(readDB) > 0 && readDB[0] != nil {
		r.readDB = readDB[0]
	}
	return r
}

func (r *ClimateHistoryRepository) GetSettings(ctx context.Context) (domclimate.Settings, error) {
	var raw string
	err := r.readDB.QueryRowContext(ctx, "SELECT config_json FROM climate_history_settings WHERE id = 1").Scan(&raw)
	if err == sql.ErrNoRows { return domclimate.DefaultSettings(), nil }
	if err != nil { return domclimate.Settings{}, fmt.Errorf("sqlite: read climate history settings: %w", err) }
	var settings domclimate.Settings
	if err := json.Unmarshal([]byte(raw), &settings); err != nil { return domclimate.Settings{}, fmt.Errorf("sqlite: decode climate history settings: %w", err) }
	if settings.IntervalMinutes == 0 { settings = domclimate.DefaultSettings() }
	return settings, nil
}

func (r *ClimateHistoryRepository) SaveSettings(ctx context.Context, settings domclimate.Settings) error {
	if err := settings.Validate(); err != nil { return err }
	raw, err := json.Marshal(settings)
	if err != nil { return fmt.Errorf("sqlite: encode climate history settings: %w", err) }
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO climate_history_settings(id, config_json, updated_at)
		VALUES(1, ?, ?)
		ON CONFLICT(id) DO UPDATE SET config_json = excluded.config_json, updated_at = excluded.updated_at
	`, string(raw), time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil { return fmt.Errorf("sqlite: save climate history settings: %w", err) }
	return nil
}

func (r *ClimateHistoryRepository) SaveSamples(ctx context.Context, samples []domclimate.Sample) error {
	if len(samples) == 0 { return nil }
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil { return fmt.Errorf("sqlite: begin climate sample batch: %w", err) }
	defer tx.Rollback()
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO climate_history_samples(
			entity_id, entity_key, entity_name, device_id, area_id, recorded_at,
			temperature, humidity, target_temperature, hvac_mode, hvac_action
		) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(entity_id, recorded_at) DO UPDATE SET
			entity_key=excluded.entity_key, entity_name=excluded.entity_name,
			device_id=excluded.device_id, area_id=excluded.area_id,
			temperature=excluded.temperature, humidity=excluded.humidity,
			target_temperature=excluded.target_temperature,
			hvac_mode=excluded.hvac_mode, hvac_action=excluded.hvac_action
	`)
	if err != nil { return fmt.Errorf("sqlite: prepare climate sample insert: %w", err) }
	defer stmt.Close()
	for _, sample := range samples {
		if sample.EntityID == "" || sample.EntityKey == "" || sample.RecordedAt.IsZero() { return fmt.Errorf("sqlite: invalid climate sample identity or timestamp") }
		_, err := stmt.ExecContext(ctx, sample.EntityID, sample.EntityKey, sample.EntityName,
			sample.DeviceID, sample.AreaID, sample.RecordedAt.UTC().Truncate(time.Second).Format(time.RFC3339),
			sample.Temperature, sample.Humidity, sample.TargetTemperature, sample.HVACMode, sample.HVACAction)
		if err != nil { return fmt.Errorf("sqlite: insert climate sample for %s: %w", sample.EntityKey, err) }
	}
	if err := tx.Commit(); err != nil { return fmt.Errorf("sqlite: commit climate sample batch: %w", err) }
	return nil
}

func climateWhere(filter domclimate.Filter, column string) (string, []any) {
	clauses := make([]string, 0, 4)
	args := make([]any, 0, 4)
	if !filter.From.IsZero() { clauses = append(clauses, column+" >= ?"); args = append(args, filter.From.UTC().Truncate(time.Second).Format(time.RFC3339)) }
	if !filter.To.IsZero() { clauses = append(clauses, column+" <= ?"); args = append(args, filter.To.UTC().Truncate(time.Second).Format(time.RFC3339)) }
	if filter.AreaID != "" { clauses = append(clauses, "area_id = ?"); args = append(args, filter.AreaID) }
	if filter.AreaIDs != nil {
		if len(filter.AreaIDs) == 0 {
			clauses = append(clauses, "1 = 0")
		} else {
			placeholders := make([]string, len(filter.AreaIDs))
			for i, id := range filter.AreaIDs {
				placeholders[i] = "?"
				args = append(args, id)
			}
			clauses = append(clauses, "area_id IN ("+strings.Join(placeholders, ",")+")")
		}
	}
	if filter.EntityID != "" { clauses = append(clauses, "entity_id = ?"); args = append(args, filter.EntityID) }
	if filter.EntityIDs != nil {
		if len(filter.EntityIDs) == 0 {
			clauses = append(clauses, "1 = 0")
		} else {
			placeholders := make([]string, len(filter.EntityIDs))
			for i, id := range filter.EntityIDs {
				placeholders[i] = "?"
				args = append(args, id)
			}
			clauses = append(clauses, "entity_id IN ("+strings.Join(placeholders, ",")+")")
		}
	}
	if len(clauses) == 0 { return "", args }
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func (r *ClimateHistoryRepository) GetHistory(ctx context.Context, filter domclimate.Filter) ([]domclimate.Point, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 10000 { limit = 2000 }
	out := make([]domclimate.Point, 0)
	for _, spec := range []struct { table, timestamp string; aggregate bool }{
		{"climate_history_samples", "recorded_at", false},
		{"climate_history_hourly", "period_start", true},
		{"climate_history_daily", "period_start", true},
	} {
		rows, err := r.loadPoints(ctx, spec.table, spec.timestamp, spec.aggregate, filter, limit)
		if err != nil { return nil, err }
		out = append(out, rows...)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].RecordedAt.Equal(out[j].RecordedAt) {
			if out[i].EntityID == out[j].EntityID {
				return out[i].AreaID < out[j].AreaID
			}
			return out[i].EntityID < out[j].EntityID
		}
		return out[i].RecordedAt.Before(out[j].RecordedAt)
	})
	if len(out) > limit { out = out[len(out)-limit:] }
	return out, nil
}

func (r *ClimateHistoryRepository) loadPoints(ctx context.Context, table, timestampColumn string, aggregate bool,
	filter domclimate.Filter, limit int) ([]domclimate.Point, error) {
	where, args := climateWhere(filter, timestampColumn)
	var columns string
	if aggregate {
		columns = "entity_id, entity_key, entity_name, device_id, area_id, period_start, " +
			"temperature_avg, temperature_min, temperature_max, humidity_avg, humidity_min, humidity_max, " +
			"target_temperature_avg, '' AS hvac_mode, '' AS hvac_action, sample_count, heating_samples, 'aggregate' AS period, temperature_count, humidity_count"
	} else {
		columns = "entity_id, entity_key, entity_name, device_id, area_id, recorded_at, " +
			"temperature AS temperature_average, temperature AS temperature_minimum, temperature AS temperature_maximum, " +
			"humidity AS humidity_average, humidity AS humidity_minimum, humidity AS humidity_maximum, " +
			"target_temperature AS target_temperature_average, hvac_mode, hvac_action, 1 AS sample_count, " +
			"CASE WHEN lower(hvac_action) LIKE '%heat%' THEN 1 ELSE 0 END AS heating_samples, 'raw' AS period, " +
			"CASE WHEN temperature IS NOT NULL THEN 1 ELSE 0 END AS temperature_count, CASE WHEN humidity IS NOT NULL THEN 1 ELSE 0 END AS humidity_count"
	}
	query := "SELECT " + columns + " FROM " + table + where + " ORDER BY " + timestampColumn + " DESC LIMIT ?"
	args = append(args, limit)
	rows, err := r.readDB.QueryContext(ctx, query, args...)
	if err != nil { return nil, fmt.Errorf("sqlite: query %s climate history: %w", table, err) }
	defer rows.Close()
	points := make([]domclimate.Point, 0)
	for rows.Next() {
		var p domclimate.Point
		var at string
		var tempAvg, tempMin, tempMax, humidityAvg, humidityMin, humidityMax, target sql.NullFloat64
		if err := rows.Scan(&p.EntityID, &p.EntityKey, &p.EntityName, &p.DeviceID, &p.AreaID, &at,
			&tempAvg, &tempMin, &tempMax, &humidityAvg, &humidityMin, &humidityMax, &target,
			&p.HVACMode, &p.HVACAction, &p.SampleCount, &p.HeatingSamples, &p.Period, &p.TemperatureCount, &p.HumidityCount); err != nil {
			return nil, fmt.Errorf("sqlite: scan climate history point: %w", err)
		}
		p.RecordedAt, _ = time.Parse(time.RFC3339Nano, at)
		if p.RecordedAt.IsZero() { p.RecordedAt, _ = time.Parse(time.RFC3339, at) }
		p.TemperatureAverage, p.TemperatureMinimum, p.TemperatureMaximum = nullablePointer(tempAvg), nullablePointer(tempMin), nullablePointer(tempMax)
		p.HumidityAverage, p.HumidityMinimum, p.HumidityMaximum = nullablePointer(humidityAvg), nullablePointer(humidityMin), nullablePointer(humidityMax)
		p.TargetTemperatureAvg = nullablePointer(target)
		points = append(points, p)
	}
	return points, rows.Err()
}

func nullablePointer(v sql.NullFloat64) *float64 {
	if !v.Valid { return nil }
	value := v.Float64
	return &value
}

// AggregateAndPrune atomically moves detailed rows to hourly and daily rollups.
func (r *ClimateHistoryRepository) AggregateAndPrune(ctx context.Context, rawCutoff, hourlyCutoff time.Time) (*domclimate.RetentionSummary, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil { return nil, fmt.Errorf("sqlite: begin climate compaction: %w", err) }
	defer tx.Rollback()
	result := &domclimate.RetentionSummary{}
	raw := rawCutoff.UTC().Truncate(time.Second).Format(time.RFC3339)
	hourly := hourlyCutoff.UTC().Truncate(time.Second).Format(time.RFC3339)

	hourlySQL := `
		INSERT INTO climate_history_hourly(
			entity_id,entity_key,entity_name,device_id,area_id,period_start,period_end,
			temperature_avg,temperature_min,temperature_max,temperature_sum,temperature_count,
			humidity_avg,humidity_min,humidity_max,humidity_sum,humidity_count,
			target_temperature_avg,target_temperature_min,target_temperature_max,target_temperature_sum,target_temperature_count,
			sample_count,heating_samples
		)
		SELECT entity_id,MAX(entity_key),MAX(entity_name),MAX(device_id),area_id,
			strftime('%Y-%m-%dT%H:00:00Z',recorded_at),strftime('%Y-%m-%dT%H:00:00Z',recorded_at,'+1 hour'),
			AVG(temperature),MIN(temperature),MAX(temperature),COALESCE(SUM(temperature),0),COUNT(temperature),
			AVG(humidity),MIN(humidity),MAX(humidity),COALESCE(SUM(humidity),0),COUNT(humidity),
			AVG(target_temperature),MIN(target_temperature),MAX(target_temperature),COALESCE(SUM(target_temperature),0),COUNT(target_temperature),
			COUNT(*),SUM(CASE WHEN lower(hvac_action) LIKE '%heat%' THEN 1 ELSE 0 END)
		FROM climate_history_samples WHERE recorded_at < ?
		GROUP BY entity_id,area_id,strftime('%Y-%m-%dT%H:00:00Z',recorded_at)
		ON CONFLICT(entity_id,area_id,period_start) DO UPDATE SET
			entity_key=excluded.entity_key,entity_name=excluded.entity_name,device_id=excluded.device_id,area_id=excluded.area_id,
			period_end=excluded.period_end,
			temperature_sum=climate_history_hourly.temperature_sum+excluded.temperature_sum,
			temperature_count=climate_history_hourly.temperature_count+excluded.temperature_count,
			temperature_avg=CASE WHEN climate_history_hourly.temperature_count+excluded.temperature_count>0 THEN (climate_history_hourly.temperature_sum+excluded.temperature_sum)/(climate_history_hourly.temperature_count+excluded.temperature_count) END,
			temperature_min=CASE WHEN climate_history_hourly.temperature_min IS NULL THEN excluded.temperature_min WHEN excluded.temperature_min IS NULL THEN climate_history_hourly.temperature_min ELSE MIN(climate_history_hourly.temperature_min,excluded.temperature_min) END,
			temperature_max=CASE WHEN climate_history_hourly.temperature_max IS NULL THEN excluded.temperature_max WHEN excluded.temperature_max IS NULL THEN climate_history_hourly.temperature_max ELSE MAX(climate_history_hourly.temperature_max,excluded.temperature_max) END,
			humidity_sum=climate_history_hourly.humidity_sum+excluded.humidity_sum,
			humidity_count=climate_history_hourly.humidity_count+excluded.humidity_count,
			humidity_avg=CASE WHEN climate_history_hourly.humidity_count+excluded.humidity_count>0 THEN (climate_history_hourly.humidity_sum+excluded.humidity_sum)/(climate_history_hourly.humidity_count+excluded.humidity_count) END,
			humidity_min=CASE WHEN climate_history_hourly.humidity_min IS NULL THEN excluded.humidity_min WHEN excluded.humidity_min IS NULL THEN climate_history_hourly.humidity_min ELSE MIN(climate_history_hourly.humidity_min,excluded.humidity_min) END,
			humidity_max=CASE WHEN climate_history_hourly.humidity_max IS NULL THEN excluded.humidity_max WHEN excluded.humidity_max IS NULL THEN climate_history_hourly.humidity_max ELSE MAX(climate_history_hourly.humidity_max,excluded.humidity_max) END,
			target_temperature_sum=climate_history_hourly.target_temperature_sum+excluded.target_temperature_sum,
			target_temperature_count=climate_history_hourly.target_temperature_count+excluded.target_temperature_count,
			target_temperature_avg=CASE WHEN climate_history_hourly.target_temperature_count+excluded.target_temperature_count>0 THEN (climate_history_hourly.target_temperature_sum+excluded.target_temperature_sum)/(climate_history_hourly.target_temperature_count+excluded.target_temperature_count) END,
			target_temperature_min=CASE WHEN climate_history_hourly.target_temperature_min IS NULL THEN excluded.target_temperature_min WHEN excluded.target_temperature_min IS NULL THEN climate_history_hourly.target_temperature_min ELSE MIN(climate_history_hourly.target_temperature_min,excluded.target_temperature_min) END,
			target_temperature_max=CASE WHEN climate_history_hourly.target_temperature_max IS NULL THEN excluded.target_temperature_max WHEN excluded.target_temperature_max IS NULL THEN climate_history_hourly.target_temperature_max ELSE MAX(climate_history_hourly.target_temperature_max,excluded.target_temperature_max) END,
			sample_count=climate_history_hourly.sample_count+excluded.sample_count,
			heating_samples=climate_history_hourly.heating_samples+excluded.heating_samples
	`
	res, err := tx.ExecContext(ctx, hourlySQL, raw)
	if err != nil { return nil, fmt.Errorf("sqlite: aggregate climate samples hourly: %w", err) }
	result.HourlyBuckets, _ = res.RowsAffected()

	dailySQL := `
		INSERT INTO climate_history_daily(
			entity_id,entity_key,entity_name,device_id,area_id,period_start,period_end,
			temperature_avg,temperature_min,temperature_max,temperature_sum,temperature_count,
			humidity_avg,humidity_min,humidity_max,humidity_sum,humidity_count,
			target_temperature_avg,target_temperature_min,target_temperature_max,target_temperature_sum,target_temperature_count,
			sample_count,heating_samples
		)
		SELECT entity_id,MAX(entity_key),MAX(entity_name),MAX(device_id),area_id,
			strftime('%Y-%m-%dT00:00:00Z',period_start),strftime('%Y-%m-%dT00:00:00Z',period_start,'+1 day'),
			CASE WHEN SUM(temperature_count)>0 THEN SUM(temperature_sum)/SUM(temperature_count) END,
			MIN(temperature_min),MAX(temperature_max),SUM(temperature_sum),SUM(temperature_count),
			CASE WHEN SUM(humidity_count)>0 THEN SUM(humidity_sum)/SUM(humidity_count) END,
			MIN(humidity_min),MAX(humidity_max),SUM(humidity_sum),SUM(humidity_count),
			CASE WHEN SUM(target_temperature_count)>0 THEN SUM(target_temperature_sum)/SUM(target_temperature_count) END,
			MIN(target_temperature_min),MAX(target_temperature_max),SUM(target_temperature_sum),SUM(target_temperature_count),
			SUM(sample_count),SUM(heating_samples)
		FROM climate_history_hourly WHERE period_start < ?
		GROUP BY entity_id,area_id,strftime('%Y-%m-%dT00:00:00Z',period_start)
		ON CONFLICT(entity_id,area_id,period_start) DO UPDATE SET
			entity_key=excluded.entity_key,entity_name=excluded.entity_name,device_id=excluded.device_id,area_id=excluded.area_id,
			period_end=excluded.period_end,
			temperature_sum=climate_history_daily.temperature_sum+excluded.temperature_sum,
			temperature_count=climate_history_daily.temperature_count+excluded.temperature_count,
			temperature_avg=CASE WHEN climate_history_daily.temperature_count+excluded.temperature_count>0 THEN (climate_history_daily.temperature_sum+excluded.temperature_sum)/(climate_history_daily.temperature_count+excluded.temperature_count) END,
			temperature_min=CASE WHEN climate_history_daily.temperature_min IS NULL THEN excluded.temperature_min WHEN excluded.temperature_min IS NULL THEN climate_history_daily.temperature_min ELSE MIN(climate_history_daily.temperature_min,excluded.temperature_min) END,
			temperature_max=CASE WHEN climate_history_daily.temperature_max IS NULL THEN excluded.temperature_max WHEN excluded.temperature_max IS NULL THEN climate_history_daily.temperature_max ELSE MAX(climate_history_daily.temperature_max,excluded.temperature_max) END,
			humidity_sum=climate_history_daily.humidity_sum+excluded.humidity_sum,
			humidity_count=climate_history_daily.humidity_count+excluded.humidity_count,
			humidity_avg=CASE WHEN climate_history_daily.humidity_count+excluded.humidity_count>0 THEN (climate_history_daily.humidity_sum+excluded.humidity_sum)/(climate_history_daily.humidity_count+excluded.humidity_count) END,
			humidity_min=CASE WHEN climate_history_daily.humidity_min IS NULL THEN excluded.humidity_min WHEN excluded.humidity_min IS NULL THEN climate_history_daily.humidity_min ELSE MIN(climate_history_daily.humidity_min,excluded.humidity_min) END,
			humidity_max=CASE WHEN climate_history_daily.humidity_max IS NULL THEN excluded.humidity_max WHEN excluded.humidity_max IS NULL THEN climate_history_daily.humidity_max ELSE MAX(climate_history_daily.humidity_max,excluded.humidity_max) END,
			target_temperature_sum=climate_history_daily.target_temperature_sum+excluded.target_temperature_sum,
			target_temperature_count=climate_history_daily.target_temperature_count+excluded.target_temperature_count,
			target_temperature_avg=CASE WHEN climate_history_daily.target_temperature_count+excluded.target_temperature_count>0 THEN (climate_history_daily.target_temperature_sum+excluded.target_temperature_sum)/(climate_history_daily.target_temperature_count+excluded.target_temperature_count) END,
			target_temperature_min=CASE WHEN climate_history_daily.target_temperature_min IS NULL THEN excluded.target_temperature_min WHEN excluded.target_temperature_min IS NULL THEN climate_history_daily.target_temperature_min ELSE MIN(climate_history_daily.target_temperature_min,excluded.target_temperature_min) END,
			target_temperature_max=CASE WHEN climate_history_daily.target_temperature_max IS NULL THEN excluded.target_temperature_max WHEN excluded.target_temperature_max IS NULL THEN climate_history_daily.target_temperature_max ELSE MAX(climate_history_daily.target_temperature_max,excluded.target_temperature_max) END,
			sample_count=climate_history_daily.sample_count+excluded.sample_count,
			heating_samples=climate_history_daily.heating_samples+excluded.heating_samples
	`
	res, err = tx.ExecContext(ctx, dailySQL, hourly)
	if err != nil { return nil, fmt.Errorf("sqlite: aggregate climate hourly buckets daily: %w", err) }
	result.DailyBuckets, _ = res.RowsAffected()

	res, err = tx.ExecContext(ctx, "DELETE FROM climate_history_samples WHERE recorded_at < ?", raw)
	if err != nil { return nil, fmt.Errorf("sqlite: purge compacted climate samples: %w", err) }
	result.RawDeleted, _ = res.RowsAffected()
	res, err = tx.ExecContext(ctx, "DELETE FROM climate_history_hourly WHERE period_start < ?", hourly)
	if err != nil { return nil, fmt.Errorf("sqlite: purge compacted climate hourly buckets: %w", err) }
	result.HourlyDeleted, _ = res.RowsAffected()
	if err := tx.Commit(); err != nil { return nil, fmt.Errorf("sqlite: commit climate compaction: %w", err) }
	return result, nil
}

var _ domclimate.Repository = (*ClimateHistoryRepository)(nil)
