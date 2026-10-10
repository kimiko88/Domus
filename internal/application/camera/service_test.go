package camera_test

import (
	"context"
	"testing"

	"github.com/domus-hub/domus/internal/application/camera"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCameraService_RegisterAndWebRTC(t *testing.T) {
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 64, Workers: 1})
	repo := camera.NewMemoryRepository()
	svc := camera.NewService(repo, bus)
	ctx := context.Background()

	cam, err := svc.RegisterCamera(ctx, "Ingresso Giardino", "rtsp://192.168.1.100:554/live", "/snapshots/cam1.jpg", nil)
	require.NoError(t, err)
	assert.NotEmpty(t, cam.ID)
	assert.Equal(t, "Ingresso Giardino", cam.Name)

	// WebRTC SDP negotiation
	session, err := svc.NegotiateWebRTC(ctx, cam.ID, "v=0\r\no=- 12345 2 IN IP4 127.0.0.1\r\ns=-\r\n")
	require.NoError(t, err)
	assert.Equal(t, cam.ID, session.CameraID)
	assert.Contains(t, session.AnswerSDP, "Ingresso Giardino")
}

func TestCameraService_FrigateEventDetection(t *testing.T) {
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 64, Workers: 1})
	repo := camera.NewMemoryRepository()
	svc := camera.NewService(repo, bus)
	ctx := context.Background()

	payload := []byte(`{
		"type": "end",
		"after": {
			"id": "1697000000.123-abc",
			"camera": "cam_front_yard",
			"label": "person",
			"top_score": 0.892
		}
	}`)

	evt, err := svc.HandleFrigateEvent(ctx, payload)
	require.NoError(t, err)
	assert.Equal(t, "cam_front_yard", evt.CameraID)
	assert.Equal(t, "person", evt.Label)
	assert.InDelta(t, 0.892, evt.Score, 0.001)

	eventsList, err := svc.ListEvents(ctx, "cam_front_yard", 10)
	require.NoError(t, err)
	assert.Len(t, eventsList, 1)
	assert.Equal(t, "person", eventsList[0].Label)
}
