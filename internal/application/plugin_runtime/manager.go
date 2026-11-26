// Package plugin_runtime owns plugin instance configuration, persistence and lifecycle.
package plugin_runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/domain/integration"
	"github.com/domus-hub/domus/internal/infrastructure/crypto"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	"github.com/domus-hub/domus/internal/plugins"
)

var (
	ErrSingleInstance = errors.New("plugin_runtime: this plugin type supports only one instance")
	ErrInternalPlugin = errors.New("plugin_runtime: internal transport plugins cannot be configured as user instances")
)

var integrationIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)

type Manager struct {
	repo integration.Repository
	cipher *crypto.EnvelopeCipher
	registry *plugins.Registry
	supervisor *plugins.Supervisor
	deps plugins.Dependencies
	mu sync.Mutex
	instanceMu sync.Mutex
	prepared map[string]plugins.Adapter
	started bool
}

type Instance struct {
	ID string `json:"id"`
	Type string `json:"type"`
	Name string `json:"name"`
	Enabled bool `json:"enabled"`
	Status string `json:"status"`
	LastError string `json:"last_error,omitempty"`
}

func NewManager(repos *sqlite.Repositories, bus events.Bus) (*Manager, error) {
	if repos == nil || repos.Integrations == nil {
		return nil, errors.New("plugin_runtime: integration repository is required")
	}
	if repos.Cipher == nil {
		return nil, errors.New("plugin_runtime: encrypted configuration storage is required")
	}
	registry := plugins.NewRegistry()
	deps := plugins.Dependencies{Bus: bus, Devices: repos.Devices, Entities: repos.Entities}
	if err := registerBuiltins(registry, builtinHost{repos: repos}); err != nil {
		return nil, fmt.Errorf("plugin_runtime: register built-ins: %w", err)
	}
	return &Manager{
		repo: repos.Integrations, cipher: repos.Cipher, registry: registry,
		supervisor: plugins.NewSupervisor(bus), deps: deps, prepared: make(map[string]plugins.Adapter),
	}, nil
}

func (m *Manager) Registry() *plugins.Registry { return m.registry }

// GetConfig returns the decrypted configuration for trusted application-layer
// consumers. It must never be wired to an HTTP response: public instance APIs
// intentionally expose metadata only and keep every secret masked.
func (m *Manager) GetConfig(ctx context.Context, id string) (json.RawMessage, error) {
	if !integrationIDPattern.MatchString(id) {
		return nil, errors.New("plugin_runtime: invalid integration instance ID")
	}
	record, err := m.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	raw, err := m.decryptConfig(record.ConfigEncrypted)
	if err != nil {
		return nil, fmt.Errorf("plugin_runtime: decrypt integration config: %w", err)
	}
	return append(json.RawMessage(nil), raw...), nil
}

// GetAdapter returns a prepared adapter by instance ID. The returned value is
// owned by the manager; callers may use its typed provider contract but must
// not call Start or Stop directly.
func (m *Manager) GetAdapter(instanceID string) (plugins.Adapter, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	adapter, ok := m.prepared[instanceID]
	return adapter, ok
}

// GetAdapterByType returns a prepared adapter by its plugin type. This is useful
// for compatibility handlers whose legacy configuration does not persist the
// runtime instance ID. It is intended for singleton provider types such as
// Tuya Cloud; callers must not use it to select among multi-instance adapters.
func (m *Manager) GetAdapterByType(pluginType string) (plugins.Adapter, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, adapter := range m.prepared {
		if adapter != nil && adapter.Type() == pluginType {
			return adapter, true
		}
	}
	return nil, false
}

// ListTypes returns only user-configurable plugin types. Internal transport
// providers remain registered for Prepare and Manager.Start, but are hidden
// from the user-facing integration catalog.
func (m *Manager) ListTypes() []plugins.Manifest {
	all := m.registry.List()
	visible := make([]plugins.Manifest, 0, len(all))
	for _, manifest := range all {
		if manifest.Internal {
			continue
		}
		publicConfig := make([]plugins.ConfigField, 0, len(manifest.Config))
		for _, field := range manifest.Config {
			if !field.Internal {
				publicConfig = append(publicConfig, field)
			}
		}
		manifest.Config = publicConfig
		visible = append(visible, manifest)
	}
	return visible
}

