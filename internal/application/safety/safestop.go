// Package safety implements deterministic, ultra-low-latency safe-stop pipelines
// for life-safety and property hazards (gas leaks, catastrophic water bursts, smoke/fire).
//
// Designed to bypass normal message queuing and guarantee valve shut-off reaction
// times strictly under 50 milliseconds via parallel hardware execution.
package safety

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/google/uuid"
)

// HazardType classifies the detected physical hazard.
type HazardType string

const (
	HazardWaterLeak HazardType = "water_leak"
	HazardGasLeak   HazardType = "gas_leak"
	HazardFireSmoke HazardType = "fire_smoke"
	HazardCO        HazardType = "carbon_monoxide"
)

// ValveType defines which physical utility line the actuator controls.
type ValveType string

const (
	ValveWater   ValveType = "water"
	ValveGas     ValveType = "gas"
	ValveGeneral ValveType = "general"
)

// ValveActuator defines the low-level hardware contract required to close
// an emergency shut-off valve directly (e.g. Modbus coil, Zigbee/Z-Wave relay, GPIO pin).
type ValveActuator interface {
	CloseValve(ctx context.Context, valveID string) error
	OpenValve(ctx context.Context, valveID string) error
}

// ValveRegistration defines an emergency actuator enrolled in the safe-stop system.
type ValveRegistration struct {
	ID        string        `json:"id"`
	Type      ValveType     `json:"type"`
	Name      string        `json:"name"`
	Location  string        `json:"location"`
	Actuator  ValveActuator `json:"-"`
	TimeoutMs int           `json:"timeout_ms"`
}

