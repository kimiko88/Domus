package tuya_local

import (
	"testing"
)

func BenchmarkTuyaPacket_Encode_v33(b *testing.B) {
	key := []byte("0123456789abcdef")
	payload := []byte(`{"dps":{"1":true,"20":100}}`)

	for i := 0; b.Loop(); i++ {
		_, _ = EncodePacket(uint32(i), CmdControl, payload, key, "3.3")
	}
}

func BenchmarkTuyaPacket_Decode_v33(b *testing.B) {
	key := []byte("0123456789abcdef")
	payload := []byte(`{"dps":{"1":true,"20":100}}`)
	encoded, _ := EncodePacket(100, CmdControl, payload, key, "3.3")

	for b.Loop() {
		_, _ = DecodePacket(encoded, key)
	}
}

func BenchmarkDPSMapper_MapDPSToEntity(b *testing.B) {
	mapper := NewDPSMapper()
	dps := map[string]any{
		"1":  true,
		"20": 85.0,
		"21": "white",
	}

	for b.Loop() {
		_ = mapper.MapDPSToEntity(CategoryLight, dps)
	}
}
