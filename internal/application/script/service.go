// Package script provides execution, concurrency management, and lifecycle for scripts.
package script

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/automation"
	"github.com/domus-hub/domus/internal/domain/events"
	domscript "github.com/domus-hub/domus/internal/domain/script"
	"github.com/google/uuid"
)

// Service coordinates script CRUD and concurrent execution.
type Service struct {
	repo       domscript.Repository
	bus        events.Bus
	mu         sync.Mutex
	activeRuns map[string]map[string]context.CancelFunc // scriptID -> runID -> cancelFunc
}

// NewService creates a new script Service instance.
func NewService(repo domscript.Repository, bus events.Bus) *Service {
	return &Service{
		repo:       repo,
		bus:        bus,
		activeRuns: make(map[string]map[string]context.CancelFunc),
	}
}

// Create validates and saves a new script.
func (s *Service) Create(ctx context.Context, sc *domscript.Script) error {
	if err := sc.Validate(); err != nil {
		return err
	}
	return s.repo.Create(ctx, sc)
}

// Get retrieves a script by ID.
func (s *Service) Get(ctx context.Context, id string) (*domscript.Script, error) {
	return s.repo.Get(ctx, id)
}

// List returns all scripts.
func (s *Service) List(ctx context.Context) ([]*domscript.Script, error) {
	return s.repo.List(ctx)
}

// Update validates and persists script changes.
func (s *Service) Update(ctx context.Context, sc *domscript.Script) error {
	if err := sc.Validate(); err != nil {
		return err
	}
	return s.repo.Update(ctx, sc)
}

// Delete removes a script by ID.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// Execute runs a script with optional input variables and concurrency mode enforcement.
func (s *Service) Execute(ctx context.Context, scriptID string, variables map[string]any) error {
	sc, err := s.repo.Get(ctx, scriptID)
	if err != nil {
		return err
	}

	runID := uuid.NewString()
	execCtx, cancel := context.WithCancel(ctx)

	// Concurrency mode control
	s.mu.Lock()
	runs, exists := s.activeRuns[sc.ID]
	if !exists {
		runs = make(map[string]context.CancelFunc)
		s.activeRuns[sc.ID] = runs
	}

	switch sc.Mode {
	case domscript.ModeSingle:
		if len(runs) > 0 {
			s.mu.Unlock()
			cancel()
			return domscript.ErrScriptAlreadyRunning
		}
	case domscript.ModeRestart:
		for _, c := range runs {
			c() // Cancel existing executions
		}
		s.activeRuns[sc.ID] = make(map[string]context.CancelFunc)
		runs = s.activeRuns[sc.ID]
	case domscript.ModeParallel, domscript.ModeQueued:
		if len(runs) >= sc.MaxRuns {
			s.mu.Unlock()
			cancel()
			return domscript.ErrMaxRunsExceeded
		}
	}

	runs[runID] = cancel
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		if r, ok := s.activeRuns[sc.ID]; ok {
			delete(r, runID)
			if len(r) == 0 {
				delete(s.activeRuns, sc.ID)
			}
		}
		s.mu.Unlock()
		cancel()
	}()

	// Broadcast script.started
	startTime := time.Now().UTC()
	if s.bus != nil {
		_ = s.bus.Publish(ctx, events.Event{
			ID:         uuid.NewString(),
			Type:       "script.started",
			OccurredAt: startTime,
			Source:     "script_runner",
			Payload: map[string]any{
				"script_id": sc.ID,
				"name":      sc.Name,
				"run_id":    runID,
				"variables": variables,
			},
		})
	}

	slog.Info("executing script", "script_id", sc.ID, "name", sc.Name, "run_id", runID)

	// Execute sequence
	for idx, act := range sc.Sequence {
		if err := execCtx.Err(); err != nil {
			s.publishFailed(ctx, sc, runID, "cancelled")
			return err
		}

		if err := s.executeAction(execCtx, act, sc, variables); err != nil {
			s.publishFailed(ctx, sc, runID, fmt.Sprintf("action %d failed: %v", idx, err))
			return err
		}
	}

	// Broadcast script.completed
	finishTime := time.Now().UTC()
	if s.bus != nil {
		_ = s.bus.Publish(ctx, events.Event{
			ID:         uuid.NewString(),
			Type:       "script.completed",
			OccurredAt: finishTime,
			Source:     "script_runner",
			Payload: map[string]any{
				"script_id":   sc.ID,
				"name":        sc.Name,
				"run_id":      runID,
				"duration_ms": finishTime.Sub(startTime).Milliseconds(),
			},
		})
	}

	return nil
}

func (s *Service) executeAction(ctx context.Context, act automation.Action, sc *domscript.Script, vars map[string]any) error {
	switch act.Type {
	case "service":
		targetKey := act.TargetEntityKey
		// Variable substitution if key contains {var}
		for k, v := range vars {
			placeholder := fmt.Sprintf("{%s}", k)
			if targetKey == placeholder {
				targetKey = fmt.Sprintf("%v", v)
			}
		}

		params := make(map[string]any)
		for k, v := range act.Params {
			params[k] = v
		}
		for k, v := range vars {
			params[k] = v
		}

		if s.bus != nil {
			return s.bus.Publish(ctx, events.Event{
				ID:         uuid.NewString(),
				Type:       events.CommandRequested,
				OccurredAt: time.Now().UTC(),
				Source:     fmt.Sprintf("script:%s", sc.ID),
				Payload: map[string]any{
					"entity_key": targetKey,
					"service":    act.Service,
					"params":     params,
				},
			})
		}

	case "delay":
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(act.DelayMs) * time.Millisecond):
		}

	case "parallel":
		var wg sync.WaitGroup
		var parallelErr error
		var errMu sync.Mutex

		for _, subAct := range act.Parallel {
			wg.Add(1)
			go func(a automation.Action) {
				defer wg.Done()
				if err := s.executeAction(ctx, a, sc, vars); err != nil {
					errMu.Lock()
					if parallelErr == nil {
						parallelErr = err
					}
					errMu.Unlock()
				}
			}(subAct)
		}
		wg.Wait()
		if parallelErr != nil {
			return parallelErr
		}
	}

	return nil
}

func (s *Service) publishFailed(ctx context.Context, sc *domscript.Script, runID, reason string) {
	if s.bus != nil {
		_ = s.bus.Publish(ctx, events.Event{
			ID:         uuid.NewString(),
			Type:       "script.failed",
			OccurredAt: time.Now().UTC(),
			Source:     "script_runner",
			Payload: map[string]any{
				"script_id": sc.ID,
				"name":      sc.Name,
				"run_id":    runID,
				"reason":    reason,
			},
		})
	}
}
