// Package history provides application services for entity state history and long-term statistics.
package history

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/domus-hub/domus/internal/domain/entity"
	domhist "github.com/domus-hub/domus/internal/domain/history"
	"github.com/google/uuid"
)

// RollupService performs downsampling aggregation of raw entity states into hourly/daily statistics,
// and manages automatic cleanup of stale high-frequency events.
type RollupService struct {
	historyRepo domhist.Repository
	statsRepo   domhist.StatisticsRepository
	entityRepo  entity.Repository
}

// NewRollupService constructs a RollupService instance.
func NewRollupService(
	historyRepo domhist.Repository,
	statsRepo domhist.StatisticsRepository,
	entityRepo entity.Repository,
) *RollupService {
	return &RollupService{
		historyRepo: historyRepo,
		statsRepo:   statsRepo,
		entityRepo:  entityRepo,
	}
}

// ExecuteHourlyRollup aggregates raw state events from the past N hours into hourly buckets.
func (s *RollupService) ExecuteHourlyRollup(ctx context.Context, lookbackHours int) (int, error) {
	if lookbackHours <= 0 {
		lookbackHours = 24
	}

	entities, err := s.entityRepo.List(ctx, entity.ListFilter{})
	if err != nil {
		return 0, fmt.Errorf("failed to list entities for rollup: %w", err)
	}

	now := time.Now().UTC()
	startWindow := now.Add(-time.Duration(lookbackHours) * time.Hour).Truncate(time.Hour)
	endWindow := now.Truncate(time.Hour)

	var allStats []*domhist.StatisticsRecord

	for _, ent := range entities {
		// Only numeric domains or sensors with unit_of_measurement
		if ent.Domain != "sensor" && ent.Domain != "number" && ent.Domain != "climate" {
			continue
		}

		rawEntries, err := s.historyRepo.ListByEntity(ctx, domhist.HistoryFilter{
			EntityID: ent.ID,
			From:     startWindow,
			To:       endWindow,
			Limit:    5000,
		})
		if err != nil || len(rawEntries) == 0 {
			continue
		}

		// Group by hour
		hourlyBuckets := make(map[time.Time][]float64)
		for _, entry := range rawEntries {
			numVal, err := strconv.ParseFloat(entry.State, 64)
			if err != nil {
				continue // Non-numeric state value (e.g. "unavailable")
			}
			hourKey := entry.RecordedAt.UTC().Truncate(time.Hour)
			hourlyBuckets[hourKey] = append(hourlyBuckets[hourKey], numVal)
		}
		unit := ent.UnitOfMeasurement
		// Calculate statistics for each hour
		for hourStart, values := range hourlyBuckets {
			if len(values) == 0 {
				continue
			}

			hourEnd := hourStart.Add(time.Hour)
			minVal := values[0]
			maxVal := values[0]
			sumVal := 0.0

			for _, v := range values {
				if v < minVal {
					minVal = v
				}
				if v > maxVal {
					maxVal = v
				}
				sumVal += v
			}
			meanVal := sumVal / float64(len(values))

			allStats = append(allStats, &domhist.StatisticsRecord{
				ID:          uuid.NewString(),
				EntityID:    ent.ID,
				Period:      "hourly",
				PeriodStart: hourStart,
				PeriodEnd:   hourEnd,
				Mean:        &meanVal,
				Min:         &minVal,
				Max:         &maxVal,
				Sum:         &sumVal,
				Count:       len(values),
				Unit:        unit,
				CreatedAt:   now,
			})
		}
	}

	if len(allStats) > 0 {
		if err := s.statsRepo.InsertOrUpdateBatch(ctx, allStats); err != nil {
			return 0, fmt.Errorf("failed to persist rollup statistics: %w", err)
		}
	}

	slog.Info("completed long-term statistics rollup", "records_created", len(allStats))
	return len(allStats), nil
}

// PurgeOldRawHistory deletes raw second-by-second records older than retentionDays (e.g. 10 days).
func (s *RollupService) PurgeOldRawHistory(ctx context.Context, retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		retentionDays = 10
	}
	cutoff := time.Now().UTC().Add(-time.Duration(retentionDays) * 24 * time.Hour)
	rowsAffected, err := s.historyRepo.PurgeOlderThan(ctx, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to purge old history: %w", err)
	}
	slog.Info("purged raw history records", "deleted_rows", rowsAffected, "older_than_days", retentionDays)
	return rowsAffected, nil
}

// Start runs periodic rollup and purging in the background until ctx is cancelled.
func (s *RollupService) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, _ = s.ExecuteHourlyRollup(ctx, 24)
			_, _ = s.PurgeOldRawHistory(ctx, 10)
		}
	}
}

