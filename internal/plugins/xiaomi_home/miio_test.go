package xiaomi_home

import (
	"bytes"
	"encoding/hex"
	"net"
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


func TestSubnetTargetsNarrowsWideNetworkToLocal24(t *testing.T) {
	ip, network, err := net.ParseCIDR("192.168.10.42/16")
	if err != nil {
		t.Fatal(err)
	}
	got := subnetTargets(ip, network)
	if len(got) != 253 {
		t.Fatalf("expected 253 usable scan targets excluding the local host, got %d", len(got))
	}
	if got[0] != "192.168.10.1" || got[len(got)-1] != "192.168.10.254" {
		t.Fatalf("unexpected bounded /24 range: first=%s last=%s", got[0], got[len(got)-1])
	}
	for _, target := range got {
		parsed := net.ParseIP(target)
		if parsed == nil || !parsed.IsPrivate() || parsed.Equal(ip) {
			t.Fatalf("unsafe or duplicate local target %q", target)
		}
	}
}

func TestSubnetTargetsRejectsIPv6AndPointToPointNetworks(t *testing.T) {
	for _, cidr := range []string{"fd00::10/64", "192.168.1.2/31", "192.168.1.2/32"} {
		ip, network, err := net.ParseCIDR(cidr)
		if err != nil {
			t.Fatal(err)
		}
		if got := subnetTargets(ip, network); len(got) != 0 {
			t.Errorf("expected no scan targets for %s, got %v", cidr, got)
		}
	}
}
