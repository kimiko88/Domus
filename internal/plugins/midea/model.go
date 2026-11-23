package midea

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
)

type Protocol string

const (
	ProtocolMSmartV1 Protocol = "midea_msmart_v1"
	ProtocolMSmartV2 Protocol = "midea_msmart_v2"
	ProtocolMSmartV3 Protocol = "midea_msmart_v3"
	ProtocolTuya     Protocol = "tuya"
	ProtocolCloud    Protocol = "midea_cloud"
)

type ApplianceKind string

const (
	KindUnknown                ApplianceKind = "unknown"
	KindAirBox                 ApplianceKind = "air_box"
	KindAirConditioner         ApplianceKind = "air_conditioner"
	KindAirPurifier            ApplianceKind = "air_purifier"
	KindBathroomMaster         ApplianceKind = "bathroom_master"
	KindClothesDryer           ApplianceKind = "clothes_dryer"
	KindDehumidifier           ApplianceKind = "dehumidifier"
	KindDishSterilizer         ApplianceKind = "dish_sterilizer"
	KindDishwasher             ApplianceKind = "dishwasher"
	KindElectricHeater         ApplianceKind = "electric_heater"
	KindElectricOven            ApplianceKind = "electric_oven"
	KindElectricPressureCooker ApplianceKind = "electric_pressure_cooker"
	KindElectricRiceCooker     ApplianceKind = "electric_rice_cooker"
	KindElectricSlowCooker     ApplianceKind = "electric_slow_cooker"
	KindElectricWaterHeater    ApplianceKind = "electric_water_heater"
	KindFan                    ApplianceKind = "fan"
	KindFreshAir               ApplianceKind = "fresh_air"
	KindFrontLoadWasher        ApplianceKind = "front_load_washer"
	KindGasStove               ApplianceKind = "gas_stove"
	KindGasWaterHeater         ApplianceKind = "gas_water_heater"
	KindHeatPumpWaterHeater    ApplianceKind = "heat_pump_water_heater"
	KindHeatPumpController     ApplianceKind = "heat_pump_wifi_controller"
	KindHeatPump               ApplianceKind = "heat_pump"
	KindHumidifier             ApplianceKind = "humidifier"
	KindIntegratedCeilingFan   ApplianceKind = "integrated_ceiling_fan"
	KindLight                  ApplianceKind = "light"
	KindMDVController          ApplianceKind = "mdv_wifi_controller"
	KindMicrowave              ApplianceKind = "microwave"
	KindMicrowaveSteamOven     ApplianceKind = "microwave_steam_oven"
	KindRangeHood              ApplianceKind = "range_hood"
	KindRefrigerator           ApplianceKind = "refrigerator"
	KindSinkDishwasher         ApplianceKind = "sink_dishwasher"
	KindToaster                ApplianceKind = "toaster"
	KindToilet                 ApplianceKind = "toilet"
	KindTopLoadWasher          ApplianceKind = "top_load_washer"
	KindWaterDrinkingAppliance ApplianceKind = "water_drinking_appliance"
)

type Capability string

const (
	CapPower              Capability = "power"
	CapMode               Capability = "mode"
	CapTargetTemperature  Capability = "target_temperature"
	CapCurrentTemperature Capability = "current_temperature"
	CapTargetHumidity     Capability = "target_humidity"
	CapCurrentHumidity    Capability = "current_humidity"
	CapFanSpeed           Capability = "fan_speed"
	CapSwing              Capability = "swing"
	CapEnergy              Capability = "energy"
	CapWaterTankFull      Capability = "water_tank_full"
	CapFilterLife         Capability = "filter_life"
	CapDoorState          Capability = "door_state"
	CapCycleState         Capability = "cycle_state"
	CapCycleProgress      Capability = "cycle_progress"
	CapRemainingTime      Capability = "remaining_time"
	CapChildLock          Capability = "child_lock"
	CapErrorCode          Capability = "error_code"
	CapTemperature        Capability = "temperature"
	CapHumidity           Capability = "humidity"
)

var deviceIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)

