package tuya_local

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
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
		// DPS 4 is the only operating-mode control currently mapped for
		// LocalTuya thermostats. Advertise it only when the device reports it.
		if _, exists := dps["4"]; exists {
			result.Attributes["hvac_modes"] = []string{"auto", "manual"}
		}
		if val, exists := dps["1"]; exists {
			result.State = boolToOnOff(val)
		}
		// Target temperature is DPS 2 on common Tuya thermostats; some
		// products expose the same value as DPS 16.
		target, exists := dps["2"]
		if !exists {
			target, exists = dps["16"]
		}
		if exists {
			if num, ok := toFloat(target); ok {
				result.Attributes["target_temperature"] = num / 10.0
				if _, hasPrimary := dps["2"]; !hasPrimary {
					result.Attributes["target_temperature_dp"] = 16
				} else {
					result.Attributes["target_temperature_dp"] = 2
				}
			}
		}
		// Current temperature: DPS 3
		if current, exists := dps["3"]; exists {
			if num, ok := toFloat(current); ok {
				result.Attributes["current_temperature"] = num / 10.0
			}
		}
		// Local thermostat mode: DPS 4 (0 = automatic, 1 = manual).
		if mode, exists := dps["4"]; exists {
			if num, ok := toFloat(mode); ok {
				switch num {
				case 0:
					result.Attributes["hvac_mode"] = "auto"
				case 1:
					result.Attributes["hvac_mode"] = "manual"
				default:
					result.Attributes["dp_4"] = mode
				}
			} else {
				result.Attributes["dp_4"] = mode
			}
		}

	default: // CategorySensor
		result.Domain = "sensor"
		result.State = "active"
		for k, v := range dps {
			result.Attributes["dp_"+k] = v
		}
		// Common Tuya thermo-hygrometers (for example WSDCGQ profiles) expose
		// temperature on DP1 and humidity on DP2. Keep the raw datapoints and
		// only infer climate semantics when the values need Tuya scaling or the
		// configured device name explicitly identifies a climate sensor.
		enrichTuyaClimateSensorAttributes(result.Attributes, "")
	}

	return result
}

// enrichTuyaClimateSensorAttributes keeps generic Tuya sensors out of Climate
// & History unless there is evidence that DP1/DP2 represent temperature and
// humidity. Existing cloud telemetry is retained and takes precedence.
func enrichTuyaClimateSensorAttributes(attributes map[string]any, deviceName string) {
	if attributes == nil {
		return
	}

	name := strings.ToLower(strings.TrimSpace(deviceName))
	nameHint := strings.Contains(name, "temp") || strings.Contains(name, "humidity") ||
		strings.Contains(name, "humedad") || strings.Contains(name, "humidité") ||
		strings.Contains(name, "umid") || strings.Contains(name, "влаж") ||
		strings.Contains(name, "feucht") || strings.Contains(name, "moisture") ||
		strings.Contains(name, "thermo") || strings.Contains(name, "termomet") ||
		strings.Contains(name, "igromet")
	alreadyMarked, _ := attributes["tuya_climate_source"].(bool)

	temperatureRaw, hasTemperature := toFloat(attributes["dp_1"])
	humidityRaw, hasHumidity := toFloat(attributes["dp_2"])
	temperature, temperatureScaled, temperatureOK := normalizeTuyaTemperatureDPS(temperatureRaw)
	humidity, humidityScaled, humidityOK := normalizeTuyaHumidityDPS(humidityRaw)

	// Some Tuya products expose only one climate measurement. Recognize those
	// when the configured name identifies the sensor or the raw value clearly
	// uses Tuya's tenths/hundredths scaling. A plausible DP1+DP2 pair is treated
	// as a thermo-hygrometer when either value provides that evidence.
	pairEvidence := hasTemperature && hasHumidity && temperatureOK && humidityOK &&
		(nameHint || temperatureScaled || humidityScaled || alreadyMarked)
	hasTemperatureSource := hasTemperature && temperatureOK &&
		(nameHint || temperatureScaled || alreadyMarked || pairEvidence)
	hasHumiditySource := hasHumidity && humidityOK &&
		(nameHint || humidityScaled || alreadyMarked || pairEvidence)
	if !hasTemperatureSource && !hasHumiditySource {
		return
	}

	kind := "temperature_humidity"
	deviceClass := "temperature"
	unit := "°C"
	if hasTemperatureSource && !hasHumiditySource {
		kind = "temperature"
	} else if hasHumiditySource && !hasTemperatureSource {
		kind = "humidity"
		deviceClass = "humidity"
		unit = "%"
	}

	// Once the DPS profile is recognized, the current local datapoints are
	// authoritative. Replace previous values so repeated LAN updates do not
	// leave Climate & History stuck on the first reading.
	if hasTemperatureSource {
		attributes["current_temperature"] = temperature
	}
	if hasHumiditySource {
		attributes["current_humidity"] = humidity
	}
	if value, ok := attributes["device_class"].(string); !ok || strings.TrimSpace(value) == "" {
		attributes["device_class"] = deviceClass
	}
	if value, ok := attributes["unit_of_measurement"].(string); !ok || strings.TrimSpace(value) == "" {
		attributes["unit_of_measurement"] = unit
	}
	attributes["tuya_climate_source"] = true
	attributes["tuya_sensor_kind"] = kind
}

