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
