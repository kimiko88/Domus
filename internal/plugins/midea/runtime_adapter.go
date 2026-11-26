package midea

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/domus-hub/domus/internal/plugins"
)

const (
	// PluginType is the public, persistent Midea/Comfee LAN integration type.
	PluginType = "midea_smarthome_local"
	// IntegrationID preserves the historical ID used by devices and entities.
	IntegrationID = "integration_midea_smarthome"
	maxManagedDevices = 64
)

// RuntimeAdapter owns the protocol drivers and the credential-free device
// command adapter. Device credentials are supplied only by the encrypted
// Plugin Manager configuration and never exposed through the adapter API.
type RuntimeAdapter struct {
	id        string
	pluginType string
	core      *Adapter
	v2        *A1LANDriver
	v3        *A1LANDriver
	mu        sync.RWMutex
	active    bool
}

var _ plugins.Adapter = (*RuntimeAdapter)(nil)

// NewRuntimeAdapter constructs an empty Midea LAN runtime. It is primarily
// useful for tests and callers which need to prepare the transport before
// devices have been configured.
func NewRuntimeAdapter(id string) (*RuntimeAdapter, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, errors.New("midea: plugin instance ID is required")
	}
	v2, err := NewA1LANDriver(ProtocolMSmartV2)
	if err != nil {
		return nil, err
	}
	v3, err := NewA1LANDriver(ProtocolMSmartV3)
	if err != nil {
		return nil, err
	}
	return &RuntimeAdapter{
		id: id, pluginType: PluginType,
		core: NewAdapter(v2, v3), v2: v2, v3: v3,
	}, nil
}

// NewRuntimeAdapterWithDevices builds the persistent single-instance Midea
// integration and restores all configured V2/V3 A1 devices before Start.
func NewRuntimeAdapterWithDevices(id string, devices []A1LANConfig) (*RuntimeAdapter, error) {
	id = strings.TrimSpace(id)
	if id != IntegrationID {
		return nil, errors.New("midea: managed plugin requires the stable integration instance ID")
	}
	if len(devices) > maxManagedDevices {
		return nil, fmt.Errorf("midea: at most %d managed devices are allowed", maxManagedDevices)
	}
	adapter, err := NewRuntimeAdapter(id)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{}, len(devices))
	for _, stored := range devices {
		deviceID := strings.TrimSpace(stored.Device.ID)
		if deviceID == "" {
			return nil, errors.New("midea: stored device ID is required")
		}
		if _, exists := seen[deviceID]; exists {
			return nil, fmt.Errorf("midea: duplicate stored device ID %q", deviceID)
		}
		seen[deviceID] = struct{}{}

		var driver *A1LANDriver
		switch stored.Device.Protocol {
		case ProtocolMSmartV2:
			driver = adapter.v2
		case ProtocolMSmartV3:
			driver = adapter.v3
		default:
			return nil, fmt.Errorf("midea: unsupported stored protocol %q", stored.Device.Protocol)
		}
		if err := driver.Register(stored); err != nil {
			return nil, fmt.Errorf("midea: restore device %q: %w", deviceID, err)
		}
		restored, err := driver.config(deviceID)
		if err != nil {
			return nil, fmt.Errorf("midea: read restored device %q: %w", deviceID, err)
		}
		if err := adapter.core.Register(restored.Device); err != nil {
			return nil, fmt.Errorf("midea: register restored device %q: %w", deviceID, err)
		}
	}
	return adapter, nil
}

func (a *RuntimeAdapter) ID() string   { return a.id }
func (a *RuntimeAdapter) Type() string { return a.pluginType }

func (a *RuntimeAdapter) Start(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	a.mu.Lock()
	a.active = true
	a.mu.Unlock()
	return nil
}

func (a *RuntimeAdapter) Stop(context.Context) error {
	a.mu.Lock()
	a.active = false
	a.mu.Unlock()
	return nil
}

func (a *RuntimeAdapter) HealthCheck(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	a.mu.RLock()
	active := a.active
	a.mu.RUnlock()
	if !active {
		return errors.New("midea: plugin provider is stopped")
	}
	return nil
}

func (a *RuntimeAdapter) Core() (*Adapter, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if !a.active {
		return nil, errors.New("midea: plugin provider is stopped")
	}
	return a.core, nil
}

func (a *RuntimeAdapter) Drivers() (*A1LANDriver, *A1LANDriver) {
	return a.v2, a.v3
}
