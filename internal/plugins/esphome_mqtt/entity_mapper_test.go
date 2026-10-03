// Package esphome_mqtt_test contains unit and integration tests for ESPHome MQTT adapter.
package esphome_mqtt_test

import (
	"testing"

	"github.com/domus-hub/domus/internal/domain/entity"
	"github.com/domus-hub/domus/internal/plugins/esphome_mqtt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEntityMapper_ParseTopic(t *testing.T) {
	mapper := esphome_mqtt.NewEntityMapper("esphome_main")

	t.Run("valid 4-part state topic", func(t *testing.T) {
		parsed, ok := mapper.ParseTopic("living_esp/sensor/temperature/state")
		require.True(t, ok)
		assert.Equal(t, "living_esp", parsed.NodeName)
		assert.Equal(t, "sensor", parsed.Domain)
		assert.Equal(t, "temperature", parsed.ObjectID)
		assert.Equal(t, "state", parsed.SubTopic)
		assert.Equal(t, "sensor.living_esp_temperature", parsed.EntityKey)
	})

	t.Run("valid switch command topic", func(t *testing.T) {
		parsed, ok := mapper.ParseTopic("kitchen_node/switch/valve/command")
		require.True(t, ok)
		assert.Equal(t, "kitchen_node", parsed.NodeName)
		assert.Equal(t, "switch", parsed.Domain)
		assert.Equal(t, "valve", parsed.ObjectID)
		assert.Equal(t, "command", parsed.SubTopic)
		assert.Equal(t, "switch.kitchen_node_valve", parsed.EntityKey)
	})

	t.Run("status topic", func(t *testing.T) {
		node, isStatus := mapper.ParseStatusTopic("living_esp/status")
		require.True(t, isStatus)
		assert.Equal(t, "living_esp", node)
	})

	t.Run("invalid topics", func(t *testing.T) {
		_, ok := mapper.ParseTopic("invalid")
		assert.False(t, ok)
		_, ok = mapper.ParseTopic("too/short/topic")
		assert.False(t, ok)
	})
}

func TestEntityMapper_MapStatePayload(t *testing.T) {
	mapper := esphome_mqtt.NewEntityMapper("esphome_main")

	t.Run("sensor float value", func(t *testing.T) {
		stateJSON, attrsJSON, err := mapper.MapState("sensor", []byte("23.85"))
		require.NoError(t, err)
		assert.Equal(t, `{"value":23.85}`, stateJSON)
		assert.Equal(t, "{}", attrsJSON)
	})

	t.Run("binary_sensor state", func(t *testing.T) {
		stateJSON, _, err := mapper.MapState("binary_sensor", []byte("ON"))
		require.NoError(t, err)
		assert.Equal(t, `{"state":true}`, stateJSON)

		stateJSON, _, err = mapper.MapState("binary_sensor", []byte("OFF"))
		require.NoError(t, err)
		assert.Equal(t, `{"state":false}`, stateJSON)
	})

	t.Run("switch state", func(t *testing.T) {
		stateJSON, _, err := mapper.MapState("switch", []byte("ON"))
		require.NoError(t, err)
		assert.Equal(t, `{"state":"ON"}`, stateJSON)

		stateJSON, _, err = mapper.MapState("switch", []byte("OFF"))
		require.NoError(t, err)
		assert.Equal(t, `{"state":"OFF"}`, stateJSON)
	})

	t.Run("light state with json payload", func(t *testing.T) {
		payload := []byte(`{"state":"ON","brightness":128}`)
		stateJSON, _, err := mapper.MapState("light", payload)
		require.NoError(t, err)
		assert.Contains(t, stateJSON, `"state":"ON"`)
		assert.Contains(t, stateJSON, `"brightness":128`)
	})
}

func TestEntityMapper_BuildCommand(t *testing.T) {
	mapper := esphome_mqtt.NewEntityMapper("esphome_main")

	t.Run("turn_on switch", func(t *testing.T) {
		cmd := entity.Command{
			EntityKey: "switch.node1_relay",
			Service:   "turn_on",
		}
		topic, payload, err := mapper.BuildCommand(cmd)
		require.NoError(t, err)
		assert.Equal(t, "node1/switch/relay/command", topic)
		assert.Equal(t, "ON", string(payload))
	})

	t.Run("turn_off switch", func(t *testing.T) {
		cmd := entity.Command{
			EntityKey: "switch.node1_relay",
			Service:   "turn_off",
		}
		topic, payload, err := mapper.BuildCommand(cmd)
		require.NoError(t, err)
		assert.Equal(t, "node1/switch/relay/command", topic)
		assert.Equal(t, "OFF", string(payload))
	})

	t.Run("light with brightness", func(t *testing.T) {
		mapper.RegisterNode("bedroom_esp")
		cmd := entity.Command{
			EntityKey: "light.bedroom_esp_ceiling",
			Service:   "turn_on",
			Params:    map[string]any{"brightness": 200},
		}
		topic, payload, err := mapper.BuildCommand(cmd)
		require.NoError(t, err)
		assert.Equal(t, "bedroom_esp/light/ceiling/command", topic)
		assert.Contains(t, string(payload), `"brightness":200`)
	})
}
