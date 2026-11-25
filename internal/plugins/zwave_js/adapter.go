package zwave_js

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/google/uuid"
)

const (
	CCBinarySwitch     = 37
	CCMultilevelSwitch = 38
	CCSensorMultilevel = 49
	CCMeter            = 50
	CCNotification     = 113
)

// Config holds one Z-Wave JS UI gateway instance.
type Config struct {
	ID                string
	WebSocketURL      string
	AuthToken         string
	ReconnectInterval time.Duration
	Logger            *slog.Logger
}

type Adapter struct {
	id     string
	wsURL  string
	bus    events.Bus
	client *Client
	logger *slog.Logger

	mu        sync.RWMutex
	active    bool
	cancel    context.CancelFunc
	done      chan struct{}
	unsubBus  func()
	nodes     map[int]*ZWaveNode
	reconnect time.Duration
}

func NewAdapter(id string, wsURL string, bus events.Bus) *Adapter {
	adapter, err := NewAdapterWithConfig(Config{ID: id, WebSocketURL: wsURL}, bus)
	if err != nil {
		// Preserve the historical constructor signature for callers. The factory
		// path uses NewAdapterWithConfig and returns validation errors to the UI.
		if id == "" { id = "zwave_js" }
		return &Adapter{id: id, wsURL: wsURL, bus: bus, client: NewClient(wsURL, nil), logger: slog.Default(), reconnect: 5 * time.Second, nodes: make(map[int]*ZWaveNode)}
	}
	return adapter
}

func NewAdapterWithConfig(cfg Config, bus events.Bus) (*Adapter, error) {
	cfg.ID = strings.TrimSpace(cfg.ID)
	if cfg.ID == "" { cfg.ID = "zwave_js" }
	rawURL := strings.TrimSpace(cfg.WebSocketURL)
	if rawURL == "" { return nil, errors.New("zwave_js: ws_url is required") }
	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "ws" && parsed.Scheme != "wss") || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" {
		return nil, errors.New("zwave_js: ws_url must be an absolute ws:// or wss:// URL without userinfo or fragment")
	}
	if cfg.ReconnectInterval == 0 { cfg.ReconnectInterval = 5 * time.Second }
	if cfg.ReconnectInterval < time.Second || cfg.ReconnectInterval > 5*time.Minute {
		return nil, errors.New("zwave_js: reconnect_interval must be between 1s and 5m")
	}
	if cfg.Logger == nil { cfg.Logger = slog.Default() }
	client := NewClient(rawURL, nil)
	client.SetAuthToken(strings.TrimSpace(cfg.AuthToken))
	a := &Adapter{
		id: cfg.ID, wsURL: rawURL, bus: bus, client: client,
		logger: cfg.Logger, reconnect: cfg.ReconnectInterval, nodes: make(map[int]*ZWaveNode),
	}
	client.handler = a.handleValueUpdated
	return a, nil
}

func (a *Adapter) ID() string   { return a.id }
func (a *Adapter) Type() string { return "zwave_js" }

func (a *Adapter) Start(parent context.Context) error {
	a.mu.Lock()
	if a.active {
		a.mu.Unlock()
		return nil
	}
	ctx, cancel := context.WithCancel(parent)
	a.cancel = cancel
	a.done = make(chan struct{})
	a.active = true
	if a.bus != nil {
		a.unsubBus = a.bus.Subscribe(events.CommandRequested, a.handleCommand)
	}
	done := a.done
	a.mu.Unlock()
	go a.reconnectLoop(ctx, done)
	a.logger.Info("zwave_js: adapter started", "integration_id", a.id, "ws_url", a.wsURL)
	return nil
}

func (a *Adapter) reconnectLoop(ctx context.Context, done chan struct{}) {
	defer close(done)
	for {
		if err := ctx.Err(); err != nil { return }
		if err := a.client.Connect(ctx); err != nil {
			if ctx.Err() != nil { return }
			a.logger.Warn("zwave_js: gateway connection failed; retry scheduled", "integration_id", a.id, "error", err)
		} else {
			a.logger.Info("zwave_js: gateway connected", "integration_id", a.id)
			ticker := time.NewTicker(500 * time.Millisecond)
			connected := true
			for connected {
				select {
				case <-ctx.Done():
					ticker.Stop()
					return
				case <-ticker.C:
					connected = a.client.Connected()
				}
			}
			ticker.Stop()
			if ctx.Err() != nil { return }
			a.logger.Warn("zwave_js: gateway disconnected; reconnecting", "integration_id", a.id)
		}
		timer := time.NewTimer(a.reconnect)
		select {
		case <-ctx.Done():
			if !timer.Stop() { <-timer.C }
			return
		case <-timer.C:
		}
	}
}

func (a *Adapter) HealthCheck(ctx context.Context) error {
	if err := ctx.Err(); err != nil { return err }
	a.mu.RLock()
	active := a.active
	a.mu.RUnlock()
	if !active { return errors.New("zwave_js: adapter is stopped") }
	if !a.client.Connected() { return errors.New("zwave_js: gateway is disconnected") }
	return nil
}

