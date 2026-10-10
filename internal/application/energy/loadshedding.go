// Package energy provides load shedding, energy monitoring, and blackout prevention.
package energy

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/google/uuid"
)

// LoadSheddingConfig defines the operational rules for load shedding.
type LoadSheddingConfig struct {
	ID                string   `json:"id"`
	MeterEntityKey    string   `json:"meter_entity_key"`    // e.g. "sensor.main_power"
	MaxPowerWatts     float64  `json:"max_power_watts"`     // e.g. 3300.0 or 4500.0
	OverloadDuration  int      `json:"overload_duration"`   // seconds of overload before shedding (e.g. 60)
	RecoveryWatts     float64  `json:"recovery_watts"`      // threshold to allow reconnecting (e.g. 2500.0)
	RecoveryDelaySec  int      `json:"recovery_delay_sec"`  // seconds under recovery before reconnecting (e.g. 180)
	PriorityLoads     []string `json:"priority_loads"`      // shedding order (least important first)
	Enabled           bool     `json:"enabled"`
}

// Controller manages real-time power monitoring and load shedding actions.
type Controller struct {
	bus    events.Bus
	config LoadSheddingConfig

	mu               sync.Mutex
	overloadStart    *time.Time
	recoveryStart    *time.Time
	currentPower     float64
	sheddedLoads     []string // loads currently turned off by the controller
	lastActionAt     time.Time
}

// NewController creates a new load shedding controller.
func NewController(cfg LoadSheddingConfig, bus events.Bus) *Controller {
	if cfg.MaxPowerWatts <= 0 {
		cfg.MaxPowerWatts = 3300.0
	}
	if cfg.OverloadDuration <= 0 {
		cfg.OverloadDuration = 60
	}
	if cfg.RecoveryWatts <= 0 {
		cfg.RecoveryWatts = 2500.0
	}
	if cfg.RecoveryDelaySec <= 0 {
		cfg.RecoveryDelaySec = 120
	}

	c := &Controller{
		bus:    bus,
		config: cfg,
	}

	if bus != nil {
		bus.Subscribe(events.EntityStateChanged, c.handleEntityStateChanged)
	}

	return c
}

// UpdateConfig updates the load shedding configuration dynamically.
func (c *Controller) UpdateConfig(cfg LoadSheddingConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.config = cfg
}

// GetStatus returns the current controller status.
func (c *Controller) GetStatus() map[string]any {
	c.mu.Lock()
	defer c.mu.Unlock()

	var overloadDurationSec float64
	if c.overloadStart != nil {
		overloadDurationSec = time.Since(*c.overloadStart).Seconds()
	}

	return map[string]any{
		"enabled":             c.config.Enabled,
		"meter_entity":        c.config.MeterEntityKey,
		"current_power_watts": c.currentPower,
		"max_power_watts":     c.config.MaxPowerWatts,
		"is_overloaded":       c.overloadStart != nil,
		"overload_seconds":    overloadDurationSec,
		"shedded_loads":       append([]string{}, c.sheddedLoads...),
		"priority_loads":      c.config.PriorityLoads,
	}
}

// ProcessPowerReading evaluates a power reading directly (useful for tests or periodic triggers).
func (c *Controller) ProcessPowerReading(ctx context.Context, power float64, now time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.config.Enabled {
		return nil
	}

	c.currentPower = power

	// Overload evaluation
	if power > c.config.MaxPowerWatts {
		c.recoveryStart = nil
		if c.overloadStart == nil {
			c.overloadStart = &now
		}

		durationOver := now.Sub(*c.overloadStart).Seconds()
		if durationOver >= float64(c.config.OverloadDuration) {
			return c.shedNextLoadLocked(ctx, now)
		}
		return nil
	}

	// Below overload threshold
	c.overloadStart = nil

	// Recovery evaluation
	if power <= c.config.RecoveryWatts && len(c.sheddedLoads) > 0 {
		if c.recoveryStart == nil {
			c.recoveryStart = &now
		}

		durationRec := now.Sub(*c.recoveryStart).Seconds()
		if durationRec >= float64(c.config.RecoveryDelaySec) {
			return c.restoreNextLoadLocked(ctx, now)
		}
	} else {
		c.recoveryStart = nil
	}

	return nil
}

