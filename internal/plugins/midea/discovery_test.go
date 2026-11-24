package midea

import (
	"bytes"
	"crypto/aes"
	"crypto/md5"
	"encoding/binary"
	"net"
	"syscall"
	"testing"
)

func makeDiscoveryFixture(t *testing.T, deviceID uint64, protocol Protocol) []byte {
	t.Helper()
	reply := make([]byte, 96)
	binary.LittleEndian.PutUint32(reply[4:8], 6444)
	copy(reply[8:40], []byte("12345678MIDEA1234AABBCCDDEEFF0000"))
	copy(reply[17:25], []byte("MDDF16WF"))
	copy(reply[40:], []byte{8})
	copy(reply[41:49], []byte("Midea_A1"))
	plain := append([]byte(nil), reply...)
	padding := aes.BlockSize - len(plain)%aes.BlockSize
	for i := 0; i < padding; i++ { plain = append(plain, byte(padding)) }
	block, err := aes.NewCipher(mideaLANKey)
	if err != nil { t.Fatal(err) }
	encrypted := make([]byte, len(plain))
	for offset := 0; offset < len(plain); offset += aes.BlockSize {
		block.Encrypt(encrypted[offset:offset+aes.BlockSize], plain[offset:offset+aes.BlockSize])
	}
	packet := make([]byte, mideaHeaderLength, mideaHeaderLength+len(encrypted)+mideaDigestLength)
	packet[0], packet[1] = 0x5A, 0x5A
	var id [8]byte
	binary.LittleEndian.PutUint64(id[:], deviceID)
	copy(packet[20:26], id[:6])
	packet = append(packet, encrypted...)
	binary.LittleEndian.PutUint16(packet[4:6], uint16(len(packet)+mideaDigestLength))
	hashInput := append(append([]byte(nil), packet...), mideaLANSalt...)
	digest := md5.Sum(hashInput)
	packet = append(packet, digest[:]...)
	if protocol == ProtocolMSmartV3 {
		wrapped := make([]byte, 8, 8+len(packet)+16)
		wrapped[0], wrapped[1] = 0x83, 0x70
		wrapped[2], wrapped[3] = byte((len(packet)+16)>>8), byte(len(packet)+16)
		wrapped = append(wrapped, packet...)
		wrapped = append(wrapped, make([]byte, 16)...)
		return wrapped
	}
	return packet
}

func TestDiscoverSocketEnablesIPv4Broadcast(t *testing.T) {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := enableMideaBroadcast(conn); err != nil {
		t.Fatalf("enable broadcast: %v", err)
	}
	raw, err := conn.SyscallConn()
	if err != nil {
		t.Fatal(err)
	}
	var enabled int
	var socketErr error
	if err := raw.Control(func(fd uintptr) {
		enabled, socketErr = syscall.GetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_BROADCAST)
	}); err != nil {
		t.Fatal(err)
	}
	if socketErr != nil {
		t.Fatal(socketErr)
	}
	if enabled != 1 {
		t.Fatalf("SO_BROADCAST = %d, want 1", enabled)
	}
}

func TestParseMideaA1DiscoveryDatagram(t *testing.T) {
	raw := makeDiscoveryFixture(t, 123456789, ProtocolMSmartV2)
	discovered, err := parseDiscoveryDatagram(raw, "192.168.1.70")
	if err != nil {
		t.Fatalf("parse discovery: %v", err)
	}
	if discovered.ID != "123456789" || discovered.TypeCode != "A1" ||
		discovered.Host != "192.168.1.70" || discovered.Port != 6444 ||
		discovered.Model != "MDDF16WF" || !discovered.Supported ||
		discovered.Protocol != ProtocolMSmartV2 {
		t.Fatalf("unexpected discovered device: %+v", discovered)
	}
}

func TestParseMideaV3DiscoveryDatagram(t *testing.T) {
	raw := makeDiscoveryFixture(t, 42, ProtocolMSmartV3)
	discovered, err := parseDiscoveryDatagram(raw, "192.168.10.20")
	if err != nil {
		t.Fatalf("parse V3 discovery: %v", err)
	}
	if discovered.Protocol != ProtocolMSmartV3 || discovered.ID != "42" {
		t.Fatalf("unexpected V3 discovery result: %+v", discovered)
	}
}

func TestParseMideaDiscoveryRejectsPublicOrTamperedDatagrams(t *testing.T) {
	raw := makeDiscoveryFixture(t, 42, ProtocolMSmartV2)
	if _, err := parseDiscoveryDatagram(raw, "8.8.8.8"); err == nil {
		t.Fatal("public source address should be rejected")
	}
	raw[len(raw)-1] ^= 0x80
	if _, err := parseDiscoveryDatagram(raw, "192.168.1.20"); err == nil {
		t.Fatal("tampered datagram should be rejected")
	}
}

func TestMideaDiscoveryPacketHasExpectedHeader(t *testing.T) {
	if len(mideaDiscoveryPacket) < 60 || !bytes.Equal(mideaDiscoveryPacket[:4], []byte{0x5A, 0x5A, 0x01, 0x11}) {
		t.Fatalf("invalid discovery request: %x", mideaDiscoveryPacket)
	}
}
