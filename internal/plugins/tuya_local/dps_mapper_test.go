package tuya_local

import (
	"testing"
)

func TestDPSMapper_MapDPSToEntity_Switch(t *testing.T) {
	mapper := NewDPSMapper()
	dps := map[string]any{
		"1":  true,
		"18": 450,  // 450 mA
		"19": 1035, // 103.5 W
		"20": 2300, // 230.0 V
	}

	res := mapper.MapDPSToEntity(CategoryPlug, dps)
	if res.Domain != "switch" {
		t.Errorf("expected domain switch, got %s", res.Domain)
	}
	if res.State != "on" {
		t.Errorf("expected state on, got %s", res.State)
	}
	if pwr, ok := res.Attributes["power_w"].(float64); !ok || pwr != 103.5 {
		t.Errorf("expected power_w 103.5, got %v", res.Attributes["power_w"])
	}
}

func TestDPSMapper_MapDPSToEntity_Light(t *testing.T) {
	mapper := NewDPSMapper()
	dps := map[string]any{
		"20": true,
		"22": 500, // half brightness (500 / 1000)
	}

	res := mapper.MapDPSToEntity(CategoryLight, dps)
	if res.Domain != "light" {
		t.Errorf("expected domain light, got %s", res.Domain)
	}
	if res.State != "on" {
		t.Errorf("expected state on, got %s", res.State)
	}
	if bri, ok := res.Attributes["brightness"].(int); !ok || bri < 125 || bri > 130 {
		t.Errorf("expected brightness around 128, got %v", res.Attributes["brightness"])
	}
}