func (c *Controller) shedNextLoadLocked(ctx context.Context, now time.Time) error {
	// Find next load from PriorityLoads that is not yet shed
	var target string
	for _, load := range c.config.PriorityLoads {
		alreadyShed := false
		for _, s := range c.sheddedLoads {
			if s == load {
				alreadyShed = true
				break
			}
		}
		if !alreadyShed {
			target = load
			break
		}
	}

	if target == "" {
		slog.Warn("load_shedding: all priority loads have already been shed, power still exceeds threshold", "power", c.currentPower)
		return nil
	}

	c.sheddedLoads = append(c.sheddedLoads, target)
	c.lastActionAt = now
	// Reset overloadStart to give the newly shed load a chance to drop the consumption
	c.overloadStart = &now

	slog.Info("load_shedding: shedding load to prevent blackout", "entity", target, "power", c.currentPower)

	if c.bus != nil {
		// 1. Dispatch turn_off command
		_ = c.bus.Publish(ctx, events.Event{
			ID:         uuid.NewString(),
			Type:       events.CommandRequested,
			OccurredAt: now,
			Source:     "energy.load_shedding",
			Payload: map[string]any{
				"entity_key": target,
				"command":    "turn_off",
				"params":     map[string]any{},
			},
		})

		// 2. Dispatch warning notification
		_ = c.bus.Publish(ctx, events.Event{
			ID:         uuid.NewString(),
			Type:       events.NotificationCreated,
			OccurredAt: now,
			Source:     "energy.load_shedding",
			Payload: map[string]any{
				"title":   "⚡ Distacco Carichi Anti-Blackout",
				"message": fmt.Sprintf("Spento %s per sovraccarico rete (%.0f W > %.0f W).", target, c.currentPower, c.config.MaxPowerWatts),
				"type":    "warning",
			},
		})
	}

	return nil
}

func (c *Controller) restoreNextLoadLocked(ctx context.Context, now time.Time) error {
	if len(c.sheddedLoads) == 0 {
		return nil
	}

	// Restore the most recently shed load (LIFO)
	lastIdx := len(c.sheddedLoads) - 1
	target := c.sheddedLoads[lastIdx]
	c.sheddedLoads = c.sheddedLoads[:lastIdx]
	c.recoveryStart = &now

	slog.Info("load_shedding: restoring shed load", "entity", target, "power", c.currentPower)

	if c.bus != nil {
		_ = c.bus.Publish(ctx, events.Event{
			ID:         uuid.NewString(),
			Type:       events.CommandRequested,
			OccurredAt: now,
			Source:     "energy.load_shedding",
			Payload: map[string]any{
				"entity_key": target,
				"command":    "turn_on",
				"params":     map[string]any{},
			},
		})

		_ = c.bus.Publish(ctx, events.Event{
			ID:         uuid.NewString(),
			Type:       events.NotificationCreated,
			OccurredAt: now,
			Source:     "energy.load_shedding",
			Payload: map[string]any{
				"title":   "✅ Ripristino Carico Elettrico",
				"message": fmt.Sprintf("Riacceso %s dopo rientro potenza (%.0f W).", target, c.currentPower),
				"type":    "info",
			},
		})
	}

	return nil
}

func (c *Controller) handleEntityStateChanged(ctx context.Context, evt events.Event) error {
	payloadBytes, err := json.Marshal(evt.Payload)
	if err != nil {
		return nil
	}

	var data struct {
		EntityKey  string         `json:"entity_key"`
		State      string         `json:"state"`
		Attributes map[string]any `json:"attributes"`
	}
	if err := json.Unmarshal(payloadBytes, &data); err != nil {
		return nil
	}

	c.mu.Lock()
	meterKey := c.config.MeterEntityKey
	c.mu.Unlock()

	if data.EntityKey != meterKey {
		return nil
	}

	// Extract power in watts
	var power float64
	if p, ok := data.Attributes["power_w"].(float64); ok {
		power = p
	} else if p, ok := data.Attributes["power"].(float64); ok {
		power = p
	} else if f, err := strconv.ParseFloat(data.State, 64); err == nil {
		power = f
	} else {
		return nil
	}

	return c.ProcessPowerReading(ctx, power, time.Now().UTC())
}
