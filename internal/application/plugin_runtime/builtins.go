// Package plugin_runtime composes built-in protocol adapters behind the Domus plugin contract.
package plugin_runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/domus-hub/domus/internal/infrastructure/mqtt"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	"github.com/domus-hub/domus/internal/plugins/weather_openmeteo"
	"github.com/domus-hub/domus/internal/plugins"
	"github.com/domus-hub/domus/internal/plugins/ble_proxy"
	"github.com/domus-hub/domus/internal/plugins/broadlink"
	"github.com/domus-hub/domus/internal/plugins/esphome_mqtt"
	"github.com/domus-hub/domus/internal/plugins/knx"
	"github.com/domus-hub/domus/internal/plugins/matter"
	"github.com/domus-hub/domus/internal/plugins/modbus_tcp"
	"github.com/domus-hub/domus/internal/plugins/mqtt_generic"
	"github.com/domus-hub/domus/internal/plugins/open_display"
	"github.com/domus-hub/domus/internal/plugins/tuya_local"
	"github.com/domus-hub/domus/internal/plugins/zwave_js"
	"github.com/domus-hub/domus/internal/plugins/voice_satellite"
	"github.com/domus-hub/domus/internal/plugins/zigbee2mqtt"
)

type builtinHost struct {
	repos *sqlite.Repositories
}

