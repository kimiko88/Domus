// Package plugin_runtime composes built-in protocol adapters behind the Domus plugin contract.
package plugin_runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/domus-hub/domus/internal/infrastructure/mqtt"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
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
	mqttClient := func(instanceID string, wire mqttWireConfig) (*managedMQTTAdapter, mqtt.Client, error) {
		brokerURL := strings.TrimSpace(wire.BrokerURL)
		if brokerURL == "" {
			return nil, nil, errors.New("mqtt: broker_url is required")
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
				return nil, nil, errors.New("mqtt: keep_alive must be between 5s and 10m")
			}
			keepAlive = parsed
		}
		client := mqtt.NewPahoClient(mqtt.Config{
			BrokerURL: brokerURL, ClientID: clientID, Username: wire.Username,
			Password: wire.Password, CleanSession: cleanSession, KeepAlive: keepAlive,
		})
		return nil, client, nil
	}
	_ = mqttClient

	if err := register(plugins.Manifest{
		APIVersion: plugins.ManifestAPIVersion, ID: "mqtt_generic", Name: "MQTT Generic",
		Version: "1.0.0", Description: "Generic MQTT state and command topics.",
		Capabilities: []plugins.Capability{plugins.CapabilityEventsRead, plugins.CapabilityEventsWrite, plugins.CapabilityDevicesRead, plugins.CapabilityDevicesWrite, plugins.CapabilityNetworkLAN},
		Config: append(append([]plugins.ConfigField{}, mqttFields...),
			plugins.ConfigField{Name: "state_topic_pattern", Type: "string", Description: "State subscription pattern; defaults to domus/+/state"},
			plugins.ConfigField{Name: "command_topic_template", Type: "string", Description: "Command topic template; defaults to domus/%s/set"},
		),
	}, func(ctx context.Context, instanceID string, raw json.RawMessage, deps plugins.Dependencies) (plugins.Adapter, error) {
		var wire mqttWireConfig
		if err := plugins.DecodeConfig(raw, &wire); err != nil { return nil, err }
		_, client, err := mqttClient(instanceID, wire)
		if err != nil { return nil, err }
		var topics struct { State string `json:"state_topic_pattern"`; Command string `json:"command_topic_template"` }
		_ = json.Unmarshal(raw, &topics)
		adapter := mqtt_generic.NewAdapter(mqtt_generic.Config{IntegrationID: instanceID, StateTopicPattern: topics.State, CommandTopicTemplate: topics.Command}, client, deps.Entities, deps.Devices, deps.Bus)
		return &managedMQTTAdapter{inner: adapter, client: client}, nil
	}); err != nil { return err }

	if err := register(plugins.Manifest{
		APIVersion: plugins.ManifestAPIVersion, ID: "zigbee2mqtt", Name: "Zigbee2MQTT",
		Version: "1.0.0", Description: "Connect a Zigbee2MQTT bridge over MQTT.",
		Capabilities: []plugins.Capability{plugins.CapabilityEventsRead, plugins.CapabilityEventsWrite, plugins.CapabilityDevicesRead, plugins.CapabilityDevicesWrite, plugins.CapabilityNetworkLAN},
		Config: append(append([]plugins.ConfigField{}, mqttFields...), plugins.ConfigField{Name: "base_topic", Type: "string", Description: "Zigbee2MQTT base topic; defaults to zigbee2mqtt"}),
	}, func(ctx context.Context, instanceID string, raw json.RawMessage, deps plugins.Dependencies) (plugins.Adapter, error) {
		var wire mqttWireConfig
		if err := plugins.DecodeConfig(raw, &wire); err != nil { return nil, err }
		_, client, err := mqttClient(instanceID, wire)
		if err != nil { return nil, err }
		var topic struct { Base string `json:"base_topic"` }
		_ = json.Unmarshal(raw, &topic)
		adapter := zigbee2mqtt.NewAdapter(zigbee2mqtt.Config{IntegrationID: instanceID, BaseTopic: topic.Base}, client, deps.Devices, deps.Entities, deps.Bus)
		return &managedMQTTAdapter{inner: adapter, client: client}, nil
	}); err != nil { return err }

	if err := register(plugins.Manifest{
		APIVersion: plugins.ManifestAPIVersion, ID: "esphome_mqtt", Name: "ESPHome MQTT",
		Version: "1.0.0", Description: "Connect ESPHome nodes via their MQTT API.",
		Capabilities: []plugins.Capability{plugins.CapabilityEventsRead, plugins.CapabilityEventsWrite, plugins.CapabilityDevicesRead, plugins.CapabilityDevicesWrite, plugins.CapabilityNetworkLAN},
		Config: append(append([]plugins.ConfigField{}, mqttFields...), plugins.ConfigField{Name: "topic_prefix", Type: "string", Description: "Optional ESPHome MQTT topic prefix"}),
	}, func(ctx context.Context, instanceID string, raw json.RawMessage, deps plugins.Dependencies) (plugins.Adapter, error) {
		var wire mqttWireConfig
		if err := plugins.DecodeConfig(raw, &wire); err != nil { return nil, err }
		_, client, err := mqttClient(instanceID, wire)
		if err != nil { return nil, err }
		var topic struct { Prefix string `json:"topic_prefix"` }
		_ = json.Unmarshal(raw, &topic)
		adapter := esphome_mqtt.NewAdapter(esphome_mqtt.Config{IntegrationID: instanceID, TopicPrefix: topic.Prefix}, client, deps.Devices, deps.Entities, deps.Bus)
		return &managedMQTTAdapter{inner: adapter, client: client}, nil
	}); err != nil { return err }

	if err := register(plugins.Manifest{
		APIVersion: plugins.ManifestAPIVersion, ID: "knx", Name: "KNXnet/IP",
		Version: "1.0.0", Description: "Connect a KNX/IP router and map group addresses to Domus entities.",
		Capabilities: []plugins.Capability{plugins.CapabilityEventsRead, plugins.CapabilityEventsWrite, plugins.CapabilityDevicesRead, plugins.CapabilityDevicesWrite, plugins.CapabilityNetworkLAN},
		Config: []plugins.ConfigField{
			{Name: "gateway_addr", Type: "string", Required: true, Description: "KNX/IP gateway host:port"},
			{Name: "entities", Type: "string", Required: true, Description: "JSON array of KNX entity mappings encoded as a string"},
		},
	}, func(ctx context.Context, instanceID string, raw json.RawMessage, deps plugins.Dependencies) (plugins.Adapter, error) {
		var wire struct {
			GatewayAddr string `json:"gateway_addr"`
			Entities []knx.EntityMapping `json:"entities"`
		}
		if err := plugins.DecodeConfig(raw, &wire); err != nil { return nil, err }
		if _, _, err := net.SplitHostPort(wire.GatewayAddr); err != nil { return nil, fmt.Errorf("knx: gateway_addr must be host:port") }
		return knx.NewAdapter(knx.Config{IntegrationID: instanceID, GatewayAddr: wire.GatewayAddr, Entities: wire.Entities}, knx.NewUDPClient(wire.GatewayAddr, deps.Logger), deps.Entities, deps.Bus, deps.Logger)
	}); err != nil { return err }

	if err := register(plugins.Manifest{
		APIVersion: plugins.ManifestAPIVersion, ID: "modbus_tcp", Name: "Modbus TCP",
		Version: "1.0.0", Description: "Poll Modbus TCP registers and expose mapped values as Domus entities.",
		Capabilities: []plugins.Capability{plugins.CapabilityEventsRead, plugins.CapabilityEventsWrite, plugins.CapabilityDevicesRead, plugins.CapabilityDevicesWrite, plugins.CapabilityNetworkLAN},
		Config: []plugins.ConfigField{
			{Name: "host", Type: "string", Required: true}, {Name: "port", Type: "integer", Description: "Default 502"},
			{Name: "unit_id", Type: "integer", Description: "Modbus unit ID; default 1"},
			{Name: "poll_interval", Type: "duration", Description: "Default 5s"},
			{Name: "registers", Type: "string", Required: true, Description: "JSON array of register mappings encoded as a string"},
		},
	}, func(ctx context.Context, instanceID string, raw json.RawMessage, deps plugins.Dependencies) (plugins.Adapter, error) {
		var wire struct {
			Host string `json:"host"`
			Port int `json:"port"`
			UnitID int `json:"unit_id"`
			PollInterval string `json:"poll_interval"`
			Registers []modbus_tcp.RegisterMapping `json:"registers"`
		}
		if err := plugins.DecodeConfig(raw, &wire); err != nil { return nil, err }
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
