package cloud_assistant

import (
	"context"
	"crypto/subtle"
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
	token := strings.TrimSpace(bearerToken)
	return &Service{
		entities:    entities,
		bus:         bus,
		bearerToken: token,
		enabled:     enabled && token != "",
	}
}

// IsEnabled returns whether cloud voice integration is active.
func (s *Service) IsEnabled() bool {
	return s.enabled
}

// ValidateToken verifies the Authorization header.
func (s *Service) ValidateToken(authHeader string) bool {
	// An enabled integration without a secret is never authenticated.
	if !s.enabled || s.bearerToken == "" {
		return false
	}
	candidate := strings.TrimSpace(authHeader)
	parts := strings.Split(candidate, " ")
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		candidate = parts[1]
	}
	return len(candidate) == len(s.bearerToken) &&
		subtle.ConstantTimeCompare([]byte(candidate), []byte(s.bearerToken)) == 1
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
		if header.Name == "SetThermostatMode" {
			return s.handleAlexaThermostatMode(ctx, req.Directive)
		}
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

func numericAttribute(attrs map[string]any, keys ...string) (float64, bool) {
	for _, key := range keys {
		value, exists := attrs[key]
		if !exists {
			continue
		}
		switch number := value.(type) {
		case float64:
			return number, true
		case float32:
			return float64(number), true
		case int:
			return float64(number), true
		case int64:
			return float64(number), true
		case json.Number:
			if parsed, err := number.Float64(); err == nil {
				return parsed, true
			}
		}
	}
	return 0, false
}

func climateSetpoint(attrs map[string]any) (float64, bool) {
	return numericAttribute(attrs, "target_temperature", "temperature", "target_temp")
}

func climateAmbientTemperature(attrs map[string]any) (float64, bool) {
	return numericAttribute(attrs, "current_temperature", "current_temp", "ambient_temperature")
}

func celsius(value float64, scale string) float64 {
	if strings.EqualFold(strings.TrimSpace(scale), "FAHRENHEIT") ||
		strings.EqualFold(strings.TrimSpace(scale), "F") {
		return (value - 32) * 5 / 9
	}
	return value
}

func (s *Service) requireClimateEntity(ctx context.Context, entityID string) error {
	ent, err := s.entities.GetByID(ctx, entityID)
	if err != nil {
		return fmt.Errorf("resolve thermostat %q: %w", entityID, err)
	}
	if ent.Domain != "climate" {
		return fmt.Errorf("entity %q is not a climate entity", entityID)
	}
	return nil
}

func climateModeValues(attrs map[string]any) []string {
	allowed := map[string]bool{"off": true, "heat": true, "cool": true, "auto": true}
	var candidates []string
	switch values := attrs["hvac_modes"].(type) {
	case []string:
		candidates = append(candidates, values...)
	case []any:
		for _, value := range values {
			if mode, ok := value.(string); ok {
				candidates = append(candidates, mode)
			}
		}
	case string:
		candidates = strings.Split(values, ",")
	}
	modes := make([]string, 0, len(candidates))
	seen := make(map[string]bool)
	for _, candidate := range candidates {
		mode := strings.ToLower(strings.TrimSpace(candidate))
		switch mode {
		case "heating":
			mode = "heat"
		case "cooling":
			mode = "cool"
		case "automatic":
			mode = "auto"
		}
		if allowed[mode] && !seen[mode] {
			modes = append(modes, mode)
			seen[mode] = true
		}
	}
	if len(modes) == 0 {
		return []string{"off", "heat", "cool", "auto"}
	}
	return modes
}

func alexaClimateModes(attrs map[string]any) []map[string]string {
	modes := climateModeValues(attrs)
	result := make([]map[string]string, 0, len(modes))
	for _, mode := range modes {
		result = append(result, map[string]string{"value": strings.ToUpper(mode)})
	}
	return result
}

