package integration

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_Serialization(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	integ := Integration{
		ID:              "integ-z2m",
		Type:            "zigbee2mqtt",
		Name:            "Zigbee Coordinator",
		Enabled:         true,
		ConfigEncrypted: []byte("super-secret-key"),
		Status:          "connected",
		LastError:       "",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	data, err := json.Marshal(integ)
	require.NoError(t, err)

	// ConfigEncrypted must NOT be leaked into JSON
	assert.NotContains(t, string(data), "super-secret-key")

	var unmarshaled Integration
	require.NoError(t, json.Unmarshal(data, &unmarshaled))
	assert.Equal(t, integ.ID, unmarshaled.ID)
	assert.Equal(t, integ.Type, unmarshaled.Type)
	assert.Equal(t, integ.Name, unmarshaled.Name)
	assert.True(t, unmarshaled.Enabled)
	assert.Equal(t, "connected", unmarshaled.Status)
}

func TestIntegration_ErrorDefinitions(t *testing.T) {
	assert.NotNil(t, ErrIntegrationNotFound)
	assert.Contains(t, ErrIntegrationNotFound.Error(), "not found")
}
