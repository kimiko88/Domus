// Package automation_test contains unit tests for the automation domain model.
package automation_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/domain/automation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAutomation_Validation(t *testing.T) {
	t.Run("valid automation", func(t *testing.T) {
		a := &automation.Automation{
			ID:          "auto-1",
			Name:        "Turn on light on motion",
			Description: "Automate hall light",
			Enabled:     true,
			Triggers: []automation.Trigger{
				{
					Platform:  "state",
					EntityKey: "binary_sensor.motion_hallway",
					To:        "ON",
				},
			},
			Conditions: []automation.Condition{
				{
					Condition: "state",
					EntityKey: "sun.sun",
					State:     "below_horizon",
				},
			},
			Actions: []automation.Action{
				{
					Type:            "service",
					Service:         "turn_on",
					TargetEntityKey: "light.hallway",
					Params:          map[string]any{"brightness": 180},
				},
			},
			ExecutionMode: "single",
		}

		err := a.Validate()
		require.NoError(t, err)
		assert.Equal(t, "single", a.ExecutionMode)
	})

	t.Run("empty name fails", func(t *testing.T) {
		a := &automation.Automation{
			ID: "auto-2",
			Triggers: []automation.Trigger{
				{Platform: "state", EntityKey: "switch.desk"},
			},
			Actions: []automation.Action{
				{Type: "service", Service: "turn_off", TargetEntityKey: "switch.desk"},
			},
		}
		err := a.Validate()
		assert.ErrorIs(t, err, automation.ErrInvalidName)
	})

	t.Run("missing triggers fails", func(t *testing.T) {
		a := &automation.Automation{
			ID:   "auto-3",
			Name: "No triggers",
			Actions: []automation.Action{
				{Type: "service", Service: "turn_on", TargetEntityKey: "light.living_room"},
			},
		}
		err := a.Validate()
		assert.ErrorIs(t, err, automation.ErrMissingTriggers)
	})

	t.Run("missing actions fails", func(t *testing.T) {
		a := &automation.Automation{
			ID:   "auto-4",
			Name: "No actions",
			Triggers: []automation.Trigger{
				{Platform: "state", EntityKey: "switch.main"},
			},
		}
		err := a.Validate()
		assert.ErrorIs(t, err, automation.ErrMissingActions)
	})

	t.Run("default execution mode is single", func(t *testing.T) {
		a := &automation.Automation{
			ID:   "auto-5",
			Name: "Default mode test",
			Triggers: []automation.Trigger{
				{Platform: "state", EntityKey: "switch.main"},
			},
			Actions: []automation.Action{
				{Type: "service", Service: "turn_on", TargetEntityKey: "light.room"},
			},
		}
		require.NoError(t, a.Validate())
		assert.Equal(t, "single", a.ExecutionMode)
	})
}

func TestAutomation_JSONSerialization(t *testing.T) {
	above := 25.0
	below := 30.0
	a := &automation.Automation{
		ID:      "auto-json-1",
		Name:    "Complex Automation",
		Enabled: true,
		Triggers: []automation.Trigger{
			{
				Platform:   "numeric_state",
				EntityKey:  "sensor.living_room_temperature",
				Above:      &above,
				Below:      &below,
				ForSeconds: 30,
			},
		},
		Conditions: []automation.Condition{
			{
				Condition: "and",
				Conditions: []automation.Condition{
					{Condition: "state", EntityKey: "switch.ac_mode", State: "cool"},
					{Condition: "time", After: "08:00:00", Before: "22:00:00"},
				},
			},
		},
		Actions: []automation.Action{
			{
				Type:            "service",
				Service:         "turn_on",
				TargetEntityKey: "climate.living_room_ac",
				Params:          map[string]any{"temperature": 23},
			},
			{
				Type:    "delay",
				DelayMs: 5000,
			},
		},
		ExecutionMode:   "restart",
		CooldownSeconds: 60,
	}

	bytes, err := json.Marshal(a)
	require.NoError(t, err)

	var unmarshaled automation.Automation
	err = json.Unmarshal(bytes, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, a.ID, unmarshaled.ID)
	assert.Equal(t, a.Name, unmarshaled.Name)
	assert.Len(t, unmarshaled.Triggers, 1)
	assert.Equal(t, "numeric_state", unmarshaled.Triggers[0].Platform)
	assert.Equal(t, 25.0, *unmarshaled.Triggers[0].Above)
	assert.Len(t, unmarshaled.Conditions, 1)
	assert.Equal(t, "and", unmarshaled.Conditions[0].Condition)
	assert.Len(t, unmarshaled.Actions, 2)
	assert.Equal(t, 5000, unmarshaled.Actions[1].DelayMs)
}

func TestAutomationRun_Model(t *testing.T) {
	now := time.Now().UTC()
	run := &automation.AutomationRun{
		ID:             "run-001",
		AutomationID:   "auto-001",
		StartedAt:      now,
		Status:         automation.RunStatusRunning,
		TriggerContext: `{"entity_key":"binary_sensor.motion"}`,
	}

	assert.Equal(t, automation.RunStatusRunning, run.Status)
	assert.Nil(t, run.FinishedAt)

	finished := now.Add(150 * time.Millisecond)
	run.Complete(finished, "Executed turn_on light")
	assert.Equal(t, automation.RunStatusCompleted, run.Status)
	assert.Equal(t, &finished, run.FinishedAt)
	assert.Equal(t, int64(150), run.DurationMs)
}
