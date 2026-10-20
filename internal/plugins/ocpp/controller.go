package ocpp

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
)

// ChargingMode defines how EV charging is throttled.
type ChargingMode string

const (
	ModeSolarOnly       ChargingMode = "solar_only"       // Charge only when surplus >= 6A (1.4kW)
	ModeSolarOptimized  ChargingMode = "solar_optimized"  // Minimum 6A, boost up with surplus
	ModeFastCharging    ChargingMode = "fast_charging"    // Max allowable current (16A - 32A)
	ModeScheduledNight  ChargingMode = "scheduled_night"  // Charge during cheap energy hours
)

// WallboxTelemetry holds live telemetry for an enrolled charge point.
type WallboxTelemetry struct {
	ChargePointID string            `json:"charge_point_id"`
	Status        ChargePointStatus `json:"status"`
	ActivePowerW  float64           `json:"active_power_w"`
	OfferedAmps   float64           `json:"offered_amps"`
	DeliveredAmps float64           `json:"delivered_amps"`
	VehicleSoC    float64           `json:"vehicle_soc_pct"`
	EnergyWh      float64           `json:"energy_wh"`
	LastSeenAt    time.Time         `json:"last_seen_at"`
}

// ControllerConfig configures smart EV modulation parameters.
type ControllerConfig struct {
	MaxContractWatts float64 // e.g. 3300W or 6000W
	GridVoltage      float64 // default 230V
	MaxCurrentAmps   float64 // default 32A (or 16A)
	MinCurrentAmps   float64 // IEC 61851 standard minimum is 6A
	DefaultMode      ChargingMode
}

// DynamicModulationResult captures the newly calculated charging profile.
type DynamicModulationResult struct {
	ChargePointID  string       `json:"charge_point_id"`
	Mode           ChargingMode `json:"mode"`
	TargetAmps     float64      `json:"target_amps"`
	ExpectedPowerW float64      `json:"expected_power_w"`
	SolarSurplusW  float64      `json:"solar_surplus_w"`
	HomeGridPowerW float64      `json:"home_grid_power_w"`
	Reason         string       `json:"reason"`
	Timestamp      time.Time    `json:"timestamp"`
}

// Controller coordinates wallboxes, solar production, and household load limits.
type Controller struct {
	mu          sync.RWMutex
	cfg         ControllerConfig
	bus         events.Bus
	mode        ChargingMode
	wallboxes   map[string]*WallboxTelemetry
	lastResults map[string]DynamicModulationResult
	sendProfile func(ctx context.Context, chargePointID string, amps float64) error
}

// NewController instantiates the smart charging optimizer.
func NewController(bus events.Bus, cfg ControllerConfig) *Controller {
	if cfg.GridVoltage <= 0 {
		cfg.GridVoltage = 230.0
	}
	if cfg.MinCurrentAmps <= 0 {
		cfg.MinCurrentAmps = 6.0
	}
	if cfg.MaxCurrentAmps <= 0 {
		cfg.MaxCurrentAmps = 32.0
	}
	if cfg.MaxContractWatts <= 0 {
		cfg.MaxContractWatts = 3300.0
	}
	if cfg.DefaultMode == "" {
		cfg.DefaultMode = ModeSolarOptimized
	}

	return &Controller{
		cfg:         cfg,
		bus:         bus,
		mode:        cfg.DefaultMode,
		wallboxes:   make(map[string]*WallboxTelemetry),
		lastResults: make(map[string]DynamicModulationResult),
	}
}

// WithProfileSender attaches the hardware command function to transmit SetChargingProfile.
func (c *Controller) WithProfileSender(sender func(ctx context.Context, chargePointID string, amps float64) error) *Controller {
	c.sendProfile = sender
	return c
}

// SetMode updates the global charging strategy.
func (c *Controller) SetMode(m ChargingMode) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.mode = m
	slog.Info("OCPP charging mode changed", "mode", m)
}

