package midea

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"
)

type Command struct {
	Values map[string]any `json:"values,omitempty"`
	Action string         `json:"action,omitempty"`
}

// ValidateCommand only accepts writes explicitly advertised by this appliance.
func ValidateCommand(device Device, command Command) error {
	if len(command.Values) == 0 && strings.TrimSpace(command.Action) == "" {
		return fmt.Errorf("%w: empty command", ErrInvalidCommand)
	}
	if command.Action != "" && !containsString(device.Actions, command.Action) {
		return fmt.Errorf("%w: action %q is not advertised", ErrInvalidCommand, command.Action)
	}
	for field, value := range command.Values {
		capability := Capability(field)
		if !containsCapability(device.Writable, capability) {
			return fmt.Errorf("%w: field %q is not writable", ErrInvalidCommand, field)
		}
		if err := validateValue(device, capability, value); err != nil {
			return fmt.Errorf("%w: %s: %v", ErrInvalidCommand, field, err)
		}
	}
	return nil
}

func validateValue(device Device, capability Capability, value any) error {
	switch capability {
	case CapPower, CapSwing, CapChildLock, CapAnion, CapPump:
		if _, ok := value.(bool); !ok {
			return errors.New("must be boolean")
		}
	case CapMode, CapFanSpeed:
		text, ok := value.(string)
		if !ok || strings.TrimSpace(text) == "" {
			return errors.New("must be a non-empty string")
		}
		if capability == CapMode && len(device.Modes) > 0 && !containsString(device.Modes, text) {
			return errors.New("mode is not supported by this model")
		}
		if capability == CapFanSpeed && len(device.FanSpeeds) > 0 && !containsString(device.FanSpeeds, text) {
			return errors.New("fan speed is not supported by this model")
		}
	case CapTargetTemperature:
		number, ok := numeric(value)
		if !ok || math.IsNaN(number) || math.IsInf(number, 0) {
			return errors.New("must be a finite number")
		}
		if device.MaxTemperature > 0 && (number < device.MinTemperature || number > device.MaxTemperature) {
			return errors.New("outside device temperature limits")
		}
		if device.TemperatureStep > 0 && math.Abs(math.Mod((number-device.MinTemperature)/device.TemperatureStep, 1)) > 0.000001 {
			return errors.New("does not match device temperature step")
		}
	case CapTargetHumidity:
		number, ok := numeric(value)
		if !ok || math.IsNaN(number) || math.IsInf(number, 0) || number < 0 || number > 100 {
			return errors.New("must be a percentage between 0 and 100")
		}
		if device.MaxHumidity > 0 && (number < device.MinHumidity || number > device.MaxHumidity) {
			return errors.New("outside device humidity limits")
		}
	case CapWaterLevelSet:
		var level int
		switch typed := value.(type) {
		case string:
			switch typed {
			case "25":
				level = 25
			case "50":
				level = 50
			case "75":
				level = 75
			case "100":
				level = 100
			default:
				return errors.New("must be one of 25, 50, 75 or 100")
			}
		default:
			number, ok := numeric(value)
			if !ok || math.IsNaN(number) || math.IsInf(number, 0) || math.Trunc(number) != number {
				return errors.New("must be an integer water level")
			}
			level = int(number)
		}
		if level != 25 && level != 50 && level != 75 && level != 100 {
			return errors.New("must be one of 25, 50, 75 or 100")
		}
	case CapCycleState:
		program, ok := value.(string)
		if !ok || strings.TrimSpace(program) == "" {
			return errors.New("must be a non-empty program/state")
		}
		if len(device.Programs) > 0 && !containsString(device.Programs, program) {
			return errors.New("program is not supported by this model")
		}
	default:
		switch value.(type) {
		case bool, string, int, int32, int64, float32, float64:
			return nil
		default:
			return errors.New("must be a scalar JSON value")
		}
	}
	return nil
}

func numeric(value any) (float64, bool) {
	switch number := value.(type) {
	case int:
		return float64(number), true
	case int32:
		return float64(number), true
	case int64:
		return float64(number), true
	case float32:
		return float64(number), true
	case float64:
		return number, true
	default:
		return 0, false
	}
}

func valuesEqual(expected, actual any) bool {
	if reflect.DeepEqual(expected, actual) {
		return true
	}
	expectedNumber, expectedOK := numeric(expected)
	actualNumber, actualOK := numeric(actual)
	return expectedOK && actualOK && math.Abs(expectedNumber-actualNumber) < 0.000001
}
