// Package scene provides application services for managing and applying scene presets.
package scene

import (
	"context"
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

	now := time.Now().UTC()
	for _, action := range sc.Actions {
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
		_ = s.bus.Publish(ctx, cmd)
	}

	return s.bus.Publish(ctx, events.Event{
		ID:         uuid.NewString(),
		Type:       events.SceneActivated,
		OccurredAt: now,
		Source:     "scene_service",
		Payload: map[string]any{
			"scene_id": sc.ID,
			"name":     sc.Name,
		},
	})
}
