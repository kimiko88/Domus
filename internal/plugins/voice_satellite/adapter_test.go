package voice_satellite

import (
	"bytes"
	"context"
	"encoding/binary"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewAdapterValidatesSecuritySensitiveConfiguration(t *testing.T) {
	base := Config{
		SatelliteID: "sat_kitchen",
		ServerURL: "https://domus.example.ts.net:8080",
		BearerToken: "session-secret",
		WakeAddress: "127.0.0.1:10400",
	}
	adapter, err := NewAdapter(base, nil)
	require.NoError(t, err)
	require.Equal(t, "sat_kitchen", adapter.ID())
	require.Equal(t, "voice_satellite", adapter.Type())
	require.NotContains(t, adapter.endpoint, "session-secret")

	base.ServerURL = "javascript:alert(1)"
	_, err = NewAdapter(base, nil)
	require.Error(t, err)
	base.ServerURL = "https://domus.example.ts.net:8080"
	base.BearerToken = ""
	_, err = NewAdapter(base, nil)
	require.Error(t, err)
	base.BearerToken = "token\r\nX-Evil: yes"
	_, err = NewAdapter(base, nil)
	require.Error(t, err)
}

func TestCaptureCommandStopsAfterSpeechAndTrailingSilence(t *testing.T) {
	adapter, err := NewAdapter(Config{
		SatelliteID: "sat_test", ServerURL: "http://127.0.0.1:8080",
		BearerToken: "test-token", WakeAddress: "127.0.0.1:10400",
		FrameDuration: 100*time.Millisecond, SilenceDuration: 300*time.Millisecond,
		MaxCommandDuration: 2*time.Second, SpeechThreshold: 0.01,
	}, nil)
	require.NoError(t, err)

	const frameSize = 3200
	audio := make([]byte, frameSize*8)
	for frame := 0; frame < 2; frame++ {
		for offset := frame*frameSize; offset < (frame+1)*frameSize; offset += 2 {
			binary.LittleEndian.PutUint16(audio[offset:offset+2], uint16(int16(12000)))
		}
	}
	pcm, speech, err := adapter.captureCommand(context.Background(), bytes.NewReader(audio), frameSize)
	require.NoError(t, err)
	require.True(t, speech)
	require.NotEmpty(t, pcm)
	require.LessOrEqual(t, len(pcm), frameSize*5) // 2 speech frames + 3 silence frames
}

func TestCaptureCommandReturnsEmptyForSilence(t *testing.T) {
	adapter, err := NewAdapter(Config{
		SatelliteID: "sat_test", ServerURL: "http://127.0.0.1:8080",
		BearerToken: "test-token", WakeAddress: "127.0.0.1:10400",
		FrameDuration: 100*time.Millisecond, MaxCommandDuration: 2*time.Second,
	}, nil)
	require.NoError(t, err)
	pcm, speech, err := adapter.captureCommand(context.Background(), bytes.NewReader(make([]byte, 3200*40)), 3200)
	require.NoError(t, err)
	require.False(t, speech)
	require.Nil(t, pcm)
}
