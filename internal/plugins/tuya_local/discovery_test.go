package tuya_local

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"encoding/json"
	"hash/crc32"
	"testing"
)

func TestValidDeviceID(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want bool
	}{
		{name: "valid alphanumeric", id: "bf1234567890abcd", want: true},
		{name: "valid separators", id: "device_01-test", want: true},
		{name: "empty", id: "", want: false},
		{name: "whitespace", id: "device 01", want: false},
		{name: "control character", id: "device\n01", want: false},
		{name: "too long", id: string(make([]byte, 129)), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validDeviceID(tt.id); got != tt.want {
				t.Fatalf("validDeviceID(%q) = %v, want %v", tt.id, got, tt.want)
			}
		})
	}
}

func TestDecodeTuyaDiscoveryPacket_PlainJSON(t *testing.T) {
	raw := []byte(`{"gwId":"device_123","version":"3.3","productKey":"product","active":1}`)
	dev, err := decodeTuyaDiscoveryPacket(raw)
	if err != nil {
		t.Fatalf("decode plain discovery JSON: %v", err)
	}
	if dev.DeviceID != "device_123" || dev.Version != "3.3" || dev.ProductKey != "product" || dev.Active != 1 {
		t.Fatalf("unexpected decoded device: %+v", dev)
	}
}

func TestDecodeTuyaDiscoveryPacket_55AAEncryptedWithReturnCode(t *testing.T) {
	rawJSON := []byte(`{"gwId":"device_55aa","version":"3.3","productKey":"pk"}`)
	encrypted, err := encryptAES_ECB(rawJSON, tuyaDiscoveryKey[:])
	if err != nil {
		t.Fatalf("encrypt discovery payload: %v", err)
	}
	frame := make55AADiscoveryTestFrame(encrypted, true)
	dev, err := decodeTuyaDiscoveryPacket(frame)
	if err != nil {
		t.Fatalf("decode encrypted 55AA discovery: %v", err)
	}
	if dev.DeviceID != "device_55aa" || dev.Version != "3.3" {
		t.Fatalf("unexpected decoded device: %+v", dev)
	}
}

func TestDecodeTuyaDiscoveryPacket_55AAPlaintextWithoutReturnCode(t *testing.T) {
	rawJSON := []byte(`{"gwId":"device_6666","version":"3.1"}`)
	frame := make55AADiscoveryTestFrame(rawJSON, false)
	dev, err := decodeTuyaDiscoveryPacket(frame)
	if err != nil {
		t.Fatalf("decode plaintext 55AA discovery: %v", err)
	}
	if dev.DeviceID != "device_6666" || dev.Version != "3.1" {
		t.Fatalf("unexpected decoded device: %+v", dev)
	}
}

func TestDecodeTuyaDiscoveryPacket_6699GCMWithReturnCode(t *testing.T) {
	rawJSON := []byte(`{"gwId":"device_6699","version":"3.5","productKey":"pk35"}`)
	plaintext := append([]byte{0, 0, 0, 0}, rawJSON...)
	frame := make6699DiscoveryTestFrame(t, plaintext)

	dev, err := decodeTuyaDiscoveryPacket(frame)
	if err != nil {
		t.Fatalf("decode authenticated 6699 discovery: %v", err)
	}
	if dev.DeviceID != "device_6699" || dev.Version != "3.5" || dev.ProductKey != "pk35" {
		t.Fatalf("unexpected decoded device: %+v", dev)
	}
}

func TestDecodeTuyaDiscoveryPacket_6699GCMWithoutReturnCode(t *testing.T) {
	rawJSON := []byte(`{"gwId":"device_6699_no_rc","version":"3.5"}`)
	frame := make6699DiscoveryTestFrame(t, rawJSON)

	dev, err := decodeTuyaDiscoveryPacket(frame)
	if err != nil {
		t.Fatalf("decode 6699 discovery without return code: %v", err)
	}
	if dev.DeviceID != "device_6699_no_rc" {
		t.Fatalf("unexpected decoded device: %+v", dev)
	}
}

