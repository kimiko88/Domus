package cloud_assistant_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	appCloud "github.com/domus-hub/domus/internal/application/cloud_assistant"
	domainCloud "github.com/domus-hub/domus/internal/domain/cloud_assistant"
	"github.com/domus-hub/domus/internal/domain/entity"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockEntityRepo struct {
	items map[string]*entity.Entity
}

func newMockEntityRepo() *mockEntityRepo {
	return &mockEntityRepo{
		items: map[string]*entity.Entity{
			"light.living_room": {
				ID:         "light.living_room",
				Name:       "Luce Soggiorno",
				Domain:     "light",
				State:      `"on"`,
				Attributes: `{"brightness": 204.0}`,
			},
			"switch.coffee": {
				ID:     "switch.coffee",
				Name:   "Macchina Caffè",
				Domain: "switch",
				State:  `"off"`,
			},
			"climate.bedroom": {
				ID:         "climate.bedroom",
				Name:       "Termostato Camera",
				Domain:     "climate",
				State:      `"auto"`,
				Attributes: `{"temperature": 21.5, "current_temperature": 20.2}`,
			},
		},
	}
}

func (m *mockEntityRepo) Create(ctx context.Context, e *entity.Entity) error {
	m.items[e.ID] = e
	return nil
}

func (m *mockEntityRepo) GetByID(ctx context.Context, id string) (*entity.Entity, error) {
	if it, ok := m.items[id]; ok {
		return it, nil
	}
	return nil, entity.ErrEntityNotFound
}

func (m *mockEntityRepo) GetByKey(ctx context.Context, key string) (*entity.Entity, error) {
	for _, it := range m.items {
		if it.EntityKey == key {
			return it, nil
		}
	}
	return nil, entity.ErrEntityNotFound
}

func (m *mockEntityRepo) List(ctx context.Context, filter entity.ListFilter) ([]*entity.Entity, error) {
	res := make([]*entity.Entity, 0, len(m.items))
	for _, it := range m.items {
		if filter.Domain != "" && it.Domain != filter.Domain {
			continue
		}
		res = append(res, it)
	}
	return res, nil
}

func (m *mockEntityRepo) Update(ctx context.Context, e *entity.Entity) error {
	m.items[e.ID] = e
	return nil
}

func (m *mockEntityRepo) UpdateState(ctx context.Context, id string, stateJSON, attributesJSON string, available bool, at time.Time) error {
	if it, ok := m.items[id]; ok {
		it.State = stateJSON
		it.Attributes = attributesJSON
		it.Available = available
		it.LastUpdatedAt = at
		return nil
	}
	return entity.ErrEntityNotFound
}

func (m *mockEntityRepo) Delete(ctx context.Context, id string) error {
	delete(m.items, id)
	return nil
}

func TestCloudAssistant_AlexaDirectives(t *testing.T) {
	repo := newMockEntityRepo()
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 64, Workers: 1})
	svc := appCloud.NewService(repo, bus, "my-secret-token", true)

	assert.True(t, svc.ValidateToken("Bearer my-secret-token"))
	assert.False(t, svc.ValidateToken("Bearer wrong-token"))

	// 1. Alexa.Discovery
	discoveryPayload := []byte(`{
		"directive": {
			"header": {
				"namespace": "Alexa.Discovery",
				"name": "Discover",
				"payloadVersion": "3",
				"messageId": "msg-1",
				"correlationToken": "token-1"
			},
			"payload": {}
		}
	}`)
	resp, err := svc.HandleAlexa(context.Background(), discoveryPayload)
	require.NoError(t, err)
	assert.Equal(t, "Discover.Response", resp.Event.Header.Name)
	assert.Equal(t, "token-1", resp.Event.Header.CorrelationToken)

	// 2. Alexa.PowerController TurnOn
	powerOnPayload := []byte(`{
		"directive": {
			"header": {
				"namespace": "Alexa.PowerController",
				"name": "TurnOn",
				"payloadVersion": "3",
				"messageId": "msg-2",
				"correlationToken": "token-2"
			},
			"endpoint": {
				"endpointId": "switch.coffee"
			},
			"payload": {}
		}
	}`)
	resp, err = svc.HandleAlexa(context.Background(), powerOnPayload)
	require.NoError(t, err)
	assert.Equal(t, "Response", resp.Event.Header.Name)
	require.NotNil(t, resp.Context)
	assert.Equal(t, "ON", resp.Context.Properties[0].Value)

	// 3. Alexa.BrightnessController SetBrightness
	brightPayload := []byte(`{
		"directive": {
			"header": {
				"namespace": "Alexa.BrightnessController",
				"name": "SetBrightness",
				"payloadVersion": "3",
				"messageId": "msg-3",
				"correlationToken": "token-3"
			},
			"endpoint": {
				"endpointId": "light.living_room"
			},
			"payload": {
				"brightness": 75
			}
		}
	}`)
	resp, err = svc.HandleAlexa(context.Background(), brightPayload)
	require.NoError(t, err)
	assert.Equal(t, "Response", resp.Event.Header.Name)
	assert.Equal(t, 75, resp.Context.Properties[0].Value)

	// 4. Alexa.ThermostatController SetTargetTemperature
	thermoPayload := []byte(`{
		"directive": {
			"header": {
				"namespace": "Alexa.ThermostatController",
				"name": "SetTargetTemperature",
				"payloadVersion": "3",
				"messageId": "msg-4",
				"correlationToken": "token-4"
			},
			"endpoint": {
				"endpointId": "climate.bedroom"
			},
			"payload": {
				"targetSetpoint": {
					"value": 22.0,
					"scale": "CELSIUS"
				}
			}
		}
	}`)
	resp, err = svc.HandleAlexa(context.Background(), thermoPayload)
	require.NoError(t, err)
	assert.Equal(t, "Response", resp.Event.Header.Name)

	// 5. Alexa ReportState
	reportPayload := []byte(`{
		"directive": {
			"header": {
				"namespace": "Alexa",
				"name": "ReportState",
				"payloadVersion": "3",
				"messageId": "msg-5",
				"correlationToken": "token-5"
			},
			"endpoint": {
				"endpointId": "light.living_room"
			},
			"payload": {}
		}
	}`)
	resp, err = svc.HandleAlexa(context.Background(), reportPayload)
	require.NoError(t, err)
	assert.Equal(t, "StateReport", resp.Event.Header.Name)
	assert.Equal(t, "ON", resp.Context.Properties[0].Value)
}

