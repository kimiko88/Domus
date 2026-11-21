package xiaomi_home

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestValidateTargetOnlyAllowsPrivateIPLiteral(t *testing.T) {
	for _, target := range []string{"example.com","8.8.8.8","127.0.0.1","169.254.169.254","http://192.168.1.4","192.168.1.4:54321"} {
		if _, err := validateTarget(target); err == nil { t.Errorf("expected %q to be rejected", target) }
	}
	for _, target := range []string{"192.168.1.4","10.0.0.8","172.16.1.2"} {
		if _, err := validateTarget(target); err != nil { t.Errorf("expected %q to be accepted: %v", target, err) }
	}
}

func TestMiIOPayloadEncryptionAndAuthenticatedPacket(t *testing.T) {
	token, err := hex.DecodeString("00112233445566778899aabbccddeeff")
	if err != nil { t.Fatal(err) }
	plain := []byte(`{"id":7,"method":"miIO.info","params":[]}`)
	encrypted, err := encryptPayload(plain, token)
	if err != nil { t.Fatal(err) }
	packet, err := buildPacket(0x12345678, 42, encrypted, token)
	if err != nil { t.Fatal(err) }
	got, err := parseResponse(packet, token)
	if err != nil { t.Fatal(err) }
	if !bytes.Equal(got, plain) { t.Fatalf("round trip mismatch: got %q want %q", got, plain) }

	packet[16] ^= 0xff
	if _, err := parseResponse(packet, token); err == nil { t.Fatal("expected checksum tampering to be rejected") }
}

func TestDecodeTokenRequiresExactly16Bytes(t *testing.T) {
	if _, err := decodeToken("not-a-token"); err == nil { t.Fatal("expected malformed token to be rejected") }
	if token, err := decodeToken("00112233445566778899aabbccddeeff"); err != nil || len(token) != 16 {
		t.Fatalf("expected 16-byte token, got %d bytes, err=%v", len(token), err)
	}
}
