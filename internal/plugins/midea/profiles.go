package midea

import (
	"errors"
	"fmt"
	"strings"
	"sync"
)

// Profile is a model-family baseline. Drivers must intersect it with the
// capability report / appliance thing-model before exposing controls.
type Profile struct {
	Kind             ApplianceKind
	Capabilities     []Capability
	Writable         []Capability
	Modes            []string
	Programs         []string
	FanSpeeds        []string
	Actions          []string
	MinTemperature   float64
	MaxTemperature   float64
	TemperatureStep  float64
	MinHumidity      float64
	MaxHumidity      float64
}

type Registry struct {
	mu       sync.RWMutex
	profiles map[string]Profile
}

func NewRegistry() *Registry {
	registry := &Registry{profiles: make(map[string]Profile)}
	_ = registry.Register("A1", Profile{
		Kind: KindDehumidifier,
		Capabilities: []Capability{
			CapPower, CapMode, CapTargetHumidity, CapCurrentHumidity,
			CapCurrentTemperature, CapFanSpeed, CapWaterTankFull, CapSwing,
			CapChildLock, CapAnion, CapPump, CapWaterLevelSet, CapFilterReminder,
		},
		Writable: []Capability{
			CapPower, CapMode, CapTargetHumidity, CapFanSpeed, CapSwing,
			CapChildLock, CapAnion, CapPump, CapWaterLevelSet,
		},
		Modes: []string{"manual", "continuous", "auto", "clothes_dry", "shoes_dry", "silent"},
		FanSpeeds: []string{"lowest", "low", "medium", "high", "auto", "off"},
		Actions: []string{"reset_filter"},
		MinHumidity: 35, MaxHumidity: 100,
	})
	_ = registry.Register("AC", Profile{
		Kind: KindAirConditioner,
		Capabilities: []Capability{
			CapPower, CapMode, CapTargetTemperature, CapCurrentTemperature,
			CapCurrentHumidity, CapFanSpeed, CapSwing, CapEnergy, CapFilterLife, CapErrorCode,
		},
		Writable: []Capability{CapPower, CapMode, CapTargetTemperature, CapFanSpeed, CapSwing},
	})
	return registry
}

// Register adds or replaces a type-code profile. Type codes are case-insensitive.
func (r *Registry) Register(typeCode string, profile Profile) error {
	if r == nil {
		return errors.New("midea: nil profile registry")
	}
	code := strings.ToUpper(strings.TrimSpace(typeCode))
	if code == "" || len(code) > 32 || profile.Kind == "" {
		return errors.New("midea: profile type code and appliance kind are required")
	}
	for _, capability := range profile.Writable {
		if !containsCapability(profile.Capabilities, capability) {
			return fmt.Errorf("midea: profile %q marks unadvertised capability %q writable", code, capability)
		}
	}
	if profile.MinHumidity < 0 || profile.MaxHumidity < 0 || profile.MaxHumidity > 100 ||
		(profile.MaxHumidity > 0 && profile.MinHumidity > profile.MaxHumidity) {
		return fmt.Errorf("midea: profile %q has invalid humidity limits", code)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.profiles == nil {
		r.profiles = make(map[string]Profile)
	}
	r.profiles[code] = cloneProfile(profile)
	return nil
}

func (r *Registry) Resolve(typeCode string) (Profile, bool) {
	if r == nil {
		return Profile{}, false
	}
	r.mu.RLock()
	profile, ok := r.profiles[strings.ToUpper(strings.TrimSpace(typeCode))]
	r.mu.RUnlock()
	if !ok {
		return Profile{}, false
	}
	return cloneProfile(profile), true
}

func (r *Registry) Apply(device *Device) bool {
	if device == nil {
		return false
	}
	profile, ok := r.Resolve(device.TypeCode)
	if !ok {
		return false
	}
	applyProfile(device, profile)
	return true
}

// ApplyReported applies a model profile but narrows its capabilities to the
// features actually reported by the connected appliance. It never adds a
// capability the device did not advertise.
func (r *Registry) ApplyReported(device *Device, reported []Capability) bool {
	if !r.Apply(device) {
		return false
	}
	allowed := make(map[Capability]struct{}, len(reported))
	for _, capability := range reported {
		allowed[capability] = struct{}{}
	}
	device.Capabilities = intersectCapabilities(device.Capabilities, allowed)
	device.Writable = intersectCapabilities(device.Writable, allowed)
	return true
}

func applyProfile(device *Device, profile Profile) {
	device.Kind = profile.Kind
	device.Capabilities = append([]Capability(nil), profile.Capabilities...)
	device.Writable = append([]Capability(nil), profile.Writable...)
	device.Modes = append([]string(nil), profile.Modes...)
	device.Programs = append([]string(nil), profile.Programs...)
	device.FanSpeeds = append([]string(nil), profile.FanSpeeds...)
	device.Actions = append([]string(nil), profile.Actions...)
	device.MinTemperature = profile.MinTemperature
	device.MaxTemperature = profile.MaxTemperature
	device.TemperatureStep = profile.TemperatureStep
	device.MinHumidity = profile.MinHumidity
	device.MaxHumidity = profile.MaxHumidity
}

func intersectCapabilities(values []Capability, allowed map[Capability]struct{}) []Capability {
	result := make([]Capability, 0, len(values))
	for _, value := range values {
		if _, ok := allowed[value]; ok {
			result = append(result, value)
		}
	}
	return result
}

func cloneProfile(profile Profile) Profile {
	profile.Capabilities = append([]Capability(nil), profile.Capabilities...)
	profile.Writable = append([]Capability(nil), profile.Writable...)
	profile.Modes = append([]string(nil), profile.Modes...)
	profile.Programs = append([]string(nil), profile.Programs...)
	profile.FanSpeeds = append([]string(nil), profile.FanSpeeds...)
	profile.Actions = append([]string(nil), profile.Actions...)
	return profile
}
