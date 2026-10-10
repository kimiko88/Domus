package tuya_local

import (
	"encoding/json"
	"testing"
)

func FuzzDecodePacket(f *testing.F) {
	key := []byte("0123456789abcdef")

	// Seed corpus with valid packets
	pkt1, _ := EncodePacket(1, CmdHeartbeat, nil, key, "3.3")
	if len(pkt1) > 0 {
		f.Add(pkt1, key)
	}

	pkt2, _ := EncodePacket(2, CmdDpQuery, []byte(`{"gwId":"dev1","devId":"dev1"}`), key, "3.3")
	if len(pkt2) > 0 {
		f.Add(pkt2, key)
	}

	pkt3, _ := EncodePacket(3, CmdControl, []byte(`{"dps":{"1":true}}`), key, "3.1")
	if len(pkt3) > 0 {
		f.Add(pkt3, key)
	}

	// Malformed seeds
	f.Add([]byte{0x00, 0x00, 0x55, 0xAA, 0x00, 0x00}, key)
	f.Add([]byte("random invalid string that is not a tuya packet"), key)
	f.Add([]byte{}, []byte{})

	f.Fuzz(func(t *testing.T, data []byte, localKey []byte) {
		_, _ = DecodePacket(data, localKey)
	})
}

func FuzzDPSMapper(f *testing.F) {
	mapper := NewDPSMapper()
	categories := []DeviceCategory{CategorySwitch, CategoryLight, CategoryClimate, CategorySensor}

	f.Add([]byte(`{"1":true,"2":50,"3":"cold"}`), 0)
	f.Add([]byte(`{"20":true,"21":"white","22":100}`), 1)
	f.Add([]byte(`{"1":false}`), 0)
	f.Add([]byte(`{}`), 3)
	f.Add([]byte(`{"invalid":[`), 2)

	f.Fuzz(func(t *testing.T, payload []byte, catIdx int) {
		var dps map[string]any
		if err := json.Unmarshal(payload, &dps); err == nil {
			idx := catIdx
			if idx < 0 {
				idx = -idx
			}
			cat := categories[idx%len(categories)]
			_ = mapper.MapDPSToEntity(cat, dps)
		}
	})
}