func (m *Manager) isInternalType(pluginType string) bool {
	for _, manifest := range m.registry.List() {
		if manifest.ID == pluginType {
			return manifest.Internal
		}
	}
	return false
}

func (m *Manager) Prepare(ctx context.Context, pluginType, instanceID string, raw json.RawMessage) (plugins.Adapter, error) {
	m.instanceMu.Lock()
	defer m.instanceMu.Unlock()
	return m.prepareLocked(ctx, pluginType, instanceID, raw)
}

// prepareLocked is called only while instanceMu is held. It keeps all manager
// mutations ordered with startup, enable/disable, reconfiguration and deletion.
func (m *Manager) prepareLocked(ctx context.Context, pluginType, instanceID string, raw json.RawMessage) (plugins.Adapter, error) {
	if !integrationIDPattern.MatchString(instanceID) {
		return nil, errors.New("plugin_runtime: invalid integration instance ID")
	}
	m.mu.Lock()
	if _, exists := m.prepared[instanceID]; exists {
		m.mu.Unlock()
		return nil, plugins.ErrAdapterExists
	}
	started := m.started
	m.mu.Unlock()
	if err := m.validateInstanceMultiplicity(ctx, pluginType, instanceID); err != nil {
		return nil, err
	}
	adapter, err := m.registry.Create(ctx, pluginType, instanceID, raw, m.deps)
	if err != nil {
		return nil, err
	}
	if err := m.supervisor.Register(adapter); err != nil {
		_ = adapter.Stop(context.Background())
		return nil, err
	}
	m.mu.Lock()
	m.prepared[instanceID] = adapter
	m.mu.Unlock()
	if started {
		if err := m.supervisor.StartAdapter(ctx, instanceID); err != nil {
			stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			stopErr := m.supervisor.Unregister(stopCtx, instanceID)
			cancel()
			if stopErr == nil {
				m.mu.Lock()
				delete(m.prepared, instanceID)
				m.mu.Unlock()
			}
			return nil, errors.Join(err, stopErr)
		}
	}
	return adapter, nil
}

// Start prepares enabled persisted instances and starts all prepared adapters.
func (m *Manager) Start(ctx context.Context) error {
	m.instanceMu.Lock()
	defer m.instanceMu.Unlock()

	m.mu.Lock()
	if m.started {
		m.mu.Unlock()
		return nil
	}
	m.mu.Unlock()

	records, err := m.repo.List(ctx)
	if err != nil { return fmt.Errorf("plugin_runtime: list integrations: %w", err) }
	for _, record := range records {
		if record == nil || !record.Enabled {
			continue
		}
		m.mu.Lock()
		_, alreadyPrepared := m.prepared[record.ID]
		m.mu.Unlock()
		if alreadyPrepared { continue }
		// Integrations not registered as protocol plugins remain owned by their
		// domain-specific application service and are intentionally left alone.
		if !m.hasType(record.Type) { continue }
		raw, err := m.decryptConfig(record.ConfigEncrypted)
		if err != nil {
			_ = m.repo.SetStatus(ctx, record.ID, "error", "stored plugin configuration could not be decrypted")
			continue
		}
		if _, err := m.prepareLocked(ctx, record.Type, record.ID, raw); err != nil {
			_ = m.repo.SetStatus(ctx, record.ID, "error", safeError(err))
		}
	}
	if err := m.supervisor.Start(ctx); err != nil { return err }
	m.mu.Lock()
	m.started = true
	m.mu.Unlock()
	m.syncStatuses(ctx)
	return nil
}

