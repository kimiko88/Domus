package xiaomi_home

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
)

// MIoTDeviceSpec is the public MIoT-Spec-V2 instance document returned for a device URN.
type MIoTDeviceSpec struct {
	Type        string          `json:"type"`
	Description string          `json:"description,omitempty"`
	Services    []MIoTService   `json:"services"`
}

type MIoTService struct {
	IID         int               `json:"iid"`
	Type        string            `json:"type"`
	Description string            `json:"description,omitempty"`
	Properties  []MIoTProperty    `json:"properties,omitempty"`
	Actions     []MIoTAction      `json:"actions,omitempty"`
	Events      []MIoTEvent       `json:"events,omitempty"`
}

type MIoTProperty struct {
	IID         int                 `json:"iid"`
	Type        string              `json:"type"`
	Description string              `json:"description,omitempty"`
	Format      string              `json:"format"`
	Access      []string            `json:"access"`
	ValueRange  []float64           `json:"value-range,omitempty"`
	ValueList   []MIoTValueOption   `json:"value-list,omitempty"`
	Unit        string              `json:"unit,omitempty"`
	MaxLength   int                 `json:"max-length,omitempty"`
}

type MIoTValueOption struct {
	Value       any    `json:"value"`
	Description string `json:"description,omitempty"`
}

type MIoTAction struct {
	IID         int    `json:"iid"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	In          []int  `json:"in,omitempty"`
	Out         []int  `json:"out,omitempty"`
}

type MIoTEvent struct {
	IID         int    `json:"iid"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	Arguments   []int  `json:"arguments,omitempty"`
}

func (s MIoTDeviceSpec) Properties() []MIoTPropertyRef {
	var result []MIoTPropertyRef
	for _, service := range s.Services {
		for _, property := range service.Properties {
			result = append(result, MIoTPropertyRef{ServiceID: service.IID, Property: property})
		}
	}
	return result
}

type MIoTPropertyRef struct {
	ServiceID int
	Property  MIoTProperty
}

func (s MIoTDeviceSpec) FindProperty(serviceID, propertyID int) (*MIoTPropertyRef, bool) {
	for _, service := range s.Services {
		if service.IID != serviceID { continue }
		for _, property := range service.Properties {
			if property.IID == propertyID {
				return &MIoTPropertyRef{ServiceID: service.IID, Property: property}, true
			}
		}
	}
	return nil, false
}

func (s MIoTDeviceSpec) FindAction(serviceID, actionID int) (*MIoTAction, bool) {
	for _, service := range s.Services {
		if service.IID != serviceID { continue }
		for _, action := range service.Actions {
			if action.IID == actionID { return &action, true }
		}
	}
	return nil, false
}

func (s MIoTDeviceSpec) FindEvent(serviceID, eventID int) (*MIoTEvent, bool) {
	for _, service := range s.Services {
		if service.IID != serviceID { continue }
		for _, event := range service.Events {
			if event.IID == eventID { return &event, true }
		}
	}
	return nil, false
}

func (p MIoTProperty) CanRead() bool { return hasMIoTAccess(p.Access, "read") }
func (p MIoTProperty) CanWrite() bool { return hasMIoTAccess(p.Access, "write") }
func (p MIoTProperty) CanNotify() bool { return hasMIoTAccess(p.Access, "notify") }

