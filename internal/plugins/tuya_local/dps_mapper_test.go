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
	manual := mapper.MapDPSToEntity(CategoryClimate, map[string]any{"4": 1})
	if manual.Attributes["hvac_mode"] != "manual" {
		t.Fatalf("expected DP4=1 to map to manual, got %+v", manual.Attributes)
	}
}
