package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

type PackageType string

const (
	TypeWasmPlugin PackageType = "wasm_plugin"
	TypeBlueprint  PackageType = "blueprint"
	TypeWidget     PackageType = "widget"
)

type PackageTier string

const (
	TierVerified  PackageTier = "verified"
	TierCommunity PackageTier = "community"
)

type ReviewStatus string

const (
	ReviewStatusApproved ReviewStatus = "approved"
	ReviewStatusPending  ReviewStatus = "pending_review"
	ReviewStatusRejected ReviewStatus = "rejected"
)

type Package struct {
	ID            string       `json:"id"`
	PackageID     string       `json:"package_id"`
	Name          string       `json:"name"`
	Type          PackageType  `json:"type"`
	Tier          PackageTier  `json:"tier"`
	ReviewStatus  ReviewStatus `json:"review_status"`
	Version       string       `json:"version"`
	Author        string       `json:"author"`
	Description   string       `json:"description"`
	RepositoryURL string       `json:"repository_url"`
	License       string       `json:"license"`
	Capabilities  []string     `json:"capabilities,omitempty"`
	Installed     bool         `json:"installed"`
	InstalledAt   *time.Time   `json:"installed_at,omitempty"`
	DownloadURL   string       `json:"download_url,omitempty"`
	Icon          string       `json:"icon,omitempty"`
}

