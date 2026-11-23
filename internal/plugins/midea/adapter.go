package midea

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// Driver implements one concrete Midea/Tuya transport protocol. Implementations
// must not retry a physical write after a timeout or an uncertain response.
type Driver interface {
	Protocol() Protocol
	Discover(context.Context) ([]Device, error)
	Read(context.Context, Device) (map[string]any, error)
	Execute(context.Context, Device, Command) error
}

type Adapter struct {
	mu      sync.RWMutex
	drivers map[Protocol]Driver
	devices map[string]Device
	locks   map[string]*sync.Mutex
}

func NewAdapter(drivers ...Driver) *Adapter {
	adapter := &Adapter{
		drivers: make(map[Protocol]Driver),
		devices: make(map[string]Device),
		locks:   make(map[string]*sync.Mutex),
	}
	for _, driver := range drivers {
		if driver != nil && driver.Protocol() != "" {
			adapter.drivers[driver.Protocol()] = driver
		}
	}
	return adapter
}

func (a *Adapter) Discover(ctx context.Context) ([]Device, error) {
	a.mu.RLock()
	type driverEntry struct {
		protocol Protocol
		driver   Driver
	}
	drivers := make([]driverEntry, 0, len(a.drivers))
	for protocol, driver := range a.drivers {
		drivers = append(drivers, driverEntry{protocol: protocol, driver: driver})
	}
	a.mu.RUnlock()
	sort.Slice(drivers, func(i, j int) bool { return drivers[i].protocol < drivers[j].protocol })

	found := make(map[string]Device)
	for _, entry := range drivers {
		driver := entry.driver
		devices, err := driver.Discover(ctx)
		if err != nil {
			return nil, fmt.Errorf("midea: discovery via %s: %w", driver.Protocol(), err)
		}
		for _, device := range devices {
			if device.Protocol == "" {
				device.Protocol = driver.Protocol()
			}
			if device.Protocol != driver.Protocol() {
				return nil, fmt.Errorf("midea: driver %s returned device for protocol %s", driver.Protocol(), device.Protocol)
			}
			if err := device.Validate(); err != nil {
				return nil, fmt.Errorf("midea: rejected discovered device %q: %w", device.ID, err)
			}
			if previous, exists := found[device.ID]; exists && previous.Protocol != device.Protocol {
				return nil, fmt.Errorf("midea: device id %q was discovered over conflicting protocols %s and %s", device.ID, previous.Protocol, device.Protocol)
			}
			found[device.ID] = cloneDevice(device)
		}
	}

	result := make([]Device, 0, len(found))
	for _, device := range found {
		result = append(result, device)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })

	a.mu.Lock()
	for id, device := range found {
		a.devices[id] = cloneDevice(device)
	}
	a.mu.Unlock()
	return result, nil
}

func (a *Adapter) Register(device Device) error {
	if err := device.Validate(); err != nil {
		return err
	}
	a.mu.RLock()
	_, ok := a.drivers[device.Protocol]
	a.mu.RUnlock()
	if !ok {
		return ErrNoDriver
	}
	a.mu.Lock()
	a.devices[device.ID] = cloneDevice(device)
	a.mu.Unlock()
	return nil
}

func (a *Adapter) ListDevices() []Device {
	a.mu.RLock()
	defer a.mu.RUnlock()
	result := make([]Device, 0, len(a.devices))
	for _, device := range a.devices {
		result = append(result, cloneDevice(device))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func (a *Adapter) Refresh(ctx context.Context, id string) (map[string]any, error) {
	unlock, device, driver, err := a.lockedDevice(id)
	if err != nil {
		return nil, err
	}
	defer unlock()

	state, err := driver.Read(ctx, device)
	if err != nil {
		return nil, fmt.Errorf("midea: read %s: %w", id, err)
	}
	device.State = cloneState(state)
	device.Online = true
	a.mu.Lock()
	a.devices[id] = device
	a.mu.Unlock()
	return cloneState(state), nil
}

// Command writes once, then verifies the result with a separate state read.
// A failed or ambiguous write is never replayed by this adapter.
func (a *Adapter) Command(ctx context.Context, id string, command Command) (map[string]any, error) {
	unlock, device, driver, err := a.lockedDevice(id)
	if err != nil {
		return nil, err
	}
	defer unlock()

	if err := ValidateCommand(device, command); err != nil {
		return nil, err
	}
	if err := driver.Execute(ctx, device, command); err != nil {
		return nil, fmt.Errorf("midea: command delivery to %s is uncertain; it will not be retried: %w", id, err)
	}

	state, err := driver.Read(ctx, device)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrReadBackUnavailable, err)
	}
	for field, expected := range command.Values {
		actual, exists := state[field]
		if !exists || !valuesEqual(expected, actual) {
			return cloneState(state), fmt.Errorf("%w: %s", ErrReadBackMismatch, field)
		}
	}

	device.State = cloneState(state)
	device.Online = true
	a.mu.Lock()
	a.devices[id] = device
	a.mu.Unlock()
	return cloneState(state), nil
}

func (a *Adapter) lockedDevice(id string) (func(), Device, Driver, error) {
	a.mu.Lock()
	if _, ok := a.devices[id]; !ok {
		a.mu.Unlock()
		return nil, Device{}, nil, ErrDeviceNotFound
	}
	lock := a.locks[id]
	if lock == nil {
		lock = &sync.Mutex{}
		a.locks[id] = lock
	}
	a.mu.Unlock()

	lock.Lock()
	a.mu.RLock()
	device, ok := a.devices[id]
	driver := a.drivers[device.Protocol]
	a.mu.RUnlock()
	if !ok {
		lock.Unlock()
		return nil, Device{}, nil, ErrDeviceNotFound
	}
	if driver == nil {
		lock.Unlock()
		return nil, Device{}, nil, ErrNoDriver
	}
	return lock.Unlock, cloneDevice(device), driver, nil
}

func cloneDevice(device Device) Device {
	device.Capabilities = append([]Capability(nil), device.Capabilities...)
	device.Writable = append([]Capability(nil), device.Writable...)
	device.Modes = append([]string(nil), device.Modes...)
	device.Programs = append([]string(nil), device.Programs...)
	device.FanSpeeds = append([]string(nil), device.FanSpeeds...)
	device.Actions = append([]string(nil), device.Actions...)
	device.State = cloneState(device.State)
	return device
}

func cloneState(state map[string]any) map[string]any {
	if state == nil {
		return nil
	}
	clone := make(map[string]any, len(state))
	for key, value := range state {
		clone[key] = value
	}
	return clone
}
