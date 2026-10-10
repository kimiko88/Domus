package tag

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/domain/tag"
	"github.com/google/uuid"
)

type Repository interface {
	SaveTag(ctx context.Context, t *tag.Tag) error
	GetTagByTagID(ctx context.Context, tagID string) (*tag.Tag, error)
	ListTags(ctx context.Context) ([]*tag.Tag, error)
	DeleteTag(ctx context.Context, id string) error
}

type MemoryRepository struct {
	mu   sync.RWMutex
	tags map[string]*tag.Tag
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		tags: make(map[string]*tag.Tag),
	}
}

func (m *MemoryRepository) SaveTag(ctx context.Context, t *tag.Tag) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tags[t.TagID] = t
	return nil
}

func (m *MemoryRepository) GetTagByTagID(ctx context.Context, tagID string) (*tag.Tag, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.tags[tagID]
	if !ok {
		return nil, fmt.Errorf("tag not found: %s", tagID)
	}
	return t, nil
}

func (m *MemoryRepository) ListTags(ctx context.Context) ([]*tag.Tag, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	list := make([]*tag.Tag, 0, len(m.tags))
	for _, t := range m.tags {
		list = append(list, t)
	}
	return list, nil
}

func (m *MemoryRepository) DeleteTag(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, v := range m.tags {
		if v.ID == id {
			delete(m.tags, k)
			break
		}
	}
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

func (s *Service) RegisterTag(ctx context.Context, name, tagID string, actionType tag.ActionType, targetID string) (*tag.Tag, error) {
	t := &tag.Tag{
		ID:             "tag_" + uuid.NewString()[:8],
		Name:           name,
		TagID:          tagID,
		ActionType:     actionType,
		ActionTargetID: targetID,
		CreatedAt:      time.Now().UTC(),
	}
	if err := s.repo.SaveTag(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *Service) ListTags(ctx context.Context) ([]*tag.Tag, error) {
	return s.repo.ListTags(ctx)
}

type ScanResult struct {
	Tag            *tag.Tag `json:"tag"`
	ActionExecuted bool     `json:"action_executed"`
	ScannedBy      string   `json:"scanned_by"`
}

func (s *Service) ScanTag(ctx context.Context, tagID, scannedBy string) (*ScanResult, error) {
	t, err := s.repo.GetTagByTagID(ctx, tagID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	t.LastScannedAt = &now
	_ = s.repo.SaveTag(ctx, t)

	res := &ScanResult{
		Tag:            t,
		ActionExecuted: true,
		ScannedBy:      scannedBy,
	}

	if s.bus != nil {
		_ = s.bus.Publish(ctx, events.Event{
			ID:         uuid.NewString(),
			Type:       events.TagScanned,
			Payload:    res,
			OccurredAt: now,
			Source:     scannedBy,
		})
	}

	return res, nil
}
