package audit

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAudit_Entry(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	entry := Entry{
		ID:         42,
		ActorType:  ActorLocalUser,
		ActorID:    "user-uuid-1",
		Action:     "device.command",
		TargetType: "entity",
		TargetID:   "light.living_room",
		Payload:    `{"service":"turn_on"}`,
		CreatedAt:  now,
	}

	data, err := json.Marshal(entry)
	require.NoError(t, err)

	var unmarshaled Entry
	require.NoError(t, json.Unmarshal(data, &unmarshaled))

	assert.Equal(t, int64(42), unmarshaled.ID)
	assert.Equal(t, ActorLocalUser, unmarshaled.ActorType)
	assert.Equal(t, "user-uuid-1", unmarshaled.ActorID)
	assert.Equal(t, "device.command", unmarshaled.Action)
	assert.Equal(t, `{"service":"turn_on"}`, unmarshaled.Payload)
}

func TestActorTypes(t *testing.T) {
	actors := []ActorType{
		ActorLocalUser,
		ActorAPIToken,
		ActorAutomation,
		ActorSystem,
	}
	for _, a := range actors {
		assert.NotEmpty(t, string(a))
	}
}
