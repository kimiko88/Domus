package store

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type PackageType string

const (
	TypeWasmPlugin PackageType = "wasm_plugin"
	TypeBlueprint  PackageType = "blueprint"
	TypeWidget     PackageType = "widget"
)

type Package struct {
	ID          string      `json:"id"`
	PackageID   string      `json:"package_id"`
	Name        string      `json:"name"`
	Type        PackageType `json:"type"`
	Version     string      `json:"version"`
	Author      string      `json:"author"`
	Description string      `json:"description"`
	Installed   bool        `json:"installed"`
	InstalledAt *time.Time  `json:"installed_at,omitempty"`
	DownloadURL string      `json:"download_url,omitempty"`
	Icon        string      `json:"icon,omitempty"`
}

type Repository interface {
	SavePackage(ctx context.Context, pkg *Package) error
	GetPackage(ctx context.Context, id string) (*Package, error)
	ListPackages(ctx context.Context, filterType string) ([]*Package, error)
	DeletePackage(ctx context.Context, id string) error
}

type MemoryRepository struct {
	mu       sync.RWMutex
	packages map[string]*Package
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		packages: make(map[string]*Package),
	}
}

func (m *MemoryRepository) SavePackage(ctx context.Context, pkg *Package) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.packages[pkg.ID] = pkg
	return nil
}

func (m *MemoryRepository) GetPackage(ctx context.Context, id string) (*Package, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.packages[id]
	if !ok {
		for _, pkg := range m.packages {
			if pkg.PackageID == id {
				return pkg, nil
			}
		}
		return nil, fmt.Errorf("package not found: %s", id)
	}
	return p, nil
}

func (m *MemoryRepository) ListPackages(ctx context.Context, filterType string) ([]*Package, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	list := make([]*Package, 0, len(m.packages))
	for _, p := range m.packages {
		if filterType == "" || string(p.Type) == filterType {
			list = append(list, p)
		}
	}
	return list, nil
}

func (m *MemoryRepository) DeletePackage(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.packages, id)
	return nil
}

type Service struct {
	repo Repository
}

func NewService(repo ...Repository) *Service {
	var r Repository
	if len(repo) > 0 && repo[0] != nil {
		r = repo[0]
	} else {
		r = NewMemoryRepository()
	}
	return &Service{repo: r}
}

func (s *Service) ListPackages(ctx context.Context, filterType string) ([]*Package, error) {
	return s.repo.ListPackages(ctx, filterType)
}

func (s *Service) InstallPackage(ctx context.Context, packageID string) (*Package, error) {
	p, err := s.repo.GetPackage(ctx, packageID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	p.Installed = true
	p.InstalledAt = &now
	if err := s.repo.SavePackage(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) UninstallPackage(ctx context.Context, packageID string) (*Package, error) {
	p, err := s.repo.GetPackage(ctx, packageID)
	if err != nil {
		return nil, err
	}
	p.Installed = false
	p.InstalledAt = nil
	if err := s.repo.SavePackage(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}
