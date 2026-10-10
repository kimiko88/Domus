package floorplan

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFloor_Validate(t *testing.T) {
	t.Run("valid floor", func(t *testing.T) {
		f := &Floor{
			ID:        "fl-1",
			Name:      "Piano Terra",
			Level:     0,
			PlanSVG:   "<svg></svg>",
			SortOrder: 1,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		require.NoError(t, f.Validate())
	})

	t.Run("empty name", func(t *testing.T) {
		f := &Floor{Name: "   "}
		assert.ErrorIs(t, f.Validate(), ErrInvalidFloorName)
	})

	t.Run("name exceeds 100 characters", func(t *testing.T) {
		f := &Floor{Name: strings.Repeat("A", 101)}
		assert.ErrorIs(t, f.Validate(), ErrInvalidFloorName)
	})

	t.Run("json serialization", func(t *testing.T) {
		f := &Floor{
			ID:    "fl-2",
			Name:  "Primo Piano",
			Level: 1,
		}
		data, err := json.Marshal(f)
		require.NoError(t, err)

		var unmarshaled Floor
		require.NoError(t, json.Unmarshal(data, &unmarshaled))
		assert.Equal(t, f.Name, unmarshaled.Name)
		assert.Equal(t, f.Level, unmarshaled.Level)
	})
}

func TestMarker_Validate(t *testing.T) {
	t.Run("valid marker", func(t *testing.T) {
		m := &Marker{
			ID:        "m-1",
			FloorID:   "fl-1",
			EntityKey: "light.living_room",
			XPercent:  45.5,
			YPercent:  60.2,
			ShowBadge: true,
		}
		require.NoError(t, m.Validate())
	})

	t.Run("missing floor id", func(t *testing.T) {
		m := &Marker{
			FloorID:   "  ",
			EntityKey: "light.living_room",
			XPercent:  50.0,
			YPercent:  50.0,
		}
		err := m.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "floor_id is required")
	})

	t.Run("missing entity key", func(t *testing.T) {
		m := &Marker{
			FloorID:   "fl-1",
			EntityKey: "   ",
			XPercent:  50.0,
			YPercent:  50.0,
		}
		assert.ErrorIs(t, m.Validate(), ErrInvalidEntityKey)
	})

	t.Run("invalid X coordinates", func(t *testing.T) {
		m1 := &Marker{FloorID: "fl-1", EntityKey: "light.a", XPercent: -0.1, YPercent: 50.0}
		assert.ErrorIs(t, m1.Validate(), ErrInvalidMarkerPos)

		m2 := &Marker{FloorID: "fl-1", EntityKey: "light.a", XPercent: 100.1, YPercent: 50.0}
		assert.ErrorIs(t, m2.Validate(), ErrInvalidMarkerPos)
	})

	t.Run("invalid Y coordinates", func(t *testing.T) {
		m1 := &Marker{FloorID: "fl-1", EntityKey: "light.a", XPercent: 50.0, YPercent: -5.0}
		assert.ErrorIs(t, m1.Validate(), ErrInvalidMarkerPos)

		m2 := &Marker{FloorID: "fl-1", EntityKey: "light.a", XPercent: 50.0, YPercent: 120.0}
		assert.ErrorIs(t, m2.Validate(), ErrInvalidMarkerPos)
	})

	t.Run("boundary coordinates valid", func(t *testing.T) {
		mMin := &Marker{FloorID: "fl-1", EntityKey: "light.a", XPercent: 0.0, YPercent: 0.0}
		assert.NoError(t, mMin.Validate())

		mMax := &Marker{FloorID: "fl-1", EntityKey: "light.a", XPercent: 100.0, YPercent: 100.0}
		assert.NoError(t, mMax.Validate())
	})
}
