package cloud_assistant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/domus-hub/domus/internal/domain/cloud_assistant"
	"github.com/domus-hub/domus/internal/domain/entity"
	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/google/uuid"
)

var (
	ErrCloudDisabled     = errors.New("cloud voice assistant integration is disabled")
	ErrUnauthorized      = errors.New("invalid or missing authorization token")
	ErrUnsupportedIntent = errors.New("unsupported directive or intent")
)

// Service translates Amazon Alexa directives and Google Assistant Smart Home actions.
type Service struct {
	entities    entity.Repository
	bus         events.Bus
	bearerToken string
	enabled     bool
}

// NewService constructs a cloud voice assistant service.
func NewService(entities entity.Repository, bus events.Bus, bearerToken string, enabled bool) *Service {
	return &Service{
		entities:    entities,
		bus:         bus,
		bearerToken: strings.TrimSpace(bearerToken),
		enabled:     enabled,
	}
}

// IsEnabled returns whether cloud voice integration is active.
func (s *Service) IsEnabled() bool {
	return s.enabled
}

// ValidateToken verifies the Authorization header.
func (s *Service) ValidateToken(authHeader string) bool {
	if s.bearerToken == "" {
		// When no token is configured, allow if enabled
		return s.enabled
	}
	parts := strings.Split(strings.TrimSpace(authHeader), " ")
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1] == s.bearerToken
	}
	return authHeader == s.bearerToken
}

// HandleAlexa processes an Alexa Smart Home directive.
func (s *Service) HandleAlexa(ctx context.Context, body []byte) (*cloud_assistant.AlexaResponseEnvelope, error) {
	if !s.enabled {
		return nil, ErrCloudDisabled
	}

	var req cloud_assistant.AlexaRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, fmt.Errorf("unmarshal alexa directive: %w", err)
	}

	header := req.Directive.Header

	switch header.Namespace {
	case "Alexa.Discovery":
		if header.Name == "Discover" {
			return s.handleAlexaDiscovery(ctx, header.CorrelationToken)
		}

	case "Alexa.PowerController":
		return s.handleAlexaPower(ctx, req.Directive)

	case "Alexa.BrightnessController":
		return s.handleAlexaBrightness(ctx, req.Directive)

	case "Alexa.ThermostatController":
		return s.handleAlexaThermostat(ctx, req.Directive)

	case "Alexa":
		if header.Name == "ReportState" {
			return s.handleAlexaReportState(ctx, req.Directive)
		}
	}

	return nil, fmt.Errorf("%w: %s.%s", ErrUnsupportedIntent, header.Namespace, header.Name)
}

func isEntityOn(ent *entity.Entity) bool {
	st := strings.ToLower(strings.Trim(ent.State, `"`))
	return st == "on" || st == "true" || strings.Contains(st, `"on":true`)
}

func parseEntityAttributes(ent *entity.Entity) map[string]any {
	var m map[string]any
	if ent.Attributes != "" {
		_ = json.Unmarshal([]byte(ent.Attributes), &m)
	}
	if m == nil {
		m = make(map[string]any)
	}
	return m
}

