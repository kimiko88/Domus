// Package chore provides application-level services and workers for managing domestic chores and deadline reminders.
package chore

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	domainchore "github.com/domus-hub/domus/internal/domain/chore"
	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/google/uuid"
)

// Service provides use cases for domestic chores.
type Service struct {
	repo domainchore.Repository
	bus  events.Bus
}

// NewService creates a new chores application service.
func NewService(repo domainchore.Repository, bus events.Bus) *Service {
	return &Service{
		repo: repo,
		bus:  bus,
	}
}

// CreateTask registers a new domestic chore.
func (s *Service) CreateTask(ctx context.Context, c *domainchore.Chore) error {
	if c.ID == "" {
		c.ID = uuid.NewString()
	}
	if err := c.Validate(); err != nil {
		return err
	}
	return s.repo.Create(ctx, c)
}

// CompleteTask marks a chore completed and provisions the next occurrence if recurring.
func (s *Service) CompleteTask(ctx context.Context, id string, completedBy string) error {
	c, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	c.Status = domainchore.StatusCompleted
	c.LastCompletedAt = &now
	c.LastCompletedBy = completedBy

	if err := s.repo.Update(ctx, c); err != nil {
		return err
	}

	// Schedule next occurrence if recurring
	nextDue := c.NextDueDate(now)
	if nextDue != nil {
		nextChore := &domainchore.Chore{
			ID:             uuid.NewString(),
			Title:          c.Title,
			Description:    c.Description,
			AreaID:         c.AreaID,
			AssignedUserID: c.AssignedUserID,
			Priority:       c.Priority,
			DueDate:        nextDue,
			RecurrenceType: c.RecurrenceType,
			RecurrenceRule: c.RecurrenceRule,
			Status:         domainchore.StatusPending,
		}
		if err := s.repo.Create(ctx, nextChore); err != nil {
			slog.Warn("chores: failed to spawn next recurring chore instance", "error", err)
		}
	}

	if s.bus != nil {
		_ = s.bus.Publish(ctx, events.Event{
			ID:         uuid.NewString(),
			Type:       events.NotificationCreated,
			OccurredAt: now,
			Source:     "domus.chores",
			Payload: map[string]any{
				"title":    "Faccenda Completata",
				"message":  c.Title,
				"type":     "success",
				"chore_id": c.ID,
			},
		})
	}

	return nil
}

// Scheduler periodically checks for due chores and dispatches reminders.
type Scheduler struct {
	repo     domainchore.Repository
	bus      events.Bus
	interval time.Duration
	stopCh   chan struct{}
	wg       sync.WaitGroup
	mu       sync.Mutex
	running  bool
}

// NewScheduler constructs a background worker for chore reminders.
func NewScheduler(repo domainchore.Repository, bus events.Bus, interval time.Duration) *Scheduler {
	if interval <= 0 {
		interval = 1 * time.Minute
	}
	return &Scheduler{
		repo:     repo,
		bus:      bus,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// Start begins the background check loop.
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return errors.New("scheduler already running")
	}
	s.running = true
	s.mu.Unlock()

	s.wg.Add(1)
	go s.loop()
	return nil
}

// Stop terminates the reminder check loop.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	close(s.stopCh)
	s.mu.Unlock()

	s.wg.Wait()
}

func (s *Scheduler) loop() {
	defer s.wg.Done()
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// Initial immediate check
	_ = s.CheckDueChores(context.Background())

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			_ = s.CheckDueChores(context.Background())
		}
	}
}

// CheckDueChores evaluates overdue tasks and broadcasts notifications.
func (s *Scheduler) CheckDueChores(ctx context.Context) error {
	now := time.Now().UTC()
	overdue, err := s.repo.GetPendingDueBefore(ctx, now)
	if err != nil {
		return err
	}

	for _, c := range overdue {
		c.Status = domainchore.StatusOverdue
		if err := s.repo.Update(ctx, c); err != nil {
			continue
		}

		if s.bus != nil {
			_ = s.bus.Publish(ctx, events.Event{
				ID:         uuid.NewString(),
				Type:       events.NotificationCreated,
				OccurredAt: now,
				Source:     "domus.chores",
				Payload: map[string]any{
					"title":    "Promemoria Faccenda: " + c.Title,
					"message":  c.Description,
					"type":     "warning",
					"chore_id": c.ID,
					"user_id":  c.AssignedUserID,
				},
			})
		}
	}

	return nil
}
