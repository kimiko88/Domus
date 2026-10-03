// Package zigbee2mqtt provides an integration adapter for Zigbee2MQTT.
package zigbee2mqtt

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/domus-hub/domus/internal/domain/device"
	"github.com/domus-hub/domus/internal/domain/entity"
	"github.com/google/uuid"
)

// BridgeDevice represents a device definition from zigbee2mqtt/bridge/devices.
type BridgeDevice struct {
	IEEEAddress  string            `json:"ieee_address"`
	Type         string            `json:"type"` // "Coordinator", "Router", "EndDevice"
	FriendlyName string            `json:"friendly_name"`
	Definition   *BridgeDefinition `json:"definition,omitempty"`
}

// BridgeDefinition describes the device hardware and capabilities.
type BridgeDefinition struct {
	Model       string         `json:"model"`
	Vendor      string         `json:"vendor"`
	Description string         `json:"description"`
	Exposes     []BridgeExpose `json:"exposes"`
}

// BridgeExpose represents a capability or sensor exposed by the device.
type BridgeExpose struct {
	Type     string         `json:"type"`     // "light", "switch", "binary", "numeric"
	Name     string         `json:"name"`     // "temperature", "humidity", "contact", "occupancy", etc.
	Property string         `json:"property"` // property name in telemetry JSON
	Unit     string         `json:"unit"`
	Features []BridgeExpose `json:"features"` // sub-features for light, switch
}

// MappedDevice pairs a Domus device with its metadata.
type MappedDevice struct {
	Device *device.Device
}

// EntityStateUpdate represents an updated state and attribute payload for a specific entity.
type EntityStateUpdate struct {
	EntityKey      string
	StateJSON      string
	AttributesJSON string
}

// DeviceMapper handles mapping between Zigbee2MQTT schemas and Domus domain models.
type DeviceMapper struct {
	integrationID string
	baseTopic     string
}

// NewDeviceMapper creates a new Zigbee2MQTT device mapper.
func NewDeviceMapper(integrationID string) *DeviceMapper {
	return &DeviceMapper{
		integrationID: integrationID,
		baseTopic:     "zigbee2mqtt",
	}
}

// SetBaseTopic configures a custom MQTT base topic (default: "zigbee2mqtt").
func (m *DeviceMapper) SetBaseTopic(baseTopic string) {
	if baseTopic != "" {
		m.baseTopic = baseTopic
	}
}

