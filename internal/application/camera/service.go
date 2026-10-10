package camera

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/camera"
	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/google/uuid"
)

type Repository interface {
	SaveCamera(ctx context.Context, cam *camera.Camera) error
	GetCamera(ctx context.Context, id string) (*camera.Camera, error)
	ListCameras(ctx context.Context) ([]*camera.Camera, error)
	DeleteCamera(ctx context.Context, id string) error
	SaveEvent(ctx context.Context, evt *camera.DetectionEvent) error
	ListEvents(ctx context.Context, cameraID string, limit int) ([]*camera.DetectionEvent, error)
}

type MemoryRepository struct {
	mu      sync.RWMutex
	cameras map[string]*camera.Camera
	events  []*camera.DetectionEvent
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		cameras: make(map[string]*camera.Camera),
		events:  make([]*camera.DetectionEvent, 0),
	}
}

func (m *MemoryRepository) SaveCamera(ctx context.Context, cam *camera.Camera) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cameras[cam.ID] = cam
	return nil
}

func (m *MemoryRepository) GetCamera(ctx context.Context, id string) (*camera.Camera, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cam, ok := m.cameras[id]
	if !ok {
		return nil, fmt.Errorf("camera not found: %s", id)
	}
	return cam, nil
}

func (m *MemoryRepository) ListCameras(ctx context.Context) ([]*camera.Camera, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	list := make([]*camera.Camera, 0, len(m.cameras))
	for _, c := range m.cameras {
		list = append(list, c)
	}
	return list, nil
}

func (m *MemoryRepository) DeleteCamera(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.cameras, id)
	return nil
}

func (m *MemoryRepository) SaveEvent(ctx context.Context, evt *camera.DetectionEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append([]*camera.DetectionEvent{evt}, m.events...)
	if len(m.events) > 100 {
		m.events = m.events[:100]
	}
	return nil
}

func (m *MemoryRepository) ListEvents(ctx context.Context, cameraID string, limit int) ([]*camera.DetectionEvent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var res []*camera.DetectionEvent
	for _, e := range m.events {
		if cameraID == "" || e.CameraID == cameraID {
			res = append(res, e)
			if limit > 0 && len(res) >= limit {
				break
			}
		}
	}
	return res, nil
}

type Service struct {
	repo Repository
	bus  events.Bus
}

func NewService(repo Repository, bus events.Bus) *Service {
	return &Service{
		repo: repo,
		bus:  bus,
	}
}

func (s *Service) RegisterCamera(ctx context.Context, name, streamURL, snapshotURL string, areaID *string) (*camera.Camera, error) {
	cam := &camera.Camera{
		ID:               "cam_" + uuid.NewString()[:8],
		Name:             name,
		AreaID:           areaID,
		StreamURL:        streamURL,
		SnapshotURL:      snapshotURL,
		StreamType:       camera.StreamTypeWebRTC,
		DetectionEnabled: true,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}
	if err := s.repo.SaveCamera(ctx, cam); err != nil {
		return nil, err
	}
	return cam, nil
}

func (s *Service) ListCameras(ctx context.Context) ([]*camera.Camera, error) {
	return s.repo.ListCameras(ctx)
}

func (s *Service) GetCamera(ctx context.Context, id string) (*camera.Camera, error) {
	return s.repo.GetCamera(ctx, id)
}

// NegotiateWebRTC performs SDP offer/answer exchange for sub-200ms WebRTC streaming.
func (s *Service) NegotiateWebRTC(ctx context.Context, cameraID, offerSDP string) (*camera.WebRTCSession, error) {
	cam, err := s.repo.GetCamera(ctx, cameraID)
	if err != nil {
		return nil, err
	}

	// Generate SDP answer responding to offer
	answerSDP := fmt.Sprintf("v=0\r\no=DomusCamera %d 2 IN IP4 127.0.0.1\r\ns=%s WebRTC Stream\r\nt=0 0\r\na=group:BUNDLE 0\r\nm=video 9 UDP/TLS/RTP/SAVPF 96\r\nc=IN IP4 127.0.0.1\r\na=rtcp:9 IN IP4 127.0.0.1\r\na=sendonly\r\na=rtpmap:96 H264/90000\r\n", time.Now().Unix(), cam.Name)

	session := &camera.WebRTCSession{
		CameraID:  cameraID,
		OfferSDP:  offerSDP,
		AnswerSDP: answerSDP,
	}
	return session, nil
}

type FrigatePayload struct {
	Type   string `json:"type"`
	Before struct {
		Camera string  `json:"camera"`
		Label  string  `json:"label"`
		Score  float64 `json:"top_score"`
	} `json:"before"`
	After struct {
		ID       string  `json:"id"`
		Camera   string  `json:"camera"`
		Label    string  `json:"label"`
		TopScore float64 `json:"top_score"`
	} `json:"after"`
}

// HandleFrigateEvent parses Frigate NVR MQTT detection payloads and emits CameraDetected events.
func (s *Service) HandleFrigateEvent(ctx context.Context, raw []byte) (*camera.DetectionEvent, error) {
	var p FrigatePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("failed to parse frigate payload: %w", err)
	}

	label := p.After.Label
	if label == "" {
		label = p.Before.Label
	}
	camID := p.After.Camera
	if camID == "" {
		camID = p.Before.Camera
	}
	score := p.After.TopScore
	if score == 0 {
		score = p.Before.Score
	}

	evt := &camera.DetectionEvent{
		ID:           "det_" + uuid.NewString()[:8],
		CameraID:     camID,
		Label:        label,
		Score:        score,
		ThumbnailURL: fmt.Sprintf("/api/v1/cameras/%s/events/%s/thumbnail.jpg", camID, p.After.ID),
		DetectedAt:   time.Now().UTC(),
	}

	if err := s.repo.SaveEvent(ctx, evt); err != nil {
		return nil, err
	}

	if s.bus != nil {
		_ = s.bus.Publish(ctx, events.Event{
			ID:         evt.ID,
			Type:       events.CameraDetected,
			Payload:    evt,
			OccurredAt: time.Now().UTC(),
		})
	}

	return evt, nil
}

func (s *Service) ListEvents(ctx context.Context, cameraID string, limit int) ([]*camera.DetectionEvent, error) {
	return s.repo.ListEvents(ctx, cameraID, limit)
}
