package chore

import (
	"context"
	"testing"
	"time"

	domainchore "github.com/domus-hub/domus/internal/domain/chore"
	"github.com/domus-hub/domus/internal/domain/events"
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

func TestChoreService_CompleteAndRecurring(t *testing.T) {
	repo := newMockChoreRepo()
	bus := &mockBus{}
	svc := NewService(repo, bus)

	ctx := context.Background()
	due := time.Now().UTC().Add(-1 * time.Hour)
	chore := &domainchore.Chore{
		ID:             "chore_1",
		Title:          "Pulizia filtri",
		DueDate:        &due,
		RecurrenceType: domainchore.RecurrenceDaily,
		Status:         domainchore.StatusPending,
	}
	_ = repo.Create(ctx, chore)

	err := svc.CompleteTask(ctx, "chore_1", "user_admin")
	if err != nil {
		t.Fatalf("complete task failed: %v", err)
	}

	if repo.items["chore_1"].Status != domainchore.StatusCompleted {
		t.Errorf("expected original chore completed, got %v", repo.items["chore_1"].Status)
	}

	// Should have spawned next recurring task
	if len(repo.items) != 2 {
		t.Fatalf("expected 2 chores in repo after recurring completion, got %d", len(repo.items))
	}
}

func TestChoreScheduler_CheckDueChores(t *testing.T) {
	repo := newMockChoreRepo()
	bus := &mockBus{}
	scheduler := NewScheduler(repo, bus, 10*time.Millisecond)

	ctx := context.Background()
	past := time.Now().UTC().Add(-10 * time.Minute)
	c := &domainchore.Chore{
		ID:      "chore_overdue",
		Title:   "Buttare la spazzatura",
		DueDate: &past,
		Status:  domainchore.StatusPending,
	}
	_ = repo.Create(ctx, c)

	err := scheduler.CheckDueChores(ctx)
	if err != nil {
		t.Fatalf("check due chores failed: %v", err)
	}

	if repo.items["chore_overdue"].Status != domainchore.StatusOverdue {
		t.Errorf("expected status overdue, got %s", repo.items["chore_overdue"].Status)
	}

	if len(bus.events) != 1 || bus.events[0].Type != events.NotificationCreated {
		t.Errorf("expected 1 notification event dispatched, got %d", len(bus.events))
	}
}
