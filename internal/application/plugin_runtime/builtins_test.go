package plugin_runtime

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/domus-hub/domus/internal/infrastructure/crypto"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	"github.com/domus-hub/domus/internal/plugins"
	"github.com/stretchr/testify/require"
 )

func TestRegisterBuiltinsExposesProtocolFamilies(t *testing.T) {
	registry := plugins.NewRegistry()
	require.NoError(t, registerBuiltins(registry, builtinHost{}))
	manifests := registry.List()
	ids := make(map[string]bool, len(manifests))
	for _, manifest := range manifests { ids[manifest.ID] = true; require.Equal(t, plugins.ManifestAPIVersion, manifest.APIVersion); require.NotEmpty(t, manifest.Capabilities) }
	multi := make(map[string]bool, len(manifests))
	for _, manifest := range manifests { multi[manifest.ID] = manifest.MultiInstance }
	for _, want := range []string{"mqtt_generic", "zigbee2mqtt", "esphome_mqtt", "knx", "modbus_tcp", "matter", "ble_proxy", "open_display"} { require.True(t, ids[want], "missing built-in plugin type %s", want) }
	for _, want := range []string{"mqtt_generic", "zigbee2mqtt", "esphome_mqtt", "knx", "modbus_tcp", "ble_proxy"} { require.True(t, multi[want], "%s must support multiple instances", want) }
	for _, want := range []string{"matter", "open_display"} { require.False(t, multi[want], "%s must remain single-instance", want) }
}

func TestRegisterBuiltinsCreatesTypedAdapters(t *testing.T) {
	registry := plugins.NewRegistry()
	cipher, err := crypto.NewEnvelopeCipher([]byte("factory-test-key"))
	require.NoError(t, err)
	repos := &sqlite.Repositories{DB: &sql.DB{}, Cipher: cipher} // Factories attach the DB; no SQL is executed here.
	require.NoError(t, registerBuiltins(registry, builtinHost{repos: repos}))
	cases := []struct { typeID, instanceID, config string }{
		{"mqtt_generic", "mqtt_kitchen", `{"broker_url":"tcp://127.0.0.1:1883","state_topic_pattern":"domus/+/state","command_topic_template":"domus/%s/set"}`},
		{"zigbee2mqtt", "zigbee_hub_upstairs", `{"broker_url":"tcp://127.0.0.1:1883","base_topic":"zigbee2mqtt_upstairs"}`},
		{"esphome_mqtt", "esphome_garage", `{"broker_url":"tcp://127.0.0.1:1883","topic_prefix":"garage"}`},
		{"knx", "knx_first_floor", `{"gateway_addr":"127.0.0.1:3671","entities":[]}`},
		{"modbus_tcp", "modbus_inverter", `{"host":"127.0.0.1","port":502,"unit_id":1,"poll_interval":"5s","registers":[]}`},
		{"matter", "matter_controller", `{}`},
		{"ble_proxy", "ble_proxy_hall", `{}`},
		{"tuya_local", "tuya_local_flat", `{}`},
		{"broadlink_local", "broadlink_living_room", `{}`},
		{"open_display", "open_display", `{"bridge_url":"http://127.0.0.1:10320","token":"test-bridge-token-0123456789abcdef0123456789"}`},
	}
	for _, tc := range cases { t.Run(tc.typeID+"/"+tc.instanceID, func(t *testing.T) { adapter, err := registry.Create(context.Background(), tc.typeID, tc.instanceID, json.RawMessage(tc.config), plugins.Dependencies{}); require.NoError(t, err); require.Equal(t, tc.instanceID, adapter.ID()); require.Equal(t, tc.typeID, adapter.Type()) }) }
}

func TestBuiltinsAllowIndependentInstancesOfSameProtocol(t *testing.T) {
	registry := plugins.NewRegistry()
	require.NoError(t, registerBuiltins(registry, builtinHost{}))
	configA := json.RawMessage(`{"broker_url":"tcp://127.0.0.1:1883","client_id":"domus_a","state_topic_pattern":"house/a/+/state"}`)
	configB := json.RawMessage(`{"broker_url":"tcp://127.0.0.1:1884","client_id":"domus_b","state_topic_pattern":"house/b/+/state"}`)
	a, err := registry.Create(context.Background(), "mqtt_generic", "mqtt_a", configA, plugins.Dependencies{}); require.NoError(t, err)
	b, err := registry.Create(context.Background(), "mqtt_generic", "mqtt_b", configB, plugins.Dependencies{}); require.NoError(t, err)
	require.NotSame(t, a, b); require.Equal(t, "mqtt_a", a.ID()); require.Equal(t, "mqtt_b", b.ID())
}

func TestBuiltinsRejectInvalidProtocolConfigurations(t *testing.T) {
	registry := plugins.NewRegistry(); require.NoError(t, registerBuiltins(registry, builtinHost{}))
	_, err := registry.Create(context.Background(), "mqtt_generic", "mqtt_bad", json.RawMessage(`{"broker_url":"tcp://127.0.0.1:1883","keep_alive":"1s"}`), plugins.Dependencies{}); require.ErrorContains(t, err, "keep_alive")
	_, err = registry.Create(context.Background(), "knx", "knx_bad", json.RawMessage(`{"gateway_addr":"not-an-address","entities":[]}`), plugins.Dependencies{}); require.ErrorContains(t, err, "gateway_addr")
	_, err = registry.Create(context.Background(), "modbus_tcp", "modbus_bad", json.RawMessage(`{"host":"127.0.0.1","port":70000,"registers":[]}`), plugins.Dependencies{}); require.ErrorContains(t, err, "port")
	_, err = registry.Create(context.Background(), "open_display", "wrong_instance", json.RawMessage(`{"bridge_url":"http://127.0.0.1:10320","token":"x"}`), plugins.Dependencies{}); require.ErrorContains(t, err, "instance ID")
}
