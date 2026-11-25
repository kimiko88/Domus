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
	mu sync.RWMutex
	active bool
}

var _ Provider = (*Adapter)(nil)

func NewAdapter(id string, cfg Config) (*Adapter, error) {
	id = strings.TrimSpace(id)
	if id == "" { id = PluginType }
	client, err := New(cfg)
	if err != nil { return nil, err }
	return &Adapter{id:id, client:client}, nil
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

func (a *Adapter) Config() Config { return a.client.Config() }
func (a *Adapter) SetCredentials(clientID, secret, baseURL, uid string) {
	a.client.SetCredentials(clientID, secret, baseURL, uid)
}
func (a *Adapter) Devices(ctx context.Context) ([]Device, error) {
	return a.client.Devices(ctx)
}
func (a *Adapter) GetDevice(ctx context.Context, id string) (*Device, error) {
	return a.client.GetDevice(ctx, id)
}
func (a *Adapter) Functions(ctx context.Context, id string) ([]Function, error) {
	return a.client.Functions(ctx, id)
}
func (a *Adapter) SendCommands(ctx context.Context, id string, commands []Command) error {
	return a.client.SendCommands(ctx, id, commands)
}

// Client exposes the concrete transport only for a temporary compatibility bridge.
// New application code should depend on Provider, not *Client.
func (a *Adapter) Client() *Client { return a.client }
