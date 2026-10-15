// Package climate provides the Better Thermostat intelligent climate compensation service.
package climate

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/climate"
	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/google/uuid"
)

// Service coordinates dynamic TRV calibration and window-open shutoff.
type Service struct {
	repo   climate.Repository
	bus    events.Bus
	logger *slog.Logger

	mu             sync.RWMutex
	byTRV          map[string]*climate.ClimateCompensation
	byRoomSensor   map[string][]*climate.ClimateCompensation
	byWindowSensor map[string][]*climate.ClimateCompensation
}

// NewService constructs a new ClimateCompensator Service and initializes indexes.
func NewService(repo climate.Repository, bus events.Bus, logger *slog.Logger) (*Service, error) {
	if logger == nil {
		logger = slog.Default()
	}

	s := &Service{
		repo:           repo,
		bus:            bus,
		logger:         logger,
		byTRV:          make(map[string]*climate.ClimateCompensation),
		byRoomSensor:   make(map[string][]*climate.ClimateCompensation),
		byWindowSensor: make(map[string][]*climate.ClimateCompensation),
	}

	// Warm up in-memory index from repository
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	items, err := repo.List(ctx)
	if err == nil {
		s.rebuildIndexes(items)
	}

	// Subscribe to EntityStateChanged events if bus is available
	if s.bus != nil {
		s.bus.Subscribe(events.EntityStateChanged, s.handleEntityStateChanged)
	}

	return s, nil
}

func (s *Service) rebuildIndexes(items []*climate.ClimateCompensation) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.byTRV = make(map[string]*climate.ClimateCompensation, len(items))
	s.byRoomSensor = make(map[string][]*climate.ClimateCompensation)
	s.byWindowSensor = make(map[string][]*climate.ClimateCompensation)

	for _, item := range items {
		s.byTRV[item.TRVEntityKey] = item
		if item.RoomTempSensorKey != "" {
			s.byRoomSensor[item.RoomTempSensorKey] = append(s.byRoomSensor[item.RoomTempSensorKey], item)
		}
		if item.WindowSensorKey != "" {
			s.byWindowSensor[item.WindowSensorKey] = append(s.byWindowSensor[item.WindowSensorKey], item)
		}
	}
}

// Save creates or updates a climate compensation config and refreshes memory index.
func (s *Service) Save(ctx context.Context, c *climate.ClimateCompensation) error {
	if c.TRVEntityKey == "" || c.RoomTempSensorKey == "" {
		return climate.ErrInvalidConfig
	}
	if c.DeltaFactor <= 0 {
		c.DeltaFactor = 1.0
	}
	if c.TargetTemperature <= 0 {
		c.TargetTemperature = 20.0
	}

	if err := s.repo.Save(ctx, c); err != nil {
		return err
	}

	items, err := s.repo.List(ctx)
	if err == nil {
		s.rebuildIndexes(items)
	}
	return nil
}

// GetByID returns compensation config by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*climate.ClimateCompensation, error) {
	return s.repo.GetByID(ctx, id)
}

// List returns all active compensations.
func (s *Service) List(ctx context.Context) ([]*climate.ClimateCompensation, error) {
	return s.repo.List(ctx)
}

// Delete removes a compensation by ID.
func (s *Service) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	items, err := s.repo.List(ctx)
	if err == nil {
		s.rebuildIndexes(items)
	}
	return nil
}

// handleEntityStateChanged processes sensor temperature updates and window contact changes.
func (s *Service) handleEntityStateChanged(ctx context.Context, evt events.Event) error {
	payload, ok := evt.Payload.(map[string]any)
	if !ok {
		return nil
	}

	entityID, _ := payload["entity_id"].(string)
	newState, _ := payload["new_state"].(string)
	if entityID == "" || newState == "" {
		return nil
	}

	// 1. Check if entity is a Window Contact Sensor
	s.mu.RLock()
	windowComps := s.byWindowSensor[entityID]
	s.mu.RUnlock()

	if len(windowComps) > 0 {
		isOpen := isWindowOpenState(newState)
		for _, comp := range windowComps {
			if !comp.Enabled {
				continue
			}
			comp.WindowOpen = isOpen
			if isOpen {
				comp.CompensatedSetpoint = 5.0
			} else {
				// Window closed, recalculate with last known or standard setpoint
				comp.CompensatedSetpoint = climate.CalculateCompensatedSetpoint(
					comp.TargetTemperature,
					comp.TargetTemperature, // fallback until next temp reading
					comp.DeltaFactor,
					false,
				)
			}
			_ = s.repo.Save(ctx, comp)
			s.dispatchTRVTemperature(ctx, comp.TRVEntityKey, comp.CompensatedSetpoint)
		}
	}

	// 2. Check if entity is an External Room Temperature Sensor
	s.mu.RLock()
	roomComps := s.byRoomSensor[entityID]
	s.mu.RUnlock()

	if len(roomComps) > 0 {
		if temp, err := strconv.ParseFloat(strings.TrimSpace(newState), 64); err == nil {
			for _, comp := range roomComps {
				if !comp.Enabled {
					continue
				}
				compensated := climate.CalculateCompensatedSetpoint(
					comp.TargetTemperature,
					temp,
					comp.DeltaFactor,
					comp.WindowOpen,
				)
				comp.CompensatedSetpoint = compensated
				_ = s.repo.Save(ctx, comp)
				s.dispatchTRVTemperature(ctx, comp.TRVEntityKey, compensated)
			}
		}
	}

	return nil
}

func (s *Service) dispatchTRVTemperature(ctx context.Context, trvKey string, targetTemp float64) {
	if s.bus == nil || trvKey == "" {
		return
	}
	_ = s.bus.Publish(ctx, events.Event{
		ID:         uuid.NewString(),
		Type:       events.CommandRequested,
		OccurredAt: time.Now().UTC(),
		Source:     "better_thermostat:climate_compensator",
		Payload: map[string]any{
			"entity_key": trvKey,
			"service":    "set_temperature",
			"params": map[string]any{
				"temperature": targetTemp,
			},
		},
	})
}

func isWindowOpenState(state string) bool {
	lower := strings.ToLower(strings.TrimSpace(state))
	return lower == "on" || lower == "open" || lower == "true" || lower == "1"
}