func (p MIoTProperty) ValidateValue(value any) error {
	switch strings.ToLower(p.Format) {
	case "bool":
		if _, ok := value.(bool); !ok { return errors.New("MIoT value must be boolean") }
	case "string":
		text, ok := value.(string)
		if !ok { return errors.New("MIoT value must be a string") }
		if p.MaxLength > 0 && len([]rune(text)) > p.MaxLength {
			return fmt.Errorf("MIoT string exceeds maximum length %d", p.MaxLength)
		}
	case "hex":
		text, ok := value.(string)
		if !ok || len(text)%2 != 0 {
			return errors.New("MIoT hex value must be an even-length hexadecimal string")
		}
		for _, ch := range text {
			if !(ch >= '0' && ch <= '9') && !(ch >= 'a' && ch <= 'f') && !(ch >= 'A' && ch <= 'F') {
				return errors.New("MIoT hex value contains a non-hexadecimal character")
			}
		}
	case "float", "uint8", "uint16", "uint32", "int8", "int16", "int32", "int64":
		number, ok := toMIoTNumber(value)
		if !ok || math.IsNaN(number) || math.IsInf(number, 0) {
			return errors.New("MIoT value must be a finite number")
		}
		if strings.HasPrefix(strings.ToLower(p.Format), "uint") && number < 0 {
			return errors.New("MIoT unsigned property cannot be negative")
		}
		if len(p.ValueRange) >= 2 {
			if number < p.ValueRange[0] || number > p.ValueRange[1] {
				return fmt.Errorf("MIoT value is outside the supported range %.4g..%.4g", p.ValueRange[0], p.ValueRange[1])
			}
			if len(p.ValueRange) >= 3 && p.ValueRange[2] > 0 {
				steps := (number - p.ValueRange[0]) / p.ValueRange[2]
				if math.Abs(steps-math.Round(steps)) > 0.0001 {
					return fmt.Errorf("MIoT value does not match the required step %.4g", p.ValueRange[2])
				}
			}
		}
		if len(p.ValueList) > 0 {
			for _, option := range p.ValueList {
				if candidate, ok := toMIoTNumber(option.Value); ok && math.Abs(candidate-number) < 0.000001 {
					return nil
				}
			}
			return errors.New("MIoT value is not in the supported value list")
		}
	default:
		return fmt.Errorf("unsupported MIoT property format %q", p.Format)
	}
	if len(p.ValueList) > 0 {
		for _, option := range p.ValueList {
			if fmt.Sprint(option.Value) == fmt.Sprint(value) { return nil }
		}
		return errors.New("MIoT value is not in the supported value list")
	}
	return nil
}

func hasMIoTAccess(access []string, wanted string) bool {
	for _, value := range access {
		if strings.EqualFold(strings.TrimSpace(value), wanted) { return true }
	}
	return false
}

func toMIoTNumber(value any) (float64, bool) {
	switch number := value.(type) {
	case float64: return number, true
	case float32: return float64(number), true
	case int: return float64(number), true
	case int32: return float64(number), true
	case int64: return float64(number), true
	case uint8: return float64(number), true
	case uint16: return float64(number), true
	case uint32: return float64(number), true
	case json.Number:
		parsed, err := number.Float64()
		return parsed, err == nil
	default: return 0, false
	}
}

func MIoTPropertyName(property MIoTProperty) string {
	parts := strings.Split(property.Type, ":")
	if len(parts) >= 5 { return parts[3] }
	if property.Description != "" { return strings.ToLower(strings.ReplaceAll(property.Description, " ", "_")) }
	return fmt.Sprintf("property_%d", property.IID)
}

func MIoTUnitName(unit string) string {
	switch strings.ToLower(unit) {
	case "celsius": return "°C"
	case "kelvin": return "K"
	case "percentage": return "%"
	case "seconds": return "s"
	case "minutes": return "min"
	case "hours": return "h"
	case "watt": return "W"
	case "pascal": return "Pa"
	case "lux": return "lx"
	case "litre": return "L"
	case "ppm": return "ppm"
	default: return unit
	}
}

func MIoTEntityDomain(property MIoTProperty) (domain string, writable bool) {
	writable = property.CanWrite()
	format := strings.ToLower(property.Format)
	switch {
	case format == "bool" && writable: return "switch", true
	case format == "bool": return "binary_sensor", false
	case len(property.ValueList) > 0 && writable: return "select", true
	case format == "string" && writable: return "text", true
	case format == "float" || strings.HasPrefix(format, "uint") || strings.HasPrefix(format, "int"):
		if writable { return "number", true }
		return "sensor", false
	default:
		return "sensor", false
	}
}
