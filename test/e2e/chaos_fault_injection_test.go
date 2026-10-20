package e2e_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/application/safety"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/domus-hub/domus/internal/interfaces/http/middleware"
	"github.com/domus-hub/domus/internal/plugins"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockChaosAdapter implements plugins.Adapter with simulated connection flapping and packet drops.
type mockChaosAdapter struct {
	mu           sync.Mutex
	id           string
	adapterType  string
	failChecks   int
	checksCount  int
}

func (m *mockChaosAdapter) ID() string   { return m.id }
func (m *mockChaosAdapter) Type() string { return m.adapterType }
func (m *mockChaosAdapter) Start(ctx context.Context) error { return nil }
func (m *mockChaosAdapter) Stop(ctx context.Context) error  { return nil }

func (m *mockChaosAdapter) HealthCheck(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checksCount++
	if m.checksCount <= m.failChecks {
		return errors.New("hardware bus connection timeout: device did not respond")
	}
	return nil
}

func TestChaos_Supervisor_HardwareFailureStorm(t *testing.T) {
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 64, Workers: 2})
	defer bus.Close()

	sup := plugins.NewSupervisor(bus)

	adapter := &mockChaosAdapter{
		id:          "modbus_inverter_01",
		adapterType: "modbus_tcp",
		failChecks:  3, // will fail first 3 health checks then recover
	}

	policy := plugins.BreakerPolicy{
		MaxFailures:         2, // trip after 2 failures
		InitialBackoff:      30 * time.Millisecond,
		MaxBackoff:          150 * time.Millisecond,
		HealthCheckInterval: 20 * time.Millisecond,
	}

	err := sup.Register(adapter, policy)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = sup.Start(ctx)
	require.NoError(t, err)
	defer func() { _ = sup.Stop(context.Background()) }()

	// Wait for probes to execute and trip the circuit breaker
	require.Eventually(t, func() bool {
		st, ok := sup.GetStatus("modbus_inverter_01")
		return ok && (st.State == plugins.StateTripped || st.State == plugins.StateDegraded)
	}, 1*time.Second, 20*time.Millisecond, "supervisor must trip or degrade adapter under failure storm")

	// Wait for adapter to recover (after failChecks == 3)
	require.Eventually(t, func() bool {
		st, ok := sup.GetStatus("modbus_inverter_01")
		return ok && st.State == plugins.StateHealthy
	}, 2*time.Second, 40*time.Millisecond, "supervisor must automatically recover adapter to Healthy once heartbeats succeed")
}

func TestChaos_Idempotency_ConcurrentStorm(t *testing.T) {
	store := middleware.NewIdempotencyStore(2 * time.Second)
	defer store.Close()

	var executionCounter int64

	handler := middleware.Idempotency(store)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate non-trivial work
		time.Sleep(10 * time.Millisecond)
		val := atomic.AddInt64(&executionCounter, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "applied",
			"exec":   val,
		})
	}))

	// Execute single initial request to populate cache
	key := "chaos-idempotency-key-001"
	reqInitial := httptest.NewRequest(http.MethodPost, "/api/v1/entities/light.salotto/command", nil)
	reqInitial.Header.Set("X-Idempotency-Key", key)
	recInitial := httptest.NewRecorder()
	handler.ServeHTTP(recInitial, reqInitial)
	require.Equal(t, http.StatusOK, recInitial.Code)
	assert.Equal(t, int64(1), atomic.LoadInt64(&executionCounter))

	// Storm: 50 concurrent goroutines with the same key
	const concurrency = 50
	var wg sync.WaitGroup
	wg.Add(concurrency)

	successCount := int64(0)
	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/entities/light.salotto/command", nil)
			req.Header.Set("X-Idempotency-Key", key)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code == http.StatusOK {
				atomic.AddInt64(&successCount, 1)
			}
		}()
	}

	wg.Wait()

	// Every request must succeed
	assert.Equal(t, int64(concurrency), successCount)
	// Execution counter MUST remain exactly 1!
	assert.Equal(t, int64(1), atomic.LoadInt64(&executionCounter), "backend handler must execute only once despite concurrent storm")
}

type mockValveActuator struct {
	closedCount int64
	openedCount int64
}

func (m *mockValveActuator) CloseValve(ctx context.Context, valveID string) error {
	atomic.AddInt64(&m.closedCount, 1)
	return nil
}

func (m *mockValveActuator) OpenValve(ctx context.Context, valveID string) error {
	atomic.AddInt64(&m.openedCount, 1)
	return nil
}

func TestChaos_SafeStop_ConcurrentEmergencyTriggers(t *testing.T) {
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 128, Workers: 4})
	defer bus.Close()

	svc := safety.NewService(bus)
	defer svc.Stop()

	waterActuator := &mockValveActuator{}
	gasActuator := &mockValveActuator{}

	_ = svc.RegisterValve(safety.ValveRegistration{
		ID:        "valve.main_water",
		Type:      safety.ValveWater,
		Name:      "Elettrovalvola Principale Acqua",
		Actuator:  waterActuator,
		TimeoutMs: 50,
	})

	_ = svc.RegisterValve(safety.ValveRegistration{
		ID:        "valve.main_gas",
		Type:      safety.ValveGas,
		Name:      "Elettrovalvola Principale Metano",
		Actuator:  gasActuator,
		TimeoutMs: 50,
	})

	// 50 concurrent water leak emergency alerts
	const numEvents = 50
	var wg sync.WaitGroup
	wg.Add(numEvents)

	ctx := context.Background()
	for i := 0; i < numEvents; i++ {
		go func(idx int) {
			defer wg.Done()
			alert := safety.EmergencyAlert{
				Hazard:       safety.HazardWaterLeak,
				SourceSensor: fmt.Sprintf("sensor.leak_%d", idx),
				Severity:     "critical",
				OccurredAt:   time.Now().UTC(),
			}
			_, _ = svc.TriggerSafeStop(ctx, alert)
		}(i)
	}

	wg.Wait()

	// Pipeline is latched: water valve must have been closed, gas valve untouched
	status := svc.Status()
	assert.True(t, status.IsLatched)
	assert.NotNil(t, status.ActiveAlert)
	assert.Equal(t, safety.HazardWaterLeak, status.ActiveAlert.Hazard)
	assert.True(t, atomic.LoadInt64(&waterActuator.closedCount) >= 1)
	assert.Equal(t, int64(0), atomic.LoadInt64(&gasActuator.closedCount), "gas valve must not close for water hazard")
}
