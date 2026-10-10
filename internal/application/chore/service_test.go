package chore

import (
	"context"
	"testing"
	"time"

	domainchore "github.com/domus-hub/domus/internal/domain/chore"
	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockChoreRepo struct {
	items map[string]*domainchore.Chore
}

func newMockChoreRepo() *mockChoreRepo {
	return &mockChoreRepo{items: make(map[string]*domainchore.Chore)}
}

func (m *mockChoreRepo) Create(ctx context.Context, c *domainchore.Chore) error {
	m.items[c.ID] = c
	return nil
}

func (m *mockChoreRepo) GetByID(ctx context.Context, id string) (*domainchore.Chore, error) {
	if c, ok := m.items[id]; ok {
		return c, nil
	}
	return nil, domainchore.ErrChoreNotFound
}

func (m *mockChoreRepo) List(ctx context.Context, f domainchore.Filter) ([]*domainchore.Chore, error) {
	var res []*domainchore.Chore
	for _, c := range m.items {
		res = append(res, c)
	}
	return res, nil
}

func (m *mockChoreRepo) Update(ctx context.Context, c *domainchore.Chore) error {
	m.items[c.ID] = c
	return nil
}

func (m *mockChoreRepo) Delete(ctx context.Context, id string) error {
	delete(m.items, id)
	return nil
}

func (m *mockChoreRepo) GetPendingDueBefore(ctx context.Context, cutoff time.Time) ([]*domainchore.Chore, error) {
	var res []*domainchore.Chore
	for _, c := range m.items {
		if c.Status == domainchore.StatusPending && c.DueDate != nil && c.DueDate.Before(cutoff) {
			res = append(res, c)
		}
	}
	return res, nil
}

type mockBus struct {
	events []events.Event
}

func (b *mockBus) Publish(ctx context.Context, evt events.Event) error {
	b.events = append(b.events, evt)
	return nil
}
func (b *mockBus) Subscribe(eventType events.EventType, handler events.Handler) func() {
	return func() {}
}
func (b *mockBus) SubscribeAll(handler events.Handler) func() {
	return func() {}
}

func TestChoreService_CreateTask(t *testing.T) {
	repo := newMockChoreRepo()
	svc := NewService(repo, nil)
	ctx := context.Background()

	t.Run("valid chore auto-generates ID", func(t *testing.T) {
		c := &domainchore.Chore{Title: "Pulizia vetri"}
		err := svc.CreateTask(ctx, c)
		require.NoError(t, err)
		assert.NotEmpty(t, c.ID)
		assert.Equal(t, domainchore.PriorityMedium, c.Priority)
		assert.Equal(t, domainchore.StatusPending, c.Status)
	})

	t.Run("invalid chore fails validation", func(t *testing.T) {
		c := &domainchore.Chore{Title: "   "}
		err := svc.CreateTask(ctx, c)
		assert.ErrorIs(t, err, domainchore.ErrInvalidTitle)
	})
}

func TestChoreService_CompleteTask(t *testing.T) {
	repo := newMockChoreRepo()
	bus := &mockBus{}
	svc := NewService(repo, bus)
	ctx := context.Background()

	t.Run("not found returns error", func(t *testing.T) {
		err := svc.CompleteTask(ctx, "non-existent-id", "admin")
		assert.ErrorIs(t, err, domainchore.ErrChoreNotFound)
	})

	t.Run("non recurring chore completes without spawning new instance", func(t *testing.T) {
		chore := &domainchore.Chore{
			ID:             "chore_single",
			Title:          "Cambio lampadina",
			RecurrenceType: domainchore.RecurrenceNone,
			Status:         domainchore.StatusPending,
		}
		_ = repo.Create(ctx, chore)

		err := svc.CompleteTask(ctx, "chore_single", "alice")
		require.NoError(t, err)

		updated := repo.items["chore_single"]
		assert.Equal(t, domainchore.StatusCompleted, updated.Status)
		assert.Equal(t, "alice", updated.LastCompletedBy)
		assert.NotNil(t, updated.LastCompletedAt)
		assert.Equal(t, 1, len(repo.items))
	})

	t.Run("recurring chore completes and spawns next instance", func(t *testing.T) {
		due := time.Now().UTC().Add(-1 * time.Hour)
		chore := &domainchore.Chore{
			ID:             "chore_recurring",
			Title:          "Pulizia filtri condizionatore",
			DueDate:        &due,
			RecurrenceType: domainchore.RecurrenceWeekly,
			Status:         domainchore.StatusPending,
		}
		_ = repo.Create(ctx, chore)

		err := svc.CompleteTask(ctx, "chore_recurring", "bob")
		require.NoError(t, err)

		assert.Equal(t, domainchore.StatusCompleted, repo.items["chore_recurring"].Status)
		// Total items should now be 3 (chore_single, chore_recurring, + new recurring instance)
		assert.Equal(t, 3, len(repo.items))
	})
}

func TestChoreScheduler_LifecycleAndDueCheck(t *testing.T) {
	repo := newMockChoreRepo()
	bus := &mockBus{}
	scheduler := NewScheduler(repo, bus, 20*time.Millisecond)

	ctx := context.Background()

	// 1. Double stop is safe
	scheduler.Stop()

	// 2. Add an overdue chore
	past := time.Now().UTC().Add(-10 * time.Minute)
	c := &domainchore.Chore{
		ID:          "chore_overdue",
		Title:       "Portare fuori il vetro",
		Description: "Bidone verde",
		DueDate:     &past,
		Status:      domainchore.StatusPending,
	}
	_ = repo.Create(ctx, c)

	// 3. Start scheduler
	err := scheduler.Start(ctx)
	require.NoError(t, err)

	// 4. Double start errors
	err = scheduler.Start(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")

	// Wait for loop tick
	time.Sleep(50 * time.Millisecond)

	// 5. Stop scheduler
	scheduler.Stop()

	assert.Equal(t, domainchore.StatusOverdue, repo.items["chore_overdue"].Status)
	assert.True(t, len(bus.events) >= 1)
	assert.Equal(t, events.NotificationCreated, bus.events[0].Type)
}