// normalizeTuyaTemperatureDPS converts a DP1 value to degrees Celsius where
// possible. Values already in the plausible range are treated as unscaled.
func normalizeTuyaTemperatureDPS(value float64) (float64, bool, bool) {
	if !math.IsNaN(value) && !math.IsInf(value, 0) && value >= -50 && value <= 70 {
		return value, false, true
	}
	for _, scale := range []float64{10, 100, 1000} {
		normalized := value / scale
		if !math.IsNaN(normalized) && !math.IsInf(normalized, 0) && normalized >= -50 && normalized <= 70 {
			return normalized, true, true
		}
	}
	return 0, false, false
}

func normalizeTuyaHumidityDPS(value float64) (float64, bool, bool) {
	if !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0 && value <= 100 {
		return value, false, true
	}
	for _, scale := range []float64{10, 100, 1000} {
		normalized := value / scale
		if !math.IsNaN(normalized) && !math.IsInf(normalized, 0) && normalized >= 0 && normalized <= 100 {
			return normalized, true, true
		}
	}
	return 0, false, false
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
			temp, ok := toFloat(params["temperature"])
			if !ok || math.IsNaN(temp) || math.IsInf(temp, 0) {
				return nil, errors.New("climate temperature must be a finite number")
			}
			dpID := 2
			if configuredDP, exists := params["dp_id"]; exists {
				value, valid := toFloat(configuredDP)
				if !valid || (value != 2 && value != 16) {
					return nil, errors.New("climate setpoint dp_id must be 2 or 16")
				}
				dpID = int(value)
			}
			scale := 1
			if configuredScale, exists := params["scale"]; exists {
				value, valid := toFloat(configuredScale)
				if !valid || value < 0 || value > 4 || math.Trunc(value) != value {
					return nil, errors.New("climate temperature scale must be an integer between 0 and 4")
				}
				scale = int(value)
			}
			dps[fmt.Sprint(dpID)] = int(math.Round(temp * math.Pow10(scale)))
		case "set_hvac_mode":
			mode, ok := params["hvac_mode"].(string)
			if !ok {
				return nil, errors.New("climate hvac_mode must be 'auto' or 'manual'")
			}
			params = map[string]any{"mode": mode}
			fallthrough
		case "set_mode":
			mode, ok := params["mode"].(string)
			if !ok {
				return nil, errors.New("climate mode must be 'auto' or 'manual'")
			}
			switch strings.ToLower(strings.TrimSpace(mode)) {
			case "auto", "automatic", "automatico":
				dps["4"] = 0
			case "manual":
				dps["4"] = 1
			default:
				return nil, fmt.Errorf("unsupported climate mode: %s", mode)
			}
		case "set_dp":
			dpID, ok := toFloat(params["dp_id"])
			value, valueOK := toFloat(params["value"])
			if !ok || dpID != 4 || !valueOK || (value != 0 && value != 1) {
				return nil, errors.New("LocalTuya climate set_dp only accepts dp_id=4 and value 0 (automatic) or 1 (manual)")
			}
			dps["4"] = int(value)
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
	case string:
		if f, err := strconv.ParseFloat(strings.TrimSpace(n), 64); err == nil {
			return f, true
		}
	}
	return 0, false
}