func (s *Service) handleAlexaDiscovery(ctx context.Context, correlationToken string) (*cloud_assistant.AlexaResponseEnvelope, error) {
	allEntities, err := s.entities.List(ctx, entity.ListFilter{})
	if err != nil {
		return nil, err
	}

	endpoints := make([]cloud_assistant.AlexaDiscoveredEndpoint, 0)
	for _, ent := range allEntities {
		category := "OTHER"
		capabilities := []cloud_assistant.AlexaCapability{
			{
				Type:      "AlexaInterface",
				Interface: "Alexa",
				Version:   "3",
			},
		}

		switch ent.Domain {
		case "light":
			category = "LIGHT"
			capabilities = append(capabilities,
				cloud_assistant.AlexaCapability{
					Type:      "AlexaInterface",
					Interface: "Alexa.PowerController",
					Version:   "3",
					Properties: &cloud_assistant.AlexaProperties{
						Supported:   []cloud_assistant.AlexaSupportedProperty{{Name: "powerState"}},
						Retrievable: true,
					},
				},
				cloud_assistant.AlexaCapability{
					Type:      "AlexaInterface",
					Interface: "Alexa.BrightnessController",
					Version:   "3",
					Properties: &cloud_assistant.AlexaProperties{
						Supported:   []cloud_assistant.AlexaSupportedProperty{{Name: "brightness"}},
						Retrievable: true,
					},
				},
			)
		case "switch":
			category = "SMARTPLUG"
			capabilities = append(capabilities, cloud_assistant.AlexaCapability{
				Type:      "AlexaInterface",
				Interface: "Alexa.PowerController",
				Version:   "3",
				Properties: &cloud_assistant.AlexaProperties{
					Supported:   []cloud_assistant.AlexaSupportedProperty{{Name: "powerState"}},
					Retrievable: true,
				},
			})
		case "climate":
			category = "THERMOSTAT"
			capabilities = append(capabilities, cloud_assistant.AlexaCapability{
				Type:      "AlexaInterface",
				Interface: "Alexa.ThermostatController",
				Version:   "3",
				Properties: &cloud_assistant.AlexaProperties{
					Supported: []cloud_assistant.AlexaSupportedProperty{
						{Name: "targetSetpoint"},
						{Name: "thermostatMode"},
					},
					Retrievable: true,
				},
			})
		default:
			continue
		}

		endpoints = append(endpoints, cloud_assistant.AlexaDiscoveredEndpoint{
			EndpointID:        ent.ID,
			ManufacturerName:  "Domus",
			FriendlyName:      ent.Name,
			Description:       ent.Name + " controllato da Domus Hub",
			DisplayCategories: []string{category},
			Capabilities:      capabilities,
		})
	}

	resp := &cloud_assistant.AlexaResponseEnvelope{
		Event: cloud_assistant.AlexaEvent{
			Header:  cloud_assistant.NewAlexaHeader("Alexa.Discovery", "Discover.Response", correlationToken),
			Payload: map[string]any{"endpoints": endpoints},
		},
	}
	return resp, nil
}

func (s *Service) handleAlexaPower(ctx context.Context, dir cloud_assistant.AlexaDirective) (*cloud_assistant.AlexaResponseEnvelope, error) {
	if dir.Endpoint == nil {
		return nil, errors.New("missing endpoint in alexa directive")
	}
	entID := dir.Endpoint.EndpointID

	powerState := "OFF"
	serviceName := "turn_off"
	if dir.Header.Name == "TurnOn" {
		powerState = "ON"
		serviceName = "turn_on"
	}

	parts := strings.Split(entID, ".")
	domain := "switch"
	if len(parts) > 0 {
		domain = parts[0]
	}

	if s.bus != nil {
		_ = s.bus.Publish(ctx, events.Event{
			ID:         uuid.NewString(),
			Type:       events.CommandRequested,
			OccurredAt: time.Now().UTC(),
			Source:     "alexa",
			Payload: map[string]any{
				"entity_id": entID,
				"domain":    domain,
				"service":   serviceName,
			},
		})
	}

	nowISO := time.Now().UTC().Format(time.RFC3339)
	return &cloud_assistant.AlexaResponseEnvelope{
		Event: cloud_assistant.AlexaEvent{
			Header:   cloud_assistant.NewAlexaHeader("Alexa", "Response", dir.Header.CorrelationToken),
			Endpoint: dir.Endpoint,
			Payload:  map[string]any{},
		},
		Context: &cloud_assistant.AlexaResponseCtx{
			Properties: []cloud_assistant.AlexaResponseContextProperty{
				{
					Namespace:                 "Alexa.PowerController",
					Name:                      "powerState",
					Value:                     powerState,
					TimeOfSample:              nowISO,
					UncertaintyInMilliseconds: 50,
				},
			},
		},
	}, nil
}

