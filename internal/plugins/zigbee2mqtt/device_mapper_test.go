// Package zigbee2mqtt_test contains unit and integration tests for Zigbee2MQTT adapter components.
package zigbee2mqtt_test

import (
	"testing"

	"github.com/domus-hub/domus/internal/domain/entity"
	"github.com/domus-hub/domus/internal/plugins/zigbee2mqtt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeviceMapper_MapBridgeDevices(t *testing.T) {
	// Sample realistic Zigbee2MQTT bridge/devices JSON payload
	bridgeDevicesJSON := `[
		{
			"ieee_address": "0x00124b0018e21a22",
			"type": "Coordinator",
			"friendly_name": "Coordinator"
		},
		{
			"ieee_address": "0x00158d0001a2b3c4",
			"type": "Router",
			"friendly_name": "living_room_light",
			"definition": {
				"model": "LED1624G9",
				"vendor": "IKEA",
				"description": "TRADFRI LED bulb E27 1000 lumen",
				"exposes": [
					{
						"type": "light",
						"features": [
							{"name": "state", "property": "state", "type": "binary"},
							{"name": "brightness", "property": "brightness", "type": "numeric"},
							{"name": "color_temp", "property": "color_temp", "type": "numeric"}
						]
					},
					{"name": "linkquality", "property": "linkquality", "type": "numeric"}
				]
			}
		},
		{
			"ieee_address": "0x00158d0002b3c4d5",
			"type": "EndDevice",
			"friendly_name": "bedroom_sensor",
			"definition": {
				"model": "WSDCGQ11LM",
				"vendor": "Aqara",
				"description": "Temperature and humidity sensor",
				"exposes": [
					{"name": "temperature", "property": "temperature", "type": "numeric", "unit": "°C"},
					{"name": "humidity", "property": "humidity", "type": "numeric", "unit": "%"},
					{"name": "battery", "property": "battery", "type": "numeric", "unit": "%"},
					{"name": "linkquality", "property": "linkquality", "type": "numeric"}
				]
			}
		},
		{
			"ieee_address": "0x00158d0003c4d5e6",
			"type": "EndDevice",
			"friendly_name": "front_door_sensor",
			"definition": {
				"model": "MCCGQ11LM",
				"vendor": "Aqara",
				"description": "Door and window contact sensor",
				"exposes": [
					{"name": "contact", "property": "contact", "type": "binary"},
					{"name": "battery", "property": "battery", "type": "numeric", "unit": "%"}
				]
			}
		},
		{
			"ieee_address": "0x00158d0004d5e6f7",
			"type": "Router",
			"friendly_name": "kitchen_plug",
			"definition": {
				"model": "SP-EUC01",
				"vendor": "Aqara",
				"description": "Smart wall plug",
				"exposes": [
					{
						"type": "switch",
						"features": [
							{"name": "state", "property": "state", "type": "binary"}
						]
					},
					{"name": "power", "property": "power", "type": "numeric", "unit": "W"}
				]
			}
		}
	]`

	mapper := zigbee2mqtt.NewDeviceMapper("z2m_integration")
	devices, entities, err := mapper.MapBridgeDevices([]byte(bridgeDevicesJSON))
	require.NoError(t, err)

	// Coordinator should be ignored, leaving 4 devices
	require.Len(t, devices, 4)

	// Verify living_room_light device & entity
	var lightDev *zigbee2mqtt.MappedDevice
	for _, d := range devices {
		if d.Device.ExternalID == "0x00158d0001a2b3c4" {
			lightDev = d
			break
		}
	}
	require.NotNil(t, lightDev)
	assert.Equal(t, "living_room_light", lightDev.Device.Name)
	assert.Equal(t, "IKEA", lightDev.Device.Manufacturer)
	assert.Equal(t, "LED1624G9", lightDev.Device.Model)
	assert.True(t, lightDev.Device.Online)

	// Verify entities generated across all devices
	entityKeys := make(map[string]*entity.Entity)
	for _, e := range entities {
		entityKeys[e.EntityKey] = e
	}

	// 1. Light entity
	require.Contains(t, entityKeys, "light.living_room_light")
	assert.Equal(t, "light", entityKeys["light.living_room_light"].Domain)
	assert.True(t, entityKeys["light.living_room_light"].Writable)

	// 2. Sensor entities for bedroom_sensor
	require.Contains(t, entityKeys, "sensor.bedroom_sensor_temperature")
	assert.Equal(t, "°C", entityKeys["sensor.bedroom_sensor_temperature"].UnitOfMeasurement)
	assert.False(t, entityKeys["sensor.bedroom_sensor_temperature"].Writable)

	require.Contains(t, entityKeys, "sensor.bedroom_sensor_humidity")
	assert.Equal(t, "%", entityKeys["sensor.bedroom_sensor_humidity"].UnitOfMeasurement)

	require.Contains(t, entityKeys, "sensor.bedroom_sensor_battery")
	assert.Equal(t, "%", entityKeys["sensor.bedroom_sensor_battery"].UnitOfMeasurement)

	// 3. Binary sensor entity for front door
	require.Contains(t, entityKeys, "binary_sensor.front_door_sensor_contact")
	assert.Equal(t, "binary_sensor", entityKeys["binary_sensor.front_door_sensor_contact"].Domain)
	assert.False(t, entityKeys["binary_sensor.front_door_sensor_contact"].Writable)

	// 4. Switch & power entity for kitchen plug
	require.Contains(t, entityKeys, "switch.kitchen_plug")
	assert.Equal(t, "switch", entityKeys["switch.kitchen_plug"].Domain)
	assert.True(t, entityKeys["switch.kitchen_plug"].Writable)

	require.Contains(t, entityKeys, "sensor.kitchen_plug_power")
	assert.Equal(t, "W", entityKeys["sensor.kitchen_plug_power"].UnitOfMeasurement)
}

