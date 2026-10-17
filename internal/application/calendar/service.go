package calendar

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/calendar"
	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/domain/schedule"
	"github.com/google/uuid"
)

type Repository interface {
	SaveSchedule(ctx context.Context, s *schedule.Schedule) error
	GetSchedule(ctx context.Context, id string) (*schedule.Schedule, error)
	ListSchedules(ctx context.Context) ([]*schedule.Schedule, error)
	DeleteSchedule(ctx context.Context, id string) error

	SaveCalendarEvent(ctx context.Context, e *calendar.Event) error
	ListCalendarEvents(ctx context.Context, from, to time.Time) ([]*calendar.Event, error)
	DeleteCalendarEvent(ctx context.Context, id string) error
}

type MemoryRepository struct {
	mu        sync.RWMutex
	schedules map[string]*schedule.Schedule
	events    map[string]*calendar.Event
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		schedules: make(map[string]*schedule.Schedule),
		events:    make(map[string]*calendar.Event),
	}
}

func (m *MemoryRepository) SaveSchedule(ctx context.Context, s *schedule.Schedule) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.schedules[s.ID] = s
	return nil
}

func (m *MemoryRepository) GetSchedule(ctx context.Context, id string) (*schedule.Schedule, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.schedules[id]
	if !ok {
		return nil, fmt.Errorf("schedule not found: %s", id)
	}
	return s, nil
}

func (m *MemoryRepository) ListSchedules(ctx context.Context) ([]*schedule.Schedule, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	list := make([]*schedule.Schedule, 0, len(m.schedules))
	for _, s := range m.schedules {
		list = append(list, s)
	}
	return list, nil
}

func (m *MemoryRepository) DeleteSchedule(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.schedules, id)
	return nil
}

func (m *MemoryRepository) SaveCalendarEvent(ctx context.Context, e *calendar.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events[e.ID] = e
	return nil
}

func (m *MemoryRepository) ListCalendarEvents(ctx context.Context, from, to time.Time) ([]*calendar.Event, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var res []*calendar.Event
	for _, e := range m.events {
		if (from.IsZero() || e.EndTime.After(from)) && (to.IsZero() || e.StartTime.Before(to)) {
			res = append(res, e)
		}
	}
	return res, nil
}

func (m *MemoryRepository) DeleteCalendarEvent(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.events, id)
	return nil
}

type Service struct {
	repo Repository
	bus  events.Bus
}

func NewService(repo Repository, bus events.Bus) *Service {
	return &Service{
		repo: repo,
		bus:  bus,
	}
}

func (s *Service) CreateSchedule(ctx context.Context, name, entityKey string, grid schedule.WeeklyGrid) (*schedule.Schedule, error) {
	sched := &schedule.Schedule{
		ID:        "sched_" + uuid.NewString()[:8],
		Name:      name,
		EntityKey: entityKey,
		Grid:      grid,
		Enabled:   true,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := s.repo.SaveSchedule(ctx, sched); err != nil {
		return nil, err
	}
	return sched, nil
}

func (s *Service) ListSchedules(ctx context.Context) ([]*schedule.Schedule, error) {
	return s.repo.ListSchedules(ctx)
}

func (s *Service) UpdateSchedule(ctx context.Context, id, name, entityKey string, enabled bool, grid schedule.WeeklyGrid) (*schedule.Schedule, error) {
	sched, err := s.repo.GetSchedule(ctx, id)
	if err != nil {
		sched = &schedule.Schedule{
			ID:        id,
			CreatedAt: time.Now().UTC(),
		}
	}
	if name != "" {
		sched.Name = name
	}
	if entityKey != "" {
		sched.EntityKey = entityKey
	}
	sched.Enabled = enabled
	if grid != nil {
		sched.Grid = grid
	}
	sched.UpdatedAt = time.Now().UTC()
	if err := s.repo.SaveSchedule(ctx, sched); err != nil {
		return nil, err
	}
	return sched, nil
}

func (s *Service) DeleteSchedule(ctx context.Context, id string) error {
	return s.repo.DeleteSchedule(ctx, id)
}

func (s *Service) EvaluateSchedule(ctx context.Context, scheduleID string, at time.Time) (*schedule.TimeSlot, bool, error) {
	sched, err := s.repo.GetSchedule(ctx, scheduleID)
	if err != nil {
		return nil, false, err
	}
	slot, active := sched.GetActiveSlot(at)
	return slot, active, nil
}

func (s *Service) CreateCalendarEvent(ctx context.Context, summary, description string, start, end time.Time, location string) (*calendar.Event, error) {
	evt := &calendar.Event{
		ID:          "evt_" + uuid.NewString()[:8],
		Summary:     summary,
		Description: description,
		StartTime:   start,
		EndTime:     end,
		Location:    location,
		Source:      "local",
		CreatedAt:   time.Now().UTC(),
	}
	if err := s.repo.SaveCalendarEvent(ctx, evt); err != nil {
		return nil, err
	}
	return evt, nil
}

func (s *Service) ImportICal(ctx context.Context, icsContent string) (int, error) {
	eventsList, err := calendar.ParseICal(icsContent)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, e := range eventsList {
		if err := s.repo.SaveCalendarEvent(ctx, e); err == nil {
			count++
		}
	}
	return count, nil
}

func (s *Service) ListEvents(ctx context.Context, from, to time.Time) ([]*calendar.Event, error) {
	return s.repo.ListCalendarEvents(ctx, from, to)
}
