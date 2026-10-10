package area

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArea_Validate(t *testing.T) {
	t.Run("valid area", func(t *testing.T) {
		a := &Area{
			ID:        "area-1",
			Name:      "Cucina",
			Icon:      "mdi:silverware-fork-knife",
			SortOrder: 1,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		require.NoError(t, a.Validate())
	})

	t.Run("empty name", func(t *testing.T) {
		a := &Area{Name: "   "}
		assert.ErrorIs(t, a.Validate(), ErrInvalidAreaName)
	})

	t.Run("name exceeds 100 characters", func(t *testing.T) {
		a := &Area{Name: strings.Repeat("K", 101)}
		assert.ErrorIs(t, a.Validate(), ErrInvalidAreaName)
	})

	t.Run("json serialization", func(t *testing.T) {
		a := &Area{
			ID:           "area-2",
			Name:         "Salotto",
			ParentAreaID: "area-1",
		}
		data, err := json.Marshal(a)
		require.NoError(t, err)

		var unmarshaled Area
		require.NoError(t, json.Unmarshal(data, &unmarshaled))
		assert.Equal(t, a.Name, unmarshaled.Name)
		assert.Equal(t, a.ParentAreaID, unmarshaled.ParentAreaID)
	})
}