func TestDeviceMapper_MapTelemetryPayload(t *testing.T) {
	mapper := zigbee2mqtt.NewDeviceMapper("z2m_integration")

	t.Run("light telemetry updates light state", func(t *testing.T) {
		payload := []byte(`{"state": "ON", "brightness": 254, "color_temp": 370, "linkquality": 80}`)
		updates, err := mapper.MapTelemetry("living_room_light", payload)
		require.NoError(t, err)
		require.NotEmpty(t, updates)

		var foundLight bool
		for _, u := range updates {
			if u.EntityKey == "light.living_room_light" {
				foundLight = true
				assert.Contains(t, u.StateJSON, `"state":"ON"`)
				assert.Contains(t, u.StateJSON, `"brightness":254`)
				assert.Contains(t, u.StateJSON, `"color_temp":370`)
				assert.Contains(t, u.AttributesJSON, `"linkquality":80`)
			}
		}
		assert.True(t, foundLight)
	})

	t.Run("sensor telemetry updates multiple sensor entities", func(t *testing.T) {
		payload := []byte(`{"temperature": 21.8, "humidity": 58.4, "battery": 92, "voltage": 3000}`)
		updates, err := mapper.MapTelemetry("bedroom_sensor", payload)
		require.NoError(t, err)

		updateMap := make(map[string]zigbee2mqtt.EntityStateUpdate)
		for _, u := range updates {
			updateMap[u.EntityKey] = u
		}

		require.Contains(t, updateMap, "sensor.bedroom_sensor_temperature")
		assert.Equal(t, `{"value":21.8}`, updateMap["sensor.bedroom_sensor_temperature"].StateJSON)

		require.Contains(t, updateMap, "sensor.bedroom_sensor_humidity")
		assert.Equal(t, `{"value":58.4}`, updateMap["sensor.bedroom_sensor_humidity"].StateJSON)

		require.Contains(t, updateMap, "sensor.bedroom_sensor_battery")
		assert.Equal(t, `{"value":92}`, updateMap["sensor.bedroom_sensor_battery"].StateJSON)
	})

	t.Run("binary sensor telemetry for contact", func(t *testing.T) {
		// Aqara contact: contact: true means closed, contact: false means open
		payload := []byte(`{"contact": false, "battery": 85}`)
		updates, err := mapper.MapTelemetry("front_door_sensor", payload)
		require.NoError(t, err)

		var foundContact bool
		for _, u := range updates {
			if u.EntityKey == "binary_sensor.front_door_sensor_contact" {
				foundContact = true
				assert.Equal(t, `{"state":false}`, u.StateJSON)
			}
		}
		assert.True(t, foundContact)
	})

	t.Run("occupancy sensor telemetry", func(t *testing.T) {
		payload := []byte(`{"occupancy": true}`)
		updates, err := mapper.MapTelemetry("hallway_motion", payload)
		require.NoError(t, err)

		require.Len(t, updates, 1)
		assert.Equal(t, "binary_sensor.hallway_motion_occupancy", updates[0].EntityKey)
		assert.Equal(t, `{"state":true}`, updates[0].StateJSON)
	})
}

func TestDeviceMapper_BuildCommandPayload(t *testing.T) {
	mapper := zigbee2mqtt.NewDeviceMapper("z2m_integration")

	t.Run("turn_on command", func(t *testing.T) {
		cmd := entity.Command{
			EntityKey: "light.living_room_light",
			Service:   "turn_on",
			Params:    map[string]any{"brightness": 128},
		}
		topic, payload, err := mapper.BuildCommand(cmd)
		require.NoError(t, err)
		assert.Equal(t, "zigbee2mqtt/living_room_light/set", topic)
		assert.Contains(t, string(payload), `"state":"ON"`)
		assert.Contains(t, string(payload), `"brightness":128`)
	})

	t.Run("turn_off command", func(t *testing.T) {
		cmd := entity.Command{
			EntityKey: "switch.kitchen_plug",
			Service:   "turn_off",
		}
		topic, payload, err := mapper.BuildCommand(cmd)
		require.NoError(t, err)
		assert.Equal(t, "zigbee2mqtt/kitchen_plug/set", topic)
		assert.Contains(t, string(payload), `"state":"OFF"`)
	})

	t.Run("toggle command", func(t *testing.T) {
		cmd := entity.Command{
			EntityKey: "light.living_room_light",
			Service:   "toggle",
		}
		topic, payload, err := mapper.BuildCommand(cmd)
		require.NoError(t, err)
		assert.Equal(t, "zigbee2mqtt/living_room_light/set", topic)
		assert.Contains(t, string(payload), `"state":"TOGGLE"`)
	})
}
