// Package tuya_cloud exposes Tuya OpenAPI as a supervised Domus service plugin.
package tuya_cloud

import (
	"context"
	"errors"
	"strings"
	"sync"
)

const PluginType = "tuya_cloud"

type Adapter struct {
	id string
	client *Client
	config Config
	mu sync.RWMutex
	active bool
}

var _ Provider = (*Adapter)(nil)

func NewAdapter(id string, cfg Config) (*Adapter, error) {
	id = strings.TrimSpace(id)
	if id == "" { id = PluginType }
	// Smart Life OAuth can be configured independently from the legacy Tuya
	// OpenAPI client credentials. Allow a dormant provider instance so the
	// encrypted Plugin Manager config can own OAuth state without substituting
	// App Authorization credentials for OpenAPI credentials.
	if strings.TrimSpace(cfg.ClientID) == "" && strings.TrimSpace(cfg.Secret) == "" {
		if strings.TrimSpace(cfg.BaseURL) == "" {
			cfg.BaseURL = "https://openapi.tuyaeu.com"
		}
		return &Adapter{id: id, config: cfg}, nil
	}
	client, err := New(cfg)
	if err != nil { return nil, err }
	return &Adapter{id:id, client:client, config:client.Config()}, nil
}

func (a *Adapter) ID() string { return a.id }
func (a *Adapter) Type() string { return PluginType }

func (a *Adapter) Start(ctx context.Context) error {
	if err := ctx.Err(); err != nil { return err }
	a.mu.Lock()
	a.active = true
	a.mu.Unlock()
	return nil
}

func (a *Adapter) Stop(ctx context.Context) error {
	a.mu.Lock()
	a.active = false
	a.mu.Unlock()
	return nil
}

func (a *Adapter) HealthCheck(ctx context.Context) error {
	if err := ctx.Err(); err != nil { return err }
	a.mu.RLock()
	active := a.active
	a.mu.RUnlock()
	if !active { return errors.New("tuya cloud: provider is stopped") }
	return nil
}

func (a *Adapter) providerClient() (*Client, error) {
	a.mu.RLock()
	client := a.client
	a.mu.RUnlock()
	if client == nil {
		return nil, errors.New("tuya cloud: OpenAPI client credentials are not configured; Smart Life OAuth credentials are stored separately")
	}
	return client, nil
}

func (a *Adapter) Config() Config {
	a.mu.RLock()
	client, cfg := a.client, a.config
	a.mu.RUnlock()
	if client != nil { return client.Config() }
	return cfg
}

func (a *Adapter) SetCredentials(clientID, secret, baseURL, uid string) {
	a.mu.Lock()
	cfg := a.config
	cfg.ClientID, cfg.Secret = strings.TrimSpace(clientID), strings.TrimSpace(secret)
	if strings.TrimSpace(baseURL) != "" { cfg.BaseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/") }
	cfg.UID = strings.TrimSpace(uid)
	a.config = cfg
	client := a.client
	if client != nil {
		client.SetCredentials(cfg.ClientID, cfg.Secret, cfg.BaseURL, cfg.UID)
		a.mu.Unlock()
		return
	}
	a.mu.Unlock()
	if cfg.ClientID == "" || cfg.Secret == "" { return }
	created, err := New(cfg)
	if err != nil { return }
	a.mu.Lock()
	if a.client == nil {
		a.client = created
		a.config = created.Config()
	}
	a.mu.Unlock()
}

func (a *Adapter) Devices(ctx context.Context) ([]Device, error) {
	client, err := a.providerClient()
	if err != nil { return nil, err }
	return client.Devices(ctx)
}
func (a *Adapter) GetDevice(ctx context.Context, id string) (*Device, error) {
	client, err := a.providerClient()
	if err != nil { return nil, err }
	return client.GetDevice(ctx, id)
}
func (a *Adapter) Functions(ctx context.Context, id string) ([]Function, error) {
	client, err := a.providerClient()
	if err != nil { return nil, err }
	return client.Functions(ctx, id)
}
func (a *Adapter) SendCommands(ctx context.Context, id string, commands []Command) error {
	client, err := a.providerClient()
	if err != nil { return err }
	return client.SendCommands(ctx, id, commands)
}

// Client exposes the concrete transport only for a temporary compatibility bridge.
// New application code should depend on Provider, not *Client.
func (a *Adapter) Client() *Client {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.client
}
