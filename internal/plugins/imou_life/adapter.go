package imou_life

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/plugins"
)

// Adapter is the supervised Imou Life provider. It owns the bounded HTTP
// transport; OAuth/user tokens remain encrypted by the integration service.
type Adapter struct {
	id string
	http *http.Client
	mu sync.RWMutex
	active bool
}

var _ plugins.Adapter = (*Adapter)(nil)

func NewAdapter(id string, timeout time.Duration) (*Adapter, error) {
	id = strings.TrimSpace(id)
	if id == "" { return nil, errors.New("imou: plugin instance ID is required") }
	if timeout <= 0 { timeout = 12 * time.Second }
	if timeout > 20*time.Second { return nil, errors.New("imou: HTTP timeout cannot exceed 20s") }
	return &Adapter{id:id, http:&http.Client{Timeout:timeout}}, nil
}

func (a *Adapter) ID() string { return a.id }
func (a *Adapter) Type() string { return "imou_life" }
func (a *Adapter) Start(ctx context.Context) error {
	if err:=ctx.Err(); err!=nil{return err}
	a.mu.Lock();a.active=true;a.mu.Unlock();return nil
}
func (a *Adapter) Stop(context.Context) error { a.mu.Lock();a.active=false;a.mu.Unlock();return nil }
func (a *Adapter) HealthCheck(ctx context.Context) error {
	if err:=ctx.Err();err!=nil{return err}
	a.mu.RLock();active:=a.active;a.mu.RUnlock()
	if !active{return errors.New("imou: provider is stopped")}
	return nil
}

// NewClient creates a credential-scoped client using the adapter's guarded
// HTTP transport. The caller must never log the credentials or returned tokens.
func (a *Adapter) NewClient(cfg Config) (*Client,error) {
	a.mu.RLock();active:=a.active;client:=a.http;a.mu.RUnlock()
	if !active{return nil,errors.New("imou: provider is stopped")}
	return NewClient(cfg,client)
}
