package vesync

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/plugins"
)

const (
	PluginType = "vesync_cloud"
	IntegrationID = "integration_vesync_cloud"
)

// Adapter owns the VeSync cloud client for the single account integration.
// Its session is persisted only in the encrypted Plugin Manager configuration.
type Adapter struct {
	id      string
	client  *Client
	mu      sync.RWMutex
	active  bool
}

var _ plugins.Adapter = (*Adapter)(nil)

func NewAdapter(id string, timeout time.Duration) (*Adapter, error) {
	return NewAdapterWithSession(id, Session{}, timeout)
}

func NewAdapterWithSession(id string, session Session, timeout time.Duration) (*Adapter, error) {
	id = strings.TrimSpace(id)
	if id != IntegrationID {
		return nil, errors.New("vesync: plugin instance ID must be integration_vesync_cloud")
	}
	if timeout <= 0 {
		timeout = 8 * time.Second
	}
	if timeout < time.Second || timeout > 20*time.Second {
		return nil, errors.New("vesync: HTTP timeout must be between 1s and 20s")
	}
	if err := validateSession(session); err != nil {
		return nil, err
	}
	return &Adapter{id: id, client: NewClient(timeout)}, nil
}

func validateSession(session Session) error {
	if len(session.AccountID) > 256 || len(session.Token) > 8192 {
		return errors.New("vesync: stored session exceeds supported limits")
	}
	if (session.AccountID == "") != (session.Token == "") {
		return errors.New("vesync: stored session must contain both account ID and token")
	}
	if session.Token == "" {
		if session.CountryCode != "" || session.Region != "" || session.TimeZone != "" {
			return errors.New("vesync: disconnected session contains partial account metadata")
		}
		return nil
	}
	if session.Region != "EU" && session.Region != "US" {
		return errors.New("vesync: stored session has an unsupported cloud region")
	}
	if session.CountryCode != "" {
		country := strings.ToUpper(strings.TrimSpace(session.CountryCode))
		if len(country) != 2 || country != session.CountryCode {
			return errors.New("vesync: stored session has an invalid country code")
		}
	}
	if session.TimeZone != "" {
		if _, err := time.LoadLocation(session.TimeZone); err != nil {
			return errors.New("vesync: stored session has an invalid time zone")
		}
	}
	return nil
}

func (a *Adapter) ID() string   { return a.id }
func (a *Adapter) Type() string { return PluginType }

func (a *Adapter) Start(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	a.mu.Lock()
	a.active = true
	a.mu.Unlock()
	return nil
}

func (a *Adapter) Stop(context.Context) error {
	a.mu.Lock()
	a.active = false
	a.mu.Unlock()
	return nil
}

func (a *Adapter) HealthCheck(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	a.mu.RLock()
	active := a.active
	a.mu.RUnlock()
	if !active {
		return errors.New("vesync: provider is stopped")
	}
	return nil
}

func (a *Adapter) Client() (*Client, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if !a.active {
		return nil, errors.New("vesync: provider is stopped")
	}
	return a.client, nil
}
