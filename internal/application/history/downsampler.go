// Package history provides telemetry recording, retention cleanup, and downsampling.
package history

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// DownsampleSummary reports how many hourly buckets were aggregated and raw records purged.
type DownsampleSummary struct {
	BucketsCreated  int64 `json:"buckets_created"`
	RawPointsPurged int64 `json:"raw_points_purged"`
}

// Downsampler aggregates historical high-frequency telemetry into hourly summary buckets.
type Downsampler struct {
	db *sql.DB
}

// NewDownsampler constructs a new Downsampler.
func NewDownsampler(db *sql.DB) *Downsampler {
	return &Downsampler{db: db}
}

// DownsampleOlderThan aggregates numeric state_history records older than cutoff into hourly_telemetry,
// then purges those raw records to reclaim storage space.
func (d *Downsampler) DownsampleOlderThan(ctx context.Context, cutoff time.Time) (*DownsampleSummary, error) {
	cutoffStr := cutoff.UTC().Format(time.RFC3339)

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("history: failed to start downsampling tx: %w", err)
	}
	defer tx.Rollback()

	// 1. Aggregate numeric states into hourly_telemetry
	insertQuery := `
		INSERT INTO hourly_telemetry (entity_id, hour_bucket, val_avg, val_min, val_max, samples)
		SELECT 
			entity_id,
			strftime('%Y-%m-%dT%H:00:00Z', recorded_at) AS hour_bucket,
			AVG(CAST(state AS REAL)) AS val_avg,
			MIN(CAST(state AS REAL)) AS val_min,
			MAX(CAST(state AS REAL)) AS val_max,
			COUNT(*) AS samples
		FROM state_history
		WHERE recorded_at < ?
		  AND (state GLOB '[0-9]*' OR state GLOB '-[0-9]*')
		GROUP BY entity_id, hour_bucket
		ON CONFLICT(entity_id, hour_bucket) DO UPDATE SET
			val_avg = (hourly_telemetry.val_avg * hourly_telemetry.samples + excluded.val_avg * excluded.samples) / (hourly_telemetry.samples + excluded.samples),
			val_min = MIN(hourly_telemetry.val_min, excluded.val_min),
			val_max = MAX(hourly_telemetry.val_max, excluded.val_max),
			samples = hourly_telemetry.samples + excluded.samples;
	`
	resInsert, err := tx.ExecContext(ctx, insertQuery, cutoffStr)
	if err != nil {
		return nil, fmt.Errorf("history: downsampling aggregation query failed: %w", err)
	}
	buckets, _ := resInsert.RowsAffected()

	// 2. Delete the raw records older than cutoff
	deleteQuery := `DELETE FROM state_history WHERE recorded_at < ?;`
	resDelete, err := tx.ExecContext(ctx, deleteQuery, cutoffStr)
	if err != nil {
		return nil, fmt.Errorf("history: failed to purge raw records after downsampling: %w", err)
	}
	purged, _ := resDelete.RowsAffected()

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("history: failed to commit downsampling tx: %w", err)
	}

	return &DownsampleSummary{
		BucketsCreated:  buckets,
		RawPointsPurged: purged,
	}, nil
}
