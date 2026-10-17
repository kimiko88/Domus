package shopping

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
	domainShop "github.com/domus-hub/domus/internal/domain/shopping"
	"github.com/google/uuid"
)

type Repository interface {
	SaveItem(ctx context.Context, item *domainShop.Item) error
	GetItem(ctx context.Context, id string) (*domainShop.Item, error)
	ListItems(ctx context.Context) ([]*domainShop.Item, error)
	DeleteItem(ctx context.Context, id string) error
	ClearCompleted(ctx context.Context) (int, error)
}

type MemoryRepository struct {
	mu    sync.RWMutex
	items map[string]*domainShop.Item
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		items: make(map[string]*domainShop.Item),
	}
}

func (m *MemoryRepository) SaveItem(ctx context.Context, item *domainShop.Item) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[item.ID] = item
	return nil
}

func (m *MemoryRepository) GetItem(ctx context.Context, id string) (*domainShop.Item, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	it, ok := m.items[id]
	if !ok {
		return nil, fmt.Errorf("shopping item not found: %s", id)
	}
	return it, nil
}

func (m *MemoryRepository) ListItems(ctx context.Context) ([]*domainShop.Item, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	list := make([]*domainShop.Item, 0, len(m.items))
	for _, it := range m.items {
		list = append(list, it)
	}
	return list, nil
}

func (m *MemoryRepository) DeleteItem(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.items, id)
	return nil
}

func (m *MemoryRepository) ClearCompleted(ctx context.Context) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	count := 0
	for k, it := range m.items {
		if it.Completed {
			delete(m.items, k)
			count++
		}
	}
	return count, nil
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

func (s *Service) AddItem(ctx context.Context, name, quantity string, category domainShop.Category, addedBy string) (*domainShop.Item, error) {
	return s.AddItemWithStore(ctx, name, quantity, category, "", addedBy)
}

func (s *Service) AddItemWithStore(ctx context.Context, name, quantity string, category domainShop.Category, supermarket, addedBy string) (*domainShop.Item, error) {
	if category == "" {
		category = domainShop.CategoryGrocery
	}
	if addedBy == "" {
		addedBy = "Famiglia"
	}

	it := &domainShop.Item{
		ID:          "shop_" + uuid.NewString()[:8],
		Name:        name,
		Quantity:    quantity,
		Category:    category,
		Supermarket: supermarket,
		Completed:   false,
		AddedBy:     addedBy,
		CreatedAt:   time.Now().UTC(),
	}

	if err := s.repo.SaveItem(ctx, it); err != nil {
		return nil, err
	}

	s.publish(ctx, "shopping.item_added", it)
	return it, nil
}

func (s *Service) ToggleItem(ctx context.Context, id, completedBy string) (*domainShop.Item, error) {
	it, err := s.repo.GetItem(ctx, id)
	if err != nil {
		return nil, err
	}

	it.Completed = !it.Completed
	now := time.Now().UTC()
	if it.Completed {
		it.CompletedAt = &now
		if completedBy != "" {
			it.CompletedBy = &completedBy
		}
	} else {
		it.CompletedAt = nil
		it.CompletedBy = nil
	}

	if err := s.repo.SaveItem(ctx, it); err != nil {
		return nil, err
	}

	s.publish(ctx, "shopping.item_toggled", it)
	return it, nil
}

func (s *Service) DeleteItem(ctx context.Context, id string) error {
	if err := s.repo.DeleteItem(ctx, id); err != nil {
		return err
	}
	s.publish(ctx, "shopping.item_deleted", map[string]string{"id": id})
	return nil
}

func (s *Service) ClearCompleted(ctx context.Context) (int, error) {
	count, err := s.repo.ClearCompleted(ctx)
	if err != nil {
		return 0, err
	}
	s.publish(ctx, "shopping.cleared_completed", map[string]int{"count": count})
	return count, nil
}

func (s *Service) ListItems(ctx context.Context) ([]*domainShop.Item, error) {
	return s.repo.ListItems(ctx)
}

func (s *Service) publish(ctx context.Context, eventType events.EventType, payload any) {
	if s.bus != nil {
		_ = s.bus.Publish(ctx, events.Event{
			ID:         uuid.NewString(),
			Type:       eventType,
			Payload:    payload,
			OccurredAt: time.Now().UTC(),
		})
	}
}
