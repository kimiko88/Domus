// Package sqlite provides SQLite persistence and cached repository decorators.
package sqlite

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/entity"
)

// CachedEntityRepository decorates an entity.Repository with an in-memory, thread-safe cache
// to deliver sub-50-microsecond reads and alleviate SQLite query load under high telemetry.
type CachedEntityRepository struct {
	base      entity.Repository
	mu        sync.RWMutex
	byID      map[string]*entity.Entity
	byKey     map[string]*entity.Entity
	populated bool
}

// NewCachedEntityRepository creates a new CachedEntityRepository.
func NewCachedEntityRepository(base entity.Repository) *CachedEntityRepository {
	return &CachedEntityRepository{
		base:  base,
		byID:  make(map[string]*entity.Entity),
		byKey: make(map[string]*entity.Entity),
	}
}

// WarmUp preloads all entities from SQLite into memory.
func (c *CachedEntityRepository) WarmUp(ctx context.Context) error {
	entities, err := c.base.List(ctx, entity.ListFilter{})
	if err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.byID = make(map[string]*entity.Entity, len(entities))
	c.byKey = make(map[string]*entity.Entity, len(entities))

	for _, e := range entities {
		cloned := cloneEntity(e)
		c.byID[cloned.ID] = cloned
		c.byKey[cloned.EntityKey] = cloned
	}
	c.populated = true
	return nil
}

// Create persists the entity to SQLite and adds it to the cache.
func (c *CachedEntityRepository) Create(ctx context.Context, e *entity.Entity) error {
	if err := c.base.Create(ctx, e); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	cloned := cloneEntity(e)
	c.byID[cloned.ID] = cloned
	c.byKey[cloned.EntityKey] = cloned
	return nil
}

// GetByID returns the entity by ID from memory cache, falling back to SQLite if absent.
func (c *CachedEntityRepository) GetByID(ctx context.Context, id string) (*entity.Entity, error) {
	c.mu.RLock()
	if c.populated {
		if e, ok := c.byID[id]; ok {
			res := cloneEntity(e)
			c.mu.RUnlock()
			return res, nil
		}
	}
	c.mu.RUnlock()

	e, err := c.base.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	cloned := cloneEntity(e)
	c.byID[cloned.ID] = cloned
	c.byKey[cloned.EntityKey] = cloned
	c.mu.Unlock()

	return cloneEntity(e), nil
}

// GetByKey returns the entity by entity_key from memory cache, falling back to SQLite if absent.
func (c *CachedEntityRepository) GetByKey(ctx context.Context, key string) (*entity.Entity, error) {
	c.mu.RLock()
	if c.populated {
		if e, ok := c.byKey[key]; ok {
			res := cloneEntity(e)
			c.mu.RUnlock()
			return res, nil
		}
	}
	c.mu.RUnlock()

	e, err := c.base.GetByKey(ctx, key)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	cloned := cloneEntity(e)
	c.byID[cloned.ID] = cloned
	c.byKey[cloned.EntityKey] = cloned
	c.mu.Unlock()

	return cloneEntity(e), nil
}

// List returns filtered entities. If the cache is warm, filtering is done directly in memory.
func (c *CachedEntityRepository) List(ctx context.Context, filter entity.ListFilter) ([]*entity.Entity, error) {
	c.mu.RLock()
	if !c.populated {
		c.mu.RUnlock()
		return c.base.List(ctx, filter)
	}

	var results []*entity.Entity
	for _, e := range c.byID {
		if filter.Domain != "" && !strings.EqualFold(e.Domain, filter.Domain) {
			continue
		}
		if filter.AreaID != "" && e.AreaID != filter.AreaID {
			continue
		}
		if filter.DeviceID != "" && e.DeviceID != filter.DeviceID {
			continue
		}
		if filter.Writable != nil && e.Writable != *filter.Writable {
			continue
		}
		results = append(results, cloneEntity(e))
	}
	c.mu.RUnlock()

	return results, nil
}

// Update persists updates to SQLite and updates the cache.
func (c *CachedEntityRepository) Update(ctx context.Context, e *entity.Entity) error {
	if err := c.base.Update(ctx, e); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	cloned := cloneEntity(e)
	c.byID[cloned.ID] = cloned
	c.byKey[cloned.EntityKey] = cloned
	return nil
}

// UpdateState writes state updates to SQLite and updates the cached entity fields.
func (c *CachedEntityRepository) UpdateState(ctx context.Context, id string, stateJSON, attributesJSON string, available bool, at time.Time) error {
	if err := c.base.UpdateState(ctx, id, stateJSON, attributesJSON, available, at); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.byID[id]; ok {
		if e.State != stateJSON {
			e.LastChangedAt = at
		}
		e.State = stateJSON
		e.Attributes = attributesJSON
		e.Available = available
		e.LastUpdatedAt = at
	}
	return nil
}

// Delete removes an entity from SQLite and the in-memory cache.
func (c *CachedEntityRepository) Delete(ctx context.Context, id string) error {
	if err := c.base.Delete(ctx, id); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.byID[id]; ok {
		delete(c.byKey, e.EntityKey)
		delete(c.byID, id)
	}
	return nil
}

func cloneEntity(e *entity.Entity) *entity.Entity {
	if e == nil {
		return nil
	}
	cloned := *e
	return &cloned
}
