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

// DefaultPackages returns curated, production-ready Domus Store extensions.
func DefaultPackages() []*Package {
	return []*Package{
		{
			ID:          "pkg_bess_arbitrage",
			PackageID:   "domus.energy.bess-arbitrage",
			Name:        "BESS Dynamic Tariff & Energy Arbitrage Optimizer",
			Type:        TypeWasmPlugin,
			Version:     "1.2.0",
			Author:      "Domus Energy Labs",
			Description: "Modulo WebAssembly per l'ottimizzazione automatica dell'accumulo fotovoltaico con tariffe dinamiche (PUN/Nord Pool): forzatura carica nelle ore a prezzo negativo o ultra-basso e scarica intelligente nei picchi serali.",
			Installed:   false,
			DownloadURL: "https://store.domus-hub.org/plugins/bess-arbitrage-1.2.0.wasm",
			Icon:        "zap",
		},
		{
			ID:          "pkg_circadian_lighting",
			PackageID:   "domus.blueprint.circadian-lighting",
			Name:        "Illuminazione Circadiana Adattiva (Sun-Sync)",
			Type:        TypeBlueprint,
			Version:     "2.1.0",
			Author:      "Domus Automation Team",
			Description: "Blueprint per sincronizzare temperatura colore (2700K al crepuscolo, 5500K a mezzogiorno) e livello di dimmer delle luci in base alla posizione astronomica del sole e all'orario biologico.",
			Installed:   false,
			DownloadURL: "https://store.domus-hub.org/blueprints/circadian-lighting-2.1.0.yaml",
			Icon:        "sun",
		},
		{
			ID:          "pkg_window_climate_guard",
			PackageID:   "domus.blueprint.window-climate-guard",
			Name:        "Guardia Clima & Infissi Intelligente",
			Type:        TypeBlueprint,
			Version:     "1.0.4",
			Author:      "GreenHome Community",
			Description: "Automazione salvavita termico: spegne valvole termostatiche TRV e split pompa di calore dopo 90s di apertura finestre, ripristinando il setpoint di comfort alla chiusura.",
			Installed:   false,
			DownloadURL: "https://store.domus-hub.org/blueprints/window-climate-guard-1.0.4.yaml",
			Icon:        "sun",
		},
		{
			ID:          "pkg_energy_flow_wheel",
			PackageID:   "domus.widget.energy-flow-wheel",
			Name:        "Power Flow Wheel & BESS Monitor",
			Type:        TypeWidget,
			Version:     "1.4.2",
			Author:      "Domus UI Studio",
			Description: "Widget SVG vettoriale animato per Dashboard: visualizzazione dinamica ad anelli delle correnti istantanee tra Solare, Batteria domestica, Rete elettrica e Consumi prioritari.",
			Installed:   false,
			DownloadURL: "https://store.domus-hub.org/widgets/energy-flow-wheel-1.4.2.js",
			Icon:        "gauge",
		},
		{
			ID:          "pkg_coral_edgetpu_ai",
			PackageID:   "domus.ai.coral-edgetpu-vision",
			Name:        "Coral EdgeTPU Local Neural Bridge",
			Type:        TypeWasmPlugin,
			Version:     "1.1.0",
			Author:      "VisionEdge AI Labs",
			Description: "Accelerazione neurale WASM sandboxed per telecamere RTSP/Frigate: inferenza locale a 100 FPS con filtraggio falsi allarmi e classificazione tempestiva di persone e veicoli.",
			Installed:   false,
			DownloadURL: "https://store.domus-hub.org/plugins/coral-edgetpu-vision-1.1.0.wasm",
			Icon:        "box",
		},
	}
}