func (s *Service) handleAlexaBrightness(ctx context.Context, dir cloud_assistant.AlexaDirective) (*cloud_assistant.AlexaResponseEnvelope, error) {
	if dir.Endpoint == nil {
		return nil, errors.New("missing endpoint in alexa directive")
	}
	entID := dir.Endpoint.EndpointID

	var payload struct {
		Brightness int `json:"brightness"`
	}
	_ = json.Unmarshal(dir.Payload, &payload)

	scaledBrightness := int(float64(payload.Brightness) * 2.55)

	if s.bus != nil {
		_ = s.bus.Publish(ctx, events.Event{
			ID:         uuid.NewString(),
			Type:       events.CommandRequested,
			OccurredAt: time.Now().UTC(),
			Source:     "alexa",
			Payload: map[string]any{
				"entity_id":  entID,
				"domain":     "light",
				"service":    "turn_on",
				"brightness": scaledBrightness,
			},
		})
	}

	nowISO := time.Now().UTC().Format(time.RFC3339)
	return &cloud_assistant.AlexaResponseEnvelope{
		Event: cloud_assistant.AlexaEvent{
			Header:   cloud_assistant.NewAlexaHeader("Alexa", "Response", dir.Header.CorrelationToken),
			Endpoint: dir.Endpoint,
			Payload:  map[string]any{},
		},
		Context: &cloud_assistant.AlexaResponseCtx{
			Properties: []cloud_assistant.AlexaResponseContextProperty{
				{
					Namespace:                 "Alexa.BrightnessController",
					Name:                      "brightness",
					Value:                     payload.Brightness,
					TimeOfSample:              nowISO,
					UncertaintyInMilliseconds: 50,
				},
			},
		},
	}, nil
}

func (s *Service) handleAlexaThermostat(ctx context.Context, dir cloud_assistant.AlexaDirective) (*cloud_assistant.AlexaResponseEnvelope, error) {
	if dir.Endpoint == nil {
		return nil, errors.New("missing endpoint in alexa directive")
	}
	entID := dir.Endpoint.EndpointID

	var payload struct {
		TargetSetpoint struct {
			Value float64 `json:"value"`
			Scale string  `json:"scale"`
		} `json:"targetSetpoint"`
	}
	_ = json.Unmarshal(dir.Payload, &payload)

	if s.bus != nil {
		_ = s.bus.Publish(ctx, events.Event{
			ID:         uuid.NewString(),
			Type:       events.CommandRequested,
			OccurredAt: time.Now().UTC(),
			Source:     "alexa",
			Payload: map[string]any{
				"entity_id":   entID,
				"domain":      "climate",
				"service":     "set_temperature",
				"temperature": payload.TargetSetpoint.Value,
			},
		})
	}

	nowISO := time.Now().UTC().Format(time.RFC3339)
	return &cloud_assistant.AlexaResponseEnvelope{
		Event: cloud_assistant.AlexaEvent{
			Header:   cloud_assistant.NewAlexaHeader("Alexa", "Response", dir.Header.CorrelationToken),
			Endpoint: dir.Endpoint,
			Payload:  map[string]any{},
		},
		Context: &cloud_assistant.AlexaResponseCtx{
			Properties: []cloud_assistant.AlexaResponseContextProperty{
				{
					Namespace:                 "Alexa.ThermostatController",
					Name:                      "targetSetpoint",
					Value:                     map[string]any{"value": payload.TargetSetpoint.Value, "scale": "CELSIUS"},
					TimeOfSample:              nowISO,
					UncertaintyInMilliseconds: 50,
				},
			},
		},
	}, nil
}