func TestCloudAssistant_GoogleActions(t *testing.T) {
	repo := newMockEntityRepo()
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 64, Workers: 1})
	svc := appCloud.NewService(repo, bus, "my-secret-token", true)

	// 1. SYNC
	syncReq := &domainCloud.GoogleRequest{
		RequestID: "req-sync-1",
		Inputs: []domainCloud.GoogleRequestInput{
			{Intent: "action.devices.SYNC"},
		},
	}
	resp, err := svc.HandleGoogle(context.Background(), syncReq)
	require.NoError(t, err)
	assert.Equal(t, "req-sync-1", resp.RequestID)

	// 2. QUERY
	queryPayload, _ := json.Marshal(map[string]any{
		"devices": []map[string]string{
			{"id": "light.living_room"},
			{"id": "switch.coffee"},
		},
	})
	queryReq := &domainCloud.GoogleRequest{
		RequestID: "req-query-1",
		Inputs: []domainCloud.GoogleRequestInput{
			{Intent: "action.devices.QUERY", Payload: queryPayload},
		},
	}
	resp, err = svc.HandleGoogle(context.Background(), queryReq)
	require.NoError(t, err)
	assert.Equal(t, "req-query-1", resp.RequestID)

	// 3. EXECUTE
	executePayload, _ := json.Marshal(map[string]any{
		"commands": []map[string]any{
			{
				"devices": []map[string]string{{"id": "switch.coffee"}},
				"execution": []map[string]any{
					{
						"command": "action.devices.commands.OnOff",
						"params":  map[string]any{"on": true},
					},
				},
			},
		},
	})
	execReq := &domainCloud.GoogleRequest{
		RequestID: "req-exec-1",
		Inputs: []domainCloud.GoogleRequestInput{
			{Intent: "action.devices.EXECUTE", Payload: executePayload},
		},
	}
	resp, err = svc.HandleGoogle(context.Background(), execReq)
	require.NoError(t, err)
	assert.Equal(t, "req-exec-1", resp.RequestID)
}

func TestCloudAssistant_Disabled(t *testing.T) {
	repo := newMockEntityRepo()
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 16, Workers: 1})
	svc := appCloud.NewService(repo, bus, "my-token", false)

	_, err := svc.HandleAlexa(context.Background(), []byte(`{}`))
	assert.ErrorIs(t, err, appCloud.ErrCloudDisabled)

	_, err = svc.HandleGoogle(context.Background(), &domainCloud.GoogleRequest{})
	assert.ErrorIs(t, err, appCloud.ErrCloudDisabled)
}


func TestCloudAssistant_EmptyTokenFailsClosed(t *testing.T) {
	svc := appCloud.NewService(newMockEntityRepo(), nil, "", true)
	assert.False(t, svc.ValidateToken(""))
	assert.False(t, svc.ValidateToken("Bearer any-value"))
}
