package ota

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/google/uuid"
)

type UpgradeState string

const (
	StateIdle        UpgradeState = "idle"
	StateDownloading UpgradeState = "downloading"
	StateFlashing    UpgradeState = "flashing"
	StateSuccess     UpgradeState = "success"
	StateFailed      UpgradeState = "failed"
)

type DeviceOTA struct {
	ID             string       `json:"id"`
	Name           string       `json:"name"`
	DeviceKey      string       `json:"device_key"`
	Platform       string       `json:"platform"` // "zigbee", "esphome", "matter", "zwave"
	CurrentVersion string       `json:"current_version"`
	LatestVersion  string       `json:"latest_version"`
	HasUpdate      bool         `json:"has_update"`
	UpgradeState   UpgradeState `json:"upgrade_state"`
	Progress       int          `json:"progress"` // 0 to 100%
	ReleaseNotes   string       `json:"release_notes,omitempty"`
	CheckedAt      time.Time    `json:"checked_at"`
}

type Repository interface {
	SaveDevice(ctx context.Context, dev *DeviceOTA) error
	GetDevice(ctx context.Context, id string) (*DeviceOTA, error)
	ListDevices(ctx context.Context) ([]*DeviceOTA, error)
	DeleteDevice(ctx context.Context, id string) error
}

type MemoryRepository struct {
	mu      sync.RWMutex
	devices map[string]*DeviceOTA
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		devices: make(map[string]*DeviceOTA),
	}
}

func (m *MemoryRepository) SaveDevice(ctx context.Context, dev *DeviceOTA) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.devices[dev.ID] = dev
	return nil
}

func (m *MemoryRepository) GetDevice(ctx context.Context, id string) (*DeviceOTA, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	d, ok := m.devices[id]
	if !ok {
		return nil, fmt.Errorf("device not found: %s", id)
	}
	return d, nil
}

func (m *MemoryRepository) ListDevices(ctx context.Context) ([]*DeviceOTA, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	list := make([]*DeviceOTA, 0, len(m.devices))
	for _, d := range m.devices {
		list = append(list, d)
	}
	return list, nil
}

func (m *MemoryRepository) DeleteDevice(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.devices, id)
	return nil
}

type Service struct {
	repo Repository
	bus  events.Bus
}

func NewService(repo Repository, bus events.Bus) *Service {
	if repo == nil {
		repo = NewMemoryRepository()
	}
	return &Service{
		repo: repo,
		bus:  bus,
	}
}

func (s *Service) ListDevices(ctx context.Context) ([]*DeviceOTA, error) {
	return s.repo.ListDevices(ctx)
}

func (s *Service) TriggerUpgrade(ctx context.Context, id string) (*DeviceOTA, error) {
	dev, err := s.repo.GetDevice(ctx, id)
	if err != nil {
		return nil, err
	}
	if !dev.HasUpdate {
		return nil, fmt.Errorf("no update available for device %s", dev.Name)
	}

	dev.UpgradeState = StateFlashing
	dev.Progress = 100
	dev.CurrentVersion = dev.LatestVersion
	dev.HasUpdate = false

	if err := s.repo.SaveDevice(ctx, dev); err != nil {
		return nil, err
	}

	if s.bus != nil {
		_ = s.bus.Publish(ctx, events.Event{
			ID:         uuid.NewString(),
			Type:       events.OTAStateChanged,
			Payload:    dev,
			OccurredAt: time.Now().UTC(),
		})
	}

	return dev, nil
}
