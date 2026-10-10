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

type Service struct {
	mu       sync.RWMutex
	packages map[string]*Package
}

func NewService() *Service {
	s := &Service{
		packages: make(map[string]*Package),
	}

	// Curated Domus Store community packages
	s.packages["pkg_aqara_wasm"] = &Package{
		ID:          "pkg_aqara_wasm",
		PackageID:   "community.aqara.cube",
		Name:        "Aqara Magic Cube Controller",
		Type:        TypeWasmPlugin,
		Version:     "1.2.0",
		Author:      "DomusCommunity",
		Description: "Plugin WASM per interpretare rotazioni, tap e scuotimento del cubo Zigbee Aqara",
		Installed:   true,
		Icon:        "box",
	}

	s.packages["pkg_bp_circadian"] = &Package{
		ID:          "pkg_bp_circadian",
		PackageID:   "blueprints.lighting.circadian",
		Name:        "Illuminazione Circadiana Dinamica",
		Type:        TypeBlueprint,
		Version:     "2.0.1",
		Author:      "SmartHomeItaly",
		Description: "Regola temperatura colore e luminosità dei faretti in base all'elevazione solare reale",
		Installed:   false,
		Icon:        "sun",
	}

	s.packages["pkg_widget_gauges"] = &Package{
		ID:          "pkg_widget_gauges",
		PackageID:   "widgets.energy.flow_circle",
		Name:        "Widget Ghiera Consumi Circolare",
		Type:        TypeWidget,
		Version:     "1.0.4",
		Author:      "DomusLabs",
		Description: "Tessera dashboard con misuratore tachimetrico rotante per carichi elettrici istantanei",
		Installed:   false,
		Icon:        "gauge",
	}

	return s
}

func (s *Service) ListPackages(ctx context.Context, filterType string) ([]*Package, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]*Package, 0, len(s.packages))
	for _, p := range s.packages {
		if filterType == "" || string(p.Type) == filterType {
			list = append(list, p)
		}
	}
	return list, nil
}

func (s *Service) InstallPackage(ctx context.Context, packageID string) (*Package, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.packages[packageID]
	if !ok {
		return nil, fmt.Errorf("package not found: %s", packageID)
	}
	now := time.Now().UTC()
	p.Installed = true
	p.InstalledAt = &now
	return p, nil
}

func (s *Service) UninstallPackage(ctx context.Context, packageID string) (*Package, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.packages[packageID]
	if !ok {
		return nil, fmt.Errorf("package not found: %s", packageID)
	}
	p.Installed = false
	p.InstalledAt = nil
	return p, nil
}
