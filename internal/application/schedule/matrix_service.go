// Package schedule provides application services for weekly matrix cronotermostato schedules.
package schedule

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/domain/schedule"
	"github.com/google/uuid"
)

// MatrixService manages the weekly schedule matrix and time-based state transitions.
type MatrixService struct {
	repo   schedule.Repository
	bus    events.Bus
	logger *slog.Logger

	mu     sync.Mutex
	stopCh chan struct{}
}

// NewMatrixService creates a new MatrixService.
func NewMatrixService(repo schedule.Repository, bus events.Bus, logger *slog.Logger) *MatrixService {
	if logger == nil {
		logger = slog.Default()
	}
	return &MatrixService{
		repo:   repo,
		bus:    bus,
		logger: logger,
	}
}

// Create inserts a new weekly schedule.
func (s *MatrixService) Create(ctx context.Context, ws *schedule.WeeklySchedule) error {
	if ws.Name == "" {
		return fmt.Errorf("schedule name cannot be empty")
	}
	if ws.Grid == nil {
		ws.Grid = make(schedule.WeeklyGrid)
	}
	return s.repo.Save(ctx, ws)
}

// Update modifies an existing weekly schedule.
func (s *MatrixService) Update(ctx context.Context, ws *schedule.WeeklySchedule) error {
	if ws.ID == "" {
		return fmt.Errorf("schedule id cannot be empty")
	}
	return s.repo.Save(ctx, ws)
}

// Get retrieves a weekly schedule by ID.
func (s *MatrixService) Get(ctx context.Context, id string) (*schedule.WeeklySchedule, error) {
	return s.repo.GetByID(ctx, id)
}

// List returns all weekly schedules.
func (s *MatrixService) List(ctx context.Context) ([]*schedule.WeeklySchedule, error) {
	return s.repo.List(ctx)
}

// Delete removes a weekly schedule by ID.
func (s *MatrixService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// EvaluateAll evaluates all registered schedules at the given timestamp and applies state transitions.
func (s *MatrixService) EvaluateAll(ctx context.Context, now time.Time) error {
	schedules, err := s.repo.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list weekly schedules for evaluation: %w", err)
	}

	for _, ws := range schedules {
		slot, newState := ws.EvaluateAt(now)
		if newState != ws.CurrentSlotState {
			oldState := ws.CurrentSlotState
			ws.CurrentSlotState = newState
			if err := s.repo.Save(ctx, ws); err != nil {
				s.logger.ErrorContext(ctx, "failed to persist evaluated schedule state",
					"id", ws.ID, "error", err)
			}

			s.logger.InfoContext(ctx, "weekly schedule slot transition",
				"id", ws.ID, "name", ws.Name, "from", oldState, "to", newState)

			s.dispatchScheduleAction(ctx, ws, slot, newState)
		}
	}

	return nil
}

func (s *MatrixService) dispatchScheduleAction(ctx context.Context, ws *schedule.WeeklySchedule, slot *schedule.TimeSlot, state string) {
	if s.bus == nil || ws.TargetEntityID == "" {
		return
	}

	params := make(map[string]any)
	serviceName := ws.TargetService
	if serviceName == "" {
		if slot != nil && slot.Value > 0 {
			serviceName = "set_temperature"
			params["temperature"] = slot.Value
		} else if state == "off" {
			serviceName = "turn_off"
		} else {
			serviceName = "turn_on"
		}
	} else if slot != nil && slot.Value > 0 {
		params["temperature"] = slot.Value
	}

	// 1. Dispatch entity command
	_ = s.bus.Publish(ctx, events.Event{
		ID:         uuid.NewString(),
		Type:       events.CommandRequested,
		OccurredAt: time.Now().UTC(),
		Source:     "schedule:weekly_matrix",
		Payload: map[string]any{
			"entity_key": ws.TargetEntityID,
			"service":    serviceName,
			"params":     params,
		},
	})

	// 2. Publish schedule triggered event
	_ = s.bus.Publish(ctx, events.Event{
		ID:         uuid.NewString(),
		Type:       events.ScheduleTriggered,
		OccurredAt: time.Now().UTC(),
		Source:     ws.ID,
		Payload: map[string]any{
			"schedule_id": ws.ID,
			"name":        ws.Name,
			"state":       state,
			"slot":        slot,
		},
	})
}