// UpdateWallboxTelemetry records incoming meter values from a wallbox.
func (c *Controller) UpdateWallboxTelemetry(cpID string, status ChargePointStatus, powerW, deliveredA, soc, energyWh float64, now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()

	wb, ok := c.wallboxes[cpID]
	if !ok {
		wb = &WallboxTelemetry{
			ChargePointID: cpID,
		}
		c.wallboxes[cpID] = wb
	}

	wb.Status = status
	wb.ActivePowerW = powerW
	wb.DeliveredAmps = deliveredA
	wb.VehicleSoC = soc
	wb.EnergyWh = energyWh
	wb.LastSeenAt = now
}

// CalculateOptimalCurrent computes the exact current limit (6A - 32A) based on solar surplus and house load.
func (c *Controller) CalculateOptimalCurrent(ctx context.Context, cpID string, solarSurplusW, totalHouseLoadW float64, now time.Time) (DynamicModulationResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	mode := c.mode
	voltage := c.cfg.GridVoltage
	minA := c.cfg.MinCurrentAmps
	maxA := c.cfg.MaxCurrentAmps
	contractW := c.cfg.MaxContractWatts

	var targetAmps float64
	var reason string

	// 1. Grid Overload Protection: total house power (including appliances) nearing contract limit
	remainingGridW := contractW - totalHouseLoadW
	maxAllowableFromGridA := math.Max(0, remainingGridW/voltage)

	switch mode {
	case ModeFastCharging:
		targetAmps = math.Min(maxA, maxAllowableFromGridA)
		reason = "Fast charging requested; capped by grid contract headroom"

	case ModeSolarOnly:
		// Strict solar surplus only
		solarA := solarSurplusW / voltage
		if solarA < minA {
			targetAmps = 0.0 // Pause charging (IEC 61851 allows 0 or >= 6A)
			reason = fmt.Sprintf("Solar surplus (%.0fW / %.1fA) below minimum 6A threshold; charging paused", solarSurplusW, solarA)
		} else {
			targetAmps = math.Min(maxA, math.Floor(solarA))
			reason = fmt.Sprintf("Modulating strictly on 100%% solar surplus (%.0fW available)", solarSurplusW)
		}

	case ModeSolarOptimized:
		// Minimum 6A guaranteed, boosted by solar surplus
		solarA := solarSurplusW / voltage
		desiredA := minA + math.Max(0, math.Floor(solarA))

		// Ensure we don't trip main breaker
		targetAmps = math.Min(maxA, desiredA)
		if targetAmps > maxAllowableFromGridA && maxAllowableFromGridA >= minA {
			targetAmps = maxAllowableFromGridA
			reason = "Solar optimized; capped by grid capacity"
		} else {
			reason = fmt.Sprintf("Solar optimized with 6A base + %.1fA solar surplus", solarA)
		}

	default:
		targetAmps = minA
		reason = "Fallback minimum current"
	}

	// Clamp to 0 or [6A, 32A]
	if targetAmps > 0 && targetAmps < minA {
		targetAmps = minA
	}
	if targetAmps > maxA {
		targetAmps = maxA
	}

	expectedPowerW := targetAmps * voltage

	res := DynamicModulationResult{
		ChargePointID:  cpID,
		Mode:           mode,
		TargetAmps:     targetAmps,
		ExpectedPowerW: expectedPowerW,
		SolarSurplusW:  solarSurplusW,
		HomeGridPowerW: totalHouseLoadW,
		Reason:         reason,
		Timestamp:      now,
	}

	c.lastResults[cpID] = res
	if wb, ok := c.wallboxes[cpID]; ok {
		wb.OfferedAmps = targetAmps
	}

	slog.Info("OCPP dynamic modulation calculated",
		"charge_point", cpID,
		"target_amps", targetAmps,
		"mode", mode,
		"reason", reason,
	)

	// Send to hardware wallbox if sender attached
	if c.sendProfile != nil {
		_ = c.sendProfile(ctx, cpID, targetAmps)
	}

	return res, nil
}

// GetStatus returns the latest telemetry and modulation state of a charge point.
func (c *Controller) GetStatus(cpID string) (WallboxTelemetry, DynamicModulationResult, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	wb, ok := c.wallboxes[cpID]
	res := c.lastResults[cpID]
	if !ok {
		return WallboxTelemetry{}, DynamicModulationResult{}, false
	}
	return *wb, res, true
}
