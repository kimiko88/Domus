package plugin_runtime

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/domus-hub/domus/internal/infrastructure/config"
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
	for _, want := range []string{"mqtt_generic", "zigbee2mqtt", "esphome_mqtt", "knx", "modbus_tcp", "matter", "ble_proxy", "open_display", "voice_satellite", "zwave_js", "weather_openmeteo", "tuya_cloud", "imou_life", "vesync_transport", "xiaomi_miio_transport", "xiaomi_home_cloud", "midea_a1_protocol"} { require.True(t, ids[want], "missing built-in plugin type %s", want) }
	internalTypes := make(map[string]bool, len(manifests))
	for _, manifest := range manifests { internalTypes[manifest.ID] = manifest.Internal }
	for _, want := range []string{"vesync_transport", "xiaomi_miio_transport", "midea_a1_protocol"} {
		require.True(t, internalTypes[want], "%s must be an internal runtime transport", want)
	}
	for _, want := range []string{"mqtt_generic", "open_display", "voice_satellite", "tuya_cloud", "xiaomi_home_cloud", "imou_life"} {
		require.False(t, internalTypes[want], "%s must remain user-configurable", want)
	}
	for _, want := range []string{"mqtt_generic", "zigbee2mqtt", "esphome_mqtt", "zwave_js", "knx", "modbus_tcp", "ble_proxy", "voice_satellite"} { require.True(t, multi[want], "%s must support multiple instances", want) }
	for _, want := range []string{"matter", "open_display", "tuya_cloud", "imou_life"} { require.False(t, multi[want], "%s must remain single-instance", want) }
}

func TestRegisterBuiltinsCreatesTypedAdapters(t *testing.T) {
	registry := plugins.NewRegistry()
	ctx := context.Background()
	db, err := sqlite.Open(config.DatabaseConfig{Path: ":memory:", WALMode: false, BusyTimeoutMs: 1000, MaxOpenConns: 1})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, sqlite.Migrate(ctx, db))
	cipher, err := crypto.NewEnvelopeCipher([]byte("factory-test-key"))
	require.NoError(t, err)
	repos := sqlite.NewRepositories(db)
	repos.Cipher = cipher
	require.NoError(t, registerBuiltins(registry, builtinHost{repos: repos}))
	cases := []struct { typeID, instanceID, config string }{
		{"mqtt_generic", "mqtt_kitchen", `{"broker_url":"tcp://127.0.0.1:1883","state_topic_pattern":"domus/+/state","command_topic_template":"domus/%s/set"}`},
		{"zigbee2mqtt", "zigbee_hub_upstairs", `{"broker_url":"tcp://127.0.0.1:1883","base_topic":"zigbee2mqtt_upstairs"}`},
		{"esphome_mqtt", "esphome_garage", `{"broker_url":"tcp://127.0.0.1:1883","topic_prefix":"garage"}`},
		{"knx", "knx_first_floor", `{"gateway_addr":"127.0.0.1:3671","entities":[]}`},
		{"modbus_tcp", "modbus_inverter", `{"host":"127.0.0.1","port":502,"unit_id":1,"poll_interval":"5s","registers":[]}`},
		{"matter", "matter_controller", `{}`},
		{"zwave_js", "zwave_upstairs", `{"ws_url":"ws://127.0.0.1:3000","auth_token":"zwave-test-secret"}`},
		{"weather_openmeteo", "weather_forecast", `{}`},
		{"tuya_cloud", "tuya_cloud", `{"client_id":"test-client","secret":"test-secret","base_url":"https://openapi.tuyaeu.com","uid":"test-user","oauth_client_id":"oauth-client","oauth_secret":"oauth-secret","oauth_identifier":"domus.test","smart_life_oauth":{"admin":{"uid":"user-1","access_token":"access","refresh_token":"refresh","expires_at":2000000000}}}`},
		{"ble_proxy", "ble_proxy_hall", `{}`},
		{"tuya_local", "tuya_local_flat", `{}`},
		{"broadlink_local", "broadlink_living_room", `{}`},
		{"open_display", "open_display", `{"bridge_url":"http://127.0.0.1:10320","token":"test-bridge-token-0123456789abcdef0123456789"}`},
		{"voice_satellite", "voice_kitchen", `{"server_url":"http://127.0.0.1:8080","bearer_token":"voice-satellite-test-token-0123456789abcdef","wake_address":"127.0.0.1:10400","wake_words":["hey_jarvis"]}`},
		{"imou_life", "integration_imou_life", `{"access_key":"project-key","secret_key":"project-secret","data_center":"eu","authorization_url":"https://auth.example/imou","public_base_url":"https://domus.example"}`},
		{"vesync_transport", "vesync_transport", `{}`},
		{"xiaomi_miio_transport", "xiaomi_miio_transport", `{}`},
		{"xiaomi_home_cloud", "integration_xiaomi_home_cloud", `{"client_id":"123456789012","client_secret":"test-secret","region":"eu","public_base_url":"https://domus.example"}`},
		{"midea_a1_protocol", "midea_a1_protocol", `{}`},
	}
	deps := plugins.Dependencies{Devices: repos.Devices, Entities: repos.Entities}
	for _, tc := range cases { t.Run(tc.typeID+"/"+tc.instanceID, func(t *testing.T) { adapter, err := registry.Create(context.Background(), tc.typeID, tc.instanceID, json.RawMessage(tc.config), deps); require.NoError(t, err); require.Equal(t, tc.instanceID, adapter.ID()); require.Equal(t, tc.typeID, adapter.Type()) }) }
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
	_, err = registry.Create(context.Background(), "xiaomi_home_cloud", "integration_xiaomi_home_cloud", json.RawMessage(`{"client_id":"123456789012","client_secret":"app-secret","region":"eu","public_base_url":"http://domus.example"}`), plugins.Dependencies{}); require.ErrorContains(t, err, "HTTPS")
	_, err = registry.Create(context.Background(), "xiaomi_home_cloud", "integration_xiaomi_home_cloud", json.RawMessage(`{"client_id":"invalid","client_secret":"app-secret","region":"eu","public_base_url":"https://domus.example"}`), plugins.Dependencies{}); require.ErrorContains(t, err, "invalid Open Platform application credentials")
	_, err = registry.Create(context.Background(), "imou_life", "integration_imou_life", json.RawMessage(`{"access_key":"project-key","secret_key":"project-secret","data_center":"eu","authorization_url":"https://auth.example/imou","public_base_url":"http://domus.example"}`), plugins.Dependencies{}); require.ErrorContains(t, err, "HTTPS")
	_, err = registry.Create(context.Background(), "imou_life", "integration_imou_life", json.RawMessage(`{"access_key":"project-key","secret_key":"project-secret","data_center":"bad center","authorization_url":"https://auth.example/imou","public_base_url":"https://domus.example"}`), plugins.Dependencies{}); require.ErrorContains(t, err, "invalid data centre")
}
