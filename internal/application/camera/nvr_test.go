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

func TestLocalNVR_FaceRecognitionAndALPR(t *testing.T) {
	repo := NewMemoryRepository()
	bus := &mockCameraBus{}
	svc := NewService(repo, bus)
	nvr := NewLocalNVR(svc, bus)
	ctx := context.Background()

	cam, err := svc.RegisterCamera(ctx, "Cancello Carrabile", "rtsp://192.168.1.51/live", "", nil)
	if err != nil {
		t.Fatalf("RegisterCamera failed: %v", err)
	}

	// 1. Enroll Family Member Face
	marcoEmbedding := []float32{0.5, 0.5, 0.5, 0.5}
	nvr.RegisterFamilyFace(FamilyFace{
		ID:        "face_marco",
		Name:      "Marco Rossi",
		Embedding: marcoEmbedding,
	})

	// Match positive: identical or close embedding
	matchedName, score := nvr.MatchFace([]float32{0.51, 0.49, 0.50, 0.50})
	if matchedName != "Marco Rossi" || score < 0.65 {
		t.Errorf("expected Marco Rossi with score >= 0.65, got name=%s, score=%f", matchedName, score)
	}

	// Match negative: orthogonal embedding
	unmatchedName, unmatchScore := nvr.MatchFace([]float32{-0.5, -0.5, 0.5, 0.5})
	if unmatchedName != "" {
		t.Errorf("expected no face match, got %s (score=%f)", unmatchedName, unmatchScore)
	}

	// 2. Enroll Authorized License Plate
	nvr.RegisterAuthorizedPlate(AuthorizedVehicle{
		PlateNumber:  "ab 123 cd", // lower case and spaces
		OwnerName:    "Marco Rossi",
		AutoOpenGate: true,
	})

	// Plate match with whitespace/case tolerance
	veh, ok := nvr.MatchPlate("AB123CD")
	if !ok || veh.OwnerName != "Marco Rossi" {
		t.Errorf("expected authorized plate match for AB123CD, got ok=%v, veh=%+v", ok, veh)
	}

	// Unknown plate
	_, unknownOk := nvr.MatchPlate("ZZ999ZZ")
	if unknownOk {
		t.Errorf("expected unknown plate to return false")
	}

	// 3. Process Detection with Face & Plate
	det := AIDetection{
		Label:        "vehicle",
		Confidence:   0.98,
		Embedding:    marcoEmbedding,
		LicensePlate: "ab123cd",
		Zone:         "Ingresso",
	}

	evt, err := nvr.ProcessDetection(ctx, cam.ID, det)
	if err != nil {
		t.Fatalf("ProcessDetection failed: %v", err)
	}

	if evt == nil {
		t.Fatalf("expected non-nil event")
	}

	// Verify last event on bus
	if len(bus.published) == 0 {
		t.Fatalf("no events published on bus")
	}
	lastEvent := bus.published[len(bus.published)-1]
	payload, ok := lastEvent.Payload.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any payload, got %T", lastEvent.Payload)
	}

	if payload["person_name"] != "Marco Rossi" {
		t.Errorf("expected person_name Marco Rossi, got %v", payload["person_name"])
	}
	if payload["face_recognized"] != true {
		t.Errorf("expected face_recognized true, got %v", payload["face_recognized"])
	}
	if payload["plate_authorized"] != true {
		t.Errorf("expected plate_authorized true, got %v", payload["plate_authorized"])
	}
}
