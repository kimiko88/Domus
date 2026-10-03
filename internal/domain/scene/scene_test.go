// Package scene_test contains unit tests for the scene domain model.
package scene_test

import (
	"encoding/json"
	"testing"

	"github.com/domus-hub/domus/internal/domain/scene"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScene_Validation(t *testing.T) {
	t.Run("valid scene", func(t *testing.T) {
		s := &scene.Scene{
			ID:   "scene-movie",
			Name: "Movie Time",
			Icon: "film",
			Actions: []scene.Action{
				{
					EntityKey: "light.living_room",
					Service:   "turn_on",
					Params:    map[string]any{"brightness": 20},
				},
				{
					EntityKey: "switch.tv_socket",
					Service:   "turn_on",
				},
			},
		}
		require.NoError(t, s.Validate())
	})

	t.Run("empty name fails", func(t *testing.T) {
		s := &scene.Scene{
			Actions: []scene.Action{
				{EntityKey: "light.1", Service: "turn_off"},
			},
		}
		err := s.Validate()
		assert.ErrorIs(t, err, scene.ErrInvalidName)
	})

	t.Run("empty actions fails", func(t *testing.T) {
		s := &scene.Scene{
			Name: "Empty Scene",
		}
		err := s.Validate()
		assert.ErrorIs(t, err, scene.ErrMissingActions)
	})
}

func TestScene_JSONSerialization(t *testing.T) {
	s := &scene.Scene{
		ID:   "scene-night",
		Name: "Goodnight",
		Icon: "moon",
		Actions: []scene.Action{
			{
				EntityKey: "light.all",
				Service:   "turn_off",
			},
		},
	}

	bytes, err := json.Marshal(s)
	require.NoError(t, err)

	var unmarshaled scene.Scene
	err = json.Unmarshal(bytes, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, s.ID, unmarshaled.ID)
	assert.Equal(t, s.Name, unmarshaled.Name)
	assert.Len(t, unmarshaled.Actions, 1)
}
