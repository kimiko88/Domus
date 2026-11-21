package broadlink

import (
	"encoding/binary"
	"net"
	"testing"
	"time"
)

func TestDeviceModel(t *testing.T) {
	tests := []struct{ typ uint16; model string; rf bool }{
		{0x2737,"RM Mini 3",false},
		{0x51da,"RM4 Mini",false},
		{0x520b,"RM4 Pro",true},
		{0x649b,"RM4 Pro",true},
		{0x6184,"RM4C Pro",true},
		{0x5211,"RM4C Mate",false},
		{0x2712,"RM Pro",true},
	}
	for _,tt := range tests {
		model,ir,rf := DeviceModel(tt.typ)
		if model != tt.model || !ir || rf != tt.rf { t.Fatalf("DeviceModel(%#x) = %q ir=%v rf=%v",tt.typ,model,ir,rf) }
	}
	if model,_,_ := DeviceModel(0x2711); model != "" { t.Fatalf("plug was incorrectly classified as RM: %s",model) }
	if !usesModernRemoteFrame(0x51da) || !usesModernRemoteFrame(0x520b) || !usesModernRemoteFrame(0x5f36) {
		t.Fatal("RM4 and new RM Mini 3 devices must use the extended remote frame")
	}
	if usesModernRemoteFrame(0x2737) || usesModernRemoteFrame(0x27c2) {
		t.Fatal("legacy RM Mini 3 devices must use the legacy remote frame")
	}
}

func TestDiscoveryPacket(t *testing.T) {
	packet := discoveryPacket(net.ParseIP("192.168.1.10"),12345,time.Date(2026,11,21,12,34,56,0,time.FixedZone("CET",3600)))
	if len(packet) != 48 { t.Fatalf("discovery packet size %d",len(packet)) }
	if packet[0x26] != 0x06 { t.Fatalf("wrong discovery command %x",packet[0x26]) }
	if packet[0x18] != 192 || packet[0x1b] != 10 { t.Fatalf("wrong source IP: %v",packet[0x18:0x1c]) }
	copyForChecksum := append([]byte(nil), packet...)
	got := binary.LittleEndian.Uint16(packet[0x20:0x22])
	copyForChecksum[0x20], copyForChecksum[0x21] = 0, 0
	if want := checksum(copyForChecksum); got != want { t.Fatalf("discovery checksum = %x, want %x",got,want) }
}

func TestEncryptDecrypt(t *testing.T) {
	plain := []byte("domus broadlink test")
	encrypted,err := encryptPayload(plain,initialKey)
	if err != nil { t.Fatal(err) }
	if len(encrypted)%16 != 0 { t.Fatalf("ciphertext is not block aligned") }
	decrypted,err := decryptPayload(encrypted,initialKey)
	if err != nil { t.Fatal(err) }
	if string(decrypted[:len(plain)]) != string(plain) { t.Fatalf("roundtrip mismatch: %q",decrypted) }
}
