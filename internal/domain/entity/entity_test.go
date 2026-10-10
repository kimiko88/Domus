package entity

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEntity_Validate(t *testing.T) {
	t.Run("valid entity with explicit domain", func(t *testing.T) {
		e := &Entity{
			ID:        "ent-1",
			EntityKey: "light.living_room",
			Domain:    "light",
			Name:      "Living Room Light",
		}
		err := e.Validate()
		require.NoError(t, err)
		assert.Equal(t, "null", e.State)
		assert.Equal(t, "{}", e.Attributes)
	})

	t.Run("valid entity infers domain from key", func(t *testing.T) {
		e := &Entity{
			ID:        "ent-2",
			EntityKey: "switch.kitchen_plug",
			Name:      "Kitchen Plug",
		}
		err := e.Validate()
		require.NoError(t, err)
		assert.Equal(t, "switch", e.Domain)
	})

	t.Run("missing dot in entity key", func(t *testing.T) {
		e := &Entity{
			EntityKey: "invalidkey",
			Name:      "Bad Key",
		}
		err := e.Validate()
		assert.ErrorIs(t, err, ErrInvalidEntityKey)
	})

	t.Run("empty prefix or suffix in key", func(t *testing.T) {
		e1 := &Entity{EntityKey: ".suffix", Name: "Test"}
		assert.ErrorIs(t, e1.Validate(), ErrInvalidEntityKey)

		e2 := &Entity{EntityKey: "prefix.", Name: "Test"}
		assert.ErrorIs(t, e2.Validate(), ErrInvalidEntityKey)
	})

	t.Run("empty name fails", func(t *testing.T) {
		e := &Entity{
			EntityKey: "sensor.temp",
			Name:      "   ",
		}
		err := e.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name cannot be empty")
	})

	t.Run("preserves existing state and attributes", func(t *testing.T) {
		e := &Entity{
			EntityKey:  "sensor.temp",
			Name:       "Temp Sensor",
			State:      "21.5",
			Attributes: `{"unit":"C"}`,
		}
		err := e.Validate()
		require.NoError(t, err)
		assert.Equal(t, "21.5", e.State)
		assert.Equal(t, `{"unit":"C"}`, e.Attributes)
	})
}

func TestTypedStates(t *testing.T) {
	t.Run("LightState json marshaling", func(t *testing.T) {
		bri := 255
		ct := 4000
		ls := LightState{
			State:      "ON",
			Brightness: &bri,
			ColorTemp:  &ct,
		}
		data, err := json.Marshal(ls)
		require.NoError(t, err)

		var unmarshaled LightState
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)
		assert.Equal(t, "ON", unmarshaled.State)
		assert.Equal(t, 255, *unmarshaled.Brightness)
		assert.Equal(t, 4000, *unmarshaled.ColorTemp)
	})

	t.Run("SwitchState json", func(t *testing.T) {
		ss := SwitchState{State: "OFF"}
		data, err := json.Marshal(ss)
		require.NoError(t, err)

		var unmarshaled SwitchState
		require.NoError(t, json.Unmarshal(data, &unmarshaled))
		assert.Equal(t, "OFF", unmarshaled.State)
	})

	t.Run("SensorState json", func(t *testing.T) {
		s := SensorState{Value: 42.195}
		data, err := json.Marshal(s)
		require.NoError(t, err)

		var unmarshaled SensorState
		require.NoError(t, json.Unmarshal(data, &unmarshaled))
		assert.Equal(t, 42.195, unmarshaled.Value)
	})

	t.Run("BinarySensorState json", func(t *testing.T) {
		bs := BinarySensorState{State: true}
		data, err := json.Marshal(bs)
		require.NoError(t, err)

		var unmarshaled BinarySensorState
		require.NoError(t, json.Unmarshal(data, &unmarshaled))
		assert.True(t, unmarshaled.State)
	})

	t.Run("Command json", func(t *testing.T) {
		cmd := Command{
			EntityKey: "light.bedroom",
			Service:   "turn_on",
			Params:    map[string]any{"brightness": 128},
		}
		data, err := json.Marshal(cmd)
		require.NoError(t, err)

		var unmarshaled Command
		require.NoError(t, json.Unmarshal(data, &unmarshaled))
		assert.Equal(t, "light.bedroom", unmarshaled.EntityKey)
		assert.Equal(t, "turn_on", unmarshaled.Service)
		assert.Equal(t, float64(128), unmarshaled.Params["brightness"])
	})
}
