package knx_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/domain/entity"
	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/domus-hub/domus/internal/plugins/knx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockKNXClient struct {
	mu         sync.Mutex
	connected  bool
	inbound    knx.InboundHandler
	sentFrames []struct {
		Dst  uint16
		APCI uint16
		Data []byte
	}
}

func newMockKNXClient() *mockKNXClient {
	return &mockKNXClient{connected: true}
}

func (m *mockKNXClient) Connect(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = true
	return nil
}

func (m *mockKNXClient) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = false
	return nil
}

func (m *mockKNXClient) IsConnected() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.connected
}

func (m *mockKNXClient) SetInboundHandler(h knx.InboundHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.inbound = h
}

func (m *mockKNXClient) SendTelegram(dstGroupAddr uint16, apci uint16, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sentFrames = append(m.sentFrames, struct {
		Dst  uint16
		APCI uint16
		Data []byte
	}{Dst: dstGroupAddr, APCI: apci, Data: data})
	return nil
}

func (m *mockKNXClient) triggerInbound(frame *knx.CEMIFrame) {
	m.mu.Lock()
	h := m.inbound
	m.mu.Unlock()
	if h != nil {
		h(frame)
	}
}

type mockEntityRepoForKNX struct {
	mu     sync.Mutex
	items  map[string]*entity.Entity
	states map[string]string
}

func newMockEntityRepoForKNX() *mockEntityRepoForKNX {
	return &mockEntityRepoForKNX{
		items:  make(map[string]*entity.Entity),
		states: make(map[string]string),
	}
}

func (r *mockEntityRepoForKNX) Create(ctx context.Context, e *entity.Entity) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[e.ID] = e
	return nil
}

func (r *mockEntityRepoForKNX) GetByID(ctx context.Context, id string) (*entity.Entity, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if it, ok := r.items[id]; ok {
		return it, nil
	}
	return nil, entity.ErrEntityNotFound
}

func (r *mockEntityRepoForKNX) GetByKey(ctx context.Context, key string) (*entity.Entity, error) {
	return nil, entity.ErrEntityNotFound
}

func (r *mockEntityRepoForKNX) List(ctx context.Context, filter entity.ListFilter) ([]*entity.Entity, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	res := make([]*entity.Entity, 0, len(r.items))
	for _, it := range r.items {
		res = append(res, it)
	}
	return res, nil
}

func (r *mockEntityRepoForKNX) Update(ctx context.Context, e *entity.Entity) error { return nil }

func (r *mockEntityRepoForKNX) UpdateState(ctx context.Context, id string, stateJSON, attributesJSON string, available bool, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.states[id] = stateJSON
	return nil
}

func (r *mockEntityRepoForKNX) Delete(ctx context.Context, id string) error { return nil }

func TestKNX_AdapterLifecycle_InboundAndOutbound(t *testing.T) {
	client := newMockKNXClient()
	repo := newMockEntityRepoForKNX()
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 64, Workers: 1})

	cfg := knx.Config{
		IntegrationID: "knx_main_building",
		GatewayAddr:   "192.168.1.200:3671",
		Entities: []knx.EntityMapping{
			{
				EntityKey:      "light.corridor",
				Name:           "Corridor Light",
				Domain:         "light",
				DPT:            knx.DPT1_Switch,
				CommandAddress: "1/1/1",
				StateAddress:   "1/1/2",
				Writable:       true,
			},
			{
				EntityKey:      "sensor.outdoor_temp",
				Name:           "Outdoor Temperature",
				Domain:         "sensor",
				DPT:            knx.DPT9_Temperature,
				StateAddress:   "1/2/1",
				Unit:           "°C",
				Writable:       false,
			},
		},
	}

	adapter, err := knx.NewAdapter(cfg, client, repo, bus, nil)
	require.NoError(t, err)

	assert.Equal(t, "knx_main_building", adapter.ID())
	assert.Equal(t, "knx", adapter.Type())
	require.NoError(t, adapter.HealthCheck(context.Background()))

	// Channel to receive emitted state events
	stateEvents := make(chan events.Event, 10)
	bus.Subscribe(events.EntityStateChanged, func(ctx context.Context, ev events.Event) error {
		stateEvents <- ev
		return nil
	})

	err = adapter.Start(context.Background())
	require.NoError(t, err)

	// 1. Simulate inbound KNX telegram for temperature (DPT 9 on 1/2/1)
	tempRaw, _ := knx.ParseGroupAddress("1/2/1")
	tempBytes, _ := knx.EncodeDPT9(18.5)
	client.triggerInbound(&knx.CEMIFrame{
		MessageCode: knx.CEMIMsgDataInd,
		Destination: tempRaw,
		IsGroup:     true,
		Data:        tempBytes,
	})

	select {
	case ev := <-stateEvents:
		assert.Equal(t, "knx", ev.Source)
		pMap := ev.Payload.(map[string]any)
		assert.Equal(t, "sensor.outdoor_temp", pMap["entity_id"])
		assert.InDelta(t, 18.5, pMap["new_state"], 0.1)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for knx inbound state event")
	}

	// 2. Simulate outbound Domus command on light.corridor (turn_on)
	cmdErr := bus.Publish(context.Background(), events.Event{
		ID:         "cmd-knx-1",
		Type:       events.CommandRequested,
		OccurredAt: time.Now().UTC(),
		Payload: map[string]any{
			"entity_id": "light.corridor",
			"service":   "turn_on",
		},
	})
	require.NoError(t, cmdErr)

	time.Sleep(50 * time.Millisecond)

	client.mu.Lock()
	require.Len(t, client.sentFrames, 1)
	expectedCmdAddr, _ := knx.ParseGroupAddress("1/1/1")
	assert.Equal(t, expectedCmdAddr, client.sentFrames[0].Dst)
	assert.Equal(t, byte(0x01), client.sentFrames[0].Data[0])
	client.mu.Unlock()

	// 3. Stop adapter
	err = adapter.Stop(context.Background())
	require.NoError(t, err)
}
