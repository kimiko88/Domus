// Package cloud provides abstractions and provider stubs for cloud-connected integrations.
package cloud

import (
	"context"
	"errors"
	"sync"
)

var (
	// ErrCloudOffline is returned when a remote cloud API is unreachable.
	ErrCloudOffline = errors.New("cloud: provider is currently offline")
)

// Provider represents a remote cloud vendor client (Tuya, Xiaomi, eWeLink, VeSync).
type Provider interface {
	// Name returns the provider identifier.
	Name() string
	// ExecuteCommand sends a control payload to the cloud API.
	ExecuteCommand(ctx context.Context, externalID string, cmd string, params map[string]any) error
	// Ping tests cloud API connectivity.
	Ping(ctx context.Context) error
}

// MockProvider is an in-memory test implementation of Provider.
type MockProvider struct {
	mu     sync.Mutex
	name   string
	online bool
}

// NewMockProvider creates a new MockProvider.
func NewMockProvider(name string) *MockProvider {
	return &MockProvider{name: name, online: true}
}

func (m *MockProvider) Name() string {
	return m.name
}

func (m *MockProvider) SetOnline(online bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.online = online
}

func (m *MockProvider) Ping(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.online {
		return ErrCloudOffline
	}
	return nil
}

func (m *MockProvider) ExecuteCommand(ctx context.Context, externalID string, cmd string, params map[string]any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.online {
		return ErrCloudOffline
	}
	return nil
}

// TuyaProvider stub for Tuya OpenAPI.
type TuyaProvider struct {
	apiKey    string
	apiSecret string
	region    string
}

// NewTuyaProvider creates a TuyaProvider stub.
func NewTuyaProvider(apiKey, apiSecret, region string) *TuyaProvider {
	return &TuyaProvider{apiKey: apiKey, apiSecret: apiSecret, region: region}
}

func (t *TuyaProvider) Name() string {
	return "tuya"
}

func (t *TuyaProvider) Ping(ctx context.Context) error {
	return nil
}

func (t *TuyaProvider) ExecuteCommand(ctx context.Context, externalID string, cmd string, params map[string]any) error {
	return nil
}

// XiaomiProvider stub for Xiaomi MiOT Cloud.
type XiaomiProvider struct {
	token  string
	server string
}

// NewXiaomiProvider creates a XiaomiProvider stub.
func NewXiaomiProvider(token, server string) *XiaomiProvider {
	return &XiaomiProvider{token: token, server: server}
}

func (x *XiaomiProvider) Name() string {
	return "xiaomi"
}

func (x *XiaomiProvider) Ping(ctx context.Context) error {
	return nil
}

func (x *XiaomiProvider) ExecuteCommand(ctx context.Context, externalID string, cmd string, params map[string]any) error {
	return nil
}

// EwelinkProvider stub for Sonoff/eWeLink Cloud.
type EwelinkProvider struct {
	email    string
	password string
	region   string
}

// NewEwelinkProvider creates an EwelinkProvider stub.
func NewEwelinkProvider(email, password, region string) *EwelinkProvider {
	return &EwelinkProvider{email: email, password: password, region: region}
}

func (e *EwelinkProvider) Name() string {
	return "ewelink"
}

func (e *EwelinkProvider) Ping(ctx context.Context) error {
	return nil
}

func (e *EwelinkProvider) ExecuteCommand(ctx context.Context, externalID string, cmd string, params map[string]any) error {
	return nil
}

// VesyncProvider stub for Levoit / Etekcity / VeSync Cloud.
type VesyncProvider struct {
	account  string
	password string
}

// NewVesyncProvider creates a VesyncProvider stub.
func NewVesyncProvider(account, password string) *VesyncProvider {
	return &VesyncProvider{account: account, password: password}
}

func (v *VesyncProvider) Name() string {
	return "vesync"
}

func (v *VesyncProvider) Ping(ctx context.Context) error {
	return nil
}

func (v *VesyncProvider) ExecuteCommand(ctx context.Context, externalID string, cmd string, params map[string]any) error {
	return nil
}
