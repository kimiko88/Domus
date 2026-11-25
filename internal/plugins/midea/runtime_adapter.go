package midea

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/domus-hub/domus/internal/plugins"
)

// RuntimeAdapter brings the M-Smart protocol engine under the common Domus
// plugin lifecycle. Credentials and device mappings remain encrypted by the
// Midea integration handler; the runtime only owns the reusable protocol drivers.
type RuntimeAdapter struct {
	id string
	core *Adapter
	v2 *A1LANDriver
	v3 *A1LANDriver
	mu sync.RWMutex
	active bool
}

var _ plugins.Adapter = (*RuntimeAdapter)(nil)

func NewRuntimeAdapter(id string)(*RuntimeAdapter,error){
	id=strings.TrimSpace(id);if id==""{return nil,errors.New("midea: plugin instance ID is required")}
	v2,err:=NewA1LANDriver(ProtocolMSmartV2);if err!=nil{return nil,err}
	v3,err:=NewA1LANDriver(ProtocolMSmartV3);if err!=nil{return nil,err}
	return &RuntimeAdapter{id:id,core:NewAdapter(v2,v3),v2:v2,v3:v3},nil
}
func(a *RuntimeAdapter)ID()string{return a.id}
func(a *RuntimeAdapter)Type()string{return "midea_a1_protocol"}
func(a *RuntimeAdapter)Start(ctx context.Context)error{if err:=ctx.Err();err!=nil{return err};a.mu.Lock();a.active=true;a.mu.Unlock();return nil}
func(a *RuntimeAdapter)Stop(context.Context)error{a.mu.Lock();a.active=false;a.mu.Unlock();return nil}
func(a *RuntimeAdapter)HealthCheck(ctx context.Context)error{if err:=ctx.Err();err!=nil{return err};a.mu.RLock();active:=a.active;a.mu.RUnlock();if !active{return errors.New("midea: protocol provider is stopped")};return nil}
func(a *RuntimeAdapter)Core()(*Adapter,error){a.mu.RLock();defer a.mu.RUnlock();if !a.active{return nil,errors.New("midea: protocol provider is stopped")};return a.core,nil}
func(a *RuntimeAdapter)Drivers()(*A1LANDriver,*A1LANDriver){return a.v2,a.v3}
