package helper

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/domain/helper"
	"github.com/google/uuid"
)

// Service coordinates User Helpers lifecycle, actions, and background timers.
type Service struct {
	repo helper.Repository
	bus  events.Bus

	mu         sync.Mutex
	timerStops map[string]chan struct{}
}

// NewService constructs a new User Helper service.
func NewService(repo helper.Repository, bus events.Bus) *Service {
	s := &Service{
		repo:       repo,
		bus:        bus,
		timerStops: make(map[string]chan struct{}),
	}
	if bus != nil {
		bus.Subscribe(events.CommandRequested, s.handleCommandRequested)
	}
	return s
}

// Create creates a helper and emits an EntityStateChanged event.
func (s *Service) Create(ctx context.Context, h *helper.Helper) error {
	if err := s.repo.Create(ctx, h); err != nil {
		return err
	}
	s.publishState(ctx, h)
	return nil
}

// GetByID returns a helper by its ID.
func (s *Service) GetByID(ctx context.Context, id string) (*helper.Helper, error) {
	return s.repo.GetByID(ctx, id)
}

// List returns all helpers, optionally by domain.
func (s *Service) List(ctx context.Context, domain string) ([]*helper.Helper, error) {
	return s.repo.List(ctx, domain)
}

// Delete removes a helper.
func (s *Service) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	if stop, ok := s.timerStops[id]; ok {
		close(stop)
		delete(s.timerStops, id)
	}
	s.mu.Unlock()
	return s.repo.Delete(ctx, id)
}

// ExecuteAction executes a domain-specific operation on a helper.
func (s *Service) ExecuteAction(ctx context.Context, entityKey string, action string, params map[string]any) error {
	h, err := s.repo.GetByEntityKey(ctx, entityKey)
	if err != nil {
		return err
	}

	switch h.Domain {
	case helper.DomainInputBoolean:
		switch action {
		case "turn_on":
			h.State = "on"
		case "turn_off":
			h.State = "off"
		case "toggle":
			h.ToggleBoolean()
		default:
			return fmt.Errorf("unknown input_boolean action: %s", action)
		}

	case helper.DomainInputNumber:
		if action == "set_value" {
			var val float64
			if v, ok := params["value"].(float64); ok {
				val = v
			} else if v, ok := params["value"].(int); ok {
				val = float64(v)
			}
			if err := h.SetNumberValue(val); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("unknown input_number action: %s", action)
		}

	case helper.DomainTimer:
		var cfg helper.TimerConfig
		_ = json.Unmarshal([]byte(h.ConfigJSON), &cfg)

		switch action {
		case "start":
			dur := cfg.DurationSeconds
			if d, ok := params["duration"].(float64); ok && d > 0 {
				dur = int(d)
			}
			cfg.DurationSeconds = dur
			cfg.RemainingSeconds = dur
			cfg.Status = "active"
			endsAt := time.Now().UTC().Add(time.Duration(dur) * time.Second)
			cfg.EndsAt = &endsAt
			b, _ := json.Marshal(cfg)
			h.ConfigJSON = string(b)
			h.State = "active"

			s.startTimerTicker(h.ID, h.EntityKey, dur)

		case "pause":
			s.stopTimerTicker(h.ID)
			cfg.Status = "paused"
			cfg.EndsAt = nil
			b, _ := json.Marshal(cfg)
			h.ConfigJSON = string(b)
			h.State = "paused"

		case "cancel":
			s.stopTimerTicker(h.ID)
			cfg.Status = "idle"
			cfg.RemainingSeconds = cfg.DurationSeconds
			cfg.EndsAt = nil
			b, _ := json.Marshal(cfg)
			h.ConfigJSON = string(b)
			h.State = "idle"

		case "finish":
			s.stopTimerTicker(h.ID)
			cfg.Status = "idle"
			cfg.RemainingSeconds = 0
			cfg.EndsAt = nil
			b, _ := json.Marshal(cfg)
			h.ConfigJSON = string(b)
			h.State = "idle"

			if s.bus != nil {
				_ = s.bus.Publish(ctx, events.Event{
					ID:         uuid.NewString(),
					Type:       "timer.finished",
					OccurredAt: time.Now().UTC(),
					Source:     "helper_service",
					Payload: map[string]any{
						"entity_key": h.EntityKey,
					},
				})
			}
		default:
			return fmt.Errorf("unknown timer action: %s", action)
		}
	}

	if err := s.repo.Update(ctx, h); err != nil {
		return err
	}
	s.publishState(ctx, h)
	return nil
}

func (s *Service) startTimerTicker(id string, entityKey string, durSec int) {
	s.mu.Lock()
	if stop, ok := s.timerStops[id]; ok {
		close(stop)
	}
	stopCh := make(chan struct{})
	s.timerStops[id] = stopCh
	s.mu.Unlock()

	go func() {
		select {
		case <-stopCh:
			return
		case <-time.After(time.Duration(durSec) * time.Second):
			ctx := context.Background()
			_ = s.ExecuteAction(ctx, entityKey, "finish", nil)
		}
	}()
}

func (s *Service) stopTimerTicker(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if stop, ok := s.timerStops[id]; ok {
		close(stop)
		delete(s.timerStops, id)
	}
}

func (s *Service) publishState(ctx context.Context, h *helper.Helper) {
	if s.bus == nil {
		return
	}
	_ = s.bus.Publish(ctx, events.Event{
		ID:         uuid.NewString(),
		Type:       events.EntityStateChanged,
		OccurredAt: time.Now().UTC(),
		Source:     "helper_service",
		Payload: map[string]any{
			"entity_key": h.EntityKey,
			"state":      h.State,
			"attributes": h.ConfigJSON,
			"available":  true,
		},
	})
}

func (s *Service) handleCommandRequested(ctx context.Context, e events.Event) error {
	payload, ok := e.Payload.(map[string]any)
	if !ok {
		return nil
	}

	entityKey, _ := payload["entity_key"].(string)
	if !strings.HasPrefix(entityKey, "input_boolean.") &&
		!strings.HasPrefix(entityKey, "input_number.") &&
		!strings.HasPrefix(entityKey, "timer.") {
		return nil
	}

	cmd, _ := payload["command"].(string)
	if cmd == "" {
		cmd, _ = payload["service"].(string)
	}
	params, _ := payload["params"].(map[string]any)

	slog.Debug("helper service handling command", "entity_key", entityKey, "cmd", cmd)
	return s.ExecuteAction(ctx, entityKey, cmd, params)
}
