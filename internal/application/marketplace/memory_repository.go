package marketplace

import (
    "context"
    "sync"
)

type MemoryRepository struct {
    mu sync.RWMutex
    sources map[string]*Source
    packages map[string]*Package
}

func NewMemoryRepository() *MemoryRepository {
    return &MemoryRepository{sources: make(map[string]*Source), packages: make(map[string]*Package)}
}

func (m *MemoryRepository) SaveSource(ctx context.Context, source *Source) error {
    m.mu.Lock(); defer m.mu.Unlock()
    copy := *source
    m.sources[source.ID] = &copy
    return nil
}
func (m *MemoryRepository) GetSource(ctx context.Context, id string) (*Source, error) {
    m.mu.RLock(); defer m.mu.RUnlock()
    source, ok := m.sources[id]
    if !ok { return nil, ErrNotFound }
    copy := *source
    return &copy, nil
}
func (m *MemoryRepository) ListSources(ctx context.Context) ([]*Source, error) {
    m.mu.RLock(); defer m.mu.RUnlock()
    result := make([]*Source, 0, len(m.sources))
    for _, value := range m.sources { copy := *value; result = append(result, &copy) }
    return result, nil
}
func (m *MemoryRepository) DeleteSource(ctx context.Context, id string) error {
    m.mu.Lock(); defer m.mu.Unlock()
    if _, ok := m.sources[id]; !ok { return ErrNotFound }
    delete(m.sources, id)
    for key, value := range m.packages { if value.RepositoryID == id { delete(m.packages, key) } }
    return nil
}
func (m *MemoryRepository) SavePackage(ctx context.Context, pkg *Package) error {
    m.mu.Lock(); defer m.mu.Unlock()
    copy := *pkg
    copy.Capabilities = append([]string(nil), pkg.Capabilities...)
    m.packages[pkg.ID] = &copy
    return nil
}
func (m *MemoryRepository) GetPackage(ctx context.Context, id string) (*Package, error) {
    m.mu.RLock(); defer m.mu.RUnlock()
    pkg, ok := m.packages[id]
    if !ok { return nil, ErrNotFound }
    copy := *pkg
    copy.Capabilities = append([]string(nil), pkg.Capabilities...)
    return &copy, nil
}
func (m *MemoryRepository) ListPackages(ctx context.Context, repositoryID string) ([]*Package, error) {
    m.mu.RLock(); defer m.mu.RUnlock()
    result := make([]*Package, 0, len(m.packages))
    for _, value := range m.packages {
        if repositoryID != "" && value.RepositoryID != repositoryID { continue }
        copy := *value
        copy.Capabilities = append([]string(nil), value.Capabilities...)
        result = append(result, &copy)
    }
    return result, nil
}
func (m *MemoryRepository) DeletePackage(ctx context.Context, id string) error {
    m.mu.Lock(); defer m.mu.Unlock()
    if _, ok := m.packages[id]; !ok { return ErrNotFound }
    delete(m.packages, id)
    return nil
}
