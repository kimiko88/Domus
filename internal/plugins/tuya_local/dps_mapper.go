package tuya_local

import (
	"encoding/json"
	"fmt"
	"math"
)

// DeviceCategory identifies the device profile for DPS mappings.
type DeviceCategory string

const (
	CategorySwitch  DeviceCategory = "switch"
	CategoryLight   DeviceCategory = "light"
	CategoryPlug    DeviceCategory = "plug"
	CategorySensor  DeviceCategory = "sensor"
	CategoryClimate DeviceCategory = "climate"
)

// DPSMapper handles bidirectional conversion between Tuya DPS dictionaries and Domus entity models.
type DPSMapper struct{}

// NewDPSMapper creates a new DPSMapper instance.
func NewDPSMapper() *DPSMapper {
	return &DPSMapper{}
}

// ParsedEntityState holds the mapped Domus state representation.
type ParsedEntityState struct {
	Domain     string         `json:"domain"`
	State      string         `json:"state"` // "on", "off", "123.45"
	Attributes map[string]any `json:"attributes"`
}

// MapDPSToEntity extracts entity state and attributes from a Tuya DPS map.
func (m *DPSMapper) MapDPSToEntity(category DeviceCategory, dps map[string]any) *ParsedEntityState {
	result := &ParsedEntityState{
		Attributes: make(map[string]any),
	}

	switch category {
	case CategorySwitch, CategoryPlug:
		result.Domain = "switch"
		// Default switch DPS is "1" or "20"
		if val, exists := dps["1"]; exists {
			result.State = boolToOnOff(val)
		} else if val, exists := dps["20"]; exists {
			result.State = boolToOnOff(val)
		} else {
			result.State = "off"
		}

		// Electrical telemetry (voltage, power, current)
		if cur, exists := dps["18"]; exists { // Current (mA)
			result.Attributes["current_ma"] = cur
		}
		if pwr, exists := dps["19"]; exists { // Power (0.1 W)
			if num, ok := toFloat(pwr); ok {
				result.Attributes["power_w"] = math.Round((num/10.0)*100) / 100
			}
		}
		if vlt, exists := dps["20"]; exists && category == CategoryPlug { // Voltage (0.1 V)
			if num, ok := toFloat(vlt); ok {
				result.Attributes["voltage_v"] = math.Round((num/10.0)*10) / 10
			}
		}

	case CategoryLight:
		result.Domain = "light"
		// Power switch: DPS 20 or 1
		if val, exists := dps["20"]; exists {
			result.State = boolToOnOff(val)
		} else if val, exists := dps["1"]; exists {
			result.State = boolToOnOff(val)
		} else {
			result.State = "off"
		}

		// Brightness: DPS 22 (range 10-1000 or 0-255)
		if bri, exists := dps["22"]; exists {
			if num, ok := toFloat(bri); ok {
				// Normalize to 0-255
				normalized := int(math.Round((num / 1000.0) * 255.0))
				if normalized > 255 {
					normalized = 255
				}
				result.Attributes["brightness"] = normalized
			}
		}

		// Color temperature: DPS 23 (range 0-1000)
		if cct, exists := dps["23"]; exists {
			result.Attributes["color_temp"] = cct
		}

		// Color mode or RGB: DPS 24
		if rgb, exists := dps["24"]; exists {
			result.Attributes["color_mode"] = rgb
		}

	case CategoryClimate:
		result.Domain = "climate"
		if val, exists := dps["1"]; exists {
			result.State = boolToOnOff(val)
		}
		// Target temperature: DPS 2 (or 16)
		if target, exists := dps["2"]; exists {
			if num, ok := toFloat(target); ok {
				result.Attributes["target_temperature"] = num / 10.0
			}
		}
		// Current temperature: DPS 3
		if current, exists := dps["3"]; exists {
			if num, ok := toFloat(current); ok {
				result.Attributes["current_temperature"] = num / 10.0
			}
		}

	default: // CategorySensor
		result.Domain = "sensor"
		result.State = "active"
		for k, v := range dps {
			result.Attributes["dp_"+k] = v
		}
	}

	return result
}

// MapCommandToDPS converts a Domus entity command into a Tuya DPS payload.
func (m *DPSMapper) MapCommandToDPS(category DeviceCategory, command string, params map[string]any) (map[string]any, error) {
	dps := make(map[string]any)

	switch category {
	case CategorySwitch, CategoryPlug:
		switch command {
		case "turn_on":
			dps["1"] = true
		case "turn_off":
			dps["1"] = false
		case "toggle":
			// Toggle requires current state from caller
			if curState, ok := params["current_state"].(string); ok && curState == "on" {
				dps["1"] = false
			} else {
				dps["1"] = true
			}
		default:
			return nil, fmt.Errorf("unsupported switch command: %s", command)
		}

	case CategoryLight:
		switch command {
		case "turn_on":
			dps["20"] = true
			if bri, ok := params["brightness"]; ok {
				if f, ok := toFloat(bri); ok {
					// Scale 0-255 to Tuya 10-1000
					scaled := int(math.Round((f / 255.0) * 1000.0))
					if scaled < 10 {
						scaled = 10
					}
					dps["22"] = scaled
				}
			}
			if cct, ok := params["color_temp"]; ok {
				dps["23"] = cct
			}
		case "turn_off":
			dps["20"] = false
		case "set_brightness":
			dps["20"] = true
			if bri, ok := params["brightness"]; ok {
				if f, ok := toFloat(bri); ok {
					scaled := int(math.Round((f / 255.0) * 1000.0))
					dps["22"] = scaled
				}
			}
		default:
			return nil, fmt.Errorf("unsupported light command: %s", command)
		}

	case CategoryClimate:
		switch command {
		case "turn_on":
			dps["1"] = true
		case "turn_off":
			dps["1"] = false
		case "set_temperature":
			if temp, ok := params["temperature"]; ok {
				if f, ok := toFloat(temp); ok {
					dps["2"] = int(f * 10.0)
				}
			}
		default:
			return nil, fmt.Errorf("unsupported climate command: %s", command)
		}

	default:
		return nil, fmt.Errorf("unsupported device category: %s", category)
	}

	return dps, nil
}

func boolToOnOff(v any) string {
	if b, ok := v.(bool); ok {
		if b {
			return "on"
		}
		return "off"
	}
	return fmt.Sprintf("%v", v)
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		if f, err := n.Float64(); err == nil {
			return f, true
		}
	}
	return 0, false
}
