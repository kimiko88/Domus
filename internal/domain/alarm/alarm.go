// Package alarm defines the Alarm Control Panel domain models and validation rules.
package alarm

import (
	"errors"
	"time"
)

var (
	ErrAlarmNotFound   = errors.New("alarm: control panel not found")
	ErrInvalidPin      = errors.New("alarm: invalid PIN code")
	ErrPinNotConfigured = errors.New("alarm: PIN has not been configured")
	ErrPinRateLimited   = errors.New("alarm: too many invalid PIN attempts")
	ErrInvalidMode     = errors.New("alarm: unsupported arm mode")
	ErrAlreadyArmed    = errors.New("alarm: already armed")
	ErrAlreadyDisarmed = errors.New("alarm: already disarmed")
)

type State string

const (
	StateDisarmed   State = "disarmed"
	StateArmedHome  State = "armed_home"
	StateArmedAway  State = "armed_away"
	StateArmedNight State = "armed_night"
	StatePending    State = "pending"
	StateTriggered  State = "triggered"
)

// ControlPanel represents an alarm system instance in Domus.
type ControlPanel struct {
	ID                  string     `json:"id"`
	Name                string     `json:"name"`
	State               State      `json:"state"`
	PinHash             string     `json:"-"`
	EntryDelaySeconds   int        `json:"entry_delay_seconds"`
	ExitDelaySeconds    int        `json:"exit_delay_seconds"`
	TriggerDelaySeconds int        `json:"trigger_delay_seconds"`
	PerimeterSensors    []string   `json:"perimeter_sensors"` // doors, windows
	InteriorSensors     []string   `json:"interior_sensors"`  // mmWave, PIR motion
	SirenEntities       []string   `json:"siren_entities"`    // sirens to turn on
	LastChangedBy       string     `json:"last_changed_by,omitempty"`
	LastTriggeredReason string     `json:"last_triggered_reason,omitempty"`
	TriggeredAt         *time.Time `json:"triggered_at,omitempty"`
	PendingEndsAt       *time.Time `json:"pending_ends_at,omitempty"`
	PendingTargetState  State      `json:"pending_target_state,omitempty"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

// IsArmed returns true if the system is armed in any mode.
func (cp *ControlPanel) IsArmed() bool {
	return cp.State == StateArmedHome || cp.State == StateArmedAway || cp.State == StateArmedNight
}

