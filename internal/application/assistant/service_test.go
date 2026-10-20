package assistant_test

import (
	"context"
	"testing"

	"github.com/domus-hub/domus/internal/application/assistant"
	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssistantService_HeuristicNLP(t *testing.T) {
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 64, Workers: 2})
	cfg := assistant.Config{
		Enabled: false, // force heuristic engine for deterministic testing
	}
	svc := assistant.NewService(cfg, bus, nil)
	ctx := context.Background()

	// Capture CommandRequested on bus
	cmdCh := make(chan map[string]any, 10)
	bus.Subscribe(events.CommandRequested, func(c context.Context, e events.Event) error {
		if payload, ok := e.Payload.(map[string]any); ok {
			cmdCh <- payload
		}
		return nil
	})

	// 1. Accendi luce in cucina
	res, err := svc.ProcessPrompt(ctx, "accendi la luce in cucina al 75%")
	require.NoError(t, err)
	assert.Contains(t, res.ResponseText, "cucina")
	assert.NotEmpty(t, res.ActionsExecuted)
	assert.Equal(t, "light.cucina", res.ActionsExecuted[0].EntityKey)
	assert.Equal(t, "turn_on", res.ActionsExecuted[0].Service)
	assert.Equal(t, 75, res.ActionsExecuted[0].Params["brightness"])

	captured := <-cmdCh
	assert.Equal(t, "light.cucina", captured["entity_key"])

	// 2. Spegni tutto
	res, err = svc.ProcessPrompt(ctx, "spegni tutto")
	require.NoError(t, err)
	assert.Contains(t, res.ResponseText, "spento tutte le luci")
	assert.Equal(t, "group.all_lights", res.ActionsExecuted[0].EntityKey)
	assert.Equal(t, "turn_off", res.ActionsExecuted[0].Service)

	// 3. Regola termostato
	res, err = svc.ProcessPrompt(ctx, "imposta la temperatura a 21.5 gradi in salotto")
	require.NoError(t, err)
	assert.Contains(t, res.ResponseText, "21.5°C")
	assert.Equal(t, "climate.salotto", res.ActionsExecuted[0].EntityKey)
	assert.Equal(t, 21.5, res.ActionsExecuted[0].Params["temperature"])

	// 4. Empty prompt
	res, err = svc.ProcessPrompt(ctx, "")
	require.NoError(t, err)
	assert.Contains(t, res.ResponseText, "Non ho capito")
}

func TestAssistantService_ToolCalling(t *testing.T) {
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 64, Workers: 2})
	cfg := assistant.Config{
		Enabled: true,
	}
	svc := assistant.NewService(cfg, bus, nil)
	ctx := context.Background()

	// 1. Verify schema definition
	tools := svc.BuildTools()
	assert.Len(t, tools, 4)

	// 2. Test direct ExecuteToolCall - control_device
	cmdCh := make(chan map[string]any, 10)
	bus.Subscribe(events.CommandRequested, func(c context.Context, e events.Event) error {
		if payload, ok := e.Payload.(map[string]any); ok {
			cmdCh <- payload
		}
		return nil
	})

	act, err := svc.ExecuteToolCall(ctx, "control_device", []byte(`{"entity_key":"light.studio","service":"turn_on","brightness":80}`))
	require.NoError(t, err)
	assert.True(t, act.Success)
	assert.Equal(t, "light.studio", act.EntityKey)
	assert.Equal(t, "turn_on", act.Service)
	assert.Equal(t, 80, act.Params["brightness"])

	captured := <-cmdCh
	assert.Equal(t, "light.studio", captured["entity_key"])
	assert.Equal(t, "turn_on", captured["service"])

	// 3. Test direct ExecuteToolCall - set_climate
	act, err = svc.ExecuteToolCall(ctx, "set_climate", []byte(`{"entity_key":"climate.camera","temperature":22.5}`))
	require.NoError(t, err)
	assert.True(t, act.Success)
	assert.Equal(t, "climate.camera", act.EntityKey)
	assert.Equal(t, "set_temperature", act.Service)
	assert.Equal(t, 22.5, act.Params["temperature"])

	capturedClimate := <-cmdCh
	assert.Equal(t, "climate.camera", capturedClimate["entity_key"])

	// 4. Test direct ExecuteToolCall - activate_scene
	act, err = svc.ExecuteToolCall(ctx, "activate_scene", []byte(`{"scene_id":"scene.cinema"}`))
	require.NoError(t, err)
	assert.True(t, act.Success)
	assert.Equal(t, "scene.cinema", act.EntityKey)
	assert.Equal(t, "activate", act.Service)

	capturedScene := <-cmdCh
	assert.Equal(t, "scene.cinema", capturedScene["entity_key"])
}
