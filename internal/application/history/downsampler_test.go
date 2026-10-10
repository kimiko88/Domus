package history

import (
	"context"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
)

func TestDownsampler_AggregatesAndPurges(t *testing.T) {
	db, err := sqlite.Open(config.DatabaseConfig{
		Path:          ":memory:",
		WALMode:       false,
		BusyTimeoutMs: 1000,
	})
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := sqlite.Migrate(ctx, db); err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	// Insert 3 raw records from 40 days ago (same hour)
	hourBase := time.Now().UTC().AddDate(0, 0, -40).Truncate(time.Hour)
	t1 := hourBase.Add(5 * time.Minute).Format(time.RFC3339)
	t2 := hourBase.Add(15 * time.Minute).Format(time.RFC3339)
	t3 := hourBase.Add(30 * time.Minute).Format(time.RFC3339)

	// Values: 20.0, 24.0, 22.0 -> Avg: 22.0, Min: 20.0, Max: 24.0
	_, err = db.ExecContext(ctx, `
		INSERT INTO state_history (entity_id, state, recorded_at) VALUES
		('sensor.temperature', '20.0', ?),
		('sensor.temperature', '24.0', ?),
		('sensor.temperature', '22.0', ?);
	`, t1, t2, t3)
	if err != nil {
		t.Fatalf("failed to insert mock history: %v", err)
	}

	downsampler := NewDownsampler(db)
	cutoff := time.Now().UTC().AddDate(0, 0, -30) // older than 30 days

	summary, err := downsampler.DownsampleOlderThan(ctx, cutoff)
	if err != nil {
		t.Fatalf("downsample failed: %v", err)
	}

	if summary.BucketsCreated != 1 {
		t.Errorf("expected 1 bucket created, got %d", summary.BucketsCreated)
	}
	if summary.RawPointsPurged != 3 {
		t.Errorf("expected 3 raw points purged, got %d", summary.RawPointsPurged)
	}

	// Verify hourly_telemetry values
	var avg, min, max float64
	var samples int
	err = db.QueryRowContext(ctx, `
		SELECT val_avg, val_min, val_max, samples
		FROM hourly_telemetry
		WHERE entity_id = 'sensor.temperature';
	`).Scan(&avg, &min, &max, &samples)
	if err != nil {
		t.Fatalf("failed to query hourly_telemetry: %v", err)
	}

	if avg != 22.0 || min != 20.0 || max != 24.0 || samples != 3 {
		t.Errorf("unexpected downsampled stats: avg=%.1f, min=%.1f, max=%.1f, samples=%d", avg, min, max, samples)
	}
}
