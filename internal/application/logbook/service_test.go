package logbook_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	applog "github.com/domus-hub/domus/internal/application/logbook"
	"github.com/domus-hub/domus/internal/domain/events"
	domlog "github.com/domus-hub/domus/internal/domain/logbook"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogbookService_EventProcessingAndQuery(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "logbook_test.db")

	db, err := sqlite.Open(config.DatabaseConfig{
		Path:          dbPath,
		WALMode:       true,
		BusyTimeoutMs: 5000,
		MaxOpenConns:  1,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, sqlite.Migrate(ctx, db))

	repo := sqlite.NewLogbookRepository(db)
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 64, Workers: 2})
	svc := applog.NewService(repo, nil, bus)
	require.NoError(t, svc.Start(ctx))
	defer func() {
		svc.Stop()
		bus.Close()
		db.Close()
	}()

	now := time.Now().UTC()

	// 1. Publish EntityStateChanged
	require.NoError(t, bus.Publish(ctx, events.Event{
		ID:         uuid.NewString(),
		Type:       events.EntityStateChanged,
		OccurredAt: now,
		Payload: map[string]any{
			"entity_key": "light.living_room",
			"state":      "on",
		},
	}))

	// 2. Publish AutomationTriggered
	require.NoError(t, bus.Publish(ctx, events.Event{
		ID:         uuid.NewString(),
		Type:       events.AutomationTriggered,
		OccurredAt: now.Add(1 * time.Second),
		Payload: map[string]any{
			"automation_id": "auto_welcome",
			"name":          "Benvenuto a Casa",
		},
	}))

	// 3. Publish PresenceUpdated
	require.NoError(t, bus.Publish(ctx, events.Event{
		ID:         uuid.NewString(),
		Type:       events.PresenceUpdated,
		OccurredAt: now.Add(2 * time.Second),
		Payload: map[string]any{
			"name":  "Marco",
			"state": "home",
		},
	}))

	// Allow event bus workers to process
	time.Sleep(100 * time.Millisecond)

	// Query all logbook entries
	entries, err := svc.Query(ctx, domlog.Filter{Limit: 10})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(entries), 3)

	// Filter by domain "automation"
	autoEntries, err := svc.Query(ctx, domlog.Filter{Domain: "automation"})
	require.NoError(t, err)
	require.Len(t, autoEntries, 1)
	assert.Equal(t, "Automazione 'Benvenuto a Casa' attivata", autoEntries[0].Message)

	// Filter by domain "person"
	personEntries, err := svc.Query(ctx, domlog.Filter{Domain: "person"})
	require.NoError(t, err)
	require.Len(t, personEntries, 1)
	assert.Equal(t, "Marco è arrivato a casa", personEntries[0].Message)
}