func (s *Service) requireClimateMode(ctx context.Context, entityID, mode string) error {
	ent, err := s.entities.GetByID(ctx, entityID)
	if err != nil {
		return fmt.Errorf("resolve thermostat %q: %w", entityID, err)
	}
	if ent.Domain != "climate" {
		return fmt.Errorf("entity %q is not a climate entity", entityID)
	}
	for _, supported := range climateModeValues(parseEntityAttributes(ent)) {
		if supported == mode {
			return nil
		}
	}
	return fmt.Errorf("thermostat %q does not support mode %q", entityID, mode)
}

func climateTemperatureRange(attrs map[string]any) (float64, float64) {
	minTemp, maxTemp := 5.0, 35.0
	if value, ok := numericAttribute(attrs, "min_temp", "min_temperature"); ok {
		minTemp = value
	}
	if value, ok := numericAttribute(attrs, "max_temp", "max_temperature"); ok {
		maxTemp = value
	}
	if minTemp < 0 || maxTemp <= minTemp {
		return 5, 35
	}
	return minTemp, maxTemp
}

func (s *Service) requireClimateSetpoint(ctx context.Context, entityID string, temperature float64) error {
	ent, err := s.entities.GetByID(ctx, entityID)
	if err != nil {
		return fmt.Errorf("resolve thermostat %q: %w", entityID, err)
	}
	if ent.Domain != "climate" {
		return fmt.Errorf("entity %q is not a climate entity", entityID)
	}
	minTemp, maxTemp := climateTemperatureRange(parseEntityAttributes(ent))
	if temperature < minTemp || temperature > maxTemp {
		return fmt.Errorf("thermostat %q setpoint must be between %.1f and %.1f °C", entityID, minTemp, maxTemp)
	}
	return nil
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
			capabilities = append(capabilities,
				cloud_assistant.AlexaCapability{
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
					Configuration: map[string]any{
						"supportsScheduling": false,
						"supportedModes": alexaClimateModes(parseEntityAttributes(ent)),
					},
				},
				cloud_assistant.AlexaCapability{
					Type:      "AlexaInterface",
					Interface: "Alexa.TemperatureSensor",
					Version:   "3",
					Properties: &cloud_assistant.AlexaProperties{
						Supported:   []cloud_assistant.AlexaSupportedProperty{{Name: "temperature"}},
						Retrievable: true,
					},
				},
			)
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
	if err := json.Unmarshal(dir.Payload, &payload); err != nil {
		return nil, fmt.Errorf("decode thermostat setpoint: %w", err)
	}
	if dir.Header.Name != "SetTargetTemperature" {
		return nil, fmt.Errorf("%w: %s.%s", ErrUnsupportedIntent, dir.Header.Namespace, dir.Header.Name)
	}
	targetCelsius := celsius(payload.TargetSetpoint.Value, payload.TargetSetpoint.Scale)
	if err := s.requireClimateSetpoint(ctx, entID, targetCelsius); err != nil {
		return nil, err
	}

	if s.bus != nil {
		if err := s.bus.Publish(ctx, events.Event{
			ID:         uuid.NewString(),
			Type:       events.CommandRequested,
			OccurredAt: time.Now().UTC(),
			Source:     "alexa",
			Payload: map[string]any{
				"entity_id":   entID,
				"domain":      "climate",
				"service":     "set_temperature",
				"temperature": targetCelsius,
			},
		}); err != nil {
			return nil, err
		}
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
					Value:                     map[string]any{"value": targetCelsius, "scale": "CELSIUS"},
					TimeOfSample:              nowISO,
					UncertaintyInMilliseconds: 50,
				},
			},
		},
	}, nil
}

