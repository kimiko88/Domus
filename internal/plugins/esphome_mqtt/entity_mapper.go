// Package esphome_mqtt provides an integration adapter for ESPHome nodes via MQTT.
package esphome_mqtt

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/domus-hub/domus/internal/domain/entity"
)

// ParsedTopic holds metadata extracted from an ESPHome MQTT topic.
type ParsedTopic struct {
	NodeName  string
	Domain    string
	ObjectID  string
	SubTopic  string // "state", "command", "attributes"
	EntityKey string // e.g. "sensor.node1_temperature"
}

// EntityMapper handles parsing and formatting for ESPHome MQTT payloads and topics.
type EntityMapper struct {
	integrationID string
	knownNodes    map[string]struct{}
}

// NewEntityMapper creates a new ESPHome entity mapper.
func NewEntityMapper(integrationID string) *EntityMapper {
	return &EntityMapper{
		integrationID: integrationID,
		knownNodes:    make(map[string]struct{}),
	}
}

// RegisterNode records a known ESPHome node name.
func (m *EntityMapper) RegisterNode(nodeName string) {
	if nodeName != "" {
		m.knownNodes[nodeName] = struct{}{}
	}
}

// ParseTopic extracts components from <node_name>/<domain>/<object_id>/<subtopic>.
func (m *EntityMapper) ParseTopic(topic string) (ParsedTopic, bool) {
	parts := strings.Split(topic, "/")
	if len(parts) < 4 {
		return ParsedTopic{}, false
	}

	nodeName := parts[0]
	m.RegisterNode(nodeName)

	domain := parts[1]
	subTopic := parts[len(parts)-1]
	objectID := strings.Join(parts[2:len(parts)-1], "_")

	entityKey := fmt.Sprintf("%s.%s_%s", domain, nodeName, objectID)

	return ParsedTopic{
		NodeName:  nodeName,
		Domain:    domain,
		ObjectID:  objectID,
		SubTopic:  subTopic,
		EntityKey: entityKey,
	}, true
}

// ParseStatusTopic checks if the topic matches <node_name>/status.
func (m *EntityMapper) ParseStatusTopic(topic string) (string, bool) {
	parts := strings.Split(topic, "/")
	if len(parts) == 2 && parts[1] == "status" {
		m.RegisterNode(parts[0])
		return parts[0], true
	}
	return "", false
}

// MapState parses the MQTT payload into Domus state and attributes JSON strings.
func (m *EntityMapper) MapState(domain string, payload []byte) (string, string, error) {
	rawStr := strings.TrimSpace(string(payload))
	attrsJSON := "{}"

	switch domain {
	case "sensor":
		if val, err := strconv.ParseFloat(rawStr, 64); err == nil {
			b, _ := json.Marshal(map[string]any{"value": val})
			return string(b), attrsJSON, nil
		}
		// String sensor value fallback
		b, _ := json.Marshal(map[string]any{"value": rawStr})
		return string(b), attrsJSON, nil

	case "binary_sensor":
		isTrue := strings.EqualFold(rawStr, "ON") || strings.EqualFold(rawStr, "true") || rawStr == "1"
		b, _ := json.Marshal(map[string]any{"state": isTrue})
		return string(b), attrsJSON, nil

	case "switch":
		state := "OFF"
		if strings.EqualFold(rawStr, "ON") || strings.EqualFold(rawStr, "true") || rawStr == "1" {
			state = "ON"
		}
		b, _ := json.Marshal(map[string]any{"state": state})
		return string(b), attrsJSON, nil

	case "light":
		if strings.HasPrefix(rawStr, "{") {
			// Already a JSON object
			return rawStr, attrsJSON, nil
		}
		state := "OFF"
		if strings.EqualFold(rawStr, "ON") {
			state = "ON"
		}
		b, _ := json.Marshal(map[string]any{"state": state})
		return string(b), attrsJSON, nil

	default:
		// Default pass-through
		if strings.HasPrefix(rawStr, "{") {
			return rawStr, attrsJSON, nil
		}
		b, _ := json.Marshal(map[string]any{"value": rawStr})
		return string(b), attrsJSON, nil
	}
}

// BuildCommand converts a Domus Command into an ESPHome command topic and payload.
func (m *EntityMapper) BuildCommand(cmd entity.Command) (string, []byte, error) {
	parts := strings.SplitN(cmd.EntityKey, ".", 2)
	if len(parts) < 2 {
		return "", nil, fmt.Errorf("esphome: invalid entity key: %s", cmd.EntityKey)
	}

	domain := parts[0]
	keyRest := parts[1] // e.g. "living_esp_ceiling" or "node1_relay"

	// Resolve nodeName and objectID using known nodes first
	nodeName := ""
	objectID := ""

	for kn := range m.knownNodes {
		prefix := kn + "_"
		if strings.HasPrefix(keyRest, prefix) {
			if len(kn) > len(nodeName) {
				nodeName = kn
				objectID = strings.TrimPrefix(keyRest, prefix)
			}
		}
	}

	if nodeName == "" {
		// Fallback to first underscore split
		subParts := strings.SplitN(keyRest, "_", 2)
		if len(subParts) < 2 {
			return "", nil, fmt.Errorf("esphome: cannot separate node and object from %s", keyRest)
		}
		nodeName = subParts[0]
		objectID = subParts[1]
	}

	return m.BuildCommandForNode(nodeName, domain, objectID, cmd)
}

// BuildCommandForNode builds an ESPHome command given explicit node, domain, and objectID.
func (m *EntityMapper) BuildCommandForNode(nodeName, domain, objectID string, cmd entity.Command) (string, []byte, error) {
	topic := fmt.Sprintf("%s/%s/%s/command", nodeName, domain, objectID)

	if len(cmd.Params) > 0 {
		// If custom parameters (like brightness, rgb) are provided, format JSON
		payloadMap := make(map[string]any)
		switch cmd.Service {
		case "turn_on":
			payloadMap["state"] = "ON"
		case "turn_off":
			payloadMap["state"] = "OFF"
		case "toggle":
			payloadMap["state"] = "TOGGLE"
		default:
			payloadMap["service"] = cmd.Service
		}
		for k, v := range cmd.Params {
			payloadMap[k] = v
		}
		bytes, err := json.Marshal(payloadMap)
		return topic, bytes, err
	}

	// Simple text command
	switch cmd.Service {
	case "turn_on":
		return topic, []byte("ON"), nil
	case "turn_off":
		return topic, []byte("OFF"), nil
	case "toggle":
		return topic, []byte("TOGGLE"), nil
	default:
		return topic, []byte(strings.ToUpper(cmd.Service)), nil
	}
}

// InferUnit attempts to determine the unit of measurement based on the entity object ID.
func InferUnit(objectID string) string {
	lower := strings.ToLower(objectID)
	switch {
	case strings.Contains(lower, "temperature") || strings.Contains(lower, "temp"):
		return "°C"
	case strings.Contains(lower, "humidity"):
		return "%"
	case strings.Contains(lower, "battery"):
		return "%"
	case strings.Contains(lower, "power") || strings.Contains(lower, "watt"):
		return "W"
	case strings.Contains(lower, "energy"):
		return "kWh"
	case strings.Contains(lower, "voltage"):
		return "V"
	case strings.Contains(lower, "current"):
		return "A"
	case strings.Contains(lower, "pressure"):
		return "hPa"
	case strings.Contains(lower, "rssi") || strings.Contains(lower, "signal"):
		return "dBm"
	default:
		return ""
	}
}
