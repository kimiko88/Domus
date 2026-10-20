// Package ocpp implements native Open Charge Point Protocol (OCPP 1.6J / 2.0.1)
// for intelligent EV charging, solar surplus modulation (6A-32A), and vehicle SoC monitoring.
package ocpp

import (
	"encoding/json"
	"fmt"
	"time"
)

// OCPP Message Type IDs
const (
	MessageTypeCall   = 2
	MessageTypeResult = 3
	MessageTypeError  = 4
)

// Common OCPP Actions
const (
	ActionBootNotification   = "BootNotification"
	ActionHeartbeat          = "Heartbeat"
	ActionStatusNotification = "StatusNotification"
	ActionMeterValues        = "MeterValues"
	ActionStartTransaction   = "StartTransaction"
	ActionStopTransaction    = "StopTransaction"
	ActionSetChargingProfile = "SetChargingProfile"
)

// ChargePointStatus defines IEC 61851 connector states.
type ChargePointStatus string

const (
	StatusAvailable    ChargePointStatus = "Available"
	StatusPreparing    ChargePointStatus = "Preparing"
	StatusCharging     ChargePointStatus = "Charging"
	StatusSuspendedEVSE ChargePointStatus = "SuspendedEVSE"
	StatusSuspendedEV  ChargePointStatus = "SuspendedEV"
	StatusFinishing    ChargePointStatus = "Finishing"
	StatusFaulted      ChargePointStatus = "Faulted"
)

// BootNotificationReq is sent by the wallbox on power up.
type BootNotificationReq struct {
	ChargePointVendor string `json:"chargePointVendor"`
	ChargePointModel  string `json:"chargePointModel"`
	ChargePointSerial string `json:"chargePointSerialNumber,omitempty"`
	FirmwareVersion   string `json:"firmwareVersion,omitempty"`
}

// BootNotificationConf is returned by Domus hub.
type BootNotificationConf struct {
	Status      string    `json:"status"` // "Accepted"
	CurrentTime time.Time `json:"currentTime"`
	Interval    int       `json:"interval"` // Heartbeat interval in seconds
}

// StatusNotificationReq reports connector status.
type StatusNotificationReq struct {
	ConnectorID int               `json:"connectorId"`
	ErrorCode   string            `json:"errorCode"`
	Status      ChargePointStatus `json:"status"`
	Info        string            `json:"info,omitempty"`
	Timestamp   time.Time         `json:"timestamp,omitempty"`
}

// SampledValue holds a single sensor measurement inside MeterValues.
type SampledValue struct {
	Value     string `json:"value"`
	Context   string `json:"context,omitempty"`
	Format    string `json:"format,omitempty"`
	Measurand string `json:"measurand,omitempty"` // "Power.Active.Import", "Current.Import", "SoC"
	Unit      string `json:"unit,omitempty"`      // "W", "A", "Percent", "Wh"
}

// MeterValue represents a periodic telemetry packet.
type MeterValue struct {
	Timestamp    time.Time      `json:"timestamp"`
	SampledValue []SampledValue `json:"sampledValue"`
}

// MeterValuesReq contains electrical telemetry from the wallbox.
type MeterValuesReq struct {
	ConnectorID   int          `json:"connectorId"`
	TransactionID int          `json:"transactionId,omitempty"`
	MeterValue    []MeterValue `json:"meterValue"`
}

// ChargingSchedulePeriod defines an allowed power limit for a duration.
type ChargingSchedulePeriod struct {
	StartPeriod int     `json:"startPeriod"` // Seconds from start
	Limit       float64 `json:"limit"`       // Amperes (e.g. 6.0 - 32.0 A)
	NumberPhases int    `json:"numberPhases,omitempty"`
}

// ChargingSchedule contains current limits.
type ChargingSchedule struct {
	ChargingRateUnit       string                   `json:"chargingRateUnit"` // "A" (Amperes) or "W" (Watts)
	ChargingSchedulePeriod []ChargingSchedulePeriod `json:"chargingSchedulePeriod"`
}

// ChargingProfile defines smart charging parameters.
type ChargingProfile struct {
	ChargingProfileID      int              `json:"chargingProfileId"`
	StackLevel             int              `json:"stackLevel"`
	ChargingProfilePurpose string           `json:"chargingProfilePurpose"` // "TxDefaultProfile", "ChargePointMaxProfile"
	ChargingProfileKind    string           `json:"chargingProfileKind"`    // "Relative", "Absolute"
	ChargingSchedule       ChargingSchedule `json:"chargingSchedule"`
}

// SetChargingProfileReq is sent by Domus to throttle wallbox current between 6A and 32A.
type SetChargingProfileReq struct {
	ConnectorID     int             `json:"connectorId"`
	ChargingProfile ChargingProfile `json:"csChargingProfiles"`
}

// ParseRawCall parses an OCPP JSON array call: [2, "<id>", "<action>", {<payload>}]
func ParseRawCall(data []byte) (msgType int, id, action string, payload []byte, err error) {
	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return 0, "", "", nil, fmt.Errorf("ocpp: invalid json array: %w", err)
	}
	if len(raw) < 3 {
		return 0, "", "", nil, fmt.Errorf("ocpp: insufficient array elements")
	}

	if err := json.Unmarshal(raw[0], &msgType); err != nil {
		return 0, "", "", nil, err
	}
	if err := json.Unmarshal(raw[1], &id); err != nil {
		return 0, "", "", nil, err
	}

	if msgType == MessageTypeCall {
		if len(raw) < 4 {
			return 0, "", "", nil, fmt.Errorf("ocpp call requires 4 elements")
		}
		if err := json.Unmarshal(raw[2], &action); err != nil {
			return 0, "", "", nil, err
		}
		payload = raw[3]
	}

	return msgType, id, action, payload, nil
}

// FormatCallResult formats a CallResult: [3, "<id>", {<payload>}]
func FormatCallResult(id string, payload any) ([]byte, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	res := []any{MessageTypeResult, id, json.RawMessage(b)}
	return json.Marshal(res)
}

// FormatCall formats a Call: [2, "<id>", "<action>", {<payload>}]
func FormatCall(id, action string, payload any) ([]byte, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	res := []any{MessageTypeCall, id, action, json.RawMessage(b)}
	return json.Marshal(res)
}
