package tuya_local

import (
	"testing"
)

func BenchmarkTuyaPacket_Encode_v33(b *testing.B) {
	key := []byte("0123456789abcdef")
	payload := []byte(`{"dps":{"1":true,"20":100}}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = EncodePacket(uint32(i), CmdControl, payload, key, "3.3")
	}
}

func BenchmarkTuyaPacket_Decode_v33(b *testing.B) {
	key := []byte("0123456789abcdef")
	payload := []byte(`{"dps":{"1":true,"20":100}}`)
	encoded, _ := EncodePacket(100, CmdControl, payload, key, "3.3")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
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

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = mapper.MapDPSToEntity(CategoryLight, dps)
	}
}