func registerBuiltins(registry *plugins.Registry, host builtinHost) error {
	register := func(manifest plugins.Manifest, factory plugins.Factory) error {
		return registry.Register(manifest, factory)
	}
	mqttFields := []plugins.ConfigField{
		{Name: "broker_url", Type: "url", Required: true, Description: "Broker URL, e.g. tcp://192.168.1.10:1883 or ssl://broker:8883"},
		{Name: "client_id", Type: "string", Description: "MQTT client identifier; defaults to the Domus instance ID"},
		{Name: "username", Type: "string", Description: "Optional broker username"},
		{Name: "password", Type: "string", Secret: true, Description: "Optional broker password"},
		{Name: "keep_alive", Type: "duration", Description: "MQTT keep-alive duration"},
		{Name: "clean_session", Type: "boolean", Description: "Start with a clean MQTT session"},
	}
	mqttClient := func(instanceID string, wire mqttWireConfig) (mqtt.Client, error) {
		brokerURL := strings.TrimSpace(wire.BrokerURL)
		if brokerURL == "" {
			return nil, errors.New("mqtt: broker_url is required")
		}
		clientID := strings.TrimSpace(wire.ClientID)
		if clientID == "" {
			clientID = "domus_" + instanceID
		}
		cleanSession := true
		if wire.CleanSession != nil {
			cleanSession = *wire.CleanSession
		}
		keepAlive := 30 * time.Second
		if wire.KeepAlive != "" {
			parsed, err := time.ParseDuration(wire.KeepAlive)
			if err != nil || parsed < 5*time.Second || parsed > 10*time.Minute {
				return nil, errors.New("mqtt: keep_alive must be between 5s and 10m")
			}
			keepAlive = parsed
		}
		client := mqtt.NewPahoClient(mqtt.Config{
			BrokerURL: brokerURL, ClientID: clientID, Username: wire.Username,
			Password: wire.Password, CleanSession: cleanSession, KeepAlive: keepAlive,
		})
		return client, nil
	}

	if err := register(plugins.Manifest{
		APIVersion: plugins.ManifestAPIVersion, ID: "mqtt_generic", Name: "MQTT Generic",
		Version: "1.0.0", Description: "Generic MQTT state and command topics.",
		MultiInstance: true,
		Capabilities: []plugins.Capability{plugins.CapabilityEventsRead, plugins.CapabilityEventsWrite, plugins.CapabilityDevicesRead, plugins.CapabilityDevicesWrite, plugins.CapabilityNetworkLAN},
		Config: append(append([]plugins.ConfigField{}, mqttFields...),
			plugins.ConfigField{Name: "state_topic_pattern", Type: "string", Description: "State subscription pattern; defaults to domus/+/state"},
			plugins.ConfigField{Name: "command_topic_template", Type: "string", Description: "Command topic template; defaults to domus/%s/set"},
		),
	}, func(ctx context.Context, instanceID string, raw json.RawMessage, deps plugins.Dependencies) (plugins.Adapter, error) {
		wire, err := plugins.DecodeConfig[mqttWireConfig](raw)
		if err != nil { return nil, err }
		client, err := mqttClient(instanceID, wire)
		if err != nil { return nil, err }
		adapter := mqtt_generic.NewAdapter(mqtt_generic.Config{IntegrationID: instanceID, StateTopicPattern: wire.StateTopicPattern, CommandTopicTemplate: wire.CommandTopicTemplate}, client, deps.Entities, deps.Devices, deps.Bus)
		return &managedMQTTAdapter{inner: adapter, client: client}, nil
	}); err != nil { return err }

	if err := register(plugins.Manifest{
		APIVersion: plugins.ManifestAPIVersion, ID: "zigbee2mqtt", Name: "Zigbee2MQTT",
		Version: "1.0.0", Description: "Connect a Zigbee2MQTT bridge over MQTT.",
		MultiInstance: true,
		Capabilities: []plugins.Capability{plugins.CapabilityEventsRead, plugins.CapabilityEventsWrite, plugins.CapabilityDevicesRead, plugins.CapabilityDevicesWrite, plugins.CapabilityNetworkLAN},
		Config: append(append([]plugins.ConfigField{}, mqttFields...), plugins.ConfigField{Name: "base_topic", Type: "string", Description: "Zigbee2MQTT base topic; defaults to zigbee2mqtt"}),
	}, func(ctx context.Context, instanceID string, raw json.RawMessage, deps plugins.Dependencies) (plugins.Adapter, error) {
		wire, err := plugins.DecodeConfig[mqttWireConfig](raw)
		if err != nil { return nil, err }
		client, err := mqttClient(instanceID, wire)
		if err != nil { return nil, err }
		adapter := zigbee2mqtt.NewAdapter(zigbee2mqtt.Config{IntegrationID: instanceID, BaseTopic: wire.BaseTopic}, client, deps.Devices, deps.Entities, deps.Bus)
		return &managedMQTTAdapter{inner: adapter, client: client}, nil
	}); err != nil { return err }

	if err := register(plugins.Manifest{
		APIVersion: plugins.ManifestAPIVersion, ID: "esphome_mqtt", Name: "ESPHome MQTT",
		Version: "1.0.0", Description: "Connect ESPHome nodes via their MQTT API.",
		MultiInstance: true,
		Capabilities: []plugins.Capability{plugins.CapabilityEventsRead, plugins.CapabilityEventsWrite, plugins.CapabilityDevicesRead, plugins.CapabilityDevicesWrite, plugins.CapabilityNetworkLAN},
		Config: append(append([]plugins.ConfigField{}, mqttFields...), plugins.ConfigField{Name: "topic_prefix", Type: "string", Description: "Optional ESPHome MQTT topic prefix"}),
	}, func(ctx context.Context, instanceID string, raw json.RawMessage, deps plugins.Dependencies) (plugins.Adapter, error) {
		wire, err := plugins.DecodeConfig[mqttWireConfig](raw)
		if err != nil { return nil, err }
		client, err := mqttClient(instanceID, wire)
		if err != nil { return nil, err }
		adapter := esphome_mqtt.NewAdapter(esphome_mqtt.Config{IntegrationID: instanceID, TopicPrefix: wire.TopicPrefix}, client, deps.Devices, deps.Entities, deps.Bus)
		return &managedMQTTAdapter{inner: adapter, client: client}, nil
	}); err != nil { return err }

	if err := register(plugins.Manifest{
		APIVersion: plugins.ManifestAPIVersion, ID: "knx", Name: "KNXnet/IP",
		Version: "1.0.0", Description: "Connect a KNX/IP router and map group addresses to Domus entities.",
		MultiInstance: true,
		Capabilities: []plugins.Capability{plugins.CapabilityEventsRead, plugins.CapabilityEventsWrite, plugins.CapabilityDevicesRead, plugins.CapabilityDevicesWrite, plugins.CapabilityNetworkLAN},
		Config: []plugins.ConfigField{
			{Name: "gateway_addr", Type: "string", Required: true, Description: "KNX/IP gateway host:port"},
			{Name: "entities", Type: "json", Required: true, Description: "Array of KNX entity mappings"},
		},
	}, func(ctx context.Context, instanceID string, raw json.RawMessage, deps plugins.Dependencies) (plugins.Adapter, error) {
		wire, err := plugins.DecodeConfig[struct {
			GatewayAddr string `json:"gateway_addr"`
			Entities []knx.EntityMapping `json:"entities"`
		}](raw)
		if err != nil { return nil, err }
		if _, _, err := net.SplitHostPort(wire.GatewayAddr); err != nil { return nil, fmt.Errorf("knx: gateway_addr must be host:port") }
		return knx.NewAdapter(knx.Config{IntegrationID: instanceID, GatewayAddr: wire.GatewayAddr, Entities: wire.Entities}, knx.NewUDPClient(wire.GatewayAddr, deps.Logger), deps.Entities, deps.Bus, deps.Logger)
	}); err != nil { return err }

	if err := register(plugins.Manifest{
		APIVersion: plugins.ManifestAPIVersion, ID: "modbus_tcp", Name: "Modbus TCP",
		Version: "1.0.0", Description: "Poll Modbus TCP registers and expose mapped values as Domus entities.",
		MultiInstance: true,
		Capabilities: []plugins.Capability{plugins.CapabilityEventsRead, plugins.CapabilityEventsWrite, plugins.CapabilityDevicesRead, plugins.CapabilityDevicesWrite, plugins.CapabilityNetworkLAN},
		Config: []plugins.ConfigField{
			{Name: "host", Type: "string", Required: true}, {Name: "port", Type: "integer", Description: "Default 502"},
			{Name: "unit_id", Type: "integer", Description: "Modbus unit ID; default 1"},
			{Name: "poll_interval", Type: "duration", Description: "Default 5s"},
			{Name: "registers", Type: "json", Required: true, Description: "Array of Modbus register mappings"},
		},
	}, func(ctx context.Context, instanceID string, raw json.RawMessage, deps plugins.Dependencies) (plugins.Adapter, error) {
		wire, err := plugins.DecodeConfig[struct {
			Host string `json:"host"`
			Port int `json:"port"`
			UnitID int `json:"unit_id"`
			PollInterval string `json:"poll_interval"`
			Registers []modbus_tcp.RegisterMapping `json:"registers"`
		}](raw)
		if err != nil { return nil, err }
		if net.ParseIP(wire.Host) == nil && (strings.ContainsAny(wire.Host, " /\\") || strings.TrimSpace(wire.Host) == "") {
			return nil, errors.New("modbus_tcp: host must be an IP address or hostname")
		}
		if wire.UnitID < 0 || wire.UnitID > 255 { return nil, errors.New("modbus_tcp: unit_id must be 0..255") }
		interval := 5 * time.Second
		if wire.PollInterval != "" {
			var err error
			interval, err = time.ParseDuration(wire.PollInterval)
			if err != nil || interval < time.Second || interval > 10*time.Minute { return nil, errors.New("modbus_tcp: poll_interval must be 1s..10m") }
		}
		port := wire.Port
		if port == 0 { port = 502 }
		if port < 1 || port > 65535 { return nil, errors.New("modbus_tcp: port must be 1..65535") }
		client := modbus_tcp.NewTCPClient(net.JoinHostPort(wire.Host, strconv.Itoa(port)), 5*time.Second)
		return modbus_tcp.NewAdapter(modbus_tcp.Config{IntegrationID: instanceID, Host: wire.Host, Port: port, UnitID: uint8(wire.UnitID), PollInterval: interval, Registers: wire.Registers}, client, deps.Entities, deps.Bus, deps.Logger), nil
	}); err != nil { return err }

	if err := register(plugins.Manifest{
		APIVersion: plugins.ManifestAPIVersion, ID: "zwave_js", Name: "Z-Wave JS UI",
		Version: "1.0.0", Description: "Connect a Z-Wave JS UI gateway over authenticated WebSocket.",
		MultiInstance: true,
		Capabilities: []plugins.Capability{plugins.CapabilityEventsRead, plugins.CapabilityEventsWrite, plugins.CapabilityDevicesRead, plugins.CapabilityDevicesWrite, plugins.CapabilityNetworkLAN},
		Config: []plugins.ConfigField{
			{Name: "ws_url", Type: "url", Required: true, Description: "Z-Wave JS UI WebSocket URL (ws:// or wss://)"},
			{Name: "auth_token", Type: "string", Secret: true, Description: "Optional Bearer token for the WebSocket bridge"},
			{Name: "reconnect_interval", Type: "duration", Description: "Reconnect delay, default 5s"},
		},
	}, func(ctx context.Context, instanceID string, raw json.RawMessage, deps plugins.Dependencies) (plugins.Adapter, error) {
		wire, err := plugins.DecodeConfig[struct {
			WSURL string `json:"ws_url"`
			AuthToken string `json:"auth_token"`
			ReconnectInterval string `json:"reconnect_interval"`
		}](raw)
		if err != nil { return nil, err }
		interval := 5 * time.Second
		if wire.ReconnectInterval != "" {
			interval, err = time.ParseDuration(wire.ReconnectInterval)
			if err != nil { return nil, fmt.Errorf("zwave_js: invalid reconnect_interval: %w", err) }
		}
		return zwave_js.NewAdapterWithConfig(zwave_js.Config{
			ID: instanceID, WebSocketURL: wire.WSURL, AuthToken: wire.AuthToken,
			ReconnectInterval: interval, Logger: deps.Logger,
		}, deps.Bus)
	}); err != nil { return err }

	if err := register(plugins.Manifest{
		APIVersion: plugins.ManifestAPIVersion, ID: "weather_openmeteo", Name: "Open-Meteo Forecast",
		Version: "1.0.0", Description: "Forecast provider for current conditions, hourly temperature, rain probability and solar radiation.",
		Capabilities: []plugins.Capability{plugins.CapabilityNetworkCloud},
	}, func(ctx context.Context, instanceID string, raw json.RawMessage, deps plugins.Dependencies) (plugins.Adapter, error) {
		if _, err := plugins.DecodeConfig[map[string]any](raw); err != nil { return nil, err }
		return weather_openmeteo.NewAdapter(instanceID), nil
	}); err != nil { return err }

	if err := register(plugins.Manifest{
		APIVersion: plugins.ManifestAPIVersion, ID: "matter", Name: "Matter / Thread",
		Version: "1.0.0", Description: "Matter controller adapter and commissioning workflow (controller transport remains experimental).",
		Capabilities: []plugins.Capability{plugins.CapabilityEventsRead, plugins.CapabilityEventsWrite, plugins.CapabilityDevicesRead, plugins.CapabilityDevicesWrite},
	}, func(ctx context.Context, instanceID string, raw json.RawMessage, deps plugins.Dependencies) (plugins.Adapter, error) {
		if _, err := plugins.DecodeConfig[map[string]any](raw); err != nil { return nil, err }
		return matter.NewAdapter(instanceID, deps.Bus), nil
	}); err != nil { return err }

	if err := register(plugins.Manifest{
		APIVersion: plugins.ManifestAPIVersion, ID: "ble_proxy", Name: "ESPHome BLE Proxy",
		Version: "1.0.0", Description: "Decode forwarded BLE advertisements (BTHome v2 and iBeacon) from LAN proxies.",
		MultiInstance: true,
		Capabilities: []plugins.Capability{plugins.CapabilityEventsRead, plugins.CapabilityDevicesWrite, plugins.CapabilityNetworkLAN},
	}, func(ctx context.Context, instanceID string, raw json.RawMessage, deps plugins.Dependencies) (plugins.Adapter, error) {
		if _, err := plugins.DecodeConfig[map[string]any](raw); err != nil { return nil, err }
		return ble_proxy.NewAdapter(instanceID, deps.Bus), nil
	}); err != nil { return err }

	if host.repos != nil && host.repos.DB != nil && host.repos.Cipher != nil {
		if err := register(plugins.Manifest{
			APIVersion: plugins.ManifestAPIVersion, ID: "tuya_local", Name: "Tuya Local",
			Version: "1.0.0", Description: "Local-first Tuya control using per-device local keys stored encrypted in Domus.",
			Capabilities: []plugins.Capability{plugins.CapabilityEventsRead, plugins.CapabilityEventsWrite, plugins.CapabilityDevicesRead, plugins.CapabilityDevicesWrite, plugins.CapabilityNetworkLAN, plugins.CapabilitySecretsRead, plugins.CapabilitySecretsWrite},
		}, func(ctx context.Context, instanceID string, raw json.RawMessage, deps plugins.Dependencies) (plugins.Adapter, error) {
			if _, err := plugins.DecodeConfig[map[string]any](raw); err != nil { return nil, err }
			return tuya_local.NewAdapter(instanceID, deps.Bus).
				WithPersistence(host.repos.DB, host.repos.Cipher).
				WithRepositories(deps.Devices, deps.Entities), nil
		}); err != nil { return err }
		if err := register(plugins.Manifest{
			APIVersion: plugins.ManifestAPIVersion, ID: "broadlink_local", Name: "BroadLink Local",
			Version: "1.0.0", Description: "Local IR/RF remote control with encrypted device keys and saved commands.",
			Capabilities: []plugins.Capability{plugins.CapabilityEventsRead, plugins.CapabilityEventsWrite, plugins.CapabilityDevicesRead, plugins.CapabilityDevicesWrite, plugins.CapabilityNetworkLAN, plugins.CapabilitySecretsRead, plugins.CapabilitySecretsWrite},
		}, func(ctx context.Context, instanceID string, raw json.RawMessage, deps plugins.Dependencies) (plugins.Adapter, error) {
			if _, err := plugins.DecodeConfig[map[string]any](raw); err != nil { return nil, err }
			return broadlink.NewAdapter(instanceID, deps.Bus).
				WithPersistence(host.repos.DB, host.repos.Cipher).
				WithRepositories(deps.Devices, deps.Entities), nil
		}); err != nil { return err }
	}

	if err := register(plugins.Manifest{
		APIVersion: plugins.ManifestAPIVersion, ID: "voice_satellite", Name: "Domus Voice Satellite",
		Version: "1.0.0", Description: "Linux microphone/speaker satellite with local Wyoming wake-word detection.",
		MultiInstance: true,
		Capabilities: []plugins.Capability{plugins.CapabilityNetworkLAN, plugins.CapabilityAudioCapture},
		Config: []plugins.ConfigField{
			{Name: "server_url", Type: "url", Required: true, Description: "Domus local HTTP(S) base URL"},
			{Name: "bearer_token", Type: "string", Required: true, Secret: true, Description: "Dedicated token for this voice satellite"},
			{Name: "wake_address", Type: "string", Required: true, Description: "Wyoming wake-word service host:port"},
			{Name: "wake_words", Type: "json", Description: "One to eight Wyoming wake-word model names"},
			{Name: "audio_device", Type: "string", Description: "ALSA capture device; defaults to default"},
			{Name: "speaker_device", Type: "string", Description: "ALSA playback device"},
			{Name: "sample_rate", Type: "integer", Description: "Only 16000 Hz is supported"},
			{Name: "frame_duration", Type: "duration"},
			{Name: "speech_threshold", Type: "number"},
			{Name: "silence_duration", Type: "duration"},
			{Name: "max_command_duration", Type: "duration"},
			{Name: "http_timeout", Type: "duration"},
		},
	}, func(ctx context.Context, instanceID string, raw json.RawMessage, deps plugins.Dependencies) (plugins.Adapter, error) {
		wire, err := plugins.DecodeConfig[struct {
			ServerURL string `json:"server_url"`
			BearerToken string `json:"bearer_token"`
			WakeAddress string `json:"wake_address"`
			WakeWords []string `json:"wake_words"`
			AudioDevice string `json:"audio_device"`
			SpeakerDevice string `json:"speaker_device"`
			SampleRate int `json:"sample_rate"`
			FrameDuration string `json:"frame_duration"`
			SpeechThreshold float64 `json:"speech_threshold"`
			SilenceDuration string `json:"silence_duration"`
			MaxCommandDuration string `json:"max_command_duration"`
			HTTPTimeout string `json:"http_timeout"`
		}](raw)
		if err != nil { return nil, err }
		parseDuration := func(name, raw string) (time.Duration, error) {
			if raw == "" { return 0, nil }
			value, err := time.ParseDuration(raw)
			if err != nil { return 0, fmt.Errorf("voice_satellite: invalid %s: %w", name, err) }
			return value, nil
		}
		frame, err := parseDuration("frame_duration", wire.FrameDuration); if err != nil { return nil, err }
		silence, err := parseDuration("silence_duration", wire.SilenceDuration); if err != nil { return nil, err }
		maxCommand, err := parseDuration("max_command_duration", wire.MaxCommandDuration); if err != nil { return nil, err }
		httpTimeout, err := parseDuration("http_timeout", wire.HTTPTimeout); if err != nil { return nil, err }
		return voice_satellite.NewAdapter(voice_satellite.Config{
			SatelliteID: instanceID, ServerURL: wire.ServerURL, BearerToken: wire.BearerToken,
			WakeAddress: wire.WakeAddress, WakeWords: wire.WakeWords, AudioDevice: wire.AudioDevice,
			SpeakerDevice: wire.SpeakerDevice, SampleRate: wire.SampleRate, FrameDuration: frame,
			SpeechThreshold: wire.SpeechThreshold, SilenceDuration: silence,
			MaxCommandDuration: maxCommand, HTTPTimeout: httpTimeout,
		}, deps.Logger)
	}); err != nil { return err }

	if err := open_display.RegisterPlugin(registry); err != nil { return err }
	return nil
}

