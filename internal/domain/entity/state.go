// Package entity provides typed state representations for standard smart home domains.
package entity

// LightState represents state for the "light" domain.
type LightState struct {
	State      string `json:"state"` // "ON", "OFF"
	Brightness *int   `json:"brightness,omitempty"`
	ColorTemp  *int   `json:"color_temp,omitempty"`
}

// SwitchState represents state for the "switch" domain.
type SwitchState struct {
	State string `json:"state"` // "ON", "OFF"
}

// SensorState represents state for the "sensor" domain.
type SensorState struct {
	Value any `json:"value"`
}

// BinarySensorState represents state for the "binary_sensor" domain.
type BinarySensorState struct {
	State bool `json:"state"` // true (detected/open/on), false (clear/closed/off)
}

// Command represents a requested control action on an entity.
type Command struct {
	EntityKey string         `json:"entity_key"`
	Service   string         `json:"service"` // e.g. "turn_on", "turn_off", "toggle", "set_brightness"
	Params    map[string]any `json:"params,omitempty"`
}
