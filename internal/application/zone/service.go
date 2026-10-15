// Package zone provides spatial geofencing services and presence resolution across multi-device trackers.
package zone

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	domzone "github.com/domus-hub/domus/internal/domain/zone"
	"github.com/domus-hub/domus/internal/domain/events"
)

// Service coordinates spatial multi-zone geofencing and multi-tracker presence resolution.
type Service struct {
	repo domzone.Repository
	bus  events.Bus
}

// NewService creates a new spatial geofencing Service instance.
func NewService(repo domzone.Repository, bus events.Bus) *Service {
	return &Service{
		repo: repo,
		bus:  bus,
	}
}

// UpdatePersonLocation evaluates the coordinates against defined spatial zones and updates person state.
func (s *Service) UpdatePersonLocation(ctx context.Context, personID string, lat, lon float64) (*domzone.Person, error) {
	person, err := s.repo.GetPerson(ctx, personID)
	if err != nil {
		return nil, err
	}

	zones, err := s.repo.ListZones(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query zones: %w", err)
	}

	newState := "not_home"
	var matchedZone *domzone.Zone

	for _, z := range zones {
		if z.Passive {
			continue
		}
		dist := z.DistanceTo(lat, lon)
		if dist <= z.Radius {
			matchedZone = z
			if strings.EqualFold(z.Name, "Casa") || z.ID == "zone_home" {
				newState = "home"
			} else {
				newState = z.Name
			}
			break
		}
	}

	oldState := person.State
	person.Latitude = &lat
	person.Longitude = &lon
	person.State = newState

	if err := s.repo.UpdatePerson(ctx, person); err != nil {
		return nil, err
	}

	if oldState != newState {
		slog.Info("person presence state updated", "person", person.Name, "from", oldState, "to", newState)
		if s.bus != nil {
			_ = s.bus.Publish(ctx, events.Event{
				ID:         uuid.NewString(),
				Type:       events.PresenceUpdated,
				OccurredAt: time.Now().UTC(),
				Source:     "zone_service",
				Payload: map[string]any{
					"person_id": person.ID,
					"name":      person.Name,
					"state":     newState,
					"latitude":  lat,
					"longitude": lon,
					"zone":      matchedZone,
				},
			})
		}
	}

	return person, nil
}