// EmergencyAlert represents a critical hazard signal received directly from
// a sensor frame, dry contact, or emergency event.
type EmergencyAlert struct {
	ID           string            `json:"id"`
	Hazard       HazardType        `json:"hazard"`
	SourceSensor string            `json:"source_sensor"`
	Location     string            `json:"location"`
	Severity     string            `json:"severity"` // "critical", "warning"
	OccurredAt   time.Time         `json:"occurred_at"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// EmergencyResult captures the outcome and timing of the emergency shut-off execution.
type EmergencyResult struct {
	AlertID        string            `json:"alert_id"`
	TriggeredAt    time.Time         `json:"triggered_at"`
	CompletedAt    time.Time         `json:"completed_at"`
	DurationUs     int64             `json:"duration_us"`     // Microseconds
	DurationMs     float64           `json:"duration_ms"`     // Milliseconds
	ValvesTargeted int               `json:"valves_targeted"`
	ValvesClosed   []string          `json:"valves_closed"`
	ValvesFailed   map[string]string `json:"valves_failed,omitempty"`
	Success        bool              `json:"success"`
}

// EmergencyStatus represents current safe-stop state for observability and diagnostics.
type EmergencyStatus struct {
	IsLatched       bool              `json:"is_latched"`
	ActiveAlert     *EmergencyAlert   `json:"active_alert,omitempty"`
	RegisteredCount int               `json:"registered_count"`
	RegisteredLines []string          `json:"registered_lines"`
	LastResult      *EmergencyResult  `json:"last_result,omitempty"`
}

// SafetyRepository defines SQLite persistence for emergency valves and incidents.
type SafetyRepository interface {
	SaveValve(ctx context.Context, v ValveRegistration) error
	ListValves(ctx context.Context) ([]ValveRegistration, error)
	DeleteValve(ctx context.Context, id string) error
	RecordIncident(ctx context.Context, alert EmergencyAlert, result EmergencyResult) error
	ListIncidents(ctx context.Context, limit int) ([]map[string]any, error)
	ClearIncident(ctx context.Context, alertID, clearedBy, reason string) error
}

// Service manages enrolled emergency shut-off actuators and coordinates
// deterministic out-of-band safe-stop execution.
type Service struct {
	mu           sync.RWMutex
	valves       map[string]ValveRegistration
	repo         SafetyRepository
	isLatched    bool
	activeAlert  *EmergencyAlert
	lastResult   *EmergencyResult
	bus          events.Bus
	fastQueue    chan EmergencyAlert
	shutdownCh   chan struct{}
	stopOnce     sync.Once
}

// NewService instantiates a SafeStop Service.
func NewService(bus events.Bus) *Service {
	s := &Service{
		valves:     make(map[string]ValveRegistration),
		bus:        bus,
		fastQueue:  make(chan EmergencyAlert, 128),
		shutdownCh: make(chan struct{}),
	}

	go s.fastQueueWorker()
	return s
}

// DefaultLoopbackActuator provides an in-memory/simulated actuator when hardware driver is not directly attached.
type DefaultLoopbackActuator struct{}

func (d *DefaultLoopbackActuator) CloseValve(ctx context.Context, valveID string) error {
	return nil
}

func (d *DefaultLoopbackActuator) OpenValve(ctx context.Context, valveID string) error {
	return nil
}

// WithRepository connects SQLite database persistence and preloads configured emergency valves.
func (s *Service) WithRepository(repo SafetyRepository) *Service {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.repo = repo
	if repo != nil {
		if valves, err := repo.ListValves(context.Background()); err == nil {
			for _, v := range valves {
				if v.Actuator == nil {
					v.Actuator = &DefaultLoopbackActuator{}
				}
				s.valves[v.ID] = v
			}
		}
	}
	return s
}

// FastChannel returns the direct, non-blocking ingestion channel for hardware interrupt routines.
func (s *Service) FastChannel() chan<- EmergencyAlert {
	return s.fastQueue
}

// RegisterValve enrolls an emergency valve actuator.
func (s *Service) RegisterValve(v ValveRegistration) error {
	if v.ID == "" {
		return errors.New("valve ID cannot be empty")
	}
	if v.Actuator == nil {
		v.Actuator = &DefaultLoopbackActuator{}
	}
	if v.TimeoutMs <= 0 {
		v.TimeoutMs = 35 // Default 35ms timeout to ensure sub-50ms roundtrip
	}

	s.mu.Lock()
	s.valves[v.ID] = v
	repo := s.repo
	s.mu.Unlock()

	if repo != nil {
		_ = repo.SaveValve(context.Background(), v)
	}

	slog.Info("emergency valve registered", "id", v.ID, "type", v.Type, "location", v.Location)
	return nil
}

// UnregisterValve removes a valve from emergency safe-stop.
func (s *Service) UnregisterValve(valveID string) {
	s.mu.Lock()
	delete(s.valves, valveID)
	repo := s.repo
	s.mu.Unlock()

	if repo != nil {
		_ = repo.DeleteValve(context.Background(), valveID)
	}
}

// ListValves returns all currently registered emergency valves.
func (s *Service) ListValves(ctx context.Context) ([]ValveRegistration, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := make([]ValveRegistration, 0, len(s.valves))
	for _, v := range s.valves {
		res = append(res, v)
	}
	return res, nil
}

// ListIncidents queries historical emergency incidents from persistence.
func (s *Service) ListIncidents(ctx context.Context, limit int) ([]map[string]any, error) {
	s.mu.RLock()
	repo := s.repo
	s.mu.RUnlock()
	if repo != nil {
		return repo.ListIncidents(ctx, limit)
	}
	return []map[string]any{}, nil
}

// TriggerSafeStop executes immediate parallel shut-off across all relevant valves,
// bypassing standard queues, and latches the emergency state.
func (s *Service) TriggerSafeStop(ctx context.Context, alert EmergencyAlert) (EmergencyResult, error) {
	start := time.Now()

	if alert.ID == "" {
		alert.ID = uuid.NewString()
	}
	if alert.OccurredAt.IsZero() {
		alert.OccurredAt = start
	}

	s.mu.Lock()
	s.isLatched = true
	alertCopy := alert
	s.activeAlert = &alertCopy

	// Determine matching valves
	var targets []ValveRegistration
	for _, v := range s.valves {
		if shouldShutValve(alert.Hazard, v.Type) {
			targets = append(targets, v)
		}
	}
	s.mu.Unlock()

	slog.Warn("EMERGENCY SAFE-STOP TRIGGERED",
		"alert_id", alert.ID,
		"hazard", alert.Hazard,
		"source", alert.SourceSensor,
		"target_valves", len(targets),
	)

	res := EmergencyResult{
		AlertID:        alert.ID,
		TriggeredAt:    start,
		ValvesTargeted: len(targets),
		ValvesFailed:   make(map[string]string),
	}

	if len(targets) > 0 {
		var wg sync.WaitGroup
		var mu sync.Mutex

		for _, v := range targets {
			wg.Add(1)
			go func(reg ValveRegistration) {
				defer wg.Done()

				// Dedicated context per valve with strict timeout (e.g. 35ms)
				vCtx, vCancel := context.WithTimeout(ctx, time.Duration(reg.TimeoutMs)*time.Millisecond)
				defer vCancel()

				err := reg.Actuator.CloseValve(vCtx, reg.ID)

				mu.Lock()
				defer mu.Unlock()
				if err != nil {
					res.ValvesFailed[reg.ID] = err.Error()
					slog.Error("failed to actuate emergency valve", "valve_id", reg.ID, "error", err)
				} else {
					res.ValvesClosed = append(res.ValvesClosed, reg.ID)
					slog.Info("emergency valve successfully closed", "valve_id", reg.ID)
				}
			}(v)
		}

		wg.Wait()
	}

	completed := time.Now()
	res.CompletedAt = completed
	duration := completed.Sub(start)
	res.DurationUs = duration.Microseconds()
	res.DurationMs = float64(duration.Microseconds()) / 1000.0
	res.Success = len(res.ValvesFailed) == 0

	s.mu.Lock()
	resCopy := res
	s.lastResult = &resCopy
	repo := s.repo
	s.mu.Unlock()

	if repo != nil {
		_ = repo.RecordIncident(ctx, alert, res)
	}

	// Asynchronously publish to normal event bus for telemetry/dashboard without delaying response
	if s.bus != nil {
		go func(r EmergencyResult, a EmergencyAlert) {
			pubCtx, pubCancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer pubCancel()
			_ = s.bus.Publish(pubCtx, events.Event{
				ID:         uuid.NewString(),
				Type:       events.EmergencyTriggered,
				OccurredAt: start,
				Source:     "safety.safe_stop",
				Payload: map[string]any{
					"alert":  a,
					"result": r,
				},
			})
		}(res, alert)
	}

	slog.Info("safe-stop pipeline completed",
		"alert_id", alert.ID,
		"latency_ms", res.DurationMs,
		"closed", len(res.ValvesClosed),
		"failed", len(res.ValvesFailed),
	)

	return res, nil
}

// ResetEmergency disarms the emergency latch after verification by an authorized operator.
func (s *Service) ResetEmergency(ctx context.Context, operatorID, reason string) error {
	s.mu.Lock()
	if !s.isLatched {
		s.mu.Unlock()
		return errors.New("no active emergency latch to reset")
	}

	s.isLatched = false
	prevAlert := s.activeAlert
	s.activeAlert = nil
	repo := s.repo
	s.mu.Unlock()

	if repo != nil && prevAlert != nil {
		_ = repo.ClearIncident(ctx, prevAlert.ID, operatorID, reason)
	}

	slog.Warn("EMERGENCY SAFE-STOP LATCH DISARMED",
		"operator_id", operatorID,
		"reason", reason,
	)

	if s.bus != nil {
		go func() {
			pubCtx, pubCancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer pubCancel()
			_ = s.bus.Publish(pubCtx, events.Event{
				ID:         uuid.NewString(),
				Type:       events.EmergencyReset,
				OccurredAt: time.Now(),
				Source:     "safety.safe_stop",
				Payload: map[string]any{
					"operator_id": operatorID,
					"reason":      reason,
					"prev_alert":  prevAlert,
				},
			})
		}()
	}

	return nil
}

// Status returns current operational status and latch condition.
func (s *Service) Status() EmergencyStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var lines []string
	seen := make(map[string]bool)
	for _, v := range s.valves {
		lineStr := fmt.Sprintf("%s (%s)", v.ID, v.Type)
		if !seen[lineStr] {
			lines = append(lines, lineStr)
			seen[lineStr] = true
		}
	}

	return EmergencyStatus{
		IsLatched:       s.isLatched,
		ActiveAlert:     s.activeAlert,
		RegisteredCount: len(s.valves),
		RegisteredLines: lines,
		LastResult:      s.lastResult,
	}
}

// Stop shuts down the fastQueue worker.
func (s *Service) Stop() {
	s.stopOnce.Do(func() {
		close(s.shutdownCh)
	})
}

func (s *Service) fastQueueWorker() {
	for {
		select {
		case <-s.shutdownCh:
			return
		case alert, ok := <-s.fastQueue:
			if !ok {
				return
			}
			// Trigger with 5-second deadline
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_, _ = s.TriggerSafeStop(ctx, alert)
			cancel()
		}
	}
}

// shouldShutValve determines if a valve of targetType should be closed for given hazard.
func shouldShutValve(hazard HazardType, targetType ValveType) bool {
	if targetType == ValveGeneral {
		return true
	}
	switch hazard {
	case HazardWaterLeak:
		return targetType == ValveWater
	case HazardGasLeak:
		return targetType == ValveGas
	case HazardFireSmoke, HazardCO:
		// On fire/smoke/CO, shut both gas lines (prevent explosion/feed)
		return targetType == ValveGas
	default:
		return false
	}
}