func (s *Service) handleAlexaReportState(ctx context.Context, dir cloud_assistant.AlexaDirective) (*cloud_assistant.AlexaResponseEnvelope, error) {
	if dir.Endpoint == nil {
		return nil, errors.New("missing endpoint in alexa directive")
	}
	entID := dir.Endpoint.EndpointID

	ent, err := s.entities.GetByID(ctx, entID)
	if err != nil {
		return nil, err
	}

	nowISO := time.Now().UTC().Format(time.RFC3339)
	powerVal := "OFF"
	if isEntityOn(ent) {
		powerVal = "ON"
	}

	return &cloud_assistant.AlexaResponseEnvelope{
		Event: cloud_assistant.AlexaEvent{
			Header:   cloud_assistant.NewAlexaHeader("Alexa", "StateReport", dir.Header.CorrelationToken),
			Endpoint: dir.Endpoint,
			Payload:  map[string]any{},
		},
		Context: &cloud_assistant.AlexaResponseCtx{
			Properties: []cloud_assistant.AlexaResponseContextProperty{
				{
					Namespace:                 "Alexa.PowerController",
					Name:                      "powerState",
					Value:                     powerVal,
					TimeOfSample:              nowISO,
					UncertaintyInMilliseconds: 50,
				},
			},
		},
	}, nil
}

// HandleGoogle processes a Google Home Smart Home Action fulfillment request.
func (s *Service) HandleGoogle(ctx context.Context, req *cloud_assistant.GoogleRequest) (*cloud_assistant.GoogleResponse, error) {
	if !s.enabled {
		return nil, ErrCloudDisabled
	}

	if len(req.Inputs) == 0 {
		return nil, errors.New("empty inputs in google request")
	}

	input := req.Inputs[0]

	switch input.Intent {
	case "action.devices.SYNC":
		return s.handleGoogleSync(ctx, req.RequestID)

	case "action.devices.QUERY":
		return s.handleGoogleQuery(ctx, req.RequestID, input.Payload)

	case "action.devices.EXECUTE":
		return s.handleGoogleExecute(ctx, req.RequestID, input.Payload)
	}

	return nil, fmt.Errorf("%w: %s", ErrUnsupportedIntent, input.Intent)
}

func (s *Service) handleGoogleSync(ctx context.Context, reqID string) (*cloud_assistant.GoogleResponse, error) {
	allEntities, err := s.entities.List(ctx, entity.ListFilter{})
	if err != nil {
		return nil, err
	}

	devices := make([]cloud_assistant.GoogleDevice, 0)
	for _, ent := range allEntities {
		var devType string
		var traits []string

		switch ent.Domain {
		case "light":
			devType = "action.devices.types.LIGHT"
			traits = []string{"action.devices.traits.OnOff", "action.devices.traits.Brightness"}
		case "switch":
			devType = "action.devices.types.OUTLET"
			traits = []string{"action.devices.traits.OnOff"}
		case "climate":
			devType = "action.devices.types.THERMOSTAT"
			traits = []string{"action.devices.traits.TemperatureSetting"}
		default:
			continue
		}

		devices = append(devices, cloud_assistant.GoogleDevice{
			ID:     ent.ID,
			Type:   devType,
			Traits: traits,
			Name: cloud_assistant.GoogleDeviceName{
				Name: ent.Name,
			},
			WillReportState: false,
			DeviceInfo: &cloud_assistant.GoogleDeviceInfo{
				Manufacturer: "Domus",
				Model:        "Hub v1",
				SwVersion:    "1.0",
			},
		})
	}

	return &cloud_assistant.GoogleResponse{
		RequestID: reqID,
		Payload: map[string]any{
			"agentUserId": "domus_user",
			"devices":     devices,
		},
	}, nil
}