type mqttWireConfig struct {
	BrokerURL string `json:"broker_url"`
	ClientID string `json:"client_id,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	KeepAlive string `json:"keep_alive,omitempty"`
	CleanSession *bool `json:"clean_session,omitempty"`
	StateTopicPattern string `json:"state_topic_pattern,omitempty"`
	CommandTopicTemplate string `json:"command_topic_template,omitempty"`
	BaseTopic string `json:"base_topic,omitempty"`
	TopicPrefix string `json:"topic_prefix,omitempty"`
}

type managedMQTTAdapter struct {
	inner plugins.Adapter
	client mqtt.Client
}

func (a *managedMQTTAdapter) ID() string { return a.inner.ID() }
func (a *managedMQTTAdapter) Type() string { return a.inner.Type() }
func (a *managedMQTTAdapter) HealthCheck(ctx context.Context) error { return a.inner.HealthCheck(ctx) }
func (a *managedMQTTAdapter) Start(ctx context.Context) error {
	if err := a.client.Connect(ctx); err != nil { return fmt.Errorf("mqtt: connect: %w", err) }
	if err := a.inner.Start(ctx); err != nil {
		a.client.Disconnect(250)
		return err
	}
	return nil
}
func (a *managedMQTTAdapter) Stop(ctx context.Context) error {
	err := a.inner.Stop(ctx)
	a.client.Disconnect(250)
	return err
}