func TestBuildTuya35DiscoveryRequest(t *testing.T) {
	const localIP = "192.168.1.20"
	packet, err := buildTuya35DiscoveryRequest(7, localIP)
	if err != nil {
		t.Fatalf("build v3.5 discovery request: %v", err)
	}
	if len(packet) < tuya6699HeaderSize+tuyaGCMNonceSize+tuyaGCMTagSize+tuya6699FooterSize {
		t.Fatalf("discovery request is too short: %d", len(packet))
	}
	if !bytes.Equal(packet[:4], []byte{0x00, 0x00, 0x66, 0x99}) {
		t.Fatalf("unexpected 6699 prefix: %x", packet[:4])
	}
	if got := binary.BigEndian.Uint32(packet[6:10]); got != 7 {
		t.Fatalf("sequence = %d, want 7", got)
	}
	if got := binary.BigEndian.Uint32(packet[10:14]); got != CmdReqDevInfo {
		t.Fatalf("command = %d, want %d", got, CmdReqDevInfo)
	}
	if got := binary.BigEndian.Uint32(packet[14:18]); int(got)+tuya6699HeaderSize+tuya6699FooterSize != len(packet) {
		t.Fatalf("invalid 6699 length field %d for packet of %d bytes", got, len(packet))
	}
	if !bytes.Equal(packet[len(packet)-4:], []byte{0x00, 0x00, 0x99, 0x66}) {
		t.Fatalf("unexpected 6699 footer: %x", packet[len(packet)-4:])
	}

	block, err := aes.NewCipher(tuyaDiscoveryKey[:])
	if err != nil {
		t.Fatal(err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatal(err)
	}
	nonce := packet[tuya6699HeaderSize : tuya6699HeaderSize+tuyaGCMNonceSize]
	sealed := packet[tuya6699HeaderSize+tuyaGCMNonceSize : len(packet)-tuya6699FooterSize]
	decoded, err := aead.Open(nil, nonce, sealed, packet[4:tuya6699HeaderSize])
	if err != nil {
		t.Fatalf("authenticate v3.5 discovery request: %v", err)
	}
	var request struct {
		From string `json:"from"`
		IP   string `json:"ip"`
	}
	if err := json.Unmarshal(decoded, &request); err != nil {
		t.Fatalf("decode solicitation JSON: %v", err)
	}
	if request.From != "app" || request.IP != localIP {
		t.Fatalf("unexpected solicitation: %+v", request)
	}
}

func TestDiscoveryListenerIncludesV35Port(t *testing.T) {
	listener := NewDiscoveryListener(nil)
	found := false
	for _, port := range listener.ports {
		if port == 7000 {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("discovery listener must bind UDP 7000 for v3.5 unicast replies")
	}
}

func make55AADiscoveryTestFrame(payload []byte, withReturnCode bool) []byte {
	frame := make([]byte, 16)
	copy(frame[:4], []byte{0x00, 0x00, 0x55, 0xaa})
	binary.BigEndian.PutUint32(frame[4:8], 1)
	binary.BigEndian.PutUint32(frame[8:12], CmdActive)
	body := payload
	if withReturnCode {
		body = append([]byte{0, 0, 0, 0}, payload...)
	}
	binary.BigEndian.PutUint32(frame[12:16], uint32(len(body)+8))
	frame = append(frame, body...)
	crc := crc32.ChecksumIEEE(frame)
	crcBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(crcBytes, crc)
	frame = append(frame, crcBytes...)
	frame = append(frame, 0x00, 0x00, 0xaa, 0x55)
	return frame
}

func make6699DiscoveryTestFrame(t *testing.T, plaintext []byte) []byte {
	t.Helper()
	block, err := aes.NewCipher(tuyaDiscoveryKey[:])
	if err != nil {
		t.Fatal(err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatal(err)
	}
	nonce := bytes.Repeat([]byte{0x42}, tuyaGCMNonceSize)
	header := make([]byte, tuya6699HeaderSize)
	copy(header[:4], []byte{0x00, 0x00, 0x66, 0x99})
	binary.BigEndian.PutUint32(header[6:10], 9)
	binary.BigEndian.PutUint32(header[10:14], CmdUDP)
	binary.BigEndian.PutUint32(header[14:18], uint32(tuyaGCMNonceSize+len(plaintext)+tuyaGCMTagSize))
	sealed := aead.Seal(nil, nonce, plaintext, header[4:])
	frame := append([]byte{}, header...)
	frame = append(frame, nonce...)
	frame = append(frame, sealed...)
	frame = append(frame, 0x00, 0x00, 0x99, 0x66)
	return frame
}

func TestDecodeTuyaDiscoveryPacket_RejectsTruncated55AAWithoutPanic(t *testing.T) {
	frame := make([]byte, 24)
	copy(frame[:4], []byte{0x00, 0x00, 0x55, 0xaa})
	binary.BigEndian.PutUint32(frame[12:16], 8)
	copy(frame[20:24], []byte{0x00, 0x00, 0xaa, 0x55})

	if _, err := decodeTuyaDiscoveryPacket(frame); err == nil {
		t.Fatal("expected truncated 55AA discovery frame to be rejected")
	}
}

func TestDecodeTuyaDiscoveryPacket_RawEncryptedJSON(t *testing.T) {
	rawJSON := []byte(`{"gwId":"device_raw_udp","version":"3.3"}`)
	encrypted, err := encryptAES_ECB(rawJSON, tuyaDiscoveryKey[:])
	if err != nil {
		t.Fatalf("encrypt raw discovery payload: %v", err)
	}

	dev, err := decodeTuyaDiscoveryPacket(encrypted)
	if err != nil {
		t.Fatalf("decode raw encrypted discovery payload: %v", err)
	}
	if dev.DeviceID != "device_raw_udp" {
		t.Fatalf("unexpected decoded device: %+v", dev)
	}
}
