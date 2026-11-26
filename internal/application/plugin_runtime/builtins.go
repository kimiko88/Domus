// Package plugin_runtime composes built-in protocol adapters behind the Domus plugin contract.
package plugin_runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/domus-hub/domus/internal/infrastructure/mqtt"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	"github.com/domus-hub/domus/internal/plugins/weather_openmeteo"
	"github.com/domus-hub/domus/internal/plugins/midea"
	"github.com/domus-hub/domus/internal/plugins/xiaomi_home"
	"github.com/domus-hub/domus/internal/plugins/vesync"
	"github.com/domus-hub/domus/internal/plugins/imou_life"
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
	"github.com/domus-hub/domus/internal/plugins/tuya_cloud"
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
		APIVersion: plugins.ManifestAPIVersion, ID: tuya_cloud.PluginType, Name: "Tuya Cloud OpenAPI",
		Version: "1.0.0", Description: "Tuya regional OpenAPI provider for account-linked device discovery and commands.",
		Capabilities: []plugins.Capability{plugins.CapabilityNetworkCloud, plugins.CapabilityDevicesRead, plugins.CapabilityDevicesWrite, plugins.CapabilitySecretsRead},
		Config: []plugins.ConfigField{
			{Name: "client_id", Type: "string", Description: "Optional Tuya IoT OpenAPI application client ID; not the Smart Life App Authorization ID"},
			{Name: "secret", Type: "string", Secret: true, Description: "Optional Tuya IoT OpenAPI application secret; not the Smart Life App Authorization secret"},
			{Name: "base_url", Type: "url", Required: true, Description: "Allowed Tuya regional HTTPS endpoint"},
			{Name: "uid", Type: "string", Description: "Optional linked Smart Life user UID"},
			{Name: "schema", Type: "string", Description: "Optional Tuya project schema"},
			{Name: "oauth_client_id", Type: "string", Secret: true, Internal: true},
			{Name: "oauth_secret", Type: "string", Secret: true, Internal: true},
			{Name: "oauth_identifier", Type: "string", Secret: true, Internal: true},
			{Name: "smart_life_oauth", Type: "json", Secret: true, Internal: true},
		},
	}, func(ctx context.Context, instanceID string, raw json.RawMessage, deps plugins.Dependencies) (plugins.Adapter, error) {
		wire, err := plugins.DecodeConfig[struct {
			ClientID string `json:"client_id"`
			Secret string `json:"secret"`
			BaseURL string `json:"base_url"`
			UID string `json:"uid"`
			Schema string `json:"schema"`
			OAuthClientID string `json:"oauth_client_id"`
			OAuthSecret string `json:"oauth_secret"`
			OAuthIdentifier string `json:"oauth_identifier"`
			SmartLifeOAuth json.RawMessage `json:"smart_life_oauth"`
		}](raw)
		if err != nil { return nil, err }
		return tuya_cloud.NewAdapter(instanceID, tuya_cloud.Config{
			ClientID: wire.ClientID, Secret: wire.Secret, BaseURL: wire.BaseURL,
			UID: wire.UID, Schema: wire.Schema,
		})
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


	// Cloud vendor clients use the same supervised runtime as LAN protocol
	// adapters. Account OAuth/session material remains in the encrypted
	// integration configuration owned by each feature handler.
	timeoutField := []plugins.ConfigField{
		{Name: "http_timeout", Type: "duration", Description: "HTTP request timeout (default 12s)"},
	}
	if err := register(plugins.Manifest{
		APIVersion: plugins.ManifestAPIVersion, ID: imou_life.PluginType, Name: "Imou Life",
		Version: "1.0.0", Description: "Imou Open Platform account, camera and event integration.",
		Capabilities: []plugins.Capability{plugins.CapabilityNetworkCloud, plugins.CapabilitySecretsRead, plugins.CapabilitySecretsWrite, plugins.CapabilityEventsWrite, plugins.CapabilityDevicesRead, plugins.CapabilityDevicesWrite},
		Config: []plugins.ConfigField{
			{Name: "access_key", Type: "string", Required: true, Description: "Imou Open Platform project access key"},
			{Name: "secret_key", Type: "string", Required: true, Secret: true, Description: "Imou Open Platform project secret"},
			{Name: "data_center", Type: "string", Required: true, Description: "Imou API data centre identifier"},
			{Name: "authorization_url", Type: "url", Required: true, Description: "Imou user-consent H5 authorization URL"},
			{Name: "public_base_url", Type: "url", Required: true, Description: "Public HTTPS origin used for OAuth and event callbacks"},
			{Name: "callback_secret", Type: "string", Secret: true, Internal: true},
			{Name: "event_secret", Type: "string", Secret: true, Internal: true},
			{Name: "push_enabled", Type: "boolean", Internal: true},
			{Name: "push_flags", Type: "json", Internal: true},
			{Name: "admin_access_token", Type: "string", Secret: true, Internal: true},
			{Name: "admin_expires_at", Type: "string", Internal: true},
			{Name: "access_token", Type: "string", Secret: true, Internal: true},
			{Name: "refresh_token", Type: "string", Secret: true, Internal: true},
			{Name: "expires_at", Type: "string", Internal: true},
			{Name: "http_timeout", Type: "duration", Description: "HTTP request timeout (default 12s)"},
		},
	}, func(ctx context.Context, instanceID string, raw json.RawMessage, deps plugins.Dependencies) (plugins.Adapter, error) {
		wire, err := plugins.DecodeConfig[struct {
			AccessKey string `json:"access_key"`
			SecretKey string `json:"secret_key"`
			DataCenter string `json:"data_center"`
			AuthorizationURL string `json:"authorization_url"`
			PublicBaseURL string `json:"public_base_url"`
			CallbackSecret string `json:"callback_secret"`
			EventSecret string `json:"event_secret"`
			PushEnabled bool `json:"push_enabled"`
			PushFlags []string `json:"push_flags"`
			AdminAccessToken string `json:"admin_access_token"`
			AdminExpiresAt string `json:"admin_expires_at"`
			AccessToken string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			ExpiresAt string `json:"expires_at"`
			HTTPTimeout string `json:"http_timeout"`
		}](raw)
		if err != nil { return nil, err }
		if err := validateImouHTTPSURL(wire.AuthorizationURL, true); err != nil {
			return nil, fmt.Errorf("imou_life: invalid authorization_url: %w", err)
		}
		if err := validateImouHTTPSURL(wire.PublicBaseURL, false); err != nil {
			return nil, fmt.Errorf("imou_life: invalid public_base_url: %w", err)
		}
		if _, err := imou_life.NewClient(imou_life.Config{
			AccessKey: wire.AccessKey, SecretKey: wire.SecretKey, DataCenter: wire.DataCenter,
		}, nil); err != nil {
			return nil, fmt.Errorf("imou_life: invalid project configuration: %w", err)
		}
		timeout := 12 * time.Second
		if wire.HTTPTimeout != "" {
			timeout, err = time.ParseDuration(wire.HTTPTimeout)
			if err != nil || timeout < time.Second || timeout > 20*time.Second {
				return nil, errors.New("imou_life: http_timeout must be between 1s and 20s")
			}
		}
		return imou_life.NewAdapter(instanceID, timeout)
	}); err != nil { return err }

	if err := register(plugins.Manifest{
		APIVersion: plugins.ManifestAPIVersion, ID: vesync.PluginType, Name: "VeSync Cloud",
		Version: "1.0.0", Description: "Persistent VeSync account session and cloud device control.",
		Capabilities: []plugins.Capability{plugins.CapabilityNetworkCloud, plugins.CapabilitySecretsRead, plugins.CapabilityDevicesRead, plugins.CapabilityDevicesWrite},
		Config: []plugins.ConfigField{
			{Name: "http_timeout", Type: "duration", Description: "Cloud API request timeout; defaults to 8s and is limited to 1s..20s"},
			{Name: "session", Type: "json", Secret: true, Internal: true, Description: "Encrypted provider session; managed by the VeSync account-linking flow"},
		},
	}, func(ctx context.Context, instanceID string, raw json.RawMessage, deps plugins.Dependencies) (plugins.Adapter, error) {
		wire, err := plugins.DecodeConfig[struct {
			Session vesync.Session `json:"session"`
			HTTPTimeout string `json:"http_timeout"`
		}](raw)
		if err != nil { return nil, err }
		timeout := 8 * time.Second
		if wire.HTTPTimeout != "" {
			timeout, err = time.ParseDuration(wire.HTTPTimeout)
			if err != nil || timeout < time.Second || timeout > 20*time.Second {
				return nil, errors.New("vesync: http_timeout must be between 1s and 20s")
			}
		}
		return vesync.NewAdapterWithSession(instanceID, wire.Session, timeout)
	}); err != nil { return err }

	if err := register(plugins.Manifest{
		APIVersion: plugins.ManifestAPIVersion, ID: "xiaomi_miio_transport", Name: "Xiaomi Home Local (miIO)",
		Internal: true,
		Version: "1.0.0", Description: "Local miIO discovery and authenticated LAN control.",
		Capabilities: []plugins.Capability{plugins.CapabilityNetworkLAN, plugins.CapabilitySecretsRead, plugins.CapabilityDevicesRead, plugins.CapabilityDevicesWrite},
		Config: timeoutField,
	}, func(ctx context.Context, instanceID string, raw json.RawMessage, deps plugins.Dependencies) (plugins.Adapter, error) {
		wire, err := plugins.DecodeConfig[struct { HTTPTimeout string `json:"http_timeout"` }](raw)
		if err != nil { return nil, err }
		timeout := 2 * time.Second
		if wire.HTTPTimeout != "" { timeout, err = time.ParseDuration(wire.HTTPTimeout); if err != nil { return nil, fmt.Errorf("xiaomi_home_local: invalid timeout: %w", err) } }
		return xiaomi_home.NewLocalAdapter(instanceID, timeout)
	}); err != nil { return err }

	if err := register(plugins.Manifest{
		APIVersion: plugins.ManifestAPIVersion, ID: xiaomi_home.CloudPluginType, Name: "Xiaomi Home Cloud",
		Version: "1.0.0", Description: "Xiaomi Open Platform OAuth account and MIoT cloud device integration.",
		Capabilities: []plugins.Capability{plugins.CapabilityNetworkCloud, plugins.CapabilitySecretsRead, plugins.CapabilitySecretsWrite, plugins.CapabilityDevicesRead, plugins.CapabilityDevicesWrite, plugins.CapabilityEventsWrite},
		Config: []plugins.ConfigField{
			{Name: "client_id", Type: "string", Required: true, Description: "Xiaomi Open Platform application ID"},
			{Name: "client_secret", Type: "string", Required: true, Secret: true, Description: "Xiaomi Open Platform application secret"},
			{Name: "region", Type: "string", Required: true, Allowed: []string{"eu", "de", "us", "sg"}, Description: "Xiaomi data center"},
			{Name: "public_base_url", Type: "url", Required: true, Description: "Public HTTPS base URL used for OAuth and event callbacks"},
			{Name: "redirect_url", Type: "url", Internal: true},
			{Name: "callback_secret", Type: "string", Secret: true, Internal: true},
			{Name: "event_secret", Type: "string", Secret: true, Internal: true},
			{Name: "oauth_state", Type: "string", Secret: true, Internal: true},
			{Name: "access_token", Type: "string", Secret: true, Internal: true},
			{Name: "refresh_token", Type: "string", Secret: true, Internal: true},
			{Name: "mac_key", Type: "string", Secret: true, Internal: true},
			{Name: "mac_algorithm", Type: "string", Internal: true},
			{Name: "expires_at", Type: "string", Internal: true},
			{Name: "user_id", Type: "string", Secret: true, Internal: true},
			{Name: "nickname", Type: "string", Internal: true},
			{Name: "http_timeout", Type: "duration", Description: "HTTP request timeout (default 12s)"},
		},
	}, func(ctx context.Context, instanceID string, raw json.RawMessage, deps plugins.Dependencies) (plugins.Adapter, error) {
		// Decode the complete persisted option flow, not only transport settings.
		// Registry validation has already checked the schema; accepting these
		// fields here ensures OAuth credentials and refreshed tokens survive the
		// adapter's atomic replacement lifecycle.
		wire, err := plugins.DecodeConfig[struct {
			ClientID string `json:"client_id"`
			ClientSecret string `json:"client_secret"`
			Region string `json:"region"`
			PublicBaseURL string `json:"public_base_url"`
			RedirectURL string `json:"redirect_url"`
			CallbackSecret string `json:"callback_secret"`
			EventSecret string `json:"event_secret"`
			OAuthState string `json:"oauth_state"`
			AccessToken string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			MacKey string `json:"mac_key"`
			MacAlgorithm string `json:"mac_algorithm"`
			ExpiresAt string `json:"expires_at"`
			UserID string `json:"user_id"`
			Nickname string `json:"nickname"`
			HTTPTimeout string `json:"http_timeout"`
		}](raw)
		if err != nil { return nil, err }
		baseURL, parseErr := url.ParseRequestURI(strings.TrimSpace(wire.PublicBaseURL))
		if parseErr != nil || baseURL.Scheme != "https" || baseURL.Host == "" || baseURL.User != nil || baseURL.RawQuery != "" || baseURL.Fragment != "" {
			return nil, errors.New("xiaomi_home_cloud: public_base_url must be an HTTPS URL without credentials, query or fragment")
		}
		redirectURL := strings.TrimSpace(wire.RedirectURL)
		if redirectURL == "" {
			redirectURL = strings.TrimRight(strings.TrimSpace(wire.PublicBaseURL), "/") + "/api/v1/integrations/xiaomi-home/cloud/callback/" + strings.Repeat("a", 64)
		}
		if _, err := xiaomi_home.NewCloudClient(xiaomi_home.CloudConfig{
			ClientID: wire.ClientID, ClientSecret: wire.ClientSecret,
			Region: wire.Region, RedirectURL: redirectURL,
		}, nil); err != nil {
			return nil, fmt.Errorf("xiaomi_home_cloud: invalid Open Platform configuration: %w", err)
		}
		timeout := 12 * time.Second
		if wire.HTTPTimeout != "" { timeout, err = time.ParseDuration(wire.HTTPTimeout); if err != nil { return nil, fmt.Errorf("xiaomi_home_cloud: invalid http_timeout: %w", err) } }
		return xiaomi_home.NewCloudAdapter(instanceID, timeout)
	}); err != nil { return err }
	if err := register(plugins.Manifest{
		APIVersion: plugins.ManifestAPIVersion, ID: "midea_a1_protocol", Name: "Midea SmartHome (M-Smart A1)",
		Internal: true,
		Version: "1.0.0", Description: "Local M-Smart V2/V3 protocol for supported A1 dehumidifiers.",
		Capabilities: []plugins.Capability{plugins.CapabilityNetworkLAN, plugins.CapabilitySecretsRead, plugins.CapabilitySecretsWrite, plugins.CapabilityDevicesRead, plugins.CapabilityDevicesWrite, plugins.CapabilityEventsRead, plugins.CapabilityEventsWrite},
	}, func(ctx context.Context, instanceID string, raw json.RawMessage, deps plugins.Dependencies) (plugins.Adapter, error) {
		if _, err := plugins.DecodeConfig[map[string]any](raw); err != nil { return nil, err }
		return midea.NewRuntimeAdapter(instanceID)
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

func validateImouHTTPSURL(raw string, allowQuery bool) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Fragment != "" || (!allowQuery && u.RawQuery != "") {
		return errors.New("URL must be absolute HTTPS without credentials or fragment (query is allowed only where documented)")
	}
	return nil
}
