package device

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDevice_Validate(t *testing.T) {
	t.Run("valid device", func(t *testing.T) {
		d := &Device{
			ID:            "dev-1",
			IntegrationID: "integ-1",
			ExternalID:    "zigbee-0x00158d0001",
			Name:          "Living Room Temperature Sensor",
			Manufacturer:  "Aqara",
			Model:         "WSDCGQ11LM",
			Online:        true,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		require.NoError(t, d.Validate())
	})

	t.Run("empty name", func(t *testing.T) {
		d := &Device{
			IntegrationID: "integ-1",
			ExternalID:    "ext-1",
			Name:          "   ",
		}
		assert.ErrorIs(t, d.Validate(), ErrInvalidDeviceName)
	})

	t.Run("missing integration id", func(t *testing.T) {
		d := &Device{
			IntegrationID: "",
			ExternalID:    "ext-1",
			Name:          "My Sensor",
		}
		err := d.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "integration_id is required")
	})

	t.Run("missing external id", func(t *testing.T) {
		d := &Device{
			IntegrationID: "integ-1",
			ExternalID:    "   ",
			Name:          "My Sensor",
		}
		err := d.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "external_id is required")
	})

	t.Run("json serialization", func(t *testing.T) {
		d := &Device{
			ID:            "dev-123",
			IntegrationID: "integ-1",
			ExternalID:    "ext-1",
			Name:          "Plug",
			Online:        true,
			Metadata:      `{"voltage":230}`,
		}
		data, err := json.Marshal(d)
		require.NoError(t, err)

		var unmarshaled Device
		require.NoError(t, json.Unmarshal(data, &unmarshaled))
		assert.Equal(t, d.ID, unmarshaled.ID)
		assert.Equal(t, d.Name, unmarshaled.Name)
		assert.True(t, unmarshaled.Online)
	})
}