func (s *Service) handleAlexaThermostatMode(ctx context.Context, dir cloud_assistant.AlexaDirective) (*cloud_assistant.AlexaResponseEnvelope, error) {
	if dir.Endpoint == nil { return nil, errors.New("missing endpoint in alexa directive") }
	var payload struct {
		ThermostatMode struct { Value string `json:"value"` } `json:"thermostatMode"`
	}
	if err := json.Unmarshal(dir.Payload, &payload); err != nil {
		return nil, fmt.Errorf("decode thermostat mode: %w", err)
	}
	mode := strings.ToLower(strings.TrimSpace(payload.ThermostatMode.Value))
	switch mode { case "off", "heat", "cool", "auto": default:
		return nil, fmt.Errorf("unsupported thermostat mode %q", payload.ThermostatMode.Value)
	}
	if err := s.requireClimateMode(ctx, dir.Endpoint.EndpointID, mode); err != nil {
		return nil, err
	}
	if s.bus != nil {
		if err := s.bus.Publish(ctx, events.Event{
			ID: uuid.NewString(), Type: events.CommandRequested, OccurredAt: time.Now().UTC(),
			Source: "alexa", Payload: map[string]any{
				"entity_id": dir.Endpoint.EndpointID, "domain": "climate",
				"service": "set_hvac_mode", "hvac_mode": mode,
			},
		}); err != nil { return nil, err }
	}
	return &cloud_assistant.AlexaResponseEnvelope{
		Event: cloud_assistant.AlexaEvent{
			Header: cloud_assistant.NewAlexaHeader("Alexa", "Response", dir.Header.CorrelationToken),
			Endpoint: dir.Endpoint, Payload: map[string]any{},
		},
		Context: &cloud_assistant.AlexaResponseCtx{Properties: []cloud_assistant.AlexaResponseContextProperty{{
			Namespace: "Alexa.ThermostatController", Name: "thermostatMode",
			Value: map[string]any{"value": strings.ToUpper(mode)},
			TimeOfSample: time.Now().UTC().Format(time.RFC3339), UncertaintyInMilliseconds: 50,
		}}},
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
	properties := make([]cloud_assistant.AlexaResponseContextProperty, 0, 3)
	if ent.Domain == "climate" {
		attrs := parseEntityAttributes(ent)
		if setpoint, ok := climateSetpoint(attrs); ok {
			properties = append(properties, cloud_assistant.AlexaResponseContextProperty{
				Namespace: "Alexa.ThermostatController",
				Name: "targetSetpoint",
				Value: map[string]any{"value": setpoint, "scale": "CELSIUS"},
				TimeOfSample: nowISO,
				UncertaintyInMilliseconds: 1000,
			})
		}
		if ambient, ok := climateAmbientTemperature(attrs); ok {
			properties = append(properties, cloud_assistant.AlexaResponseContextProperty{
				Namespace: "Alexa.TemperatureSensor",
				Name: "temperature",
				Value: map[string]any{"value": ambient, "scale": "CELSIUS"},
				TimeOfSample: nowISO,
				UncertaintyInMilliseconds: 1000,
			})
		}
		mode, _ := attrs["hvac_mode"].(string)
		if mode == "" {
			mode = strings.Trim(strings.TrimSpace(ent.State), `"`)
		}
		switch strings.ToLower(strings.TrimSpace(mode)) {
		case "off", "heat", "heating", "cool", "cooling", "auto", "automatic":
			switch strings.ToLower(strings.TrimSpace(mode)) {
			case "heating": mode = "heat"
			case "cooling": mode = "cool"
			case "automatic": mode = "auto"
			}
			properties = append(properties, cloud_assistant.AlexaResponseContextProperty{
				Namespace: "Alexa.ThermostatController", Name: "thermostatMode",
				Value: map[string]any{"value": strings.ToUpper(mode)},
				TimeOfSample: nowISO, UncertaintyInMilliseconds: 1000,
			})
		}
	} else {
		powerVal := "OFF"
		if isEntityOn(ent) {
			powerVal = "ON"
		}
		properties = append(properties, cloud_assistant.AlexaResponseContextProperty{
			Namespace: "Alexa.PowerController",
			Name: "powerState",
			Value: powerVal,
			TimeOfSample: nowISO,
			UncertaintyInMilliseconds: 50,
		})
	}

	return &cloud_assistant.AlexaResponseEnvelope{
		Event: cloud_assistant.AlexaEvent{
			Header:   cloud_assistant.NewAlexaHeader("Alexa", "StateReport", dir.Header.CorrelationToken),
			Endpoint: dir.Endpoint,
			Payload:  map[string]any{},
		},
		Context: &cloud_assistant.AlexaResponseCtx{
			Properties: properties,
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

		var attributes map[string]any
		if ent.Domain == "climate" {
			attributes = map[string]any{
				"thermostatTemperatureUnit": "C",
				"availableThermostatModes": climateModeValues(parseEntityAttributes(ent)),
				"thermostatTemperatureRange": func() map[string]any {
					minTemp, maxTemp := climateTemperatureRange(parseEntityAttributes(ent))
					return map[string]any{"minThresholdCelsius": minTemp, "maxThresholdCelsius": maxTemp}
				}(),
			}
		}
		devices = append(devices, cloud_assistant.GoogleDevice{
			ID:         ent.ID,
			Type:       devType,
			Traits:     traits,
			Attributes: attributes,
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
			if temp, ok := climateSetpoint(attrs); ok {
				st["thermostatTemperatureSetpoint"] = temp
			}
			if cur, ok := climateAmbientTemperature(attrs); ok {
				st["thermostatTemperatureAmbient"] = cur
			}
			st["thermostatTemperatureUnit"] = "C"
			mode, _ := attrs["hvac_mode"].(string)
			if mode == "" {
				mode = strings.Trim(strings.TrimSpace(ent.State), `"`)
			}
			switch strings.ToLower(mode) {
			case "off":
				st["thermostatMode"] = "off"
			case "heat", "heating":
				st["thermostatMode"] = "heat"
			case "cool", "cooling":
				st["thermostatMode"] = "cool"
			case "auto", "automatic":
				st["thermostatMode"] = "auto"
			default:
				st["thermostatMode"] = "auto"
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

			case "action.devices.commands.ThermostatSetMode":
				mode, _ := exec.Params["thermostatMode"].(string)
				mode = strings.ToLower(strings.TrimSpace(mode))
				if mode != "off" && mode != "heat" && mode != "cool" && mode != "auto" {
					return nil, fmt.Errorf("unsupported thermostat mode %q", mode)
				}
				for _, id := range targetIDs {
					if err := s.requireClimateMode(ctx, id, mode); err != nil {
						return nil, err
					}
				}
				states["thermostatMode"] = mode
				for _, id := range targetIDs {
					if s.bus != nil {
						if err := s.bus.Publish(ctx, events.Event{
							ID: uuid.NewString(), Type: events.CommandRequested, OccurredAt: time.Now().UTC(),
							Source: "google_assistant", Payload: map[string]any{
								"entity_id": id, "domain": "climate",
								"service": "set_hvac_mode", "hvac_mode": mode,
							},
						}); err != nil { return nil, err }
					}
				}

			case "action.devices.commands.ThermostatTemperatureSetpoint":
				temp, ok := exec.Params["thermostatTemperatureSetpoint"].(float64)
				if !ok {
					return nil, errors.New("thermostat setpoint is required")
				}
				for _, id := range targetIDs {
					if err := s.requireClimateSetpoint(ctx, id, temp); err != nil {
						return nil, err
					}
				}
				states["thermostatTemperatureSetpoint"] = temp

				for _, id := range targetIDs {
					if s.bus != nil {
						if err := s.bus.Publish(ctx, events.Event{
							ID: uuid.NewString(), Type: events.CommandRequested, OccurredAt: time.Now().UTC(),
							Source: "google_assistant", Payload: map[string]any{
								"entity_id": id, "domain": "climate",
								"service": "set_temperature", "temperature": temp,
							},
						}); err != nil { return nil, err }
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