func TestDPSMapper_MapCommandToDPS(t *testing.T) {
	mapper := NewDPSMapper()

	// Switch turn on
	cmdDPS, err := mapper.MapCommandToDPS(CategorySwitch, "turn_on", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmdDPS["1"] != true {
		t.Errorf("expected DPS 1 == true, got %v", cmdDPS["1"])
	}

	// Light set brightness
	lightDPS, err := mapper.MapCommandToDPS(CategoryLight, "set_brightness", map[string]any{
		"brightness": 255,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lightDPS["20"] != true || lightDPS["22"] != 1000 {
		t.Errorf("expected DPS 20: true, 22: 1000, got %v", lightDPS)
	}
}

func TestDPSMapper_MapClimateModeAndSetDP4(t *testing.T) {
	mapper := NewDPSMapper()

	auto, err := mapper.MapCommandToDPS(CategoryClimate, "set_mode", map[string]any{"mode": "auto"})
	if err != nil || auto["4"] != 0 {
		t.Fatalf("auto mode must write DP4=0, got %v, err=%v", auto, err)
	}
	manual, err := mapper.MapCommandToDPS(CategoryClimate, "set_mode", map[string]any{"mode": "manual"})
	if err != nil || manual["4"] != 1 {
		t.Fatalf("manual mode must write DP4=1, got %v, err=%v", manual, err)
	}
	direct, err := mapper.MapCommandToDPS(CategoryClimate, "set_dp", map[string]any{"dp_id": 4, "value": 0})
	if err != nil || direct["4"] != 0 {
		t.Fatalf("device.set_dp(4, 0) must write DP4=0, got %v, err=%v", direct, err)
	}
	if _, err := mapper.MapCommandToDPS(CategoryClimate, "set_dp", map[string]any{"dp_id": 2, "value": 1}); err == nil {
		t.Fatal("raw LocalTuya mode writes must reject datapoints other than DP4")
	}
}

func TestDPSMapper_MapClimateTemperature(t *testing.T) {
	mapper := NewDPSMapper()
	dps, err := mapper.MapCommandToDPS(CategoryClimate, "set_temperature", map[string]any{"temperature": 21.5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dps["2"] != 215 {
		t.Fatalf("expected temperature 21.5°C to write DP2=215, got %v", dps["2"])
	}
}

func TestDPSMapper_MapClimateModeState(t *testing.T) {
	mapper := NewDPSMapper()
	parsed := mapper.MapDPSToEntity(CategoryClimate, map[string]any{"2": 215, "3": 205, "4": 0})
	if parsed.Attributes["target_temperature"] != 21.5 || parsed.Attributes["current_temperature"] != 20.5 {
		t.Fatalf("unexpected mapped temperatures: %+v", parsed.Attributes)
	}
	if parsed.Attributes["hvac_mode"] != "auto" {
		t.Fatalf("expected DP4=0 to map to auto, got %+v", parsed.Attributes)
	}
	if got := parsed.Attributes["hvac_modes"]; len(got.([]string)) != 2 || got.([]string)[0] != "auto" || got.([]string)[1] != "manual" {
		t.Fatalf("expected only supported LocalTuya modes to be advertised, got %+v", got)
	}
	manual := mapper.MapDPSToEntity(CategoryClimate, map[string]any{"4": 1})
	if manual.Attributes["hvac_mode"] != "manual" {
		t.Fatalf("expected DP4=1 to map to manual, got %+v", manual.Attributes)
	}
}

func TestDPSMapper_MapSetHVACModeToLocalTuyaDP4(t *testing.T) {
	dps, err := NewDPSMapper().MapCommandToDPS(CategoryClimate, "set_hvac_mode", map[string]any{"hvac_mode": "manual"})
	if err != nil || dps["4"] != 1 {
		t.Fatalf("manual HVAC mode must map to DPS 4=1, got %v, err=%v", dps, err)
	}
	if _, err := NewDPSMapper().MapCommandToDPS(CategoryClimate, "set_hvac_mode", map[string]any{"hvac_mode": "heat"}); err == nil {
		t.Fatal("unsupported generic heat mode must be rejected")
	}
}

func TestDPSMapper_MapClimateSetpointDPS16(t *testing.T) {
	mapper := NewDPSMapper()
	parsed := mapper.MapDPSToEntity(CategoryClimate, map[string]any{"16": 225, "3": 205})
	if parsed.Attributes["target_temperature"] != 22.5 {
		t.Fatalf("expected DPS 16 setpoint 22.5°C, got %+v", parsed.Attributes)
	}
	if parsed.Attributes["target_temperature_dp"] != 16 {
		t.Fatalf("expected target_temperature_dp=16, got %+v", parsed.Attributes)
	}

	dps, err := mapper.MapCommandToDPS(CategoryClimate, "set_temperature", map[string]any{
		"temperature": 22.5,
		"dp_id":       16,
	})
	if err != nil {
		t.Fatalf("map DPS 16 command: %v", err)
	}
	if dps["16"] != 225 {
		t.Fatalf("expected DPS 16=225, got %+v", dps)
	}
	if _, exists := dps["2"]; exists {
		t.Fatalf("DPS 16 command must not also write DPS 2: %+v", dps)
	}
}

func TestDPSMapper_RejectsUnsupportedClimateSetpointDP(t *testing.T) {
	_, err := NewDPSMapper().MapCommandToDPS(CategoryClimate, "set_temperature", map[string]any{
		"temperature": 21.0,
		"dp_id":       3,
	})
	if err == nil {
		t.Fatal("expected unsupported setpoint datapoint to be rejected")
	}
}

func TestDPSMapper_MapClimateSetpointScale(t *testing.T) {
	mapper := NewDPSMapper()
	dps, err := mapper.MapCommandToDPS(CategoryClimate, "set_temperature", map[string]any{
		"temperature": 21.0,
		"scale":       0,
	})
	if err != nil {
		t.Fatalf("map integer-degree thermostat: %v", err)
	}
	if dps["2"] != 21 {
		t.Fatalf("expected scale 0 to write DPS2=21, got %+v", dps)
	}

	dps, err = mapper.MapCommandToDPS(CategoryClimate, "set_temperature", map[string]any{
		"temperature": 21.5,
		"scale":       2,
	})
	if err != nil {
		t.Fatalf("map centi-degree thermostat: %v", err)
	}
	if dps["2"] != 2150 {
		t.Fatalf("expected scale 2 to write DPS2=2150, got %+v", dps)
	}

	if _, err := mapper.MapCommandToDPS(CategoryClimate, "set_temperature", map[string]any{
		"temperature": 21.0,
		"scale":       5,
	}); err == nil {
		t.Fatal("unsupported temperature scales must be rejected")
	}
}


func TestDPSMapper_MapTuyaTemperatureHumiditySensor(t *testing.T) {
	mapped := NewDPSMapper().MapDPSToEntity(CategorySensor, map[string]any{
		"1": 215,
		"2": 48,
		"3": 90,
	})
	if mapped.Domain != "sensor" || mapped.State != "active" {
		t.Fatalf("sensor domain/state changed: %+v", mapped)
	}
	if mapped.Attributes["dp_1"] != 215 || mapped.Attributes["dp_2"] != 48 {
		t.Fatalf("raw DPS values must be retained: %#v", mapped.Attributes)
	}
	if mapped.Attributes["current_temperature"] != 21.5 ||
		mapped.Attributes["current_humidity"] != float64(48) {
		t.Fatalf("expected DP1/DP2 to map to 21.5°C and 48%%, got %#v", mapped.Attributes)
	}
	if mapped.Attributes["tuya_climate_source"] != true ||
		mapped.Attributes["tuya_sensor_kind"] != "temperature_humidity" ||
		mapped.Attributes["device_class"] != "temperature" ||
		mapped.Attributes["unit_of_measurement"] != "°C" {
		t.Fatalf("climate source metadata missing: %#v", mapped.Attributes)
	}
}

func TestDPSMapper_DoesNotClassifyGenericNumericSensorAsClimate(t *testing.T) {
	mapped := NewDPSMapper().MapDPSToEntity(CategorySensor, map[string]any{
		"1": 1,
		"2": 90,
	})
	if mapped.Attributes["tuya_climate_source"] == true {
		t.Fatalf("generic unscaled numeric sensor must not be classified as climate: %#v", mapped.Attributes)
	}
	if _, exists := mapped.Attributes["current_temperature"]; exists {
		t.Fatalf("generic sensor must not receive inferred temperature: %#v", mapped.Attributes)
	}
}

func TestDPSMapper_RecognizesNamedUnscaledThermoHygrometer(t *testing.T) {
	attributes := map[string]any{"dp_1": 19.5, "dp_2": 44}
	enrichTuyaClimateSensorAttributes(attributes, "Sensore temperatura e umidità")
	if attributes["current_temperature"] != 19.5 || attributes["current_humidity"] != float64(44) ||
		attributes["tuya_climate_source"] != true {
		t.Fatalf("named unscaled thermo-hygrometer was not recognized: %#v", attributes)
	}
}


func TestDPSMapper_RecognizesSingleTemperatureSensorByName(t *testing.T) {
	attributes := map[string]any{"dp_1": 215}
	enrichTuyaClimateSensorAttributes(attributes, "Tuya temperature sensor")
	if attributes["current_temperature"] != 21.5 ||
		attributes["tuya_sensor_kind"] != "temperature" ||
		attributes["unit_of_measurement"] != "°C" {
		t.Fatalf("single temperature DPS was not normalized: %#v", attributes)
	}
	if _, exists := attributes["current_humidity"]; exists {
		t.Fatalf("temperature-only source must not invent humidity: %#v", attributes)
	}
}

func TestDPSMapper_RecognizesSingleHumiditySensorByName(t *testing.T) {
	attributes := map[string]any{"dp_2": 450}
	enrichTuyaClimateSensorAttributes(attributes, "Tuya humidity sensor")
	if attributes["current_humidity"] != float64(45) ||
		attributes["tuya_sensor_kind"] != "humidity" ||
		attributes["device_class"] != "humidity" ||
		attributes["unit_of_measurement"] != "%" {
		t.Fatalf("single humidity DPS was not normalized: %#v", attributes)
	}
	if _, exists := attributes["current_temperature"]; exists {
		t.Fatalf("humidity-only source must not invent temperature: %#v", attributes)
	}
}


func TestDPSMapper_RefreshesPreviouslyMappedSensorValues(t *testing.T) {
	attributes := map[string]any{
		"dp_1": 215, "dp_2": 48,
		"current_temperature": 20.5,
		"current_humidity": 45.0,
		"tuya_climate_source": true,
		"tuya_sensor_kind": "temperature_humidity",
	}
	// Simulate a new LocalTuya packet after the sensor has already been
	// identified and the previous normalized values have been persisted.
	attributes["dp_1"] = 222
	attributes["dp_2"] = 51
	enrichTuyaClimateSensorAttributes(attributes, "Tuya room sensor")
	if attributes["current_temperature"] != 22.2 || attributes["current_humidity"] != float64(51) {
		t.Fatalf("normalized sensor values were not refreshed from the latest DPS: %#v", attributes)
	}
}
