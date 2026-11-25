package vesync

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/plugins"
)

// Adapter provides one managed VeSync cloud transport. Account credentials and
// refreshable sessions continue to be encrypted in the integration repository.
type Adapter struct {
	id string
	client *Client
	mu sync.RWMutex
	active bool
}

var _ plugins.Adapter = (*Adapter)(nil)

func NewAdapter(id string, timeout time.Duration) (*Adapter,error) {
	id = strings.TrimSpace(id)
	if id == "" { return nil,errors.New("vesync: plugin instance ID is required") }
	if timeout<=0 { timeout=8*time.Second }
	if timeout>20*time.Second { return nil,errors.New("vesync: HTTP timeout cannot exceed 20s") }
	return &Adapter{id:id,client:NewClient(timeout)},nil
}
func (a *Adapter) ID() string{return a.id}
func (a *Adapter) Type() string{return "vesync_transport"}
func (a *Adapter) Start(ctx context.Context)error{if err:=ctx.Err();err!=nil{return err};a.mu.Lock();a.active=true;a.mu.Unlock();return nil}
func (a *Adapter) Stop(context.Context)error{a.mu.Lock();a.active=false;a.mu.Unlock();return nil}
func (a *Adapter) HealthCheck(ctx context.Context)error{if err:=ctx.Err();err!=nil{return err};a.mu.RLock();active:=a.active;a.mu.RUnlock();if !active{return errors.New("vesync: provider is stopped")};return nil}
func (a *Adapter) Client()(*Client,error){a.mu.RLock();defer a.mu.RUnlock();if !a.active{return nil,errors.New("vesync: provider is stopped")};return a.client,nil}
