// Package cloud_assistant defines protocol envelopes and device representations
// for Amazon Alexa Smart Home and Google Home Smart Home Actions.
package cloud_assistant

import (
	"encoding/json"
	"time"
)

// --- Alexa Smart Home Models ---

// AlexaDirectiveHeader describes the directive action and metadata.
type AlexaDirectiveHeader struct {
	Namespace        string `json:"namespace"`
	Name             string `json:"name"`
	MessageID        string `json:"messageId"`
	PayloadVersion   string `json:"payloadVersion"`
	CorrelationToken string `json:"correlationToken,omitempty"`
}

// AlexaEndpoint describes the target device.
type AlexaEndpoint struct {
	EndpointID string            `json:"endpointId"`
	Cookie     map[string]string `json:"cookie,omitempty"`
}

// AlexaDirective is the root envelope sent by Alexa.
type AlexaDirective struct {
	Header   AlexaDirectiveHeader `json:"header"`
	Endpoint *AlexaEndpoint       `json:"endpoint,omitempty"`
	Payload  json.RawMessage      `json:"payload"`
}

// AlexaRequest is the top-level body from the Alexa Smart Home Skill adapter.
type AlexaRequest struct {
	Directive AlexaDirective `json:"directive"`
}

// AlexaCapability describes a capability supported by an endpoint.
type AlexaCapability struct {
	Type       string           `json:"type"`      // e.g. "AlexaInterface"
	Interface  string           `json:"interface"` // e.g. "Alexa.PowerController"
	Version    string           `json:"version"`   // e.g. "3"
	Properties *AlexaProperties `json:"properties,omitempty"`
}

// AlexaProperties describes whether states can be proactively reported or retrieved.
type AlexaProperties struct {
	Supported           []AlexaSupportedProperty `json:"supported"`
	ProactivelyReported bool                     `json:"proactivelyReported"`
	Retrievable         bool                     `json:"retrievable"`
}

// AlexaSupportedProperty describes a state property.
type AlexaSupportedProperty struct {
	Name string `json:"name"`
}

// AlexaDiscoveredEndpoint is an endpoint exposed in Alexa.Discovery response.
type AlexaDiscoveredEndpoint struct {
	EndpointID        string            `json:"endpointId"`
	ManufacturerName  string            `json:"manufacturerName"`
	FriendlyName      string            `json:"friendlyName"`
	Description       string            `json:"description"`
	DisplayCategories []string          `json:"displayCategories"` // e.g. ["LIGHT"], ["SMARTPLUG"], ["THERMOSTAT"]
	Capabilities      []AlexaCapability `json:"capabilities"`
	Cookie            map[string]string `json:"cookie,omitempty"`
}

// AlexaResponseContextProperty is a state property in Alexa.Response context.
type AlexaResponseContextProperty struct {
	Namespace                 string `json:"namespace"`
	Name                      string `json:"name"`
	Value                     any    `json:"value"`
	TimeOfSample              string `json:"timeOfSample"`
	UncertaintyInMilliseconds int    `json:"uncertaintyInMilliseconds"`
}

// AlexaResponseEnvelope is the top-level return to Alexa.
type AlexaResponseEnvelope struct {
	Event   AlexaEvent        `json:"event"`
	Context *AlexaResponseCtx `json:"context,omitempty"`
}

// AlexaResponseCtx contains contextual properties.
type AlexaResponseCtx struct {
	Properties []AlexaResponseContextProperty `json:"properties"`
}

// AlexaEvent is the event block inside the response.
type AlexaEvent struct {
	Header   AlexaDirectiveHeader `json:"header"`
	Endpoint *AlexaEndpoint       `json:"endpoint,omitempty"`
	Payload  any                  `json:"payload"`
}

// --- Google Home Smart Home Models ---

// GoogleRequestInput is an intent within a Google fulfillment call.
type GoogleRequestInput struct {
	Intent  string          `json:"intent"` // e.g. "action.devices.SYNC", "action.devices.QUERY", "action.devices.EXECUTE"
	Payload json.RawMessage `json:"payload"`
}

// GoogleRequest is the root fulfillment request from Google Assistant.
type GoogleRequest struct {
	RequestID string               `json:"requestId"`
	Inputs    []GoogleRequestInput `json:"inputs"`
}

// GoogleDevice describes a device in action.devices.SYNC response.
type GoogleDevice struct {
	ID              string            `json:"id"`
	Type            string            `json:"type"`   // e.g. "action.devices.types.LIGHT", "action.devices.types.OUTLET", "action.devices.types.THERMOSTAT"
	Traits          []string          `json:"traits"` // e.g. "action.devices.traits.OnOff", "action.devices.traits.Brightness"
	Name            GoogleDeviceName  `json:"name"`
	WillReportState bool              `json:"willReportState"`
	Attributes      map[string]any    `json:"attributes,omitempty"`
	DeviceInfo      *GoogleDeviceInfo `json:"deviceInfo,omitempty"`
}

// GoogleDeviceName holds labels and nicknames.
type GoogleDeviceName struct {
	Name         string   `json:"name"`
	DefaultNames []string `json:"defaultNames,omitempty"`
	Nicknames    []string `json:"nicknames,omitempty"`
}

// GoogleDeviceInfo holds metadata.
type GoogleDeviceInfo struct {
	Manufacturer string `json:"manufacturer"`
	Model        string `json:"model"`
	SwVersion    string `json:"swVersion"`
}

// GoogleCommandExecution represents an action inside action.devices.EXECUTE.
type GoogleCommandExecution struct {
	Command string         `json:"command"` // e.g. "action.devices.commands.OnOff"
	Params  map[string]any `json:"params"`
}

// GoogleCommandTarget is a device targeted by a command.
type GoogleCommandTarget struct {
	ID string `json:"id"`
}

// GoogleExecuteCommand is an item within EXECUTE payload commands.
type GoogleExecuteCommand struct {
	Devices   []GoogleCommandTarget    `json:"devices"`
	Execution []GoogleCommandExecution `json:"execution"`
}

// GoogleResponse is the root response envelope sent back to Google.
type GoogleResponse struct {
	RequestID string `json:"requestId"`
	Payload   any    `json:"payload"`
}

// NewAlexaHeader helper creates an Alexa response header.
func NewAlexaHeader(namespace, name, correlationToken string) AlexaDirectiveHeader {
	return AlexaDirectiveHeader{
		Namespace:        namespace,
		Name:             name,
		MessageID:        "msg_" + time.Now().UTC().Format("20060102150405.000000"),
		PayloadVersion:   "3",
		CorrelationToken: correlationToken,
	}
}
