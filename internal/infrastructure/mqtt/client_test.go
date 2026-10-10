package mqtt

import (
	"context"
	"crypto/tls"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPahoClient_ConfigInitialization(t *testing.T) {
	cfg := Config{
		BrokerURL:    "tcp://127.0.0.1:1883",
		ClientID:     "domus-test-client",
		Username:     "domus",
		Password:     "secretpass",
		CleanSession: true,
		KeepAlive:    15 * time.Second,
		TLSConfig:    &tls.Config{InsecureSkipVerify: true},
	}
	client := NewPahoClient(cfg)
	require.NotNil(t, client)
	assert.NotNil(t, client.client)
}

func TestMockClient_LifecycleAndPublish(t *testing.T) {
	mock := NewMockClient()
	require.True(t, mock.IsConnected())

	ctx := context.Background()

	// 1. Subscribe to a topic
	var receivedTopic string
	var receivedPayload []byte
	var wg sync.WaitGroup
	wg.Add(1)

	err := mock.Subscribe(ctx, "sensors/temp", 0, func(topic string, payload []byte) {
		receivedTopic = topic
		receivedPayload = payload
		wg.Done()
	})
	require.NoError(t, err)

	// 2. Publish a message
	payload := []byte("21.5")
	err = mock.Publish(ctx, "sensors/temp", 0, false, payload)
	require.NoError(t, err)

	wg.Wait()
	assert.Equal(t, "sensors/temp", receivedTopic)
	assert.Equal(t, "21.5", string(receivedPayload))

	// 3. Inspect published history
	pub := mock.GetPublished()
	require.Len(t, pub, 1)
	assert.Equal(t, "sensors/temp", pub[0].Topic)
	assert.Equal(t, payload, pub[0].Payload)

	// 4. Disconnect and publish should fail
	mock.Disconnect(0)
	assert.False(t, mock.IsConnected())

	err = mock.Publish(ctx, "sensors/temp", 0, false, []byte("22.0"))
	assert.ErrorIs(t, err, ErrNotConnected)

	// 5. Reconnect
	err = mock.Connect(ctx)
	require.NoError(t, err)
	assert.True(t, mock.IsConnected())
}

func TestMockClient_Unsubscribe(t *testing.T) {
	mock := NewMockClient()
	ctx := context.Background()

	var count int
	_ = mock.Subscribe(ctx, "lights/+", 0, func(topic string, payload []byte) {
		count++
	})

	mock.SimulateIncoming("lights/living_room", []byte("ON"))
	assert.Equal(t, 1, count)

	err := mock.Unsubscribe(ctx, "lights/+")
	require.NoError(t, err)

	mock.SimulateIncoming("lights/living_room", []byte("OFF"))
	assert.Equal(t, 1, count, "unsubscribed handler must not be triggered")
}

func TestMatchTopic(t *testing.T) {
	tests := []struct {
		pattern string
		topic   string
		matched bool
	}{
		{"#", "any/arbitrary/nested/topic", true},
		{"zigbee2mqtt/bridge/state", "zigbee2mqtt/bridge/state", true},
		{"zigbee2mqtt/bridge/state", "zigbee2mqtt/bridge/other", false},
		{"zigbee2mqtt/+/set", "zigbee2mqtt/light_bulb/set", true},
		{"zigbee2mqtt/+/set", "zigbee2mqtt/light_bulb/state", false},
		{"domus/room/#", "domus/room/kitchen/light", true},
		{"domus/room/#", "domus/other/kitchen/light", false},
		{"a/+/c/+", "a/b/c/d", true},
		{"a/+/c/+", "a/b/x/d", false},
		{"a/b", "a/b/extra", false},
	}

	for _, tc := range tests {
		t.Run(tc.pattern+"_vs_"+tc.topic, func(t *testing.T) {
			res := matchTopic(tc.pattern, tc.topic)
			assert.Equal(t, tc.matched, res)
		})
	}
}
