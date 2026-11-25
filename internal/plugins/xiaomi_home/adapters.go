package xiaomi_home

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/plugins"
)

// LocalAdapter is the supervised miIO LAN transport provider.
type LocalAdapter struct {
	id string
	client *Client
	mu sync.RWMutex
	active bool
}

var _ plugins.Adapter = (*LocalAdapter)(nil)

func NewLocalAdapter(id string, timeout time.Duration)(*LocalAdapter,error){
	id=strings.TrimSpace(id);if id==""{return nil,errors.New("xiaomi: local plugin instance ID is required")}
	if timeout<500*time.Millisecond||timeout>10*time.Second{timeout=2*time.Second}
	return &LocalAdapter{id:id,client:NewClient(timeout)},nil
}
func(a *LocalAdapter)ID()string{return a.id}
func(a *LocalAdapter)Type()string{return "xiaomi_miio_transport"}
func(a *LocalAdapter)Start(ctx context.Context)error{if err:=ctx.Err();err!=nil{return err};a.mu.Lock();a.active=true;a.mu.Unlock();return nil}
func(a *LocalAdapter)Stop(context.Context)error{a.mu.Lock();a.active=false;a.mu.Unlock();return nil}
func(a *LocalAdapter)HealthCheck(ctx context.Context)error{if err:=ctx.Err();err!=nil{return err};a.mu.RLock();active:=a.active;a.mu.RUnlock();if !active{return errors.New("xiaomi: local provider is stopped")};return nil}
func(a *LocalAdapter)Client()(*Client,error){a.mu.RLock();defer a.mu.RUnlock();if !a.active{return nil,errors.New("xiaomi: local provider is stopped")};return a.client,nil}

// CloudAdapter owns the bounded HTTP transport for Xiaomi Open Platform.
// OAuth tokens and app secrets remain under the encrypted integration config.
type CloudAdapter struct {
	id string
	http *http.Client
	mu sync.RWMutex
	active bool
}
var _ plugins.Adapter = (*CloudAdapter)(nil)
func NewCloudAdapter(id string,timeout time.Duration)(*CloudAdapter,error){
	id=strings.TrimSpace(id);if id==""{return nil,errors.New("xiaomi: cloud plugin instance ID is required")}
	if timeout<=0{timeout=12*time.Second};if timeout>20*time.Second{return nil,errors.New("xiaomi: cloud HTTP timeout cannot exceed 20s")}
	return &CloudAdapter{id:id,http:&http.Client{Timeout:timeout}},nil
}
func(a *CloudAdapter)ID()string{return a.id}
func(a *CloudAdapter)Type()string{return "xiaomi_open_platform_transport"}
func(a *CloudAdapter)Start(ctx context.Context)error{if err:=ctx.Err();err!=nil{return err};a.mu.Lock();a.active=true;a.mu.Unlock();return nil}
func(a *CloudAdapter)Stop(context.Context)error{a.mu.Lock();a.active=false;a.mu.Unlock();return nil}
func(a *CloudAdapter)HealthCheck(ctx context.Context)error{if err:=ctx.Err();err!=nil{return err};a.mu.RLock();active:=a.active;a.mu.RUnlock();if !active{return errors.New("xiaomi: cloud provider is stopped")};return nil}
func(a *CloudAdapter)NewClient(cfg CloudConfig)(*CloudClient,error){a.mu.RLock();active,client:=a.active,a.http;a.mu.RUnlock();if !active{return nil,errors.New("xiaomi: cloud provider is stopped")};return NewCloudClient(cfg,client)}