func (s *Service) handleGoogleQuery(ctx context.Context, reqID string, payloadBytes []byte) (*cloud_assistant.GoogleResponse, error) {
	var payload struct {
		Devices []struct {
			ID string `json:"id"`
		} `json:"devices"`
	}
	_ = json.Unmarshal(payloadBytes, &payload)

	devicesMap := make(map[string]any)
	for _, target := range payload.Devices {
		ent, err := s.entities.GetByID(ctx, target.ID)
		if err != nil {
			devicesMap[target.ID] = map[string]any{"online": false, "status": "ERROR"}
			continue
		}

		attrs := parseEntityAttributes(ent)
		st := map[string]any{
			"online": true,
			"status": "SUCCESS",
			"on":     isEntityOn(ent),
		}

		if ent.Domain == "light" {
			if br, ok := attrs["brightness"].(float64); ok {
				st["brightness"] = int(br / 2.55)
			}
		} else if ent.Domain == "climate" {
			if temp, ok := attrs["temperature"].(float64); ok {
				st["thermostatTemperatureSetpoint"] = temp
			}
			if cur, ok := attrs["current_temperature"].(float64); ok {
				st["thermostatTemperatureAmbient"] = cur
			}
		}

		devicesMap[target.ID] = st
	}

	return &cloud_assistant.GoogleResponse{
		RequestID: reqID,
		Payload: map[string]any{
			"devices": devicesMap,
		},
	}, nil
}

func (s *Service) handleGoogleExecute(ctx context.Context, reqID string, payloadBytes []byte) (*cloud_assistant.GoogleResponse, error) {
	var payload struct {
		Commands []cloud_assistant.GoogleExecuteCommand `json:"commands"`
	}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, err
	}

	type commandResult struct {
		IDs    []string       `json:"ids"`
		Status string         `json:"status"`
		States map[string]any `json:"states,omitempty"`
	}

	results := make([]commandResult, 0)

	for _, cmd := range payload.Commands {
		for _, exec := range cmd.Execution {
			targetIDs := make([]string, len(cmd.Devices))
			for i, d := range cmd.Devices {
				targetIDs[i] = d.ID
			}

			states := make(map[string]any)

			switch exec.Command {
			case "action.devices.commands.OnOff":
				turnOn, _ := exec.Params["on"].(bool)
				service := "turn_off"
				if turnOn {
					service = "turn_on"
				}
				states["on"] = turnOn

				for _, id := range targetIDs {
					parts := strings.Split(id, ".")
					domain := "switch"
					if len(parts) > 0 {
						domain = parts[0]
					}
					if s.bus != nil {
						_ = s.bus.Publish(ctx, events.Event{
							ID:         uuid.NewString(),
							Type:       events.CommandRequested,
							OccurredAt: time.Now().UTC(),
							Source:     "google_assistant",
							Payload: map[string]any{
								"entity_id": id,
								"domain":    domain,
								"service":   service,
							},
						})
					}
				}

			case "action.devices.commands.BrightnessAbsolute":
				br, _ := exec.Params["brightness"].(float64)
				states["brightness"] = int(br)
				scaled := int(br * 2.55)

				for _, id := range targetIDs {
					if s.bus != nil {
						_ = s.bus.Publish(ctx, events.Event{
							ID:         uuid.NewString(),
							Type:       events.CommandRequested,
							OccurredAt: time.Now().UTC(),
							Source:     "google_assistant",
							Payload: map[string]any{
								"entity_id":  id,
								"domain":     "light",
								"service":    "turn_on",
								"brightness": scaled,
							},
						})
					}
				}

			case "action.devices.commands.ThermostatTemperatureSetpoint":
				temp, _ := exec.Params["thermostatTemperatureSetpoint"].(float64)
				states["thermostatTemperatureSetpoint"] = temp

				for _, id := range targetIDs {
					if s.bus != nil {
						_ = s.bus.Publish(ctx, events.Event{
							ID:         uuid.NewString(),
							Type:       events.CommandRequested,
							OccurredAt: time.Now().UTC(),
							Source:     "google_assistant",
							Payload: map[string]any{
								"entity_id":   id,
								"domain":      "climate",
								"service":     "set_temperature",
								"temperature": temp,
							},
						})
					}
				}
			}

			results = append(results, commandResult{
				IDs:    targetIDs,
				Status: "SUCCESS",
				States: states,
			})
		}
	}

	return &cloud_assistant.GoogleResponse{
		RequestID: reqID,
		Payload: map[string]any{
			"commands": results,
		},
	}, nil
}
