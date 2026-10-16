package modbus_tcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/entity"
	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/plugins"
	"github.com/google/uuid"
)

// Ensure Adapter implements plugins.Adapter
var _ plugins.Adapter = (*Adapter)(nil)

// RegisterMapping maps a Modbus register or coil to a Domus Entity.
type RegisterMapping struct {
	EntityKey    string    `json:"entity_key"`
	Name         string    `json:"name"`
	FunctionCode byte      `json:"function_code"` // 0x01 (Coils), 0x03 (Holding), 0x04 (Input)
	Address      uint16    `json:"address"`
	DataType     DataType  `json:"data_type"`
	WordOrder    WordOrder `json:"word_order,omitempty"`
	Scale        float64   `json:"scale,omitempty"`
	Unit         string    `json:"unit,omitempty"`
	Writable     bool      `json:"writable"`
}

// Config defines the Modbus TCP integration settings.
type Config struct {
	IntegrationID string            `json:"integration_id"`
	Host          string            `json:"host"`
	Port          int               `json:"port"`
	UnitID        uint8             `json:"unit_id"`
	PollInterval  time.Duration     `json:"poll_interval"`
	Registers     []RegisterMapping `json:"registers"`
}

// Adapter bridges industrial Modbus TCP devices (inverters, heat pumps, power meters) into Domus.
type Adapter struct {
	cfg          Config
	client       Client
	entityRepo   entity.Repository
	bus          events.Bus
	logger       *slog.Logger
	mu           sync.Mutex
	running      bool
	cancelFn     context.CancelFunc
	unsubCommand func()
	lastValues   map[string]any
}

// NewAdapter constructs a new Modbus TCP Adapter.
func NewAdapter(
	cfg Config,
	client Client,
	entityRepo entity.Repository,
	bus events.Bus,
	logger *slog.Logger,
) *Adapter {
	if cfg.Port <= 0 {
		cfg.Port = 502
	}
	if cfg.UnitID == 0 {
		cfg.UnitID = 1
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 5 * time.Second
	}
	if logger == nil {
		logger = slog.Default()
	}

	return &Adapter{
		cfg:        cfg,
		client:     client,
		entityRepo: entityRepo,
		bus:        bus,
		logger:     logger.With("plugin", "modbus_tcp", "id", cfg.IntegrationID),
		lastValues: make(map[string]any),
	}
}

// ID returns the unique integration ID.
func (a *Adapter) ID() string {
	return a.cfg.IntegrationID
}

// Type returns the adapter type key.
func (a *Adapter) Type() string {
	return "modbus_tcp"
}

// HealthCheck verifies whether the Modbus TCP client is actively connected.
func (a *Adapter) HealthCheck(ctx context.Context) error {
	if a.client == nil || !a.client.IsConnected() {
		return fmt.Errorf("modbus_tcp: device %s:%d not connected", a.cfg.Host, a.cfg.Port)
	}
	return nil
}

// Start boots up the background polling routine and event bus command listener.
func (a *Adapter) Start(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.running {
		return nil
	}

	// 1. Attempt initial connection (non-fatal if temporarily offline, poller will retry)
	if err := a.client.Connect(ctx); err != nil {
		a.logger.Warn("initial modbus connection failed, will retry during poll", "err", err)
	}

	// 2. Ensure entities exist in repository
	if a.entityRepo != nil {
		for _, reg := range a.cfg.Registers {
			domain := "sensor"
			if reg.DataType == TypeBool {
				if reg.Writable {
					domain = "switch"
				} else {
					domain = "binary_sensor"
				}
			}

			ent := &entity.Entity{
				ID:                reg.EntityKey,
				EntityKey:         reg.EntityKey,
				Name:              reg.Name,
				Domain:            domain,
				UnitOfMeasurement: reg.Unit,
				Writable:          reg.Writable,
				State:             "null",
				Attributes:        "{}",
				Available:         a.client.IsConnected(),
				LastChangedAt:     time.Now().UTC(),
				LastUpdatedAt:     time.Now().UTC(),
			}
			_ = a.entityRepo.Create(ctx, ent)
		}
	}

	// 3. Subscribe to outbound commands
	if a.bus != nil {
		a.unsubCommand = a.bus.Subscribe(events.CommandRequested, a.handleOutboundCommand)
	}

	// 4. Start background poller
	pollCtx, cancel := context.WithCancel(context.Background())
	a.cancelFn = cancel
	a.running = true

	go a.pollLoop(pollCtx)

	a.logger.Info("modbus_tcp adapter started", "host", a.cfg.Host, "port", a.cfg.Port)
	return nil
}

// Stop shuts down the poller and closes the connection.
func (a *Adapter) Stop(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.running {
		return nil
	}

	a.running = false
	if a.cancelFn != nil {
		a.cancelFn()
	}
	if a.unsubCommand != nil {
		a.unsubCommand()
		a.unsubCommand = nil
	}

	if a.client != nil {
		_ = a.client.Close()
	}

	a.logger.Info("modbus_tcp adapter stopped")
	return nil
}

