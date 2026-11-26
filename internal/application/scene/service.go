// Package scene provides application services for managing and applying scene presets.
package scene

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/domain/scene"
	"github.com/google/uuid"
)

// Service coordinates scene operations and hardware activations.
type Service struct {
	repo scene.Repository
	bus  events.Bus
}

// NewService constructs a scene application Service.
func NewService(repo scene.Repository, bus events.Bus) *Service {
	return &Service{
		repo: repo,
		bus:  bus,
	}
}

// ApplyScene activates a scene preset by dispatching commands to all configured entities.
func (s *Service) ApplyScene(ctx context.Context, sceneID string) error {
	sc, err := s.repo.GetByID(ctx, sceneID)
	if err != nil {
		return err
	}

	if s.bus == nil { return errors.New("scene: event bus is not configured") }
	now := time.Now().UTC()
	for index, action := range sc.Actions {
		cmd := events.Event{
			ID:         uuid.NewString(),
			Type:       events.CommandRequested,
			OccurredAt: now,
			Source:     "scene:" + sc.ID,
			Payload: map[string]any{
				"entity_key": action.EntityKey,
				"service":    action.Service,
				"params":     action.Params,
			},
		}
		if err := s.bus.Publish(ctx, cmd); err != nil {
			return fmt.Errorf("scene: dispatch action %d for %s: %w", index+1, action.EntityKey, err)
		}
	}

	// SceneActivated confirms dispatch acceptance, not physical state.
	if err := s.bus.Publish(ctx, events.Event{
		ID:         uuid.NewString(),
		Type:       events.SceneActivated,
		OccurredAt: now,
		Source:     "scene_service",
		Payload: map[string]any{
			"scene_id": sc.ID,
			"name":     sc.Name,
		},
	}); err != nil {
		return fmt.Errorf("scene: publish activation event: %w", err)
	}
	return nil
}
