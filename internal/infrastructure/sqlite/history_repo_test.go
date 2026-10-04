package sqlite_test

import (
	"context"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/domain/history"
)

func TestHistoryRepository(t *testing.T) {
	ctx := context.Background()
	repos := setupTestDB(t)
	repo := repos.History

	t.Run("SaveBatch and ListByEntity", func(t *testing.T) {
		t0 := time.Date(2026, 10, 4, 10, 0, 0, 0, time.UTC)
		t1 := t0.Add(1 * time.Minute)
		t2 := t0.Add(2 * time.Minute)

		entries := []*history.StateHistory{
			{EntityID: "sensor.living_temp", State: `{"temp": 20.0}`, RecordedAt: t0},
			{EntityID: "sensor.living_temp", State: `{"temp": 21.0}`, RecordedAt: t1},
			{EntityID: "sensor.living_temp", State: `{"temp": 21.5}`, RecordedAt: t2},
			{EntityID: "sensor.kitchen_temp", State: `{"temp": 19.5}`, RecordedAt: t1},
		}

		err := repo.SaveBatch(ctx, entries)
		if err != nil {
			t.Fatalf("unexpected SaveBatch error: %v", err)
		}

		// Query living_temp
		res, err := repo.ListByEntity(ctx, history.HistoryFilter{
			EntityID: "sensor.living_temp",
		})
		if err != nil {
			t.Fatalf("unexpected ListByEntity error: %v", err)
		}

		if len(res) != 3 {
			t.Fatalf("expected 3 entries, got %d", len(res))
		}
		if res[0].RecordedAt.Format(time.RFC3339) != t0.Format(time.RFC3339) {
			t.Fatalf("expected first entry at %v, got %v", t0, res[0].RecordedAt)
		}
		if res[2].RecordedAt.Format(time.RFC3339) != t2.Format(time.RFC3339) {
			t.Fatalf("expected last entry at %v, got %v", t2, res[2].RecordedAt)
		}

		// Filter with time window
		resWindow, err := repo.ListByEntity(ctx, history.HistoryFilter{
			EntityID: "sensor.living_temp",
			From:     t0.Add(30 * time.Second),
			To:       t1.Add(30 * time.Second),
		})
		if err != nil {
			t.Fatalf("unexpected error with window: %v", err)
		}
		if len(resWindow) != 1 {
			t.Fatalf("expected 1 entry in window, got %d", len(resWindow))
		}
		if resWindow[0].State != `{"temp": 21.0}` {
			t.Fatalf("expected state temp 21.0, got %s", resWindow[0].State)
		}

		// Filter with limit
		resLimit, err := repo.ListByEntity(ctx, history.HistoryFilter{
			EntityID: "sensor.living_temp",
			Limit:    2,
		})
		if err != nil {
			t.Fatalf("unexpected error with limit: %v", err)
		}
		if len(resLimit) != 2 {
			t.Fatalf("expected 2 entries with limit, got %d", len(resLimit))
		}
	})

	t.Run("PurgeOlderThan", func(t *testing.T) {
		tOld := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		tNew := time.Date(2026, 10, 4, 10, 0, 0, 0, time.UTC)

		oldEntries := []*history.StateHistory{
			{EntityID: "sensor.old", State: `1`, RecordedAt: tOld},
			{EntityID: "sensor.old", State: `2`, RecordedAt: tOld.Add(time.Hour)},
			{EntityID: "sensor.new", State: `3`, RecordedAt: tNew},
		}

		if err := repo.SaveBatch(ctx, oldEntries); err != nil {
			t.Fatalf("SaveBatch failed: %v", err)
		}

		cutoff := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		purged, err := repo.PurgeOlderThan(ctx, cutoff)
		if err != nil {
			t.Fatalf("PurgeOlderThan failed: %v", err)
		}
		if purged != 2 {
			t.Fatalf("expected 2 purged records, got %d", purged)
		}

		remaining, err := repo.ListByEntity(ctx, history.HistoryFilter{EntityID: "sensor.old"})
		if err != nil {
			t.Fatalf("ListByEntity failed: %v", err)
		}
		if len(remaining) != 0 {
			t.Fatalf("expected 0 remaining old records, got %d", len(remaining))
		}
	})
}