func (a *Adapter) pollLoop(ctx context.Context) {
	ticker := time.NewTicker(a.cfg.PollInterval)
	defer ticker.Stop()

	// Initial immediate poll
	a.pollOnce(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.pollOnce(ctx)
		}
	}
}

func (a *Adapter) pollOnce(ctx context.Context) {
	if !a.client.IsConnected() {
		if err := a.client.Connect(ctx); err != nil {
			a.logger.Debug("modbus reconnect attempt failed", "err", err)
			return
		}
	}

	for _, reg := range a.cfg.Registers {
		val, err := a.readRegister(ctx, reg)
		if err != nil {
			a.logger.Debug("read register failed", "entity", reg.EntityKey, "err", err)
			continue
		}

		a.processValue(ctx, reg, val)
	}
}

func (a *Adapter) readRegister(ctx context.Context, reg RegisterMapping) (any, error) {
	txID := a.client.NextTxID()
	var req []byte
	var qty uint16 = 1

	switch reg.DataType {
	case TypeInt32, TypeUint32, TypeFloat32:
		qty = 2
	}

	switch reg.FunctionCode {
	case FuncReadCoils:
		req = BuildReadCoils(txID, a.cfg.UnitID, reg.Address, 1)
	case FuncReadHoldingRegisters:
		req = BuildReadHoldingRegisters(txID, a.cfg.UnitID, reg.Address, qty)
	case FuncReadInputRegisters:
		req = BuildReadInputRegisters(txID, a.cfg.UnitID, reg.Address, qty)
	default:
		return nil, fmt.Errorf("unsupported function code 0x%02X", reg.FunctionCode)
	}

	respPDU, err := a.client.RoundTrip(ctx, req)
	if err != nil {
		return nil, err
	}

	dataBytes, err := ParseReadResponse(reg.FunctionCode, respPDU)
	if err != nil {
		return nil, err
	}

	return DecodeRegisterValue(dataBytes, reg.DataType, reg.WordOrder, reg.Scale)
}

func (a *Adapter) processValue(ctx context.Context, reg RegisterMapping, val any) {
	a.mu.Lock()
	prevVal, exists := a.lastValues[reg.EntityKey]
	a.lastValues[reg.EntityKey] = val
	a.mu.Unlock()

	stateStr := fmt.Sprintf("%v", val)
	if b, ok := val.(bool); ok {
		if b {
			stateStr = "on"
		} else {
			stateStr = "off"
		}
	}

	stateJSON, _ := json.Marshal(stateStr)

	// Update entity repository
	if a.entityRepo != nil {
		_ = a.entityRepo.UpdateState(ctx, reg.EntityKey, string(stateJSON), "{}", true, time.Now().UTC())
	}

	// Publish state change if changed or first sample
	if !exists || prevVal != val {
		if a.bus != nil {
			_ = a.bus.Publish(ctx, events.Event{
				ID:         uuid.NewString(),
				Type:       events.EntityStateChanged,
				OccurredAt: time.Now().UTC(),
				Source:     "modbus_tcp",
				Payload: map[string]any{
					"entity_id": reg.EntityKey,
					"old_state": prevVal,
					"new_state": val,
					"unit":      reg.Unit,
					"last_seen": time.Now().UTC().Format(time.RFC3339),
				},
			})
		}
	}
}

func (a *Adapter) handleOutboundCommand(ctx context.Context, ev events.Event) error {
	payloadMap, ok := ev.Payload.(map[string]any)
	if !ok {
		return nil
	}

	entityID, _ := payloadMap["entity_id"].(string)
	service, _ := payloadMap["service"].(string)

	var targetReg *RegisterMapping
	for _, reg := range a.cfg.Registers {
		if reg.EntityKey == entityID && reg.Writable {
			targetReg = &reg
			break
		}
	}

	if targetReg == nil {
		return nil // Not handled by this adapter
	}

	txID := a.client.NextTxID()

	if targetReg.FunctionCode == FuncReadCoils || targetReg.DataType == TypeBool {
		on := (service == "turn_on")
		req := BuildWriteSingleCoil(txID, a.cfg.UnitID, targetReg.Address, on)
		_, err := a.client.RoundTrip(ctx, req)
		if err != nil {
			a.logger.Error("write coil failed", "entity", targetReg.EntityKey, "err", err)
			return err
		}
		a.processValue(ctx, *targetReg, on)
		return nil
	}

	if valNum, ok := payloadMap["value"].(float64); ok {
		rawVal := uint16(valNum)
		if targetReg.Scale > 0 {
			rawVal = uint16(valNum / targetReg.Scale)
		}
		req := BuildWriteSingleRegister(txID, a.cfg.UnitID, targetReg.Address, rawVal)
		_, err := a.client.RoundTrip(ctx, req)
		if err != nil {
			a.logger.Error("write register failed", "entity", targetReg.EntityKey, "err", err)
			return err
		}
		a.processValue(ctx, *targetReg, valNum)
	}

	return nil
}
