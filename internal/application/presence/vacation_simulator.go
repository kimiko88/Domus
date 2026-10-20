// Package presence provides smart presence simulation for vacation mode
// and dynamic commute tracking with ETA calculation.
package presence

import (
	"context"
	"log/slog"
	"math/rand"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/google/uuid"
)

// SimulatedAction defines a recorded user habit to be replayed with pseudo-random jitter.
type SimulatedAction struct {
	ID           string        `json:"id"`
	EntityID     string        `json:"entity_id"`
	Command      string        `json:"command"`       // e.g. "turn_on", "turn_off", "open", "close"
	TimeOfDaySec int           `json:"time_of_day_s"` // Seconds from 00:00:00 (e.g. 20:30 = 73800)
	JitterMin    int           `json:"jitter_min"`    // Random delta (+/- minutes, default 15)
	DaysOfWeek   []time.Weekday `json:"days_of_week"`  // Applicable days (empty = every day)
	LastExecuted time.Time     `json:"last_executed,omitempty"`
}

// VacationSimulatorConfig configures the smart presence simulator.
type VacationSimulatorConfig struct {
	Enabled       bool
	JitterMinutes int // Default +/- 15 min
}

// VacationSimulator learns or replays realistic user habits when the home is unoccupied.
type VacationSimulator struct {
	mu          sync.RWMutex
	cfg         VacationSimulatorConfig
	actions     map[string]SimulatedAction
	bus         events.Bus
	rnd         *rand.Rand
	tickerStop  chan struct{}
	stopOnce    sync.Once
	commandSink func(ctx context.Context, entityID, cmd string) error
}

// NewVacationSimulator creates an instance of the presence simulator.
func NewVacationSimulator(bus events.Bus, cfg VacationSimulatorConfig) *VacationSimulator {
	if cfg.JitterMinutes <= 0 {
		cfg.JitterMinutes = 15
	}

	return &VacationSimulator{
		cfg:        cfg,
		actions:    make(map[string]SimulatedAction),
		bus:        bus,
		rnd:        rand.New(rand.NewSource(time.Now().UnixNano())),
		tickerStop: make(chan struct{}),
	}
}

// WithCommandSink attaches a command dispatcher (e.g. entity service or bus).
func (v *VacationSimulator) WithCommandSink(sink func(ctx context.Context, entityID, cmd string) error) *VacationSimulator {
	v.commandSink = sink
	return v
}

// RegisterAction enrolls a habit for replay during vacation mode.
func (v *VacationSimulator) RegisterAction(act SimulatedAction) {
	if act.ID == "" {
		act.ID = uuid.NewString()
	}
	if act.JitterMin < 0 {
		act.JitterMin = v.cfg.JitterMinutes
	}

	v.mu.Lock()
	defer v.mu.Unlock()
	v.actions[act.ID] = act
}

// SetVacationMode enables or disables the simulator.
func (v *VacationSimulator) SetVacationMode(enabled bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.cfg.Enabled = enabled
	slog.Info("vacation presence simulator state changed", "enabled", enabled)
}

// IsEnabled checks if vacation mode simulation is currently active.
func (v *VacationSimulator) IsEnabled() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.cfg.Enabled
}

// EvaluateNow checks which actions are scheduled for the given time and executes them with jitter.
func (v *VacationSimulator) EvaluateNow(ctx context.Context, now time.Time) []SimulatedAction {
	v.mu.Lock()
	defer v.mu.Unlock()

	if !v.cfg.Enabled {
		return nil
	}

	currentSec := now.Hour()*3600 + now.Minute()*60 + now.Second()
	currentWeekday := now.Weekday()

	var executed []SimulatedAction

	for id, act := range v.actions {
		// Day of week filter
		if len(act.DaysOfWeek) > 0 {
			dayMatch := false
			for _, d := range act.DaysOfWeek {
				if d == currentWeekday {
					dayMatch = true
					break
				}
			}
			if !dayMatch {
				continue
			}
		}

		// Prevent executing more than once on the same day
		if !act.LastExecuted.IsZero() && act.LastExecuted.YearDay() == now.YearDay() && act.LastExecuted.Year() == now.Year() {
			continue
		}

		// Calculate target window with optional jitter
		jitterSec := 0
		if act.JitterMin > 0 {
			jitterSec = (v.rnd.Intn(act.JitterMin*2+1) - act.JitterMin) * 60
		}
		targetSec := act.TimeOfDaySec + jitterSec

		// If current time is within +/- 60s of targetSec
		if currentSec >= targetSec-60 && currentSec <= targetSec+60 {
			act.LastExecuted = now
			v.actions[id] = act
			executed = append(executed, act)

			slog.Info("vacation simulator replaying routine",
				"entity_id", act.EntityID,
				"command", act.Command,
				"jitter_sec", jitterSec,
			)

			// Dispatch command
			if v.commandSink != nil {
				_ = v.commandSink(ctx, act.EntityID, act.Command)
			}
			if v.bus != nil {
				go v.publishCommand(act)
			}
		}
	}

	return executed
}

// Start launches the 1-minute ticker routine.
func (v *VacationSimulator) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-v.tickerStop:
				return
			case <-ctx.Done():
				return
			case t := <-ticker.C:
				runCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				v.EvaluateNow(runCtx, t)
				cancel()
			}
		}
	}()
}

// Stop cleanly terminates the simulator.
func (v *VacationSimulator) Stop() {
	v.stopOnce.Do(func() {
		close(v.tickerStop)
	})
}

func (v *VacationSimulator) publishCommand(act SimulatedAction) {
	pubCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = v.bus.Publish(pubCtx, events.Event{
		ID:         uuid.NewString(),
		Type:       events.CommandRequested,
		OccurredAt: time.Now(),
		Source:     "presence.vacation_simulator",
		Payload: map[string]any{
			"entity_id": act.EntityID,
			"command":   act.Command,
			"simulated": true,
		},
	})
}
