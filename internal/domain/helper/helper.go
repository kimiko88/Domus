// Package helper defines domain models and interfaces for user virtual entities (helpers).
package helper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

var (
	ErrHelperNotFound = errors.New("helper: not found")
	ErrInvalidDomain  = errors.New("helper: invalid domain, must be input_boolean, input_number, timer, or climate_room_compensator")
	ErrInvalidName    = errors.New("helper: name cannot be empty")
)

const (
	DomainInputBoolean       = "input_boolean"
	DomainInputNumber        = "input_number"
	DomainTimer              = "timer"
	DomainClimateCompensator = "climate_room_compensator"
)

// Helper represents a virtual user-configurable entity.
type Helper struct {
	ID         string    `json:"id"`
	EntityKey  string    `json:"entity_key"`
	Name       string    `json:"name"`
	Domain     string    `json:"domain"`
	State      string    `json:"state"`
	ConfigJSON string    `json:"config_json"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// InputBooleanConfig holds icon and UI preferences.
type InputBooleanConfig struct {
	Icon string `json:"icon,omitempty"`
}

// InputNumberConfig defines bounds and stepping for numeric sliders.
type InputNumberConfig struct {
	Min  float64 `json:"min"`
	Max  float64 `json:"max"`
	Step float64 `json:"step"`
	Unit string  `json:"unit,omitempty"`
}

// TimerConfig defines countdown settings.
type TimerConfig struct {
	DurationSeconds  int        `json:"duration_seconds"`
	RemainingSeconds int        `json:"remaining_seconds"`
	Status           string     `json:"status"` // "idle", "active", "paused"
	EndsAt           *time.Time `json:"ends_at,omitempty"`
}

// ClimateCompensatorConfig defines Better Thermostat room offset settings.
type ClimateCompensatorConfig struct {
	TargetTRVKey        string  `json:"target_trv_key"`
	RoomSensorKey       string  `json:"room_sensor_key"`
	WindowSensorKey     string  `json:"window_sensor_key,omitempty"`
	TargetTemperature   float64 `json:"target_temperature"`
	CurrentRoomTemp     float64 `json:"current_room_temp"`
	CalibrationFactor   float64 `json:"calibration_factor"` // default 1.5
	CompensatedSetpoint float64 `json:"compensated_setpoint"`
	WindowOpen          bool    `json:"window_open"`
	Active              bool    `json:"active"`
}

// CalculateCompensatedSetpoint computes the dynamic TRV setpoint.
func CalculateCompensatedSetpoint(target, roomTemp, factor float64, windowOpen bool) float64 {
	if windowOpen {
		return 5.0 // frost protection
	}
	if factor <= 0 {
		factor = 1.0
	}
	delta := target - roomTemp
	compensated := target + (delta * factor)
	if compensated < 5.0 {
		compensated = 5.0
	}
	if compensated > 30.0 {
		compensated = 30.0
	}
	return math.Round(compensated*2) / 2
}

// NewClimateCompensator creates a new Better Thermostat helper.
func NewClimateCompensator(id, name, entityKey, trvKey, roomSensorKey, windowSensorKey string, targetTemp, factor float64) *Helper {
	if entityKey == "" {
		entityKey = fmt.Sprintf("climate.%s", sanitizeKey(name))
	}
	if factor <= 0 {
		factor = 1.5
	}
	if targetTemp <= 0 {
		targetTemp = 20.0
	}
	cfg, _ := json.Marshal(ClimateCompensatorConfig{
		TargetTRVKey:        trvKey,
		RoomSensorKey:       roomSensorKey,
		WindowSensorKey:     windowSensorKey,
		TargetTemperature:   targetTemp,
		CurrentRoomTemp:     targetTemp,
		CalibrationFactor:   factor,
		CompensatedSetpoint: targetTemp,
		Active:              true,
	})
	now := time.Now().UTC()
	return &Helper{
		ID:         id,
		EntityKey:  entityKey,
		Name:       name,
		Domain:     DomainClimateCompensator,
		State:      fmt.Sprintf("%.1f", targetTemp),
		ConfigJSON: string(cfg),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// NewInputBoolean creates a new virtual boolean helper.
func NewInputBoolean(id, name, entityKey, icon string, initialState bool) *Helper {
	if entityKey == "" {
		entityKey = fmt.Sprintf("input_boolean.%s", sanitizeKey(name))
	}
	cfg, _ := json.Marshal(InputBooleanConfig{Icon: icon})
	state := "off"
	if initialState {
		state = "on"
	}
	now := time.Now().UTC()
	return &Helper{
		ID:         id,
		EntityKey:  entityKey,
		Name:       name,
		Domain:     DomainInputBoolean,
		State:      state,
		ConfigJSON: string(cfg),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// NewInputNumber creates a new virtual numeric slider helper.
func NewInputNumber(id, name, entityKey string, min, max, step, initialVal float64, unit string) *Helper {
	if entityKey == "" {
		entityKey = fmt.Sprintf("input_number.%s", sanitizeKey(name))
	}
	if step <= 0 {
		step = 1
	}
	if initialVal < min {
		initialVal = min
	}
	if initialVal > max {
		initialVal = max
	}
	cfg, _ := json.Marshal(InputNumberConfig{
		Min:  min,
		Max:  max,
		Step: step,
		Unit: unit,
	})
	now := time.Now().UTC()
	return &Helper{
		ID:         id,
		EntityKey:  entityKey,
		Name:       name,
		Domain:     DomainInputNumber,
		State:      fmt.Sprintf("%.2f", initialVal),
		ConfigJSON: string(cfg),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// NewTimer creates a new virtual countdown timer helper.
func NewTimer(id, name, entityKey string, durationSec int) *Helper {
	if entityKey == "" {
		entityKey = fmt.Sprintf("timer.%s", sanitizeKey(name))
	}
	if durationSec <= 0 {
		durationSec = 60
	}
	cfg, _ := json.Marshal(TimerConfig{
		DurationSeconds:  durationSec,
		RemainingSeconds: durationSec,
		Status:           "idle",
	})
	now := time.Now().UTC()
	return &Helper{
		ID:         id,
		EntityKey:  entityKey,
		Name:       name,
		Domain:     DomainTimer,
		State:      "idle",
		ConfigJSON: string(cfg),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// SetNumberValue safely updates the numeric helper value clamping between min and max.
func (h *Helper) SetNumberValue(val float64) error {
	if h.Domain != DomainInputNumber {
		return errors.New("helper: not an input_number")
	}
	var cfg InputNumberConfig
	if err := json.Unmarshal([]byte(h.ConfigJSON), &cfg); err != nil {
		return err
	}
	clamped := math.Max(cfg.Min, math.Min(cfg.Max, val))
	h.State = fmt.Sprintf("%.2f", clamped)
	h.UpdatedAt = time.Now().UTC()
	return nil
}

// ToggleBoolean flips the on/off state.
func (h *Helper) ToggleBoolean() {
	if h.Domain != DomainInputBoolean {
		return
	}
	if h.State == "on" {
		h.State = "off"
	} else {
		h.State = "on"
	}
	h.UpdatedAt = time.Now().UTC()
}

func sanitizeKey(name string) string {
	res := strings.ToLower(strings.TrimSpace(name))
	res = strings.ReplaceAll(res, " ", "_")
	res = strings.ReplaceAll(res, "-", "_")
	return res
}

// AsNumber returns the float64 value if input_number.
func (h *Helper) AsNumber() float64 {
	val, _ := strconv.ParseFloat(h.State, 64)
	return val
}

// Repository defines storage operations for User Helpers.
type Repository interface {
	Create(ctx context.Context, h *Helper) error
	GetByID(ctx context.Context, id string) (*Helper, error)
	GetByEntityKey(ctx context.Context, entityKey string) (*Helper, error)
	List(ctx context.Context, domain string) ([]*Helper, error)
	Update(ctx context.Context, h *Helper) error
	Delete(ctx context.Context, id string) error
}
