package plugins

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/device"
	"github.com/domus-hub/domus/internal/domain/entity"
	"github.com/domus-hub/domus/internal/domain/events"
)

const (
	ManifestAPIVersion = "domus.plugin/v1"
	MaxPluginConfigBytes = 64 << 10
)

var (
	ErrPluginNotFound = errors.New("plugins: plugin type is not registered")
	ErrPluginExists = errors.New("plugins: plugin type already registered")
	ErrInvalidManifest = errors.New("plugins: invalid plugin manifest")
	ErrInvalidPluginConfig = errors.New("plugins: invalid plugin configuration")
	ErrAdapterMismatch = errors.New("plugins: factory returned an adapter that does not match its manifest")
	pluginIDPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{1,62}$`)
	instanceIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)
	pluginVersionPattern = regexp.MustCompile(`^v?[0-9]+\.[0-9]+\.[0-9]+(?:[-+][A-Za-z0-9.-]+)?$`)
	configKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)
)

// Capability is a declared permission needed by a plugin. For native Go plugins
// this is metadata and review guidance, not an OS sandbox.
type Capability string

const (
	CapabilityEventsRead   Capability = "events:read"
	CapabilityEventsWrite  Capability = "events:write"
	CapabilityDevicesRead  Capability = "devices:read"
	CapabilityDevicesWrite Capability = "devices:write"
	CapabilityNetworkLAN   Capability = "network:lan"
	CapabilityNetworkCloud Capability = "network:cloud"
	CapabilitySecretsRead  Capability = "secrets:read"
	CapabilitySecretsWrite Capability = "secrets:write"
	CapabilityStorageRead  Capability = "storage:read"
	CapabilityStorageWrite Capability = "storage:write"
)

var knownCapabilities = map[Capability]struct{}{
	CapabilityEventsRead: {}, CapabilityEventsWrite: {},
	CapabilityDevicesRead: {}, CapabilityDevicesWrite: {},
	CapabilityNetworkLAN: {}, CapabilityNetworkCloud: {},
	CapabilitySecretsRead: {}, CapabilitySecretsWrite: {},
	CapabilityStorageRead: {}, CapabilityStorageWrite: {},
}

// ConfigField documents and validates one plugin setting. Supported types are
// string, boolean, integer, number, duration and url.
type ConfigField struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Required    bool     `json:"required,omitempty"`
	Secret      bool     `json:"secret,omitempty"`
	Description string   `json:"description,omitempty"`
	Allowed     []string `json:"allowed,omitempty"`
}

// Manifest is the declarative description of an integration plugin.
type Manifest struct {
	APIVersion   string        `json:"api_version"`
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Version      string        `json:"version"`
	Description  string        `json:"description,omitempty"`
	Author       string        `json:"author,omitempty"`
	Capabilities []Capability  `json:"capabilities,omitempty"`
	Config       []ConfigField `json:"config,omitempty"`
}

func (m Manifest) Validate() error {
	if m.APIVersion != ManifestAPIVersion {
		return fmt.Errorf("%w: api_version must be %q", ErrInvalidManifest, ManifestAPIVersion)
	}
	if !pluginIDPattern.MatchString(m.ID) {
		return fmt.Errorf("%w: id must be a lowercase plugin key (2-63 characters)", ErrInvalidManifest)
	}
	if strings.TrimSpace(m.Name) == "" || len(m.Name) > 128 {
		return fmt.Errorf("%w: name is required and must be at most 128 bytes", ErrInvalidManifest)
	}
	if !pluginVersionPattern.MatchString(m.Version) {
		return fmt.Errorf("%w: version must be semver (for example 1.0.0)", ErrInvalidManifest)
	}
	seenCapabilities := make(map[Capability]struct{}, len(m.Capabilities))
	for _, capability := range m.Capabilities {
		if _, ok := knownCapabilities[capability]; !ok {
			return fmt.Errorf("%w: unsupported capability %q", ErrInvalidManifest, capability)
		}
		if _, duplicate := seenCapabilities[capability]; duplicate {
			return fmt.Errorf("%w: duplicate capability %q", ErrInvalidManifest, capability)
		}
		seenCapabilities[capability] = struct{}{}
	}
	seenFields := make(map[string]struct{}, len(m.Config))
	for _, field := range m.Config {
		if !configKeyPattern.MatchString(field.Name) {
			return fmt.Errorf("%w: invalid config field name %q", ErrInvalidManifest, field.Name)
		}
		switch field.Type {
		case "string", "boolean", "integer", "number", "duration", "url":
		default:
			return fmt.Errorf("%w: config field %q has unsupported type %q", ErrInvalidManifest, field.Name, field.Type)
		}
		if _, duplicate := seenFields[field.Name]; duplicate {
			return fmt.Errorf("%w: duplicate config field %q", ErrInvalidManifest, field.Name)
		}
		seenFields[field.Name] = struct{}{}
		if len(field.Allowed) > 0 && field.Type != "string" {
			return fmt.Errorf("%w: allowed values are supported only for string fields (%q)", ErrInvalidManifest, field.Name)
		}
	}
	return nil
}

// Dependencies are the supported host services offered to in-process plugins.
// A plugin should retain only the dependencies it needs and must honour ctx.
type Dependencies struct {
	Bus      events.Bus
	Entities entity.Repository
	Devices  device.Repository
	Logger   *slog.Logger
}

// Factory creates one configured adapter instance. It must not start background
// work; the Supervisor owns Start/Stop and recovery.
type Factory func(ctx context.Context, instanceID string, config json.RawMessage, deps Dependencies) (Adapter, error)

type pluginRegistration struct {
	manifest Manifest
	factory  Factory
}

// Registry maps plugin type IDs to validated manifests and adapter factories.
type Registry struct {
	mu       sync.RWMutex
	entries  map[string]pluginRegistration
}

func NewRegistry() *Registry {
	return &Registry{entries: make(map[string]pluginRegistration)}
}

func (r *Registry) Register(manifest Manifest, factory Factory) error {
	if r == nil {
		return fmt.Errorf("plugins: nil registry")
	}
	if err := manifest.Validate(); err != nil {
		return err
	}
	if factory == nil {
		return fmt.Errorf("%w: factory is nil", ErrInvalidManifest)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.entries[manifest.ID]; exists {
		return ErrPluginExists
	}
	manifest.Capabilities = append([]Capability(nil), manifest.Capabilities...)
	manifest.Config = append([]ConfigField(nil), manifest.Config...)
	r.entries[manifest.ID] = pluginRegistration{manifest: manifest, factory: factory}
	return nil
}

// List returns stable, defensive copies of the registered manifests.
func (r *Registry) List() []Manifest {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := make([]string, 0, len(r.entries))
	for id := range r.entries {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	result := make([]Manifest, 0, len(ids))
	for _, id := range ids {
		m := r.entries[id].manifest
		m.Capabilities = append([]Capability(nil), m.Capabilities...)
		m.Config = append([]ConfigField(nil), m.Config...)
		result = append(result, m)
	}
	return result
}

// Create validates the supplied settings and returns an adapter. Start is
// deliberately not called here; register the result with Supervisor afterwards.
func (r *Registry) Create(ctx context.Context, pluginID, instanceID string, rawConfig json.RawMessage, deps Dependencies) (Adapter, error) {
	if r == nil {
		return nil, fmt.Errorf("plugins: nil registry")
	}
	if !instanceIDPattern.MatchString(instanceID) {
		return nil, fmt.Errorf("%w: invalid instance ID", ErrInvalidPluginConfig)
	}
	r.mu.RLock()
	registration, ok := r.entries[pluginID]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrPluginNotFound, pluginID)
	}
	if err := validateConfig(registration.manifest.Config, rawConfig); err != nil {
		return nil, err
	}
	if deps.Logger == nil {
		deps.Logger = slog.Default()
	}
	adapter, err := registration.factory(ctx, instanceID, rawConfig, deps)
	if err != nil {
		return nil, fmt.Errorf("plugins: create %s instance %s: %w", pluginID, instanceID, err)
	}
	if adapter == nil || adapter.ID() != instanceID || adapter.Type() != pluginID {
		if adapter != nil {
			stopCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			_ = adapter.Stop(stopCtx)
			cancel()
		}
		return nil, ErrAdapterMismatch
	}
	return adapter, nil
}

func validateConfig(fields []ConfigField, raw json.RawMessage) error {
	if len(raw) > MaxPluginConfigBytes {
		return fmt.Errorf("%w: configuration exceeds %d bytes", ErrInvalidPluginConfig, MaxPluginConfigBytes)
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		raw = json.RawMessage("{}")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var values map[string]json.RawMessage
	if err := decoder.Decode(&values); err != nil || values == nil {
		return fmt.Errorf("%w: expected a JSON object", ErrInvalidPluginConfig)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return fmt.Errorf("%w: trailing JSON content", ErrInvalidPluginConfig)
	}
	schema := make(map[string]ConfigField, len(fields))
	for _, field := range fields {
		schema[field.Name] = field
		if field.Required {
			if _, exists := values[field.Name]; !exists {
				return fmt.Errorf("%w: required field %q is missing", ErrInvalidPluginConfig, field.Name)
			}
		}
	}
	for name, rawValue := range values {
		field, ok := schema[name]
		if !ok {
			return fmt.Errorf("%w: unknown field %q", ErrInvalidPluginConfig, name)
		}
		var value any
		valueDecoder := json.NewDecoder(bytes.NewReader(rawValue))
		valueDecoder.UseNumber()
		if err := valueDecoder.Decode(&value); err != nil {
			return fmt.Errorf("%w: field %q: %v", ErrInvalidPluginConfig, name, err)
		}
		switch field.Type {
		case "string", "duration", "url":
			text, ok := value.(string)
			if !ok || strings.TrimSpace(text) == "" {
				return fmt.Errorf("%w: field %q must be a non-empty string", ErrInvalidPluginConfig, name)
			}
			if field.Type == "duration" {
				if _, err := time.ParseDuration(text); err != nil {
					return fmt.Errorf("%w: field %q must be a Go duration: %v", ErrInvalidPluginConfig, name, err)
				}
			}
			if field.Type == "url" {
				u, err := url.ParseRequestURI(text)
				if err != nil || u.Scheme == "" || u.Host == "" {
					return fmt.Errorf("%w: field %q must be an absolute URL", ErrInvalidPluginConfig, name)
				}
			}
			if len(field.Allowed) > 0 {
				allowed := false
				for _, candidate := range field.Allowed {
					if text == candidate { allowed = true; break }
				}
				if !allowed { return fmt.Errorf("%w: field %q has an unsupported value", ErrInvalidPluginConfig, name) }
			}
		case "boolean":
			if _, ok := value.(bool); !ok { return fmt.Errorf("%w: field %q must be a boolean", ErrInvalidPluginConfig, name) }
		case "integer":
			number, ok := value.(json.Number)
			if !ok { return fmt.Errorf("%w: field %q must be an integer", ErrInvalidPluginConfig, name) }
			if _, err := strconv.ParseInt(number.String(), 10, 64); err != nil { return fmt.Errorf("%w: field %q must be an integer", ErrInvalidPluginConfig, name) }
		case "number":
			number, ok := value.(json.Number)
			if !ok { return fmt.Errorf("%w: field %q must be a number", ErrInvalidPluginConfig, name) }
			if _, err := strconv.ParseFloat(number.String(), 64); err != nil { return fmt.Errorf("%w: field %q must be a number", ErrInvalidPluginConfig, name) }
		}
	}
	return nil
}

// DecodeConfig decodes a plugin's typed settings and rejects unknown JSON keys.
func DecodeConfig[T any](raw json.RawMessage) (T, error) {
	var config T
	if len(raw) > MaxPluginConfigBytes {
		return config, fmt.Errorf("%w: configuration exceeds %d bytes", ErrInvalidPluginConfig, MaxPluginConfigBytes)
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		raw = json.RawMessage("{}")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return config, fmt.Errorf("%w: %v", ErrInvalidPluginConfig, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return config, fmt.Errorf("%w: trailing JSON content", ErrInvalidPluginConfig)
	}
	return config, nil
}
