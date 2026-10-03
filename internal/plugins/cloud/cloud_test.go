// Package cloud_test tests cloud providers and the resilience adapter.
package cloud_test

import (
	"context"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/domus-hub/domus/internal/plugins/cloud"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCloudProviders_InterfaceAndStubs(t *testing.T) {
	providers := []cloud.Provider{
		cloud.NewTuyaProvider("key", "secret", "eu"),
		cloud.NewXiaomiProvider("token", "server_eu"),
		cloud.NewEwelinkProvider("email", "password", "eu"),
		cloud.NewVesyncProvider("account", "password"),
	}

	expectedNames := []string{"tuya", "xiaomi", "ewelink", "vesync"}

	for i, p := range providers {
		assert.Equal(t, expectedNames[i], p.Name())
		require.NoError(t, p.Ping(context.Background()))

		err := p.ExecuteCommand(context.Background(), "dev_123", "turn_on", map[string]any{"brightness": 100})
		require.NoError(t, err)
	}
}

func TestCloudAdapter_ResilienceAndDegradation(t *testing.T) {
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 32, Workers: 1})
	defer bus.Close()

	mockProvider := cloud.NewMockProvider("mock_cloud")

	adapter := cloud.NewAdapter(cloud.Config{
		IntegrationID:    "cloud_tuya",
		Provider:         mockProvider,
		FailureThreshold: 2,
		ResetTimeout:     100 * time.Millisecond,
	}, bus)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	require.NoError(t, adapter.Start(ctx))
	defer adapter.Stop(ctx)

	degradedChan := make(chan events.Event, 5)
	unsub := bus.Subscribe(events.IntegrationStateChanged, func(c context.Context, e events.Event) error {
		degradedChan <- e
		return nil
	})
	defer unsub()

	// Healthy state
	require.NoError(t, adapter.HealthCheck(ctx))
	err := adapter.Execute(ctx, "dev_001", "turn_on", nil)
	require.NoError(t, err)

	// Simulate cloud provider going offline
	mockProvider.SetOnline(false)

	// First failure
	err1 := adapter.Execute(ctx, "dev_001", "turn_on", nil)
	assert.Error(t, err1)

	// Second failure trips circuit breaker
	err2 := adapter.Execute(ctx, "dev_001", "turn_on", nil)
	assert.Error(t, err2)

	select {
	case evt := <-degradedChan:
		payload := evt.Payload.(map[string]any)
		assert.Equal(t, "degraded", payload["status"])
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for degraded event")
	}

	// Breaker is now open: calls immediately return ErrCircuitBreakerOpen without blocking
	start := time.Now()
	err3 := adapter.Execute(ctx, "dev_001", "turn_on", nil)
	elapsed := time.Since(start)

	assert.ErrorIs(t, err3, cloud.ErrCircuitBreakerOpen)
	assert.Less(t, elapsed, 10*time.Millisecond, "Open circuit breaker must fail fast without waiting")

	// Health check reflects degraded state
	healthErr := adapter.HealthCheck(ctx)
	assert.ErrorIs(t, healthErr, cloud.ErrCircuitBreakerOpen)

	// Bring cloud back online and wait for reset timeout
	mockProvider.SetOnline(true)
	time.Sleep(120 * time.Millisecond)

	// Breaker resets upon successful execution
	errRecover := adapter.Execute(ctx, "dev_001", "turn_on", nil)
	require.NoError(t, errRecover)
	assert.Nil(t, adapter.HealthCheck(ctx))
}
