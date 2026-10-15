// Package climate defines domain models for climate control and intelligent thermostatic valve compensation.
package climate

import (
	"context"
	"errors"
	"math"
	"time"
)

var (
	ErrCompensationNotFound = errors.New("climate compensation not found")
	ErrInvalidConfig        = errors.New("invalid climate compensation configuration")
)

// ClimateCompensation stores the calibration profile linking a TRV valve to an external room sensor and window contact.
type ClimateCompensation struct {
	ID                  string    `json:"id"`
	TRVEntityKey        string    `json:"trv_entity_key"`
	RoomTempSensorKey   string    `json:"room_temp_sensor_key"`
	WindowSensorKey     string    `json:"window_sensor_key,omitempty"`
	TargetTemperature   float64   `json:"target_temperature"`
	DeltaFactor         float64   `json:"delta_factor"` // standard 1.0 to 2.0
	CompensatedSetpoint float64   `json:"compensated_setpoint"`
	WindowOpen          bool      `json:"window_open"`
	Enabled             bool      `json:"enabled"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// CalculateCompensatedSetpoint determines the target TRV setpoint to achieve room comfort.
// If window is open, it returns 5.0 (frost protection / shutoff).
func CalculateCompensatedSetpoint(target, roomTemp, deltaFactor float64, windowOpen bool) float64 {
	if windowOpen {
		return 5.0
	}
	if deltaFactor <= 0 {
		deltaFactor = 1.0
	}
	// e.g. target 21.0, room sensor reads 19.0 -> error = +2.0 -> valve set to 21 + 2.0*1.5 = 24.0
	delta := target - roomTemp
	compensated := target + (delta * deltaFactor)
	if compensated < 5.0 {
		compensated = 5.0
	}
	if compensated > 30.0 {
		compensated = 30.0
	}
	return math.Round(compensated*2) / 2
}

// Repository defines storage interface for ClimateCompensation configurations.
type Repository interface {
	Save(ctx context.Context, comp *ClimateCompensation) error
	GetByID(ctx context.Context, id string) (*ClimateCompensation, error)
	GetByTRV(ctx context.Context, trvKey string) (*ClimateCompensation, error)
	List(ctx context.Context) ([]*ClimateCompensation, error)
	Delete(ctx context.Context, id string) error
}
