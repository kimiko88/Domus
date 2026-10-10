package camera

import (
	"context"
	"testing"

	"github.com/domus-hub/domus/internal/domain/events"
)

type mockCameraBus struct {
	published []events.Event
}

func (m *mockCameraBus) Publish(ctx context.Context, evt events.Event) error {
	m.published = append(m.published, evt)
	return nil
}
func (m *mockCameraBus) Subscribe(t events.EventType, h events.Handler) func() { return func() {} }
func (m *mockCameraBus) SubscribeAll(h events.Handler) func()                  { return func() {} }
func (m *mockCameraBus) Close() error                                          { return nil }

func TestLocalNVR_StatusAndDetection(t *testing.T) {
	repo := NewMemoryRepository()
	bus := &mockCameraBus{}
	svc := NewService(repo, bus)
	nvr := NewLocalNVR(svc, bus)

	ctx := context.Background()
	cam, err := svc.RegisterCamera(ctx, "Ingresso Principale", "rtsp://192.168.1.50/live", "", nil)
	if err != nil {
		t.Fatalf("RegisterCamera failed: %v", err)
	}

	// 1. Check NVR Status
	status := nvr.GetStatus(ctx)
	if status.TotalCameras != 1 {
		t.Errorf("expected 1 camera in NVR status, got %d", status.TotalCameras)
	}

	// 2. Add Motion Zone
	zone := MotionZone{
		CameraID:    cam.ID,
		Name:        "Portone",
		Sensitivity: 90,
		Labels:      []string{"person", "package"},
		Enabled:     true,
	}
	nvr.AddZone(zone)

	zones := nvr.GetZones(cam.ID)
	if len(zones) != 1 || zones[0].Name != "Portone" {
		t.Errorf("expected Portone zone, got %+v", zones)
	}

	// 3. Process AI Detection
	det := AIDetection{
		Label:      "person",
		Confidence: 0.94,
		Box: BoundingBox{
			X:      0.25,
			Y:      0.30,
			Width:  0.20,
			Height: 0.50,
		},
		Zone: "Portone",
	}

	evt, err := nvr.ProcessDetection(ctx, cam.ID, det)
	if err != nil {
		t.Fatalf("ProcessDetection failed: %v", err)
	}

	if evt.Label != "person" || evt.Score != 0.94 {
		t.Errorf("unexpected event data: %+v", evt)
	}

	// Verify event bus publication
	if len(bus.published) == 0 {
		t.Errorf("expected event published on bus")
	}

	// 4. Get Event Clip
	clip, err := nvr.GetEventClip(evt.ID)
	if err != nil {
		t.Fatalf("GetEventClip failed: %v", err)
	}
	if clip.CameraID != cam.ID {
		t.Errorf("expected camera ID %s in clip, got %s", cam.ID, clip.CameraID)
	}
}