// MapBridgeDevices parses the bridge/devices payload into Domus devices and entities.
func (m *DeviceMapper) MapBridgeDevices(payload []byte) ([]*MappedDevice, []*entity.Entity, error) {
	var bridgeDevs []BridgeDevice
	if err := json.Unmarshal(payload, &bridgeDevs); err != nil {
		return nil, nil, fmt.Errorf("zigbee2mqtt: failed to unmarshal bridge devices: %w", err)
	}

	now := time.Now().UTC()
	var mappedDevices []*MappedDevice
	var allEntities []*entity.Entity

	for _, bd := range bridgeDevs {
		if strings.EqualFold(bd.Type, "Coordinator") {
			continue // Skip zigbee coordinator bridge itself
		}

		devID := uuid.NewString()
		manufacturer := ""
		model := ""
		if bd.Definition != nil {
			manufacturer = bd.Definition.Vendor
			model = bd.Definition.Model
		}

		dev := &device.Device{
			ID:            devID,
			IntegrationID: m.integrationID,
			ExternalID:    bd.IEEEAddress,
			Name:          bd.FriendlyName,
			Manufacturer:  manufacturer,
			Model:         model,
			Online:        true,
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		mappedDevices = append(mappedDevices, &MappedDevice{Device: dev})

		if bd.Definition == nil {
			continue
		}

		// Extract entities from exposes
		for _, exp := range bd.Definition.Exposes {
			switch exp.Type {
			case "light":
				ent := &entity.Entity{
					ID:            uuid.NewString(),
					DeviceID:      devID,
					EntityKey:     fmt.Sprintf("light.%s", bd.FriendlyName),
					Domain:        "light",
					Name:          bd.FriendlyName,
					Writable:      true,
					State:         `{"state":"OFF"}`,
					Attributes:    "{}",
					Available:     true,
					LastChangedAt: now,
					LastUpdatedAt: now,
				}
				allEntities = append(allEntities, ent)

			case "switch":
				ent := &entity.Entity{
					ID:            uuid.NewString(),
					DeviceID:      devID,
					EntityKey:     fmt.Sprintf("switch.%s", bd.FriendlyName),
					Domain:        "switch",
					Name:          bd.FriendlyName,
					Writable:      true,
					State:         `{"state":"OFF"}`,
					Attributes:    "{}",
					Available:     true,
					LastChangedAt: now,
					LastUpdatedAt: now,
				}
				allEntities = append(allEntities, ent)

			case "binary":
				domain := "binary_sensor"
				key := fmt.Sprintf("%s.%s_%s", domain, bd.FriendlyName, exp.Name)
				ent := &entity.Entity{
					ID:            uuid.NewString(),
					DeviceID:      devID,
					EntityKey:     key,
					Domain:        domain,
					Name:          fmt.Sprintf("%s %s", bd.FriendlyName, exp.Name),
					Writable:      false,
					State:         `{"state":false}`,
					Attributes:    "{}",
					Available:     true,
					LastChangedAt: now,
					LastUpdatedAt: now,
				}
				allEntities = append(allEntities, ent)

			case "numeric":
				domain := "sensor"
				key := fmt.Sprintf("%s.%s_%s", domain, bd.FriendlyName, exp.Name)
				ent := &entity.Entity{
					ID:                uuid.NewString(),
					DeviceID:          devID,
					EntityKey:         key,
					Domain:            domain,
					Name:              fmt.Sprintf("%s %s", bd.FriendlyName, exp.Name),
					UnitOfMeasurement: exp.Unit,
					Writable:          false,
					State:             `{"value":null}`,
					Attributes:        "{}",
					Available:         true,
					LastChangedAt:     now,
					LastUpdatedAt:     now,
				}
				allEntities = append(allEntities, ent)

			default:
				// Check property or name if type is not explicit
				name := exp.Name
				if name == "" {
					name = exp.Property
				}
				if name == "temperature" || name == "humidity" || name == "battery" || name == "power" || name == "voltage" {
					unit := exp.Unit
					if unit == "" {
						switch name {
						case "temperature":
							unit = "°C"
						case "humidity", "battery":
							unit = "%"
						case "power":
							unit = "W"
						case "voltage":
							unit = "mV"
						}
					}
					ent := &entity.Entity{
						ID:                uuid.NewString(),
						DeviceID:          devID,
						EntityKey:         fmt.Sprintf("sensor.%s_%s", bd.FriendlyName, name),
						Domain:            "sensor",
						Name:              fmt.Sprintf("%s %s", bd.FriendlyName, name),
						UnitOfMeasurement: unit,
						Writable:          false,
						State:             `{"value":null}`,
						Attributes:        "{}",
						Available:         true,
						LastChangedAt:     now,
						LastUpdatedAt:     now,
					}
					allEntities = append(allEntities, ent)
				} else if name == "contact" || name == "occupancy" || name == "motion" || name == "water_leak" {
					ent := &entity.Entity{
						ID:            uuid.NewString(),
						DeviceID:      devID,
						EntityKey:     fmt.Sprintf("binary_sensor.%s_%s", bd.FriendlyName, name),
						Domain:        "binary_sensor",
						Name:          fmt.Sprintf("%s %s", bd.FriendlyName, name),
						Writable:      false,
						State:         `{"state":false}`,
						Attributes:    "{}",
						Available:     true,
						LastChangedAt: now,
						LastUpdatedAt: now,
					}
					allEntities = append(allEntities, ent)
				}
			}
		}
	}

	return mappedDevices, allEntities, nil
}

// MapTelemetry transforms a friendly_name MQTT telemetry message into entity state updates.
func (m *DeviceMapper) MapTelemetry(friendlyName string, payload []byte) ([]EntityStateUpdate, error) {
	var raw map[string]any
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil, fmt.Errorf("zigbee2mqtt: failed to unmarshal telemetry for %s: %w", friendlyName, err)
	}

	var updates []EntityStateUpdate

	// Common attributes to preserve
	attrMap := make(map[string]any)
	for k, v := range raw {
		if k == "linkquality" || k == "voltage" || k == "update" {
			attrMap[k] = v
		}
	}
	attrBytes, _ := json.Marshal(attrMap)
	attrJSON := string(attrBytes)

	// 1. Light or Switch state
	if stateVal, hasState := raw["state"]; hasState {
		stateStr, _ := stateVal.(string)
		if raw["brightness"] != nil || raw["color_temp"] != nil {
			// Light
			lightData := map[string]any{"state": stateStr}
			if b, ok := raw["brightness"]; ok {
				lightData["brightness"] = b
			}
			if ct, ok := raw["color_temp"]; ok {
				lightData["color_temp"] = ct
			}
			b, _ := json.Marshal(lightData)
			updates = append(updates, EntityStateUpdate{
				EntityKey:      fmt.Sprintf("light.%s", friendlyName),
				StateJSON:      string(b),
				AttributesJSON: attrJSON,
			})
		} else {
			// Switch or basic Light
			switchData := map[string]any{"state": stateStr}
			b, _ := json.Marshal(switchData)
			updates = append(updates, EntityStateUpdate{
				EntityKey:      fmt.Sprintf("switch.%s", friendlyName),
				StateJSON:      string(b),
				AttributesJSON: attrJSON,
			})
			// Also support light if entity might be light
			updates = append(updates, EntityStateUpdate{
				EntityKey:      fmt.Sprintf("light.%s", friendlyName),
				StateJSON:      string(b),
				AttributesJSON: attrJSON,
			})
		}
	}

	// 2. Numeric sensors
	numericSensors := []string{"temperature", "humidity", "battery", "power", "illuminance", "pressure"}
	for _, s := range numericSensors {
		if val, exists := raw[s]; exists {
			sJSON, _ := json.Marshal(map[string]any{"value": val})
			updates = append(updates, EntityStateUpdate{
				EntityKey:      fmt.Sprintf("sensor.%s_%s", friendlyName, s),
				StateJSON:      string(sJSON),
				AttributesJSON: attrJSON,
			})
		}
	}

	// 3. Binary sensors
	binarySensors := []string{"contact", "occupancy", "motion", "water_leak", "smoke"}
	for _, b := range binarySensors {
		if val, exists := raw[b]; exists {
			boolVal, ok := val.(bool)
			if !ok {
				// Sometimes "ON"/"OFF" or 1/0
				if strVal, isStr := val.(string); isStr {
					boolVal = strings.EqualFold(strVal, "ON") || strings.EqualFold(strVal, "true")
				} else if numVal, isNum := val.(float64); isNum {
					boolVal = numVal != 0
				}
			}
			sJSON, _ := json.Marshal(map[string]any{"state": boolVal})
			updates = append(updates, EntityStateUpdate{
				EntityKey:      fmt.Sprintf("binary_sensor.%s_%s", friendlyName, b),
				StateJSON:      string(sJSON),
				AttributesJSON: attrJSON,
			})
		}
	}

	return updates, nil
}

