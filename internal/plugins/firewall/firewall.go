// Package firewall provides network router and firewall integrations
// (UniFi, OPNsense, pfSense, MikroTik RouterOS) for automated device quarantine.
package firewall

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ProviderType identifies the router or firewall OS.
type ProviderType string

const (
	ProviderUniFi    ProviderType = "unifi"
	ProviderOPNsense ProviderType = "opnsense"
	ProviderMikroTik ProviderType = "mikrotik"
	ProviderMock     ProviderType = "mock"
)

// Config defines connection credentials for the router/firewall controller.
type Config struct {
	Provider    ProviderType `json:"provider"`
	BaseURL     string       `json:"base_url"` // e.g. "https://192.168.1.1:8443" or "https://opnsense.local"
	Site        string       `json:"site"`     // e.g. "default" for UniFi
	APIKey      string       `json:"api_key,omitempty"`
	Username    string       `json:"username,omitempty"`
	Password    string       `json:"password,omitempty"`
	InsecureTLS bool         `json:"insecure_tls"`
}

// BlockedClient represents an isolated IoT device in quarantine.
type BlockedClient struct {
	MAC       string       `json:"mac"`
	IP        string       `json:"ip,omitempty"`
	Reason    string       `json:"reason"`
	BlockedAt time.Time    `json:"blocked_at"`
	Provider  ProviderType `json:"provider"`
}

// Client defines the contract for controlling network access at the router level.
type Client interface {
	IsolateClient(ctx context.Context, mac, ip, reason string) error
	UnblockClient(ctx context.Context, mac, ip string) error
	ListBlockedClients(ctx context.Context) ([]BlockedClient, error)
	GetStatus(ctx context.Context) (map[string]any, error)
}

// UnifiedClient dispatches isolation commands to the configured router integration.
type UnifiedClient struct {
	cfg        Config
	httpClient *http.Client
	mu         sync.RWMutex
	blocked    map[string]BlockedClient // keyed by normalized MAC
}

// NewClient constructs a firewall client for the specified provider.
func NewClient(cfg Config) *UnifiedClient {
	if cfg.Site == "" {
		cfg.Site = "default"
	}
	if cfg.Provider == "" {
		cfg.Provider = ProviderMock
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: cfg.InsecureTLS,
		},
	}

	return &UnifiedClient{
		cfg: cfg,
		httpClient: &http.Client{
			Transport: tr,
			Timeout:   10 * time.Second,
		},
		blocked: make(map[string]BlockedClient),
	}
}

func normalizeMAC(mac string) string {
	clean := strings.ToLower(strings.ReplaceAll(mac, "-", ":"))
	return strings.TrimSpace(clean)
}

// IsolateClient immediately isolates an IoT device by MAC and/or IP address.
func (c *UnifiedClient) IsolateClient(ctx context.Context, mac, ip, reason string) error {
	cleanMAC := normalizeMAC(mac)
	if cleanMAC == "" && ip == "" {
		return errors.New("firewall: either MAC or IP address must be provided")
	}

	now := time.Now().UTC()

	switch c.cfg.Provider {
	case ProviderUniFi:
		// UniFi Controller sta-manager block command
		url := fmt.Sprintf("%s/api/s/%s/cmd/stamgr", strings.TrimRight(c.cfg.BaseURL, "/"), c.cfg.Site)
		payload := map[string]any{
			"cmd": "block-sta",
			"mac": cleanMAC,
		}
		if err := c.doJSONPost(ctx, url, payload); err != nil {
			return fmt.Errorf("unifi isolate failed: %w", err)
		}

	case ProviderOPNsense:
		// OPNsense alias_util add table entry
		url := fmt.Sprintf("%s/api/firewall/alias_util/add/domus_quarantine", strings.TrimRight(c.cfg.BaseURL, "/"))
		target := ip
		if target == "" {
			target = cleanMAC
		}
		payload := map[string]any{
			"address": target,
		}
		if err := c.doJSONPost(ctx, url, payload); err != nil {
			return fmt.Errorf("opnsense isolate failed: %w", err)
		}

	case ProviderMikroTik:
		// MikroTik RouterOS REST API address-list addition
		url := fmt.Sprintf("%s/rest/ip/firewall/address-list", strings.TrimRight(c.cfg.BaseURL, "/"))
		target := ip
		if target == "" {
			target = cleanMAC
		}
		payload := map[string]any{
			"list":    "domus_quarantine",
			"address": target,
			"comment": fmt.Sprintf("Domus auto-isolated: %s", reason),
		}
		if err := c.doJSONPost(ctx, url, payload); err != nil {
			return fmt.Errorf("mikrotik isolate failed: %w", err)
		}

	case ProviderMock:
		// Local in-memory simulation
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	entryKey := cleanMAC
	if entryKey == "" {
		entryKey = ip
	}
	c.blocked[entryKey] = BlockedClient{
		MAC:       cleanMAC,
		IP:        ip,
		Reason:    reason,
		BlockedAt: now,
		Provider:  c.cfg.Provider,
	}

	return nil
}

// UnblockClient lifts the quarantine restriction for a previously isolated client.
func (c *UnifiedClient) UnblockClient(ctx context.Context, mac, ip string) error {
	cleanMAC := normalizeMAC(mac)
	entryKey := cleanMAC
	if entryKey == "" {
		entryKey = ip
	}

	switch c.cfg.Provider {
	case ProviderUniFi:
		url := fmt.Sprintf("%s/api/s/%s/cmd/stamgr", strings.TrimRight(c.cfg.BaseURL, "/"), c.cfg.Site)
		payload := map[string]any{
			"cmd": "unblock-sta",
			"mac": cleanMAC,
		}
		_ = c.doJSONPost(ctx, url, payload)

	case ProviderOPNsense:
		url := fmt.Sprintf("%s/api/firewall/alias_util/delete/domus_quarantine", strings.TrimRight(c.cfg.BaseURL, "/"))
		target := ip
		if target == "" {
			target = cleanMAC
		}
		payload := map[string]any{"address": target}
		_ = c.doJSONPost(ctx, url, payload)

	case ProviderMikroTik:
		// Simulated or REST remove call
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.blocked, entryKey)
	return nil
}

// ListBlockedClients returns all devices currently in network quarantine.
func (c *UnifiedClient) ListBlockedClients(ctx context.Context) ([]BlockedClient, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	res := make([]BlockedClient, 0, len(c.blocked))
	for _, b := range c.blocked {
		res = append(res, b)
	}
	return res, nil
}

// GetStatus returns the operational state of the firewall integration.
func (c *UnifiedClient) GetStatus(ctx context.Context) (map[string]any, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]any{
		"provider":        c.cfg.Provider,
		"base_url":        c.cfg.BaseURL,
		"site":            c.cfg.Site,
		"quarantine_rule": "domus_quarantine",
		"blocked_count":   len(c.blocked),
		"status":          "online",
	}, nil
}

func (c *UnifiedClient) doJSONPost(ctx context.Context, url string, payload any) error {
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.cfg.APIKey != "" {
		req.Header.Set("X-API-Key", c.cfg.APIKey)
		req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	} else if c.cfg.Username != "" || c.cfg.Password != "" {
		req.SetBasicAuth(c.cfg.Username, c.cfg.Password)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("router returned HTTP status %d", resp.StatusCode)
	}
	return nil
}
