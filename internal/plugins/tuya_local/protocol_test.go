package tuya_local

import (
	"bytes"
	"testing"
)

func TestTuyaProtocol_EncodeDecode(t *testing.T) {
	localKey := []byte("0123456789abcdef") // 16 bytes key
	payload := []byte(`{"devId":"test_device_123","dps":{"1":true,"2":100}}`)
	seq := uint32(42)
	cmd := CmdControl

	encoded, err := EncodePacket(seq, cmd, payload, localKey, "3.3")
	if err != nil {
		t.Fatalf("failed to encode packet: %v", err)
	}

	if len(encoded) < 16 {
		t.Fatalf("encoded packet too short: %d bytes", len(encoded))
	}

	// Verify prefix and suffix
	if !bytes.Equal(encoded[:4], PrefixHeader) {
		t.Fatalf("wrong prefix: %v", encoded[:4])
	}
	if !bytes.Equal(encoded[len(encoded)-4:], SuffixHeader) {
		t.Fatalf("wrong suffix: %v", encoded[len(encoded)-4:])
	}

	// Decode
	packet, err := DecodePacket(encoded, localKey)
	if err != nil {
		t.Fatalf("failed to decode packet: %v", err)
	}

	if packet.Sequence != seq {
		t.Errorf("expected sequence %d, got %d", seq, packet.Sequence)
	}
	if packet.Command != cmd {
		t.Errorf("expected command %d, got %d", cmd, packet.Command)
	}
	if !bytes.Equal(packet.Payload, payload) {
		t.Errorf("payload mismatch: expected %s, got %s", string(payload), string(packet.Payload))
	}
}

func TestTuyaProtocol_CorruptedCRC(t *testing.T) {
	localKey := []byte("0123456789abcdef")
	payload := []byte(`{"ping":true}`)
	encoded, err := EncodePacket(1, CmdHeartbeat, payload, localKey, "3.3")
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	// Corrupt payload byte
	encoded[17] ^= 0xFF

	_, err = DecodePacket(encoded, localKey)
	if err == nil {
		t.Fatalf("expected CRC failure, got nil error")
	}
}