// BuildCommand formats a Domus entity command into a Zigbee2MQTT topic and JSON payload.
func (m *DeviceMapper) BuildCommand(cmd entity.Command) (string, []byte, error) {
	parts := strings.SplitN(cmd.EntityKey, ".", 2)
	if len(parts) < 2 {
		return "", nil, fmt.Errorf("zigbee2mqtt: invalid entity key %s", cmd.EntityKey)
	}

	friendlyName := parts[1]
	// If entity key has property suffix e.g. switch.kitchen_plug or light.living_room_light
	// extract the friendly_name
	topic := fmt.Sprintf("%s/%s/set", m.baseTopic, friendlyName)

	payloadMap := make(map[string]any)

	switch cmd.Service {
	case "turn_on":
		payloadMap["state"] = "ON"
		for k, v := range cmd.Params {
			payloadMap[k] = v
		}
	case "turn_off":
		payloadMap["state"] = "OFF"
	case "toggle":
		payloadMap["state"] = "TOGGLE"
	case "set_brightness":
		payloadMap["state"] = "ON"
		if b, ok := cmd.Params["brightness"]; ok {
			payloadMap["brightness"] = b
		}
	case "set_color_temp":
		if ct, ok := cmd.Params["color_temp"]; ok {
			payloadMap["color_temp"] = ct
		}
	default:
		// Pass-through service and params
		for k, v := range cmd.Params {
			payloadMap[k] = v
		}
	}

	payloadBytes, err := json.Marshal(payloadMap)
	if err != nil {
		return "", nil, fmt.Errorf("zigbee2mqtt: failed to marshal command payload: %w", err)
	}

	return topic, payloadBytes, nil
}
