package history_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	apphistory "github.com/domus-hub/domus/internal/application/history"
	domhistory "github.com/domus-hub/domus/internal/domain/history"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
)

type mockHistoryRepo struct {
	mu      sync.Mutex
	batches [][]*domhistory.StateHistory
}

func (m *mockHistoryRepo) SaveBatch(ctx context.Context, entries []*domhistory.StateHistory) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	copied := make([]*domhistory.StateHistory, len(entries))
	copy(copied, entries)
	m.batches = append(m.batches, copied)
	return nil
}

func (m *mockHistoryRepo) ListByEntity(ctx context.Context, filter domhistory.HistoryFilter) ([]*domhistory.StateHistory, error) {
	return nil, nil
}

func (m *mockHistoryRepo) PurgeOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	return 0, nil
}

func TestRecorder_BatchedFlush(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 100, Workers: 1})
	defer bus.Close()

	repo := &mockHistoryRepo{}
	recorder := apphistory.NewRecorder(repo, bus, apphistory.RecorderConfig{
		BatchSize:     3,
		FlushInterval: 50 * time.Millisecond,
	})

	if err := recorder.Start(ctx); err != nil {
		t.Fatalf("failed to start recorder: %v", err)
	}
	defer recorder.Stop()

	// Publish 2 events (less than BatchSize)
	state1, _ := json.Marshal("on")
	state2, _ := json.Marshal("off")
	recorder.RecordState("light.salon", string(state1), time.Now())
	recorder.RecordState("light.kitchen", string(state2), time.Now())

	// Give time for interval flush
	time.Sleep(100 * time.Millisecond)

	repo.mu.Lock()
	countBatches := len(repo.batches)
	var totalEntries int
	for _, b := range repo.batches {
		totalEntries += len(b)
	}
	repo.mu.Unlock()

	if countBatches == 0 || totalEntries != 2 {
		t.Fatalf("expected 2 entries flushed via interval, got %d entries in %d batches", totalEntries, countBatches)
	}

	// Publish 3 events rapidly to trigger immediate batch flush
	for i := 0; i < 3; i++ {
		recorder.RecordState("sensor.temp", `{"val":20}`, time.Now())
	}

	// Wait briefly for worker
	time.Sleep(30 * time.Millisecond)

	repo.mu.Lock()
	var totalAfter int
	for _, b := range repo.batches {
		totalAfter += len(b)
	}
	repo.mu.Unlock()

	if totalAfter != 5 {
		t.Fatalf("expected 5 total entries, got %d", totalAfter)
	}
}

func TestRecorder_FlushOnStop(t *testing.T) {
	ctx := context.Background()
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 100, Workers: 1})
	defer bus.Close()

	repo := &mockHistoryRepo{}
	recorder := apphistory.NewRecorder(repo, bus, apphistory.RecorderConfig{
		BatchSize:     100,
		FlushInterval: 10 * time.Second, // Long interval
	})

	if err := recorder.Start(ctx); err != nil {
		t.Fatalf("failed to start recorder: %v", err)
	}

	recorder.RecordState("switch.fan", `"on"`, time.Now())
	recorder.Stop() // Should flush remaining items

	repo.mu.Lock()
	defer repo.mu.Unlock()
	if len(repo.batches) != 1 || len(repo.batches[0]) != 1 {
		t.Fatalf("expected 1 entry flushed on stop, got %v", repo.batches)
	}
}