var (
	ErrInvalidDevice       = errors.New("midea: invalid device")
	ErrNoDriver            = errors.New("midea: no driver registered for device protocol")
	ErrDeviceNotFound      = errors.New("midea: device not found")
	ErrInvalidCommand      = errors.New("midea: command is not supported by this device")
	ErrReadBackMismatch    = errors.New("midea: device state does not match requested command")
	ErrReadBackUnavailable = errors.New("midea: command was sent but state verification failed; delivery is unknown")
)

// Device is the public, credential-free device description stored by Domus.
// V3 tokens and keys must never be placed in this structure or API responses.
type Device struct {
	ID               string         `json:"id"`
	Name             string         `json:"name"`
	Brand            string         `json:"brand,omitempty"`
	Model            string         `json:"model,omitempty"`
	TypeCode         string         `json:"type_code,omitempty"`
	Kind             ApplianceKind  `json:"kind"`
	Protocol         Protocol       `json:"protocol"`
	ProtocolVersion  string         `json:"protocol_version,omitempty"`
	Host             string         `json:"host,omitempty"`
	Port             int            `json:"port,omitempty"`
	Capabilities     []Capability   `json:"capabilities,omitempty"`
	Writable         []Capability   `json:"writable,omitempty"`
	Modes            []string       `json:"modes,omitempty"`
	Programs         []string       `json:"programs,omitempty"`
	FanSpeeds        []string       `json:"fan_speeds,omitempty"`
	Actions           []string       `json:"actions,omitempty"`
	MinTemperature   float64        `json:"min_temperature,omitempty"`
	MaxTemperature   float64        `json:"max_temperature,omitempty"`
	TemperatureStep  float64        `json:"temperature_step,omitempty"`
	MinHumidity      float64        `json:"min_humidity,omitempty"`
	MaxHumidity      float64        `json:"max_humidity,omitempty"`
	Online           bool           `json:"online"`
	State            map[string]any `json:"state,omitempty"`
}

func (d Device) Validate() error {
	if !deviceIDPattern.MatchString(strings.TrimSpace(d.ID)) {
		return fmt.Errorf("%w: invalid device id", ErrInvalidDevice)
	}
	if d.Kind == "" {
		return fmt.Errorf("%w: appliance kind is required", ErrInvalidDevice)
	}
	if d.Protocol != ProtocolMSmartV1 && d.Protocol != ProtocolMSmartV2 &&
		d.Protocol != ProtocolMSmartV3 && d.Protocol != ProtocolTuya &&
		d.Protocol != ProtocolCloud {
		return fmt.Errorf("%w: unknown protocol %q", ErrInvalidDevice, d.Protocol)
	}
	if d.Port < 0 || d.Port > 65535 {
		return fmt.Errorf("%w: invalid port", ErrInvalidDevice)
	}
	if d.Protocol == ProtocolCloud {
		if d.Host != "" || d.Port != 0 {
			return fmt.Errorf("%w: cloud devices must not expose a LAN target", ErrInvalidDevice)
		}
	} else {
		ip := net.ParseIP(strings.TrimSpace(d.Host))
		if ip == nil || !ip.IsPrivate() || ip.IsLoopback() || ip.IsUnspecified() || d.Port == 0 {
			return fmt.Errorf("%w: LAN target must use a private non-loopback IP and a valid port", ErrInvalidDevice)
		}
	}
	if d.MinTemperature < 0 || d.MaxTemperature < 0 ||
		(d.MaxTemperature > 0 && d.MinTemperature > d.MaxTemperature) ||
		d.TemperatureStep < 0 {
		return fmt.Errorf("%w: invalid temperature limits", ErrInvalidDevice)
	}
	if d.MinHumidity < 0 || d.MaxHumidity < 0 ||
		(d.MaxHumidity > 0 && d.MinHumidity > d.MaxHumidity) || d.MaxHumidity > 100 {
		return fmt.Errorf("%w: invalid humidity limits", ErrInvalidDevice)
	}
	for _, capability := range d.Writable {
		if !containsCapability(d.Capabilities, capability) {
			return fmt.Errorf("%w: writable capability %q is not advertised", ErrInvalidDevice, capability)
		}
	}
	return nil
}

func containsCapability(values []Capability, wanted Capability) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