func (a *Adapter) Stop(ctx context.Context) error {
	a.mu.Lock()
	if !a.active {
		a.mu.Unlock()
		return nil
	}
	a.active = false
	cancel, done, unsubscribe := a.cancel, a.done, a.unsubBus
	a.cancel, a.done, a.unsubBus = nil, nil, nil
	a.mu.Unlock()
	if cancel != nil { cancel() }
	if unsubscribe != nil { unsubscribe() }
	closeErr := a.client.Close()
	if done != nil {
		select {
		case <-done:
		case <-ctx.Done():
			if closeErr != nil { return closeErr }
			return ctx.Err()
		}
	}
	return closeErr
}

// ProcessValueUpdate routes incoming CC property updates to Domus entities.
func (a *Adapter) ProcessValueUpdate(ctx context.Context, nodeID int, cc int, prop string, val any) {
	a.handleValueUpdated(nodeID, cc, prop, val)
}

func (a *Adapter) entityPrefix() string {
	if a.id == "zwave_js" || a.id == "zwave" { return "zwave" }
	return "zwave_" + a.id
}

func (a *Adapter) handleValueUpdated(nodeID int, cc int, prop string, val any) {
	if a.bus == nil { return }
	ctx := context.Background()
	now := time.Now().UTC()
	prefix := a.entityPrefix()
	source := a.id

	switch cc {
	case CCBinarySwitch:
		state := "off"
		if boolVal, ok := val.(bool); ok && boolVal { state = "on" }
		_ = a.bus.Publish(ctx, events.Event{
			ID: uuid.NewString(), Type: events.EntityStateChanged, OccurredAt: now, Source: source,
			Payload: map[string]any{"entity_key": fmt.Sprintf("switch.%s_node_%d", prefix, nodeID), "state": state, "available": true},
		})
	case CCMultilevelSwitch:
		numVal, _ := val.(float64)
		state := "off"
		if numVal > 0 { state = "on" }
		attrs, _ := json.Marshal(map[string]any{"level": int(numVal)})
		_ = a.bus.Publish(ctx, events.Event{
			ID: uuid.NewString(), Type: events.EntityStateChanged, OccurredAt: now, Source: source,
			Payload: map[string]any{"entity_key": fmt.Sprintf("light.%s_node_%d", prefix, nodeID), "state": state, "attributes": string(attrs), "available": true},
		})
	case CCSensorMultilevel:
		numVal, ok := val.(float64)
		if !ok { return }
		entityKey := fmt.Sprintf("sensor.%s_node_%d_%s", prefix, nodeID, safeProperty(prop))
		_ = a.bus.Publish(ctx, events.Event{
			ID: uuid.NewString(), Type: events.EntityStateChanged, OccurredAt: now, Source: source,
			Payload: map[string]any{"entity_key": entityKey, "state": fmt.Sprintf("%.2f", numVal), "available": true},
		})
	case CCMeter:
		numVal, ok := val.(float64)
		if !ok { return }
		entityKey := fmt.Sprintf("sensor.%s_node_%d_power", prefix, nodeID)
		if strings.Contains(strings.ToLower(prop), "energy") || strings.Contains(strings.ToLower(prop), "kwh") {
			entityKey = fmt.Sprintf("sensor.%s_node_%d_energy", prefix, nodeID)
		}
		_ = a.bus.Publish(ctx, events.Event{
			ID: uuid.NewString(), Type: events.EntityStateChanged, OccurredAt: now, Source: source,
			Payload: map[string]any{"entity_key": entityKey, "state": fmt.Sprintf("%.1f", numVal), "available": true},
		})
	}
}

func safeProperty(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	var b strings.Builder
	for _, r := range raw {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_' { b.WriteRune(r) } else if b.Len() > 0 { b.WriteByte('_') }
	}
	return strings.Trim(b.String(), "_")
}

func (a *Adapter) handleCommand(ctx context.Context, e events.Event) error {
	payload, ok := e.Payload.(map[string]any)
	if !ok { return nil }
	entityKey, _ := payload["entity_key"].(string)
	prefix := a.entityPrefix()
	marker := prefix + "_node_"
	if !strings.Contains(entityKey, "."+marker) { return nil }
	parts := strings.SplitN(entityKey, "."+marker, 2)
	if len(parts) != 2 { return nil }
	nodeID, err := strconv.Atoi(strings.SplitN(parts[1], "_", 2)[0])
	if err != nil || nodeID <= 0 { return nil }
	cmd, _ := payload["command"].(string)
	if cmd == "" { cmd, _ = payload["service"].(string) }
	if strings.HasPrefix(entityKey, "switch.") {
		return a.client.SetValue(ctx, nodeID, CCBinarySwitch, "currentValue", cmd == "turn_on")
	}
	if strings.HasPrefix(entityKey, "light.") {
		level := 0
		if cmd == "turn_on" {
			level = 99
			if params, ok := payload["params"].(map[string]any); ok {
				if brightness, ok := params["brightness"].(float64); ok { level = int(brightness * 99.0 / 255.0) }
			}
		}
		return a.client.SetValue(ctx, nodeID, CCMultilevelSwitch, "currentValue", level)
	}
	return nil
}
