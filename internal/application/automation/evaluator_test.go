// Package automation_test provides unit tests for the automation evaluation engine.
package automation_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	appauto "github.com/domus-hub/domus/internal/application/automation"
	"github.com/domus-hub/domus/internal/domain/automation"
	"github.com/domus-hub/domus/internal/domain/device"
	"github.com/domus-hub/domus/internal/domain/entity"
	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/domain/integration"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvaluator_Triggers(t *testing.T) {
	evaluator := appauto.NewEvaluator()

	t.Run("state trigger exact match", func(t *testing.T) {
		trig := automation.Trigger{
			Platform:  "state",
			EntityKey: "binary_sensor.motion",
			To:        "ON",
		}
		evt := events.Event{
			Type: events.EntityStateChanged,
			Payload: map[string]any{
				"entity_key": "binary_sensor.motion",
				"state":      `{"state":"ON"}`,
			},
		}
		assert.True(t, evaluator.EvaluateTrigger(trig, evt))

		// Mismatched state
		evtOff := events.Event{
			Type: events.EntityStateChanged,
			Payload: map[string]any{
				"entity_key": "binary_sensor.motion",
				"state":      `{"state":"OFF"}`,
			},
		}
		assert.False(t, evaluator.EvaluateTrigger(trig, evtOff))
	})

	t.Run("numeric_state trigger above threshold", func(t *testing.T) {
		above := 25.0
		trig := automation.Trigger{
			Platform:  "numeric_state",
			EntityKey: "sensor.temp",
			Above:     &above,
		}
		evt := events.Event{
			Type: events.EntityStateChanged,
			Payload: map[string]any{
				"entity_key": "sensor.temp",
				"state":      `{"value":26.5}`,
			},
		}
		assert.True(t, evaluator.EvaluateTrigger(trig, evt))

		evtBelow := events.Event{
			Type: events.EntityStateChanged,
			Payload: map[string]any{
				"entity_key": "sensor.temp",
				"state":      `{"value":24.0}`,
			},
		}
		assert.False(t, evaluator.EvaluateTrigger(trig, evtBelow))
	})

	t.Run("event trigger matches event type", func(t *testing.T) {
		trig := automation.Trigger{
			Platform:  "event",
			EventType: "custom.button_click",
		}
		evt := events.Event{
			Type: "custom.button_click",
		}
		assert.True(t, evaluator.EvaluateTrigger(trig, evt))
	})
}

func TestEvaluator_Conditions(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "eval_cond_test.db")

	db, err := sqlite.Open(config.DatabaseConfig{
		Path:          dbPath,
		WALMode:       true,
		BusyTimeoutMs: 5000,
		MaxOpenConns:  1,
	})
	require.NoError(t, err)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, sqlite.Migrate(ctx, db))

	repos := sqlite.NewRepositories(db)
	now := time.Now().UTC()

	// Seed integration, device, entity
	in := &integration.Integration{ID: "in1", Type: "test", Name: "Test In", Status: "connected", CreatedAt: now, UpdatedAt: now}
	require.NoError(t, repos.Integrations.Create(ctx, in))
	dev := &device.Device{ID: "dev1", IntegrationID: "in1", ExternalID: "ext1", Name: "Dev 1", Online: true, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, repos.Devices.Create(ctx, dev))
	entTemp := &entity.Entity{
		ID:        uuid.NewString(),
		DeviceID:  dev.ID,
		EntityKey: "sensor.temperature",
		Domain:    "sensor",
		Name:      "Temp",
		State:     `{"value":22.5}`,
		Available: true,
	}
	require.NoError(t, repos.Entities.Create(ctx, entTemp))

	evaluator := appauto.NewEvaluator()

	t.Run("numeric_state condition below threshold", func(t *testing.T) {
		below := 25.0
		cond := automation.Condition{
			Condition: "numeric_state",
			EntityKey: "sensor.temperature",
			Below:     &below,
		}
		ok, err := evaluator.EvaluateCondition(ctx, cond, repos.Entities)
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("composite and condition", func(t *testing.T) {
		below := 25.0
		above := 20.0
		cond := automation.Condition{
			Condition: "and",
			Conditions: []automation.Condition{
				{Condition: "numeric_state", EntityKey: "sensor.temperature", Below: &below},
				{Condition: "numeric_state", EntityKey: "sensor.temperature", Above: &above},
			},
		}
		ok, err := evaluator.EvaluateCondition(ctx, cond, repos.Entities)
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("composite not condition", func(t *testing.T) {
		above := 30.0
		cond := automation.Condition{
			Condition: "not",
			Conditions: []automation.Condition{
				{Condition: "numeric_state", EntityKey: "sensor.temperature", Above: &above},
			},
		}
		ok, err := evaluator.EvaluateCondition(ctx, cond, repos.Entities)
		require.NoError(t, err)
		assert.True(t, ok)
	})
}
