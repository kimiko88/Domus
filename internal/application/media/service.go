package media

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/domain/media"
	"github.com/google/uuid"
)

type Repository interface {
	SavePlayer(ctx context.Context, p *media.MediaPlayer) error
	GetPlayer(ctx context.Context, id string) (*media.MediaPlayer, error)
	ListPlayers(ctx context.Context) ([]*media.MediaPlayer, error)
	DeletePlayer(ctx context.Context, id string) error
}

type MemoryRepository struct {
	mu      sync.RWMutex
	players map[string]*media.MediaPlayer
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		players: make(map[string]*media.MediaPlayer),
	}
}

func (m *MemoryRepository) SavePlayer(ctx context.Context, p *media.MediaPlayer) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.players[p.ID] = p
	return nil
}

func (m *MemoryRepository) GetPlayer(ctx context.Context, id string) (*media.MediaPlayer, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.players[id]
	if !ok {
		return nil, fmt.Errorf("media player not found: %s", id)
	}
	return p, nil
}

func (m *MemoryRepository) ListPlayers(ctx context.Context) ([]*media.MediaPlayer, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	list := make([]*media.MediaPlayer, 0, len(m.players))
	for _, p := range m.players {
		list = append(list, p)
	}
	return list, nil
}

func (m *MemoryRepository) DeletePlayer(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.players, id)
	return nil
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

func (s *Service) RegisterPlayer(ctx context.Context, name string, platform media.Platform, areaID *string) (*media.MediaPlayer, error) {
	p := &media.MediaPlayer{
		ID:        "media_" + uuid.NewString()[:8],
		Name:      name,
		Platform:  platform,
		AreaID:    areaID,
		State:     media.StateIdle,
		Volume:    0.5,
		Muted:     false,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.repo.SavePlayer(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) ListPlayers(ctx context.Context) ([]*media.MediaPlayer, error) {
	return s.repo.ListPlayers(ctx)
}

func (s *Service) DeletePlayer(ctx context.Context, id string) error {
	return s.repo.DeletePlayer(ctx, id)
}

func (s *Service) SetState(ctx context.Context, id string, state media.PlayerState) (*media.MediaPlayer, error) {
	p, err := s.repo.GetPlayer(ctx, id)
	if err != nil {
		return nil, err
	}
	p.State = state
	if err := s.repo.SavePlayer(ctx, p); err != nil {
		return nil, err
	}
	s.publishChange(ctx, p)
	return p, nil
}

func (s *Service) SetVolume(ctx context.Context, id string, volume float64) (*media.MediaPlayer, error) {
	p, err := s.repo.GetPlayer(ctx, id)
	if err != nil {
		return nil, err
	}
	if volume < 0 {
		volume = 0
	}
	if volume > 1.0 {
		volume = 1.0
	}
	p.Volume = volume
	if err := s.repo.SavePlayer(ctx, p); err != nil {
		return nil, err
	}
	s.publishChange(ctx, p)
	return p, nil
}

func (s *Service) BroadcastAnnouncement(ctx context.Context, announcement *media.BroadcastAnnouncement) (int, error) {
	players, err := s.repo.ListPlayers(ctx)
	if err != nil {
		return 0, err
	}

	targetMap := make(map[string]bool)
	for _, id := range announcement.TargetPlayerIDs {
		targetMap[id] = true
	}

	notified := 0
	for _, p := range players {
		if len(announcement.TargetPlayerIDs) > 0 && !targetMap[p.ID] {
			continue
		}
		// In a real device this pushes audio URL and sets volume
		p.CurrentTitle = announcement.Message
		_ = s.repo.SavePlayer(ctx, p)
		s.publishChange(ctx, p)
		notified++
	}

	return notified, nil
}

func (s *Service) publishChange(ctx context.Context, p *media.MediaPlayer) {
	if s.bus != nil {
		_ = s.bus.Publish(ctx, events.Event{
			ID:         uuid.NewString(),
			Type:       events.MediaStateChanged,
			Payload:    p,
			OccurredAt: time.Now().UTC(),
		})
	}
}