func (m *Manager) validateInstanceMultiplicity(ctx context.Context, pluginType, instanceID string) error {
	var multiInstance bool
	foundType := false
	for _, manifest := range m.registry.List() {
		if manifest.ID == pluginType {
			foundType = true
			multiInstance = manifest.MultiInstance
			break
		}
	}
	if !foundType || multiInstance {
		return nil
	}
	records, err := m.repo.List(ctx)
	if err != nil {
		return fmt.Errorf("plugin_runtime: check existing plugin instances: %w", err)
	}
	for _, record := range records {
		if record != nil && record.Type == pluginType && record.ID != instanceID {
			return ErrSingleInstance
		}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, adapter := range m.prepared {
		if id != instanceID && adapter != nil && adapter.Type() == pluginType {
			return ErrSingleInstance
		}
	}
	return nil
}

func (m *Manager) hasType(id string) bool {
	for _, manifest := range m.registry.List() {
		if manifest.ID == id { return true }
	}
	return false
}

func (m *Manager) decryptConfig(raw []byte) (json.RawMessage, error) {
	if len(raw) == 0 { return json.RawMessage(`{}`), nil }
	plain, err := m.cipher.Decrypt(string(raw))
	if err != nil { return nil, err }
	if !json.Valid([]byte(plain)) { return nil, errors.New("stored plugin config is not valid JSON") }
	return json.RawMessage(plain), nil
}

func (m *Manager) Create(ctx context.Context, id, pluginType, name string, config json.RawMessage, enabled bool) (Instance, error) {
	m.instanceMu.Lock()
	defer m.instanceMu.Unlock()
	id, name = strings.TrimSpace(id), strings.TrimSpace(name)
	if !integrationIDPattern.MatchString(id) || name == "" || len(name) > 128 {
		return Instance{}, errors.New("plugin_runtime: integration ID or name is invalid")
	}
	if len(config) == 0 { config = json.RawMessage(`{}`) }
	if m.isInternalType(pluginType) {
		return Instance{}, ErrInternalPlugin
	}
	if err := m.validateInstanceMultiplicity(ctx, pluginType, id); err != nil { return Instance{}, err }
	adapter, err := m.registry.Create(ctx, pluginType, id, config, m.deps)
	if err != nil { return Instance{}, err }
	encrypted, err := m.cipher.Encrypt(string(config))
	if err != nil {
		_ = adapter.Stop(context.Background())
		return Instance{}, err
	}
	now := time.Now().UTC()
	record := &integration.Integration{
		ID: id, Type: pluginType, Name: name, Enabled: enabled,
		ConfigEncrypted: []byte(encrypted), Status: "disconnected", CreatedAt: now, UpdatedAt: now,
	}
	if err := m.repo.Create(ctx, record); err != nil {
		_ = adapter.Stop(context.Background())
		return Instance{}, err
	}
	if enabled {
		if err := m.supervisor.Register(adapter); err != nil {
			_ = m.repo.Delete(ctx, id)
			_ = adapter.Stop(context.Background())
			return Instance{}, err
		}
		m.mu.Lock()
		m.prepared[id] = adapter
		started := m.started
		m.mu.Unlock()
		if started {
			if err := m.supervisor.StartAdapter(ctx, id); err != nil {
				_ = m.repo.SetStatus(ctx, id, "error", safeError(err))
			} else {
				_ = m.repo.SetStatus(ctx, id, "connected", "")
			}
		}
	} else {
		// Validation and construction are performed for disabled instances too;
		// release any factory-owned resources without starting background work.
		_ = adapter.Stop(context.Background())
	}
	return m.toInstance(ctx, record)
}

// UpdateConfig validates and atomically replaces a persisted plugin configuration.
// The previous adapter/configuration is restored when the replacement cannot start
// or the encrypted configuration cannot be committed.
func (m *Manager) UpdateConfig(ctx context.Context, id string, config json.RawMessage) (Instance, error) {
	m.instanceMu.Lock()
	defer m.instanceMu.Unlock()
	return m.updateConfigLocked(ctx, id, config)
}

// UpdateConfigMutate atomically reads, mutates and replaces one encrypted
// plugin configuration under the same instance lifecycle lock. It prevents
// OAuth account updates from overwriting concurrent administrator edits.
func (m *Manager) UpdateConfigMutate(ctx context.Context, id string, mutate func(map[string]json.RawMessage) error) (Instance, error) {
	if mutate == nil {
		return Instance{}, errors.New("plugin_runtime: config mutator is required")
	}
	m.instanceMu.Lock()
	defer m.instanceMu.Unlock()
	if !integrationIDPattern.MatchString(id) {
		return Instance{}, errors.New("plugin_runtime: invalid integration instance ID")
	}
	record, err := m.repo.GetByID(ctx, id)
	if err != nil {
		return Instance{}, err
	}
	raw, err := m.decryptConfig(record.ConfigEncrypted)
	if err != nil {
		return Instance{}, fmt.Errorf("plugin_runtime: decrypt integration config: %w", err)
	}
	var config map[string]json.RawMessage
	if err := json.Unmarshal(raw, &config); err != nil {
		return Instance{}, fmt.Errorf("plugin_runtime: decode integration config: %w", err)
	}
	if config == nil {
		config = make(map[string]json.RawMessage)
	}
	if err := mutate(config); err != nil {
		return Instance{}, err
	}
	updated, err := json.Marshal(config)
	if err != nil {
		return Instance{}, fmt.Errorf("plugin_runtime: encode mutated integration config: %w", err)
	}
	return m.updateConfigLocked(ctx, id, updated)
}

func (m *Manager) updateConfigLocked(ctx context.Context, id string, config json.RawMessage) (Instance, error) {
record, err := m.repo.GetByID(ctx, id)
	if err != nil {
		return Instance{}, err
	}
	if !integrationIDPattern.MatchString(id) {
		return Instance{}, errors.New("plugin_runtime: invalid integration instance ID")
	}
	if len(config) == 0 {
		config = json.RawMessage(`{}`)
	}
	candidate, err := m.registry.Create(ctx, record.Type, record.ID, config, m.deps)
	if err != nil {
		return Instance{}, err
	}
	encrypted, err := m.cipher.Encrypt(string(config))
	if err != nil {
		_ = candidate.Stop(context.Background())
		return Instance{}, err
	}

	m.mu.Lock()
	started := m.started
	oldAdapter := m.prepared[id]
	m.mu.Unlock()

	if !record.Enabled {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		stopErr := candidate.Stop(stopCtx)
		cancel()
		if stopErr != nil {
			return Instance{}, stopErr
		}
		record.ConfigEncrypted = []byte(encrypted)
		record.UpdatedAt = time.Now().UTC()
		if err := m.repo.Update(ctx, record); err != nil {
			return Instance{}, err
		}
		return m.toInstance(ctx, record)
	}

	_, registered := m.supervisor.GetStatus(id)
	if registered {
		err = m.supervisor.Replace(ctx, id, candidate)
	} else {
		err = m.supervisor.Register(candidate)
		if err == nil && started {
			err = m.supervisor.StartAdapter(ctx, id)
		}
	}
	if err != nil {
		if !registered {
			stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			stopErr := m.supervisor.Unregister(stopCtx, id)
			cancel()
			if stopErr != nil {
				// Supervisor keeps the adapter registered while Stop is failing.
				m.mu.Lock()
				m.prepared[id] = candidate
				m.mu.Unlock()
				return Instance{}, errors.Join(err, stopErr)
			}
			// Unregister performed the successful Stop; do not stop twice.
		}
		// A registered replacement is cleaned up (or rolled back) by Supervisor.Replace.
		return Instance{}, err
	}

	m.mu.Lock()
	m.prepared[id] = candidate
	m.mu.Unlock()

	record.ConfigEncrypted = []byte(encrypted)
	record.UpdatedAt = time.Now().UTC()
	if err := m.repo.Update(ctx, record); err != nil {
		var rollbackErr error
		if oldAdapter != nil && registered {
			rollbackErr = m.supervisor.Replace(context.Background(), id, oldAdapter)
		} else {
			rollbackErr = m.supervisor.Unregister(context.Background(), id)
		}
		m.mu.Lock()
		if rollbackErr == nil {
			if oldAdapter != nil {
				m.prepared[id] = oldAdapter
			} else {
				delete(m.prepared, id)
			}
		}
		m.mu.Unlock()
		if rollbackErr != nil {
			_ = m.repo.SetStatus(ctx, id, "error", safeError(rollbackErr))
			return Instance{}, errors.Join(err, fmt.Errorf("plugin_runtime: configuration rollback failed: %w", rollbackErr))
		}
		return Instance{}, err
	}

	status := "disconnected"
	if started {
		status = "connected"
	}
	if err := m.repo.SetStatus(ctx, id, status, ""); err != nil {
		return Instance{}, err
	}
	return m.toInstance(ctx, record)
}
func (m *Manager) SetEnabled(ctx context.Context, id string, enabled bool) error {
	m.instanceMu.Lock()
	defer m.instanceMu.Unlock()

	record, err := m.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if record.Enabled == enabled {
		return nil
	}
	m.mu.Lock()
	started := m.started
	oldAdapter := m.prepared[id]
	m.mu.Unlock()

	if !enabled {
		if _, registered := m.supervisor.GetStatus(id); registered {
			stopCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			err = m.supervisor.Unregister(stopCtx, id)
			cancel()
			if err != nil {
				return err
			}
		}
		record.Enabled = false
		record.UpdatedAt = time.Now().UTC()
		if err := m.repo.Update(ctx, record); err != nil {
			if oldAdapter != nil {
				restoreErr := m.supervisor.Register(oldAdapter)
				if restoreErr == nil && started {
					restoreErr = m.supervisor.StartAdapter(ctx, id)
				}
				if restoreErr != nil {
					_ = m.repo.SetStatus(ctx, id, "error", safeError(restoreErr))
					return errors.Join(err, fmt.Errorf("plugin_runtime: could not restore adapter after failed disable: %w", restoreErr))
				}
			}
			return err
		}
		m.mu.Lock()
		delete(m.prepared, id)
		m.mu.Unlock()
		return m.repo.SetStatus(ctx, id, "disconnected", "")
	}

	raw, err := m.decryptConfig(record.ConfigEncrypted)
	if err != nil {
		_ = m.repo.SetStatus(ctx, id, "error", "stored plugin configuration could not be decrypted")
		return err
	}
	adapter, err := m.registry.Create(ctx, record.Type, record.ID, raw, m.deps)
	if err != nil {
		_ = m.repo.SetStatus(ctx, id, "error", safeError(err))
		return err
	}
	if err := m.supervisor.Register(adapter); err != nil {
		_ = adapter.Stop(context.Background())
		return err
	}
	if started {
		if err := m.supervisor.StartAdapter(ctx, id); err != nil {
			stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			stopErr := m.supervisor.Unregister(stopCtx, id)
			cancel()
			if stopErr != nil {
				m.mu.Lock()
				m.prepared[id] = adapter
				m.mu.Unlock()
			}
			_ = m.repo.SetStatus(ctx, id, "error", safeError(errors.Join(err, stopErr)))
			return errors.Join(err, stopErr)
		}
	}
	record.Enabled = true
	record.UpdatedAt = time.Now().UTC()
	if err := m.repo.Update(ctx, record); err != nil {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		stopErr := m.supervisor.Unregister(stopCtx, id)
		cancel()
		if stopErr != nil {
			m.mu.Lock()
			m.prepared[id] = adapter
			m.mu.Unlock()
			_ = m.repo.SetStatus(ctx, id, "error", safeError(errors.Join(err, stopErr)))
		}
		return errors.Join(err, stopErr)
	}
	m.mu.Lock()
	m.prepared[id] = adapter
	m.mu.Unlock()
	status := "disconnected"
	if started {
		status = "connected"
	}
	return m.repo.SetStatus(ctx, id, status, "")
}

func (m *Manager) Restart(ctx context.Context, id string) error {
	m.instanceMu.Lock()
	defer m.instanceMu.Unlock()
	record, err := m.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if !record.Enabled {
		return errors.New("plugin_runtime: disabled instance cannot be restarted")
	}
	m.mu.Lock()
	started := m.started
	m.mu.Unlock()
	if !started {
		return errors.New("plugin_runtime: runtime is not running")
	}
	if err := m.supervisor.Restart(ctx, id); err != nil {
		_ = m.repo.SetStatus(ctx, id, "error", safeError(err))
		return err
	}
	return m.repo.SetStatus(ctx, id, "connected", "")
}

func (m *Manager) Delete(ctx context.Context, id string) error {
	m.instanceMu.Lock()
	defer m.instanceMu.Unlock()
	record, err := m.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	m.mu.Lock()
	started := m.started
	adapter := m.prepared[id]
	m.mu.Unlock()
	if _, exists := m.supervisor.GetStatus(id); exists {
		if err := m.supervisor.Unregister(ctx, id); err != nil {
			return err
		}
	}
	if err := m.repo.Delete(ctx, id); err != nil {
		if record.Enabled && adapter != nil {
			restoreErr := m.supervisor.Register(adapter)
			if restoreErr == nil && started {
				restoreErr = m.supervisor.StartAdapter(ctx, id)
			}
			if restoreErr != nil {
				_ = m.repo.SetStatus(ctx, id, "error", safeError(restoreErr))
				return errors.Join(err, fmt.Errorf("plugin_runtime: could not restore adapter after failed deletion: %w", restoreErr))
			}
		}
		return err
	}
	m.mu.Lock()
	delete(m.prepared, id)
	m.mu.Unlock()
	return nil
}

func (m *Manager) ListInstances(ctx context.Context) ([]Instance, error) {
	records, err := m.repo.List(ctx)
	if err != nil { return nil, err }
	out := make([]Instance, 0, len(records))
	for _, record := range records {
		if record == nil { continue }
		instance, convertErr := m.toInstance(ctx, record)
		if convertErr != nil { return nil, convertErr }
		out = append(out, instance)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (m *Manager) toInstance(ctx context.Context, record *integration.Integration) (Instance, error) {
	status := record.Status
	lastError := ""
	if current, ok := m.supervisor.GetStatus(record.ID); ok {
		switch current.State {
		case plugins.StateHealthy: status = "connected"
		case plugins.StateStarting: status = "starting"
		case plugins.StateDegraded, plugins.StateTripped:
			status = "error"
			lastError = "adapter health check failed"
		case plugins.StateStopped: status = "disconnected"
		}
		if current.LastError != "" { lastError = "adapter reported a connection or health error" }
	}
	return Instance{ID: record.ID, Type: record.Type, Name: record.Name, Enabled: record.Enabled, Status: status, LastError: lastError}, nil
}

func (m *Manager) syncStatuses(ctx context.Context) {
	for _, status := range m.supervisor.ListStatus() {
		if status.State == plugins.StateHealthy {
			_ = m.repo.SetStatus(ctx, status.ID, "connected", "")
		} else {
			_ = m.repo.SetStatus(ctx, status.ID, "error", "adapter failed to start or pass its initial health check")
		}
	}
}

func (m *Manager) Stop(ctx context.Context) error {
	m.instanceMu.Lock()
	defer m.instanceMu.Unlock()
	m.mu.Lock()
	started := m.started
	m.started = false
	m.mu.Unlock()
	if !started {
		return nil
	}
	return m.supervisor.Stop(ctx)
}

func safeError(err error) string {
	if err == nil { return "" }
	message := strings.TrimSpace(err.Error())
	if len(message) > 240 { message = message[:240] }
	// Do not persist full remote response bodies, URLs with credentials or plugin config.
	return "plugin startup/configuration failed; inspect the local Domus logs"
}