type SubmitPackageRequest struct {
	PackageID     string      `json:"package_id"`
	Name          string      `json:"name"`
	Type          PackageType `json:"type"`
	Version       string      `json:"version"`
	Author        string      `json:"author"`
	Description   string      `json:"description"`
	RepositoryURL string      `json:"repository_url"`
	License       string      `json:"license"`
	Capabilities  []string    `json:"capabilities"`
	Icon          string      `json:"icon,omitempty"`
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

func (s *Service) ListPackagesWithOptions(ctx context.Context, filterType, tier, status string) ([]*Package, error) {
	all, err := s.repo.ListPackages(ctx, filterType)
	if err != nil {
		return nil, err
	}
	filtered := make([]*Package, 0, len(all))
	for _, p := range all {
		if tier != "" && string(p.Tier) != tier {
			continue
		}
		if status != "" && string(p.ReviewStatus) != status {
			continue
		}
		filtered = append(filtered, p)
	}
	return filtered, nil
}

func (s *Service) SubmitCommunityPackage(ctx context.Context, req SubmitPackageRequest) (*Package, error) {
	if strings.TrimSpace(req.PackageID) == "" || strings.TrimSpace(req.Name) == "" {
		return nil, errors.New("package_id and name are required")
	}
	repoURL := strings.TrimSpace(req.RepositoryURL)
	if repoURL == "" || (!strings.HasPrefix(repoURL, "http://") && !strings.HasPrefix(repoURL, "https://")) {
		return nil, errors.New("valid open source repository_url (e.g. GitHub/GitLab) is required for community packages")
	}
	license := strings.TrimSpace(req.License)
	if license == "" {
		return nil, errors.New("open source license is required for community packages")
	}

	pType := req.Type
	if pType == "" {
		pType = TypeWasmPlugin
	}
	ver := strings.TrimSpace(req.Version)
	if ver == "" {
		ver = "1.0.0"
	}
	icon := req.Icon
	if icon == "" {
		icon = "box"
	}

	cleanID := "pkg_comm_" + strings.ReplaceAll(strings.ToLower(req.PackageID), ".", "_")
	pkg := &Package{
		ID:            cleanID,
		PackageID:     req.PackageID,
		Name:          req.Name,
		Type:          pType,
		Tier:          TierCommunity,
		ReviewStatus:  ReviewStatusPending,
		Version:       ver,
		Author:        req.Author,
		Description:   req.Description,
		RepositoryURL: repoURL,
		License:       license,
		Capabilities:  req.Capabilities,
		Installed:     false,
		Icon:          icon,
	}

	if err := s.repo.SavePackage(ctx, pkg); err != nil {
		return nil, err
	}
	return pkg, nil
}

func (s *Service) ApprovePackage(ctx context.Context, packageID string) (*Package, error) {
	p, err := s.repo.GetPackage(ctx, packageID)
	if err != nil {
		return nil, err
	}
	p.ReviewStatus = ReviewStatusApproved
	if err := s.repo.SavePackage(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) RejectPackage(ctx context.Context, packageID string, reason string) (*Package, error) {
	p, err := s.repo.GetPackage(ctx, packageID)
	if err != nil {
		return nil, err
	}
	p.ReviewStatus = ReviewStatusRejected
	if err := s.repo.SavePackage(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
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

// DefaultPackages returns curated, production-ready Domus Store extensions and WASI plugins.
func DefaultPackages() []*Package {
	return []*Package{
		// 1. BESS Arbitrage (WASI, Verified)
		{
			ID:            "pkg_bess_arbitrage",
			PackageID:     "domus.energy.bess-arbitrage",
			Name:          "BESS Dynamic Tariff & Energy Arbitrage Optimizer",
			Type:          TypeWasmPlugin,
			Tier:          TierVerified,
			ReviewStatus:  ReviewStatusApproved,
			Version:       "1.2.0",
			Author:        "Domus Energy Labs",
			Description:   "Modulo WebAssembly per l'ottimizzazione automatica dell'accumulo fotovoltaico con tariffe dinamiche (PUN/Nord Pool): forzatura carica nelle ore a prezzo negativo o ultra-basso e scarica intelligente nei picchi serali.",
			RepositoryURL: "https://github.com/domus-hub/extensions/tree/main/wasm-bess-optimizer",
			License:       "Apache-2.0",
			Capabilities:  []string{"energy:read", "energy:control", "tariffs:dynamic", "event_bus:subscribe"},
			Installed:     false,
			DownloadURL:   "https://store.domus-hub.org/plugins/bess-arbitrage-1.2.0.wasm",
			Icon:          "zap",
		},
		// 2. SunSpec Modbus RTU / RS485 (WASI, Verified)
		{
			ID:            "pkg_wasi_sunspec_modbus",
			PackageID:     "domus.wasi.sunspec-modbus",
			Name:          "SunSpec Modbus RTU / RS485 Inverter & Heat Pump Driver",
			Type:          TypeWasmPlugin,
			Tier:          TierVerified,
			ReviewStatus:  ReviewStatusApproved,
			Version:       "1.0.0",
			Author:        "Domus Energy Labs",
			Description:   "Driver sandboxed WASI per inverter solari e pompe di calore SunSpec Alliance (Fronius, SolarEdge, SMA, Huawei, Daikin) su linea seriale RS485 e Modbus TCP.",
			RepositoryURL: "https://github.com/domus-hub/extensions/tree/main/wasm-sunspec-modbus",
			License:       "Apache-2.0",
			Capabilities:  []string{"wasi:serial-port", "energy:read", "energy:control", "events:write"},
			Installed:     false,
			DownloadURL:   "https://store.domus-hub.org/plugins/sunspec-modbus-1.0.0.wasm",
			Icon:          "cpu",
		},
		// 3. Wireless M-Bus & OMS Smart Metering (WASI, Verified)
		{
			ID:            "pkg_wasi_wmbus_meters",
			PackageID:     "domus.wasi.wmbus-meters",
			Name:          "Wireless M-Bus & OMS Smart Metering (868 MHz)",
			Type:          TypeWasmPlugin,
			Tier:          TierVerified,
			ReviewStatus:  ReviewStatusApproved,
			Version:       "1.0.0",
			Author:        "Domus IoT Labs",
			Description:   "Decodifica radio crittografata EN 13757-4 OMS (Open Metering System) su 868 MHz per contatori smart di acqua, gas, energia termica e ripartitori di calore (Kamstrup, Diehl, B-Meters).",
			RepositoryURL: "https://github.com/domus-hub/extensions/tree/main/wasm-wmbus-meters",
			License:       "Apache-2.0",
			Capabilities:  []string{"wasi:serial-port", "crypto:aes128", "events:write"},
			Installed:     false,
			DownloadURL:   "https://store.domus-hub.org/plugins/wmbus-meters-1.0.0.wasm",
			Icon:          "radio",
		},
		// 4. Solar Irradiance Local Predictor (WASI, Verified)
		{
			ID:            "pkg_wasi_solar_irradiance",
			PackageID:     "domus.wasi.solar-irradiance",
			Name:          "Solar Irradiance & Clear-Sky Local Predictor",
			Type:          TypeWasmPlugin,
			Tier:          TierVerified,
			ReviewStatus:  ReviewStatusApproved,
			Version:       "1.0.0",
			Author:        "Domus Energy Labs",
			Description:   "Algoritmo fisico astronomico locale (Perez/PVLib) per la stima accurata della produzione fotovoltaica a 24-48 ore completamente offline senza dipendenze cloud.",
			RepositoryURL: "https://github.com/domus-hub/extensions/tree/main/wasm-solar-irradiance",
			License:       "Apache-2.0",
			Capabilities:  []string{"wasi:clock", "energy:read", "events:write"},
			Installed:     false,
			DownloadURL:   "https://store.domus-hub.org/plugins/solar-irradiance-1.0.0.wasm",
			Icon:          "sun",
		},
		// 5. Meshtastic LoRa Off-Grid (WASI, Community Approved)
		{
			ID:            "pkg_wasi_meshtastic_lora",
			PackageID:     "domus.wasi.meshtastic-lora",
			Name:          "Meshtastic & LoRa Off-Grid Emergency Beacon",
			Type:          TypeWasmPlugin,
			Tier:          TierCommunity,
			ReviewStatus:  ReviewStatusApproved,
			Version:       "1.0.0",
			Author:        "OffGrid Mesh Community",
			Description:   "Bridge radio LoRa (868/915 MHz) per reti mesh decentralizzate Meshtastic: invia allarmi critici e riceve comandi di emergenza anche in blackout elettrico e cellulare.",
			RepositoryURL: "https://github.com/domus-community/wasm-meshtastic-lora",
			License:       "GPL-3.0",
			Capabilities:  []string{"wasi:serial-port", "alarm:read", "events:write"},
			Installed:     false,
			DownloadURL:   "https://store.domus-hub.org/plugins/meshtastic-lora-1.0.0.wasm",
			Icon:          "share-2",
		},
		// 6. mmWave Radar DSP (WASI, Verified)
		{
			ID:            "pkg_wasi_mmwave_dsp",
			PackageID:     "domus.wasi.mmwave-radar-dsp",
			Name:          "mmWave Radar Point-Cloud DSP & Micro-Presence Filter",
			Type:          TypeWasmPlugin,
			Tier:          TierVerified,
			ReviewStatus:  ReviewStatusApproved,
			Version:       "1.0.0",
			Author:        "Domus Sensing Labs",
			Description:   "Elaborazione in tempo reale del segnale grezzo FMCW (sensori radar 24GHz/60GHz come HLK-LD2410, Seeed) per micro-presenza, rilevamento respiro e allarme caduta anziani.",
			RepositoryURL: "https://github.com/domus-hub/extensions/tree/main/wasm-mmwave-dsp",
			License:       "Apache-2.0",
			Capabilities:  []string{"wasi:serial-port", "presence:write", "events:write"},
			Installed:     false,
			DownloadURL:   "https://store.domus-hub.org/plugins/mmwave-dsp-1.0.0.wasm",
			Icon:          "activity",
		},
		// 7. Acoustic Event Classifier (WASI, Verified)
		{
			ID:            "pkg_wasi_acoustic_classifier",
			PackageID:     "domus.wasi.acoustic-classifier",
			Name:          "Local Acoustic Event & Emergency Siren Classifier",
			Type:          TypeWasmPlugin,
			Tier:          TierVerified,
			ReviewStatus:  ReviewStatusApproved,
			Version:       "1.0.0",
			Author:        "Domus AI Labs",
			Description:   "Classificatore acustico neurale WASI-NN locale per flussi audio RTSP e microfoni: riconosce sirene fumo/CO T3-T4, rottura cristalli e allarmi perdite idriche.",
			RepositoryURL: "https://github.com/domus-hub/extensions/tree/main/wasm-acoustic-classifier",
			License:       "Apache-2.0",
			Capabilities:  []string{"wasi:nn", "safety:trigger", "events:write"},
			Installed:     false,
			DownloadURL:   "https://store.domus-hub.org/plugins/acoustic-classifier-1.0.0.wasm",
			Icon:          "volume-2",
		},
		// 8. GSM/LTE AT-Command Modem & SMS Failover (WASI, Community Approved)
		{
			ID:            "pkg_wasi_gsm_sms_failover",
			PackageID:     "domus.wasi.gsm-sms-failover",
			Name:          "Local GSM/LTE AT-Command Modem & SMS Failover",
			Type:          TypeWasmPlugin,
			Tier:          TierCommunity,
			ReviewStatus:  ReviewStatusApproved,
			Version:       "1.0.0",
			Author:        "Domus Hardware Lab",
			Description:   "Controllo diretto via comandi AT di modem cellulari USB/mini-PCIe (SIMCom, Quectel) per invio SMS di emergenza e failover connettività quando la linea internet fissa è interrotta.",
			RepositoryURL: "https://github.com/domus-community/wasm-gsm-sms-failover",
			License:       "MIT",
			Capabilities:  []string{"wasi:serial-port", "notification:send", "network:failover"},
			Installed:     false,
			DownloadURL:   "https://store.domus-hub.org/plugins/gsm-sms-failover-1.0.0.wasm",
			Icon:          "smartphone",
		},
		// 9. Day-Ahead Dynamic Tariffs Engine (WASI, Verified)
		{
			ID:            "pkg_wasi_dynamic_tariffs",
			PackageID:     "domus.wasi.dynamic-tariffs",
			Name:          "Day-Ahead Dynamic Energy Tariffs Engine (PUN/Nord Pool/EPEX)",
			Type:          TypeWasmPlugin,
			Tier:          TierVerified,
			ReviewStatus:  ReviewStatusApproved,
			Version:       "1.0.0",
			Author:        "Domus Energy Labs",
			Description:   "Download, parsing e indicizzazione oraria dei prezzi dell'energia elettrica all'ingrosso (PUN GME italiano, Nord Pool, EPEX Spot) per alimentare le automazioni di accumulo e carichi prioritari.",
			RepositoryURL: "https://github.com/domus-hub/extensions/tree/main/wasm-dynamic-tariffs",
			License:       "Apache-2.0",
			Capabilities:  []string{"wasi:http:fetch", "energy:write", "events:write"},
			Installed:     false,
			DownloadURL:   "https://store.domus-hub.org/plugins/dynamic-tariffs-1.0.0.wasm",
			Icon:          "trending-down",
		},
		// 10. Coral EdgeTPU Vision (WASI, Verified)
		{
			ID:            "pkg_coral_edgetpu_ai",
			PackageID:     "domus.ai.coral-edgetpu-vision",
			Name:          "Coral EdgeTPU Local Neural Bridge",
			Type:          TypeWasmPlugin,
			Tier:          TierVerified,
			ReviewStatus:  ReviewStatusApproved,
			Version:       "1.1.0",
			Author:        "VisionEdge AI Labs",
			Description:   "Accelerazione neurale WASM sandboxed per telecamere RTSP/Frigate: inferenza locale a 100 FPS con filtraggio falsi allarmi e classificazione tempestiva di persone e veicoli.",
			RepositoryURL: "https://github.com/visionedge-ai/domus-coral-tpu",
			License:       "Apache-2.0",
			Capabilities:  []string{"camera:read", "wasi:nn", "events:write"},
			Installed:     false,
			DownloadURL:   "https://store.domus-hub.org/plugins/coral-edgetpu-vision-1.1.0.wasm",
			Icon:          "box",
		},
		// 11. Circadian Lighting (Blueprint, Verified)
		{
			ID:            "pkg_circadian_lighting",
			PackageID:     "domus.blueprint.circadian-lighting",
			Name:          "Illuminazione Circadiana Adattiva (Sun-Sync)",
			Type:          TypeBlueprint,
			Tier:          TierVerified,
			ReviewStatus:  ReviewStatusApproved,
			Version:       "2.1.0",
			Author:        "Domus Automation Team",
			Description:   "Blueprint per sincronizzare temperatura colore (2700K al crepuscolo, 5500K a mezzogiorno) e livello di dimmer delle luci in base alla posizione astronomica del sole e all'orario biologico.",
			RepositoryURL: "https://github.com/domus-hub/blueprints/tree/main/circadian-lighting",
			License:       "Apache-2.0",
			Installed:     false,
			DownloadURL:   "https://store.domus-hub.org/blueprints/circadian-lighting-2.1.0.yaml",
			Icon:          "sun",
		},
		// 12. Window Climate Guard (Blueprint, Community Approved)
		{
			ID:            "pkg_window_climate_guard",
			PackageID:     "domus.blueprint.window-climate-guard",
			Name:          "Guardia Clima & Infissi Intelligente",
			Type:          TypeBlueprint,
			Tier:          TierCommunity,
			ReviewStatus:  ReviewStatusApproved,
			Version:       "1.0.4",
			Author:        "GreenHome Community",
			Description:   "Automazione salvavita termico: spegne valvole termostatiche TRV e split pompa di calore dopo 90s di apertura finestre, ripristinando il setpoint di comfort alla chiusura.",
			RepositoryURL: "https://github.com/greenhome-community/domus-window-guard",
			License:       "MIT",
			Installed:     false,
			DownloadURL:   "https://store.domus-hub.org/blueprints/window-climate-guard-1.0.4.yaml",
			Icon:          "sun",
		},
		// 13. Power Flow Wheel (Widget, Verified)
		{
			ID:            "pkg_energy_flow_wheel",
			PackageID:     "domus.widget.energy-flow-wheel",
			Name:          "Power Flow Wheel & BESS Monitor",
			Type:          TypeWidget,
			Tier:          TierVerified,
			ReviewStatus:  ReviewStatusApproved,
			Version:       "1.4.2",
			Author:        "Domus UI Studio",
			Description:   "Widget SVG vettoriale animato per Dashboard: visualizzazione dinamica ad anelli delle correnti istantanee tra Solare, Batteria domestica, Rete elettrica e Consumi prioritari.",
			RepositoryURL: "https://github.com/domus-hub/widgets/tree/main/energy-flow-wheel",
			License:       "Apache-2.0",
			Installed:     false,
			DownloadURL:   "https://store.domus-hub.org/widgets/energy-flow-wheel-1.4.2.js",
			Icon:          "gauge",
		},
		// 14. Demo Community Pending Review Package
		{
			ID:            "pkg_comm_esp32_cam_ble",
			PackageID:     "domus.community.esp32-cam-ble",
			Name:          "ESP32-CAM BLE Proxy & Vision Streamer",
			Type:          TypeWasmPlugin,
			Tier:          TierCommunity,
			ReviewStatus:  ReviewStatusPending,
			Version:       "0.9.0",
			Author:        "OpenESP Community",
			Description:   "Plugin proposto dalla community per integrare moduli telecamera low-cost ESP32-CAM via Bluetooth LE e streaming locale compresso in WASI.",
			RepositoryURL: "https://github.com/domus-community/wasm-esp32-cam-ble",
			License:       "MIT",
			Capabilities:  []string{"wasi:sockets", "camera:read"},
			Installed:     false,
			Icon:          "camera",
		},
	}
}
