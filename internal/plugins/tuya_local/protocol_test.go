package tuya_local

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
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

func TestDecodeResponsePacketExtractsReturnCode(t *testing.T) {
	key := []byte("0123456789abcdef")
	payload := []byte(`{"dps":{"4":1}}`)
	outgoing, err := EncodePacket(7, CmdControl, payload, key, "3.3")
	if err != nil { t.Fatal(err) }

	// Device responses prepend a uint32 return code to the payload.
	bodyEnd := len(outgoing) - 8
	response := make([]byte, 0, len(outgoing)+4)
	response = append(response, outgoing[:16]...)
	response = append(response, 0, 0, 0, 0)
	response = append(response, outgoing[16:bodyEnd]...)
	binary.BigEndian.PutUint32(response[12:16], binary.BigEndian.Uint32(outgoing[12:16])+4)
	crc := crc32.ChecksumIEEE(response)
	var checksum [4]byte
	binary.BigEndian.PutUint32(checksum[:], crc)
	response = append(response, checksum[:]...)
	response = append(response, SuffixHeader...)

	packet, err := DecodeResponsePacket(response, key)
	if err != nil { t.Fatalf("decode device response: %v", err) }
	if packet.ReturnCode != 0 { t.Fatalf("return code = %d, want 0", packet.ReturnCode) }
	if packet.Sequence != 7 || packet.Command != CmdControl { t.Fatalf("unexpected response header: seq=%d command=%d", packet.Sequence, packet.Command) }
	if !bytes.Equal(packet.Payload, payload) { t.Fatalf("payload mismatch: got %q want %q", packet.Payload, payload) }
}
