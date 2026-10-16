package modbus_tcp_test

import (
	"context"
	"encoding/binary"
	"errors"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/domain/entity"
	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/domus-hub/domus/internal/plugins/modbus_tcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockModbusClient struct {
	mu        sync.Mutex
	connected bool
	txCounter uint16
	responses map[uint16][]byte
	lastWrite []byte
}

func newMockModbusClient() *mockModbusClient {
	return &mockModbusClient{
		connected: true,
		responses: make(map[uint16][]byte),
	}
}

func (m *mockModbusClient) NextTxID() uint16 {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.txCounter++
	return m.txCounter
}

func (m *mockModbusClient) Connect(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = true
	return nil
}

func (m *mockModbusClient) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = false
	return nil
}

func (m *mockModbusClient) IsConnected() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.connected
}

func (m *mockModbusClient) RoundTrip(ctx context.Context, req []byte) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.connected {
		return nil, errors.New("disconnected")
	}

	m.lastWrite = req
	if len(req) < 10 {
		return nil, errors.New("request too short")
	}

	funcCode := req[7]
	address := binary.BigEndian.Uint16(req[8:10])

	// If predefined response exists for this address
	if resp, ok := m.responses[address]; ok {
		return resp, nil
	}

	// Default responses based on function code
	switch funcCode {
	case modbus_tcp.FuncReadHoldingRegisters, modbus_tcp.FuncReadInputRegisters:
		// Return 4 bytes payload with float32 5000.0
		val := math.Float32bits(5000.0)
		buf := make([]byte, 6)
		buf[0] = funcCode
		buf[1] = 4 // 4 bytes data
		binary.BigEndian.PutUint32(buf[2:6], val)
		return buf, nil

	case modbus_tcp.FuncReadCoils:
		// Return 1 byte coil ON
		return []byte{modbus_tcp.FuncReadCoils, 0x01, 0x01}, nil

	case modbus_tcp.FuncWriteSingleCoil, modbus_tcp.FuncWriteSingleRegister:
		// Echo request PDU as response
		return req[7:], nil
	}

	return nil, errors.New("unhandled mock modbus function")
}

type mockEntityRepo struct {
	mu     sync.Mutex
	items  map[string]*entity.Entity
	states map[string]string
}

func newMockEntityRepo() *mockEntityRepo {
	return &mockEntityRepo{
		items:  make(map[string]*entity.Entity),
		states: make(map[string]string),
	}
}

func (r *mockEntityRepo) Create(ctx context.Context, e *entity.Entity) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[e.ID] = e
	return nil
}

func (r *mockEntityRepo) GetByID(ctx context.Context, id string) (*entity.Entity, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if it, ok := r.items[id]; ok {
		return it, nil
	}
	return nil, entity.ErrEntityNotFound
}

func (r *mockEntityRepo) GetByKey(ctx context.Context, key string) (*entity.Entity, error) {
	return nil, entity.ErrEntityNotFound
}

func (r *mockEntityRepo) List(ctx context.Context, filter entity.ListFilter) ([]*entity.Entity, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	res := make([]*entity.Entity, 0, len(r.items))
	for _, it := range r.items {
		res = append(res, it)
	}
	return res, nil
}

func (r *mockEntityRepo) Update(ctx context.Context, e *entity.Entity) error { return nil }

func (r *mockEntityRepo) UpdateState(ctx context.Context, id string, stateJSON, attributesJSON string, available bool, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.states[id] = stateJSON
	return nil
}

func (r *mockEntityRepo) Delete(ctx context.Context, id string) error { return nil }

func TestModbusTCP_AdapterLifecycleAndPolling(t *testing.T) {
	client := newMockModbusClient()
	repo := newMockEntityRepo()
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 64, Workers: 1})

	cfg := modbus_tcp.Config{
		IntegrationID: "inverter_huawei",
		Host:          "192.168.1.150",
		Port:          502,
		UnitID:        1,
		PollInterval:  50 * time.Millisecond,
		Registers: []modbus_tcp.RegisterMapping{
			{
				EntityKey:    "sensor.inverter_power",
				Name:         "Inverter Power",
				FunctionCode: modbus_tcp.FuncReadHoldingRegisters,
				Address:      32064,
				DataType:     modbus_tcp.TypeFloat32,
				WordOrder:    modbus_tcp.BigEndian,
				Scale:        1.0,
				Unit:         "W",
				Writable:     false,
			},
			{
				EntityKey:    "switch.heat_pump_boost",
				Name:         "Heat Pump Boost",
				FunctionCode: modbus_tcp.FuncReadCoils,
				Address:      100,
				DataType:     modbus_tcp.TypeBool,
				Scale:        1.0,
				Writable:     true,
			},
		},
	}

	adapter := modbus_tcp.NewAdapter(cfg, client, repo, bus, nil)
	assert.Equal(t, "inverter_huawei", adapter.ID())
	assert.Equal(t, "modbus_tcp", adapter.Type())

	require.NoError(t, adapter.HealthCheck(context.Background()))

	// Channel to catch state changed event
	evChan := make(chan events.Event, 10)
	bus.Subscribe(events.EntityStateChanged, func(ctx context.Context, ev events.Event) error {
		evChan <- ev
		return nil
	})

	err := adapter.Start(context.Background())
	require.NoError(t, err)

	// Wait for poller to emit events
	select {
	case ev := <-evChan:
		assert.Equal(t, "modbus_tcp", ev.Source)
		pMap, ok := ev.Payload.(map[string]any)
		require.True(t, ok)
		assert.NotEmpty(t, pMap["entity_id"])
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for modbus polling state event")
	}

	// Test outbound command to turn off switch
	err = bus.Publish(context.Background(), events.Event{
		ID:         "cmd-1",
		Type:       events.CommandRequested,
		OccurredAt: time.Now().UTC(),
		Payload: map[string]any{
			"entity_id": "switch.heat_pump_boost",
			"service":   "turn_off",
		},
	})
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	// Stop adapter
	err = adapter.Stop(context.Background())
	require.NoError(t, err)
}
