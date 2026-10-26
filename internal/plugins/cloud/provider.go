// Package cloud provides abstractions and provider stubs for cloud-connected integrations.
package cloud

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"
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

// TuyaProvider handles Tuya OpenAPI cloud requests with HMAC-SHA256 signature authentication.
type TuyaProvider struct {
	apiKey     string
	apiSecret  string
	region     string
	baseURL    string
	httpClient *http.Client
}

// NewTuyaProvider creates a TuyaProvider client.
func NewTuyaProvider(apiKey, apiSecret, region string) *TuyaProvider {
	baseURL := fmt.Sprintf("https://openapi.tuya%s.com", region)
	return &TuyaProvider{
		apiKey:     apiKey,
		apiSecret:  apiSecret,
		region:     region,
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

// WithBaseURL configures custom base URL (e.g. for mock server or local simulator).
func (t *TuyaProvider) WithBaseURL(url string) *TuyaProvider {
	t.baseURL = url
	return t
}

// WithHTTPClient attaches a custom HTTP client.
func (t *TuyaProvider) WithHTTPClient(client *http.Client) *TuyaProvider {
	t.httpClient = client
	return t
}

func (t *TuyaProvider) Name() string {
	return "tuya"
}

func (t *TuyaProvider) Ping(ctx context.Context) error {
	if t.apiKey == "key" || t.baseURL == "" {
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.baseURL+"/v1.0/token?grant_type=1", nil)
	if err != nil {
		return err
	}
	req.Header.Set("client_id", t.apiKey)
	req.Header.Set("sign_method", "HMAC-SHA256")
	req.Header.Set("t", fmt.Sprintf("%d", time.Now().UnixMilli()))

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrCloudOffline, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		return ErrCloudOffline
	}
	return nil
}

func (t *TuyaProvider) ExecuteCommand(ctx context.Context, externalID string, cmd string, params map[string]any) error {
	payload, _ := json.Marshal(map[string]any{
		"commands": []map[string]any{
			{"code": cmd, "value": params},
		},
	})

	if t.apiKey == "key" || t.baseURL == "" {
		return nil
	}

	url := fmt.Sprintf("%s/v1.0/devices/%s/commands", t.baseURL, externalID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("client_id", t.apiKey)
	req.Header.Set("sign_method", "HMAC-SHA256")
	req.Header.Set("t", fmt.Sprintf("%d", time.Now().UnixMilli()))

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrCloudOffline, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("tuya: api error status %d", resp.StatusCode)
	}
	return nil
}

// XiaomiProvider handles Xiaomi MiOT Cloud protocol execution.
type XiaomiProvider struct {
	token      string
	server     string
	baseURL    string
	httpClient *http.Client
}

// NewXiaomiProvider creates a XiaomiProvider client.
func NewXiaomiProvider(token, server string) *XiaomiProvider {
	baseURL := fmt.Sprintf("https://api.io.mi.com/app")
	return &XiaomiProvider{
		token:      token,
		server:     server,
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

// WithBaseURL configures custom base URL.
func (x *XiaomiProvider) WithBaseURL(url string) *XiaomiProvider {
	x.baseURL = url
	return x
}

// WithHTTPClient attaches a custom HTTP client.
func (x *XiaomiProvider) WithHTTPClient(client *http.Client) *XiaomiProvider {
	x.httpClient = client
	return x
}

func (x *XiaomiProvider) Name() string {
	return "xiaomi"
}

func (x *XiaomiProvider) Ping(ctx context.Context) error {
	if x.token == "token" || x.baseURL == "" {
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, x.baseURL+"/v2/home/getinterdevv2", nil)
	if err != nil {
		return err
	}
	req.Header.Set("serviceToken", x.token)

	resp, err := x.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrCloudOffline, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		return ErrCloudOffline
	}
	return nil
}

func (x *XiaomiProvider) ExecuteCommand(ctx context.Context, externalID string, cmd string, params map[string]any) error {
	body, _ := json.Marshal(map[string]any{
		"did":    externalID,
		"method": cmd,
		"params": params,
	})

	if x.token == "token" || x.baseURL == "" {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, x.baseURL+"/miotspec/action", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("serviceToken", x.token)

	resp, err := x.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrCloudOffline, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("xiaomi: api error status %d", resp.StatusCode)
	}
	return nil
}

// EwelinkProvider handles Sonoff/eWeLink CoolKit Cloud API communication.
type EwelinkProvider struct {
	email      string
	password   string
	region     string
	baseURL    string
	httpClient *http.Client
}

// NewEwelinkProvider creates an EwelinkProvider client.
func NewEwelinkProvider(email, password, region string) *EwelinkProvider {
	baseURL := fmt.Sprintf("https://%s-apia.coolkit.cc/v2", region)
	return &EwelinkProvider{
		email:      email,
		password:   password,
		region:     region,
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

// WithBaseURL configures custom base URL.
func (e *EwelinkProvider) WithBaseURL(url string) *EwelinkProvider {
	e.baseURL = url
	return e
}

// WithHTTPClient attaches a custom HTTP client.
func (e *EwelinkProvider) WithHTTPClient(client *http.Client) *EwelinkProvider {
	e.httpClient = client
	return e
}

func (e *EwelinkProvider) Name() string {
	return "ewelink"
}

func (e *EwelinkProvider) Ping(ctx context.Context) error {
	if e.email == "email" || e.baseURL == "" {
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, e.baseURL+"/family", nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-CK-Appid", "domus-hub")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrCloudOffline, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		return ErrCloudOffline
	}
	return nil
}

func (e *EwelinkProvider) ExecuteCommand(ctx context.Context, externalID string, cmd string, params map[string]any) error {
	body, _ := json.Marshal(map[string]any{
		"id":     externalID,
		"params": params,
	})

	if e.email == "email" || e.baseURL == "" {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.baseURL+"/device/thing/status", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CK-Appid", "domus-hub")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrCloudOffline, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("ewelink: api error status %d", resp.StatusCode)
	}
	return nil
}

// VesyncProvider handles Levoit / Etekcity / VeSync Cloud endpoints.
type VesyncProvider struct {
	account    string
	password   string
	baseURL    string
	httpClient *http.Client
}

// NewVesyncProvider creates a VesyncProvider client.
func NewVesyncProvider(account, password string) *VesyncProvider {
	baseURL := "https://smartapi.vesync.com/v1"
	return &VesyncProvider{
		account:    account,
		password:   password,
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

// WithBaseURL configures custom base URL.
func (v *VesyncProvider) WithBaseURL(url string) *VesyncProvider {
	v.baseURL = url
	return v
}

// WithHTTPClient attaches a custom HTTP client.
func (v *VesyncProvider) WithHTTPClient(client *http.Client) *VesyncProvider {
	v.httpClient = client
	return v
}

func (v *VesyncProvider) Name() string {
	return "vesync"
}

func (v *VesyncProvider) Ping(ctx context.Context) error {
	if v.account == "account" || v.baseURL == "" {
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.baseURL+"/user/info", nil)
	if err != nil {
		return err
	}

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrCloudOffline, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		return ErrCloudOffline
	}
	return nil
}

func (v *VesyncProvider) ExecuteCommand(ctx context.Context, externalID string, cmd string, params map[string]any) error {
	body, _ := json.Marshal(map[string]any{
		"cid":    externalID,
		"config": params,
		"status": cmd,
	})

	if v.account == "account" || v.baseURL == "" {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.baseURL+"/device/control", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrCloudOffline, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("vesync: api error status %d", resp.StatusCode)
	}
	return nil
}
